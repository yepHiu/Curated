package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/tools"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
)

// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
type answerStreamer func(context.Context, llm.TurnRequest, func(string)) (llm.AssistantTurn, error)

// StreamTurn 把受控测试回调适配成模型流接口。
func (f answerStreamer) StreamTurn(ctx context.Context, req llm.TurnRequest, delta func(string)) (llm.AssistantTurn, error) {
	return f(ctx, req, delta)
}

// answerGateway 创建只使用合成影片的查询与发布网关。
func answerGateway(t *testing.T) *core.Gateway {
	t.Helper()
	reg := core.NewRegistry()
	q := stubQueryForLoop{movie: contracts.MovieListItemDTO{ID: "m1", Code: "TEST-101", Title: "Recorded title", Actors: []string{"Recorded actor"}, RuntimeMinutes: 85}}
	if err := tools.RegisterQueryTools(reg, q); err != nil {
		t.Fatal(err)
	}
	gw := core.NewGateway(reg, nil, nil, nil)
	if err := tools.RegisterPresentTools(reg, gw.MovieRefs()); err != nil {
		t.Fatal(err)
	}
	return gw
}

// firstAnswerRef 读取模型实际收到的工具引用，避免在测试中假造有效 ID。
func firstAnswerRef(t *testing.T, req llm.TurnRequest) string {
	t.Helper()
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role != "tool" {
			continue
		}
		s := strings.TrimSuffix(strings.TrimPrefix(req.Messages[i].Content, "<source>\n"), "\n</source>")
		var result core.Result
		if json.Unmarshal([]byte(s), &result) == nil && len(result.AnswerRefs) > 0 {
			return result.AnswerRefs[0].RefID
		}
	}
	t.Fatal("no current reference supplied to model")
	return ""
}

// TestStructuredAnswerPublishesServerFactsAndSnapshot 验证 Structured Answer Publishes Server Facts And Snapshot 的行为与失败边界，使用隔离测试数据。
func TestStructuredAnswerPublishesServerFactsAndSnapshot(t *testing.T) {
	for _, level := range []string{core.SanitizeFull, core.SanitizeMinimal} {
		// 分别验证该输入或隐私模式下的发布边界。
		t.Run(level, func(t *testing.T) {
			calls := 0
			// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
			streamer := answerStreamer(func(_ context.Context, req llm.TurnRequest, delta func(string)) (llm.AssistantTurn, error) {
				calls++
				if delta != nil || req.OnThinking != nil {
					t.Fatal("raw stream can escape")
				}
				if calls == 1 {
					return llm.AssistantTurn{Content: "FORGED-999 before query", ToolCalls: []llm.ToolCall{{ID: "read", Function: llm.ToolCallFunction{Name: "search_movies", Arguments: `{}`}}}}, nil
				}
				if level == core.SanitizeMinimal && strings.Contains(req.Messages[len(req.Messages)-1].Content, "Recorded title") {
					t.Fatal("minimal leaked title")
				}
				ref := firstAnswerRef(t, req)
				return llm.AssistantTurn{Content: "FORGED-999 after query", ToolCalls: []llm.ToolCall{{ID: "answer", Function: llm.ToolCallFunction{Name: core.SubmitAnswerName, Arguments: fmt.Sprintf(`{"items":[{"refId":%q,"fields":["code","title","actors"],"reasonFacts":["runtimeMinutes"]}]}`, ref)}}}}, nil
			})
			events := collectEvents(t, NewLoop(answerGateway(t), streamer, level, "zh-CN"), []llm.ChatMessage{{Role: "user", Content: "推荐一部"}})
			raw, _ := json.Marshal(events)
			if strings.Contains(string(raw), "FORGED-999") {
				t.Fatalf("draft leaked: %s", raw)
			}
			done := events[len(events)-1]
			if calls != 2 || done.AnswerEvidence == nil || done.AnswerEvidence.Version != 1 || len(done.AnswerEvidence.Items) != 1 {
				t.Fatal(calls, done)
			}
			if done.AnswerEvidence.Items[0].Fields["title"] != "Recorded title" {
				t.Fatal(done.AnswerEvidence)
			}
			var text string
			var cards []contracts.AIAgentMovieCardDTO
			for _, event := range events {
				if event.Type == "text_delta" {
					text += event.Delta
				}
				if event.Type == "movie_cards" {
					cards = event.Movies
				}
			}
			if !strings.Contains(text, "TEST-101") || !strings.Contains(text, "85") || !strings.Contains(text, "Recorded actor") || len(cards) != 1 || cards[0].Title != "Recorded title" {
				t.Fatal(text, cards)
			}
		})
	}
}

// TestUnsupportedProseIsNeverPublishedAndRepairIsBounded 验证 Unsupported Prose Is Never Published And Repair Is Bounded 的行为与失败边界，使用隔离测试数据。
func TestUnsupportedProseIsNeverPublishedAndRepairIsBounded(t *testing.T) {
	for _, content := range []string{"Try MADEUP-999", "ＴＥＳＴ－９９９", "FC2-PPV-9999999", "123456_789", "Try **FAKE**-999", "[a title](https://example.com/fake)", "<img src=x>", "TE\u200bST-999"} {
		// 分别验证该输入或隐私模式下的发布边界。
		t.Run(content, func(t *testing.T) {
			calls := 0
			// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
			s := answerStreamer(func(context.Context, llm.TurnRequest, func(string)) (llm.AssistantTurn, error) {
				calls++
				return llm.AssistantTurn{Content: content}, nil
			})
			events := collectEvents(t, NewLoop(answerGateway(t), s, core.SanitizeFull, "en"), []llm.ChatMessage{{Role: "user", Content: content}})
			raw, _ := json.Marshal(events)
			if strings.Contains(string(raw), content) || calls != 2 || events[len(events)-1].Outcome.ReasonCode != "answer_rejected" {
				t.Fatal(calls, string(raw))
			}
		})
	}
}

// TestAnswerCancellationAndProviderFailureDiscardDraft 验证 Answer Cancellation And Provider Failure Discard Draft 的行为与失败边界，使用隔离测试数据。
func TestAnswerCancellationAndProviderFailureDiscardDraft(t *testing.T) {
	for _, cancelRun := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
		s := answerStreamer(func(_ context.Context, _ llm.TurnRequest, delta func(string)) (llm.AssistantTurn, error) {
			if delta != nil {
				delta("FAKE-")
				delta("999")
			}
			if cancelRun {
				cancel()
			}
			return llm.AssistantTurn{Content: "FAKE-999"}, errors.New("upstream FAKE-999")
		})
		var events []contracts.AIChatSSEEvent
		// 收集用户实际可见的事件，检查草稿不会通过旁路发布。
		_ = NewLoop(answerGateway(t), s, core.SanitizeFull, "en").Run(ctx, "same", "msg", nil, nil, func(e contracts.AIChatSSEEvent) { events = append(events, e) })
		cancel()
		raw, _ := json.Marshal(events)
		if strings.Contains(string(raw), "FAKE-") {
			t.Fatal(string(raw))
		}
		want := "failed"
		if cancelRun {
			want = "cancelled"
		}
		if events[len(events)-1].Outcome.Status != want {
			t.Fatal(events)
		}
	}
}

// TestConcurrentSameSessionCannotUseOtherRunsReference 验证 Concurrent Same Session Cannot Use Other Runs Reference 的行为与失败边界，使用隔离测试数据。
func TestConcurrentSameSessionCannotUseOtherRunsReference(t *testing.T) {
	gw := answerGateway(t)
	ready := make(chan string, 1)
	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	// 独立运行生产者；上下文取消后结束，所有响应仍由接收方串行写出。
	go func() {
		defer wg.Done()
		calls := 0
		// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
		s := answerStreamer(func(_ context.Context, req llm.TurnRequest, _ func(string)) (llm.AssistantTurn, error) {
			calls++
			if calls == 1 {
				return llm.AssistantTurn{ToolCalls: []llm.ToolCall{{ID: "read", Function: llm.ToolCallFunction{Name: "search_movies", Arguments: `{}`}}}}, nil
			}
			ready <- firstAnswerRef(t, req)
			<-release
			return llm.AssistantTurn{Content: "Ready."}, nil
		})
		_ = NewLoop(gw, s, core.SanitizeFull, "en").Run(context.Background(), "same", "a", nil, nil, nil)
	}()
	ref := <-ready
	// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
	s := answerStreamer(func(context.Context, llm.TurnRequest, func(string)) (llm.AssistantTurn, error) {
		return llm.AssistantTurn{ToolCalls: []llm.ToolCall{{ID: "answer", Function: llm.ToolCallFunction{Name: core.SubmitAnswerName, Arguments: fmt.Sprintf(`{"items":[{"refId":%q}]}`, ref)}}}}, nil
	})
	var events []contracts.AIChatSSEEvent
	// 收集用户实际可见的事件，检查草稿不会通过旁路发布。
	_ = NewLoop(gw, s, core.SanitizeFull, "en").Run(context.Background(), "same", "b", nil, nil, func(e contracts.AIChatSSEEvent) { events = append(events, e) })
	close(release)
	wg.Wait()
	for _, event := range events {
		if len(event.Movies) > 0 || event.AnswerEvidence != nil {
			t.Fatal("cross-request publication", event)
		}
	}
}

// TestWriteDraftRejectsNewCodesButKeepsUserText 验证 Write Draft Rejects New Codes But Keeps User Text 的行为与失败边界，使用隔离测试数据。
func TestWriteDraftRejectsNewCodesButKeepsUserText(t *testing.T) {
	refs := core.NewAnswerRefStore()
	if writeHasUnsupportedCodes(json.RawMessage(`{"movieId":"syn-001","body":"My note"}`), refs, "save My note to the selected movie") {
		t.Fatal("local foreign key was mistaken for invented prose")
	}
	if !writeHasUnsupportedCodes(json.RawMessage(`{"body":"Try FAKE-999"}`), refs, "save a note") {
		t.Fatal("fabricated draft accepted")
	}
	if writeHasUnsupportedCodes(json.RawMessage(`{"body":"Investigate FAKE-999"}`), refs, "Write: Investigate FAKE-999") {
		t.Fatal("user-authored draft changed")
	}
}

// TestMixedFinalizationDoesNotExecuteOtherTools 验证 Mixed Finalization Does Not Execute Other Tools 的行为与失败边界，使用隔离测试数据。
func TestMixedFinalizationDoesNotExecuteOtherTools(t *testing.T) {
	gw := answerGateway(t)
	s := &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{
		{ToolCalls: []llm.ToolCall{{ID: "read", Function: llm.ToolCallFunction{Name: "search_movies", Arguments: `{}`}}, {ID: "final", Function: llm.ToolCallFunction{Name: core.SubmitAnswerName, Arguments: `{"items":[{"refId":"forged"}]}`}}}},
		{Content: "I need a verified record first."},
	}}
	events := collectEvents(t, NewLoop(gw, s, core.SanitizeFull, "en"), nil)
	for _, event := range events {
		if event.Type == "tool_call_started" {
			t.Fatal("mixed finalization executed a tool", event)
		}
	}
}

// TestSelectedIDWithoutReadCannotPublishFacts 验证 Selected IDWithout Read Cannot Publish Facts 的行为与失败边界，使用隔离测试数据。
func TestSelectedIDWithoutReadCannotPublishFacts(t *testing.T) {
	s := &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{{ToolCalls: []llm.ToolCall{{ID: "present", Function: llm.ToolCallFunction{Name: core.PresentMoviesName, Arguments: `{"items":[{"movieId":"m1","reason":"FAKE-999"}]}`}}}}, {Content: "Please retrieve the details first."}}}
	var events []contracts.AIChatSSEEvent
	// 收集用户实际可见的事件，检查草稿不会通过旁路发布。
	_ = NewLoop(answerGateway(t), s, core.SanitizeFull, "en").Run(context.Background(), "s", "m", nil, &contracts.AIChatContext{SelectedMovieIDs: []string{"m1"}}, func(e contracts.AIChatSSEEvent) { events = append(events, e) })
	for _, e := range events {
		if len(e.Movies) > 0 {
			t.Fatal("unread selection published", e)
		}
	}
}

// TestProviderAnswerKeepsIdentityAndEscapesSourceMarkup 验证 Provider Answer Keeps Identity And Escapes Source Markup 的行为与失败边界，使用隔离测试数据。
func TestProviderAnswerKeepsIdentityAndEscapesSourceMarkup(t *testing.T) {
	store := core.NewAnswerRefStore()
	hints := store.Capture(core.SearchProviderTitlesName, core.Result{OK: true, Data: map[string]any{"source": map[string]any{"items": []any{map[string]any{"code": "TEST-102", "title": "[image](https://evil.example) <script>", "provider": "Synthetic", "inLibrary": false, "homepage": "https://example.com/record"}}}}})
	ref, _ := store.Lookup(hints[0].RefID)
	text, cards, evidence := renderAnswer(&core.AnswerSubmission{Items: []core.AnswerItem{{Ref: ref, Fields: []string{"code", "title"}}}}, "en")
	if len(cards) != 0 || !strings.Contains(text, "Not in library") || strings.Contains(text, "<script>") || strings.Contains(text, "](https:") {
		t.Fatal(text, cards)
	}
	if evidence.Items[0].Source != "provider" || evidence.Items[0].Fields["homepage"] != "https://example.com/record" {
		t.Fatal(evidence)
	}
}

// TestUnverifiedQueryQuotationDoesNotCreateEvidence 验证 Unverified Query Quotation Does Not Create Evidence 的行为与失败边界，使用隔离测试数据。
func TestUnverifiedQueryQuotationDoesNotCreateEvidence(t *testing.T) {
	s := &llm.ScriptedStreamer{Turns: []llm.AssistantTurn{{ToolCalls: []llm.ToolCall{{ID: "query", Function: llm.ToolCallFunction{Name: core.SubmitAnswerName, Arguments: `{"queryRef":"user_input"}`}}}}}}
	events := collectEvents(t, NewLoop(answerGateway(t), s, core.SanitizeFull, "en"), []llm.ChatMessage{{Role: "user", Content: "Find FAKE-999"}})
	var text string
	for _, e := range events {
		if e.Type == "text_delta" {
			text += e.Delta
		}
		if e.AnswerEvidence != nil || len(e.Movies) > 0 {
			t.Fatal("query became verified", e)
		}
	}
	if !strings.Contains(text, "unverified") || !strings.Contains(text, "FAKE-999") || events[len(events)-1].Outcome.Status != "needs_input" {
		t.Fatal(events)
	}
}

// TestSingleStepLimitStopsBeforeUnverifiedFinalAnswer 验证 Single Step Limit Stops Before Unverified Final Answer 的行为与失败边界，使用隔离测试数据。
func TestSingleStepLimitStopsBeforeUnverifiedFinalAnswer(t *testing.T) {
	reg := core.NewRegistry()
	_ = tools.RegisterQueryTools(reg, stubQueryForLoop{movie: contracts.MovieListItemDTO{ID: "m1", Code: "TEST-101"}})
	// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
	gw := core.NewGateway(reg, nil, nil, func() core.Settings { return core.Settings{StepLimit: 1} })
	_ = tools.RegisterPresentTools(reg, gw.MovieRefs())
	calls := 0
	// 为当前隔离用例提供可控响应或收集结果，验证发布与取消边界。
	s := answerStreamer(func(context.Context, llm.TurnRequest, func(string)) (llm.AssistantTurn, error) {
		calls++
		return llm.AssistantTurn{ToolCalls: []llm.ToolCall{{ID: "read", Function: llm.ToolCallFunction{Name: "search_movies", Arguments: `{}`}}}}, nil
	})
	events := collectEvents(t, NewLoop(gw, s, core.SanitizeFull, "en"), nil)
	if calls != 1 || events[len(events)-1].Outcome.ReasonCode != "tool_step_limit" {
		t.Fatal(calls, events)
	}
}
