package run

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

const maxRecentMessages = 24

type Emitter func(contracts.AIChatSSEEvent)

type Loop struct {
	gateway  *core.Gateway
	streamer llm.Streamer
	sanitize string
	locale   string
}

// NewLoop 构造使用当前网关和隐私策略的单轮执行器。
func NewLoop(gateway *core.Gateway, streamer llm.Streamer, sanitize, locale string) *Loop {
	if sanitize == "" {
		sanitize = core.SanitizeFull
	}
	return &Loop{gateway: gateway, streamer: streamer, sanitize: sanitize, locale: locale}
}

// Run 执行请求隔离的查询与发布流程，所有模型正文必须先经过展示校验。
func (l *Loop) Run(ctx context.Context, sessionID, messageID string, history []llm.ChatMessage, page *contracts.AIChatContext, emit Emitter) error {
	if l == nil || l.gateway == nil || l.streamer == nil {
		return fmt.Errorf("agent loop is not configured")
	}
	if emit == nil {
		emit = func(contracts.AIChatSSEEvent) {}
	}
	refs := core.NewAnswerRefStore()
	ctx = core.WithAnswerRefs(ctx, refs)
	scope := refs.Scope()
	// 清理本次请求的临时锚点与预算，不清理同会话的其他请求。
	defer func() {
		l.gateway.ResetSessionSteps(scope)
		l.gateway.ResetMovieRefs(scope)
		l.gateway.ResetBookRefs(scope)
		l.gateway.ResetActorRefs(scope)
		l.gateway.ResetSourceURLs(scope)
	}()
	page = core.MoviePageContext(page)
	seedTurnEntities(l.gateway, scope, page)
	seq := 0
	// 为本次流式响应分配单调事件序号。
	nextSeq := func() int {
		seq++
		return seq
	}
	emit(contracts.AIChatSSEEvent{Type: "message_start", SessionID: sessionID, MessageID: messageID, Seq: nextSeq()})

	messages := buildMessages(history, page, l.locale)
	tools := l.toolSpecs()
	steps := 0
	stepLimit := l.gateway.StepLimit()
	hadFailure := false
	hadTruncation := false
	needsInput := false
	repairs := 0
	var answerEvidence *contracts.AIAnswerEvidenceDTO
	userText := ""
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			userText = history[i].Content
			break
		}
	}
	refs.SetQuery(userText)
	// 仅发布已经校验的正文或服务端生成的固定提示。
	emitText := func(text string) {
		if text != "" {
			emit(contracts.AIChatSSEEvent{Type: "text_delta", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(), Delta: text})
		}
	}
	// 将最终状态及实际展示的来源快照一同持久化。
	emitDone := func(status, reason string, retryable bool, reasonCode string) {
		emit(contracts.AIChatSSEEvent{
			Type: "message_done", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
			Outcome:        &contracts.AIChatOutcomeDTO{Status: status, Reason: reason, Retryable: retryable, ReasonCode: reasonCode},
			AnswerEvidence: answerEvidence,
		})
	}
	// 最多允许一次修正，之后以固定说明结束且不发布被拒绝草稿。
	reject := func() bool {
		if repairs > 0 || (stepLimit > 0 && steps >= stepLimit) {
			emitText(answerCopy(l.locale, "I could not verify the proposed movie answer. Retrieved source records remain available; please narrow the request or select a movie.", "未能核实拟回答的作品信息。已查到的来源记录仍可查看，请缩小范围或选择具体作品。", "回答に含まれる作品情報を確認できませんでした。取得済みの記録を確認し、条件を絞るか作品を選択してください。"))
			emitDone("partial", "Unverified draft was not published.", false, "answer_rejected")
			return false
		}
		repairs++
		messages = append(messages, llm.ChatMessage{Role: "system", Content: "Your draft was not published. You have one correction attempt. For movie facts call submit_answer ALONE with current answerRefs and available field keys. Do not repeat codes/titles in prose. If no evidence exists, explain the retrieval limitation without naming invented works. Do not perform more retrieval or writes."})
		return true
	}

	for {
		if ctx.Err() != nil {
			emitDone("cancelled", "The user cancelled this request before it finished.", false, "")
			return nil
		}
		if stepLimit > 0 && steps >= stepLimit {
			// End before another model call: the last batch may contain unexecuted
			// calls, which must not be sent back as an incomplete tool conversation.
			emitDone("partial", fmt.Sprintf("The configured limit of %d tool calls was reached. Completed results are preserved; the task is not finished.", stepLimit), true, "tool_step_limit")
			return nil
		}
		availableTools := tools
		choice := ""
		if repairs > 0 || (stepLimit > 0 && steps == stepLimit-1 && len(refs.All()) > 0) {
			availableTools = nil
			for _, spec := range tools {
				if spec.Name == core.SubmitAnswerName {
					availableTools = append(availableTools, spec)
				}
			}
			if len(availableTools) == 0 {
				choice = "none"
			}
		}
		if estimatedRequestTokens(messages, availableTools) > requestTokenEstimateBudget {
			status := "needs_input"
			if steps > 0 {
				status = "partial"
			}
			emitDone(status, "The context budget was reached. Narrow the request or start a new conversation; completed results remain available.", false, "")
			return nil
		}
		// Never publish raw thinking or partial prose, including on cancellation.
		emit(contracts.AIChatSSEEvent{Type: "answer_progress", SessionID: sessionID, MessageID: messageID, Seq: nextSeq()})
		modelCtx, cancelModel := context.WithTimeout(ctx, 2*time.Minute)
		turn, err := l.streamer.StreamTurn(modelCtx, llm.TurnRequest{Messages: messages, Tools: availableTools, ToolChoice: choice, MaxOutputBytes: 256 * 1024}, nil)
		cancelModel()
		if ctx.Err() != nil {
			emitDone("cancelled", "The user cancelled this request before it finished.", false, "")
			return nil
		}
		if err != nil {
			if ctx.Err() != nil {
				emitDone("cancelled", "The user cancelled this request before it finished.", false, "")
				return nil
			}
			emitDone("failed", "The model response could not be completed.", true, "")
			return fmt.Errorf("model response could not be completed")
		}
		if len(turn.ToolCalls) == 0 {
			if proseNeedsReferences(turn.Content, refs) {
				if reject() {
					continue
				}
				return nil
			}
			emitText(turn.Content)
			if strings.TrimSpace(turn.Content) == "" {
				emitDone("failed", "The model returned no answer.", true, "")
			} else if needsInput {
				emitDone("needs_input", "A local entity needs the user's selection or a more specific name.", false, "")
			} else if hadFailure || hadTruncation {
				reason := "Some requested evidence could not be fully retrieved."
				if hadFailure {
					reason = "One or more retrievals failed; the answer contains only confirmed results."
				}
				emitDone("partial", reason, hadFailure, "")
			} else {
				emitDone("completed", "", false, "")
			}
			return nil
		}

		invalidBatch := false
		for _, call := range turn.ToolCalls {
			if (call.Name() == core.SubmitAnswerName && len(turn.ToolCalls) != 1) || (repairs > 0 && call.Name() != core.SubmitAnswerName) {
				invalidBatch = true
			}
		}
		if invalidBatch {
			if reject() {
				continue
			}
			return nil
		}
		for i := range turn.ToolCalls {
			if turn.ToolCalls[i].ID == "" {
				turn.ToolCalls[i].ID = fmt.Sprintf("call_%d_%d", steps, i)
			}
		}
		assistant := llm.ChatMessage{Role: "assistant", ToolCalls: turn.ToolCalls}
		messages = append(messages, assistant)
		for _, call := range turn.ToolCalls {
			steps++
			toolCallID := call.ID
			if toolCallID == "" {
				toolCallID = fmt.Sprintf("call_%d", steps)
			}
			emit(contracts.AIChatSSEEvent{
				Type: "tool_call_started", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
				ToolCallID: toolCallID, Name: call.Name(),
			})
			token, args := extractConfirmToken(nonzeroJSON(call.Args()))
			result := core.Result{}
			def, _ := l.gateway.Registry().Get(call.Name())
			if def.Permission == core.PermissionWritePreview && writeHasUnsupportedCodes(args, refs, userText) {
				result = rejectedAnswerResult()
			} else {
				result = l.gateway.Invoke(ctx, core.Call{
					Name:       call.Name(),
					Args:       args,
					SessionID:  sessionID,
					Channel:    core.ChannelChat,
					ConfirmTok: token,
					Sanitize:   l.sanitize,
				})
			}
			if result.OK {
				if isBookLibraryTool(call.Name()) {
					l.gateway.RememberBookRefs(scope, core.ExtractBookRefs(result))
				} else {
					l.gateway.RememberMovieRefs(scope, core.ExtractMovieRefs(result))
					l.gateway.RememberActorNames(scope, core.ExtractActorNames(result))
					l.gateway.RememberSourceURLs(scope, core.ExtractSourceURLs(result))
				}
			}
			if !result.OK {
				hadFailure = true
			}
			if result.Truncated {
				hadTruncation = true
			}
			resolution := entityResolutionFromResult(call.Name(), result)
			if resolution != nil {
				if resolution.Status == "matched" {
					for _, candidate := range resolution.Candidates {
						if candidate.MovieID != "" {
							l.gateway.RememberMovieRefs(scope, []core.MovieRef{{ID: candidate.MovieID, Title: candidate.Title, Code: candidate.Code}})
						}
						if candidate.ActorName != "" {
							l.gateway.RememberActorNames(scope, []string{candidate.ActorName})
						}
					}
				} else {
					needsInput = true
				}
			}
			summary := "Retrieved records are available."
			if !result.OK {
				summary = "The tool could not complete this request."
			}
			if result.ConfirmToken != "" {
				summary = "A draft is waiting for confirmation."
			}
			ok := result.OK
			movies := presentMovieCards(call.Name(), result)
			books := presentBookCards(call.Name(), result)
			if len(movies) > 0 {
				publication := &core.AnswerSubmission{}
				for _, movie := range movies {
					if ref, ok := refs.LocalMovie(movie.MovieID); ok {
						publication.Items = append(publication.Items, core.AnswerItem{Ref: ref})
					}
				}
				_, movies, answerEvidence = renderAnswer(publication, l.locale)
			}
			providerRows := providerTitleRows(call.Name(), result)
			toolEvidence := evidenceForTool(call.Name(), result)
			if call.Name() == core.SubmitAnswerName {
				toolEvidence = nil
			} // May contain multiple sources; use the final per-record snapshot.
			emit(contracts.AIChatSSEEvent{
				Type: "tool_call_result", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
				ToolCallID: toolCallID, Name: call.Name(), OK: &ok, Summary: summary, Truncated: result.Truncated,
				Movies: movies, Books: books, ProviderRows: providerRows, Resolution: resolution, Evidence: toolEvidence,
			})
			if len(movies) > 0 {
				emit(contracts.AIChatSSEEvent{
					Type: "movie_cards", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
					ToolCallID: toolCallID, Name: call.Name(), Movies: movies,
				})
			}
			if len(books) > 0 {
				emit(contracts.AIChatSSEEvent{
					Type: "book_cards", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
					ToolCallID: toolCallID, Name: call.Name(), Books: books,
				})
			}
			if result.Answer != nil && result.OK && call.Name() == core.SubmitAnswerName {
				text, cards, snapshot := renderAnswer(result.Answer, l.locale)
				answerEvidence = snapshot
				emitText(text)
				if len(cards) > 0 {
					emit(contracts.AIChatSSEEvent{Type: "movie_cards", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(), Name: call.Name(), ToolCallID: toolCallID, Movies: cards})
				}
				status := "completed"
				if result.Answer.Query != "" {
					status = "needs_input"
				}
				if hadFailure || hadTruncation {
					status = "partial"
				}
				if needsInput {
					status = "needs_input"
				}
				emitDone(status, "", false, "")
				return nil
			}
			if result.ConfirmToken != "" {
				confirmArgs := result.ConfirmArgs
				if len(confirmArgs) == 0 {
					confirmArgs = args
				}
				emit(contracts.AIChatSSEEvent{
					Type: "confirm_required", SessionID: sessionID, MessageID: messageID, Seq: nextSeq(),
					ToolCallID: toolCallID, Name: call.Name(),
					ConfirmToken: result.ConfirmToken, ExpiresAt: result.ExpiresAt,
					Changes:   confirmChanges(result.Changes),
					Arguments: confirmArgs,
					Summary:   summary,
					OK:        &ok,
				})
				emitDone("needs_confirmation", "A write preview is ready. Confirm it to save the changes.", false, "confirmation_required")
				return nil
			}
			payload, _ := json.Marshal(result)
			messages = append(messages, llm.ChatMessage{
				Role:       "tool",
				ToolCallID: toolCallID,
				Content:    wrapToolContent(payload),
			})
			if call.Name() == core.SubmitAnswerName || (result.Error != nil && result.Error.Code == "AI_ANSWER_REJECTED") {
				if len(turn.ToolCalls) > 1 {
					emitDone("partial", "A proposed answer could not be verified.", false, "answer_rejected")
					return nil
				}
				if !reject() {
					return nil
				}
				break
			}
			if stepLimit > 0 && steps >= stepLimit {
				break
			}
		}
	}
}

func providerTitleRows(name string, result core.Result) []contracts.AIAgentProviderTitleDTO {
	if name != core.SearchProviderTitlesName || !result.OK || result.Data == nil {
		return nil
	}
	raw, err := json.Marshal(result.Data)
	if err != nil {
		return nil
	}
	var envelope struct {
		Source struct {
			Items []contracts.AIAgentProviderTitleDTO `json:"items"`
		} `json:"source"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil
	}
	return envelope.Source.Items
}

func entityResolutionFromResult(name string, result core.Result) *contracts.AIEntityResolutionDTO {
	if name != "resolve_entities" || !result.OK || result.Data == nil {
		return nil
	}
	raw, err := json.Marshal(result.Data)
	if err != nil {
		return nil
	}
	var envelope struct {
		Source contracts.AIEntityResolutionDTO `json:"source"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Source.Status == "" {
		return nil
	}
	return &envelope.Source
}

func evidenceForTool(name string, result core.Result) *contracts.AIEvidenceDTO {
	evidence := &contracts.AIEvidenceDTO{
		Source:      evidenceSource(name),
		RetrievedAt: time.Now().UTC().Format(time.RFC3339),
		Truncated:   result.Truncated,
		NextCursor:  result.NextCursor,
	}
	if result.Error != nil {
		evidence.Failed = true
		evidence.ErrorCode = result.Error.Code
	}
	if result.Data != nil {
		evidence.Filters = evidenceFilters(result.Data)
	}
	return evidence
}

func evidenceSource(name string) string {
	switch name {
	case core.SearchProviderTitlesName:
		return "provider"
	case core.GetSourcePageName:
		return "source_page"
	default:
		return "local"
	}
}

func evidenceFilters(data any) map[string]string {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	var envelope struct {
		Source map[string]any `json:"source"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Source == nil {
		return nil
	}
	query, ok := envelope.Source["query"].(map[string]any)
	if !ok || len(query) == 0 {
		return nil
	}
	out := make(map[string]string, len(query))
	for key, value := range query {
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				out[key] = strings.TrimSpace(v)
			}
		case float64:
			out[key] = fmt.Sprintf("%v", v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (l *Loop) toolSpecs() []llm.ToolSpec {
	defs := l.gateway.Registry().ListByPermission(map[string]bool{
		core.PermissionRead:         true,
		core.PermissionWritePreview: true,
	})
	out := make([]llm.ToolSpec, 0, len(defs))
	for _, def := range defs {
		out = append(out, llm.ToolSpec{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.ParamsSchema.JSONSchemaMap(),
		})
	}
	return out
}

func buildMessages(history []llm.ChatMessage, page *contracts.AIChatContext, locale string) []llm.ChatMessage {
	trimmed, omitted := boundedHistory(history)
	out := make([]llm.ChatMessage, 0, len(trimmed)+1)
	out = append(out, llm.ChatMessage{Role: "system", Content: prompts.SystemPrompt(locale, page)})
	if omitted {
		out[0].Content += "\nEarlier conversation messages were omitted to fit the context budget. Do not assume missing facts or permissions; ask for clarification when needed."
	}
	out = append(out, trimmed...)
	return out
}

func nonzeroJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	return raw
}

func extractConfirmToken(raw string) (string, json.RawMessage) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", json.RawMessage(`{}`)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil || obj == nil {
		return "", json.RawMessage(trimmed)
	}
	token := ""
	if v, ok := obj["confirmToken"].(string); ok {
		token = strings.TrimSpace(v)
		delete(obj, "confirmToken")
	}
	encoded, err := json.Marshal(obj)
	if err != nil {
		return token, json.RawMessage(trimmed)
	}
	return token, encoded
}

func wrapToolContent(payload []byte) string {
	return "<source>\n" + string(payload) + "\n</source>"
}

func toolSummary(name string, result core.Result) string {
	if result.Error != nil {
		return result.Error.Message
	}
	if !result.OK {
		return "failed"
	}
	raw, err := json.Marshal(result.Data)
	if err != nil {
		return name + " ok"
	}
	text := string(raw)
	if utf8.RuneCountInString(text) > 180 {
		runes := []rune(text)
		text = string(runes[:180]) + "…"
	}
	if result.Truncated {
		return name + " truncated: " + text
	}
	return name + ": " + text
}

func presentMovieCards(name string, result core.Result) []contracts.AIAgentMovieCardDTO {
	if name != core.PresentMoviesName || !result.OK {
		return nil
	}
	refs := core.ExtractMovieRefs(result)
	if len(refs) == 0 {
		return nil
	}
	out := make([]contracts.AIAgentMovieCardDTO, 0, len(refs))
	for _, ref := range refs {
		out = append(out, contracts.AIAgentMovieCardDTO{
			MovieID:  ref.ID,
			Title:    ref.Title,
			Code:     ref.Code,
			Actors:   ref.Actors,
			CoverURL: ref.CoverURL,
			ThumbURL: ref.ThumbURL,
			Reason:   ref.Reason,
		})
	}
	return out
}

func presentBookCards(name string, result core.Result) []contracts.AIAgentBookCardDTO {
	if (name != core.PresentComicsName && name != core.PresentPhotosName) || !result.OK {
		return nil
	}
	refs := core.ExtractBookRefs(result)
	if len(refs) == 0 {
		return nil
	}
	out := make([]contracts.AIAgentBookCardDTO, 0, len(refs))
	for _, ref := range refs {
		card := contracts.AIAgentBookCardDTO{
			Kind:     ref.Kind,
			Title:    ref.Title,
			CoverURL: ref.CoverURL,
			Tags:     ref.Tags,
		}
		if ref.Kind == "comic" {
			card.ComicID = ref.ID
		} else {
			card.PhotoID = ref.ID
		}
		out = append(out, card)
	}
	return out
}

func isBookLibraryTool(name string) bool {
	switch name {
	case "search_comics", "get_comic_detail", core.PresentComicsName, core.SaveComicCommentName,
		"search_photos", "get_photo_detail", core.PresentPhotosName, core.SavePhotoCommentName,
		core.UpdateComicTitleName, core.UpdatePhotoTitleName:
		return true
	default:
		return false
	}
}

func confirmChanges(changes []core.Change) []contracts.AIConfirmChangeDTO {
	if len(changes) == 0 {
		return nil
	}
	out := make([]contracts.AIConfirmChangeDTO, 0, len(changes))
	for _, change := range changes {
		out = append(out, contracts.AIConfirmChangeDTO{
			Path:   change.Path,
			Before: change.Before,
			After:  change.After,
		})
	}
	return out
}

func seedTurnEntities(gateway *core.Gateway, sessionID string, page *contracts.AIChatContext) {
	page = core.MoviePageContext(page)
	if gateway == nil || page == nil {
		return
	}
	if id := strings.TrimSpace(page.MovieID); id != "" {
		gateway.RememberMovieRefs(sessionID, []core.MovieRef{{ID: id}})
	}
	if name := strings.TrimSpace(page.ActorName); name != "" {
		gateway.RememberActorNames(sessionID, []string{name})
	}
	for _, rawID := range page.SelectedMovieIDs {
		if id := strings.TrimSpace(rawID); id != "" {
			gateway.RememberMovieRefs(sessionID, []core.MovieRef{{ID: id}})
		}
	}
	gateway.RememberActorNames(sessionID, page.SelectedActors)
	if id := strings.TrimSpace(page.ComicID); id != "" {
		gateway.RememberBookRefs(sessionID, []core.BookRef{{Kind: "comic", ID: id}})
	}
	if id := strings.TrimSpace(page.PhotoID); id != "" {
		gateway.RememberBookRefs(sessionID, []core.BookRef{{Kind: "photo", ID: id}})
	}
	for _, rawID := range page.SelectedComicIDs {
		if id := strings.TrimSpace(rawID); id != "" {
			gateway.RememberBookRefs(sessionID, []core.BookRef{{Kind: "comic", ID: id}})
		}
	}
	for _, rawID := range page.SelectedPhotoIDs {
		if id := strings.TrimSpace(rawID); id != "" {
			gateway.RememberBookRefs(sessionID, []core.BookRef{{Kind: "photo", ID: id}})
		}
	}
	for _, mention := range page.Mentions {
		kind := strings.ToLower(strings.TrimSpace(mention.Kind))
		switch kind {
		case "movie":
			if id := strings.TrimSpace(mention.ID); id != "" {
				gateway.RememberMovieRefs(sessionID, []core.MovieRef{{ID: id}})
			}
		case "actor":
			names := make([]string, 0, 2)
			if label := strings.TrimSpace(mention.Label); label != "" {
				names = append(names, label)
			}
			if id := strings.TrimSpace(mention.ID); id != "" {
				names = append(names, id)
			}
			gateway.RememberActorNames(sessionID, names)
		case "comic":
			if id := strings.TrimSpace(mention.ID); id != "" {
				gateway.RememberBookRefs(sessionID, []core.BookRef{{Kind: "comic", ID: id, Title: strings.TrimSpace(mention.Label)}})
			}
		case "photo":
			if id := strings.TrimSpace(mention.ID); id != "" {
				gateway.RememberBookRefs(sessionID, []core.BookRef{{Kind: "photo", ID: id, Title: strings.TrimSpace(mention.Label)}})
			}
		}
	}
}
