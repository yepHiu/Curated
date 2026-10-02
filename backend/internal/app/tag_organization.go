package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/agent/prompts"
	"curated-backend/internal/agent/run"
	"curated-backend/internal/contracts"
	"curated-backend/internal/llm"
	"curated-backend/internal/storage"
)

// tagOrganizationRuntime 让任务独立于聊天连接，同时服从服务生命周期。
type tagOrganizationRuntime struct {
	mu        sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}
	wake      chan struct{}
	runningID string
	cancelJob context.CancelFunc
}

// StartTagOrganization 固定用户选择范围并发起有恢复记录的后台任务。
func (a *App) StartTagOrganization(ctx context.Context, req contracts.TagOrganizationRequest) (contracts.TagOrganizationJobDTO, error) {
	if err := a.aiPermission(true); err != nil {
		return contracts.TagOrganizationJobDTO{}, err
	}
	if _, err := normalizeAIProviderConfig(a.currentAIProviderConfig()); err != nil {
		return contracts.TagOrganizationJobDTO{}, err
	}
	if strings.TrimSpace(a.currentAIProviderConfig().Model) == "" {
		return contracts.TagOrganizationJobDTO{}, ErrAIProviderNotConfigured
	}
	if req.RequestID == "" || len(req.RequestID) > 128 {
		return contracts.TagOrganizationJobDTO{}, &core.ToolError{Code: "BAD_REQUEST", Message: "requestId required"}
	}
	locale := req.Locale
	if locale == "" || locale == "zh" {
		locale = "zh-CN"
	}
	if locale != "zh-CN" && locale != "en" && locale != "ja" {
		return contracts.TagOrganizationJobDTO{}, &core.ToolError{Code: "BAD_REQUEST", Message: "unsupported label locale"}
	}
	id := newAgentID("tag-org-")
	var err error
	switch req.Scope {
	case "all":
		id, err = a.store.CreateAllTagOrganization(ctx, id, req.RequestID, "manual", locale)
	case "unorganized", "outdated":
		id, err = a.store.CreateRemainingTagOrganization(ctx, id, req.RequestID, "manual", locale, req.Scope)
	case "selected":
		if len(req.MovieIDs) == 0 || len(req.MovieIDs) > 600 {
			return contracts.TagOrganizationJobDTO{}, &core.ToolError{Code: "BAD_REQUEST", Message: "select 1 to 600 movies"}
		}
		id, err = a.store.CreateTagOrganization(ctx, id, req.RequestID, "manual", locale, req.MovieIDs)
	default:
		return contracts.TagOrganizationJobDTO{}, &core.ToolError{Code: "BAD_REQUEST", Message: "invalid scope"}
	}
	if errors.Is(err, storage.ErrNoOrganizationMovies) {
		return contracts.TagOrganizationJobDTO{}, &core.ToolError{Code: "AI_ORGANIZATION_NO_MOVIES", Message: "no movies to organize"}
	}
	if err != nil {
		return contracts.TagOrganizationJobDTO{}, err
	}
	a.wakeTagOrganization()
	return a.store.GetTagOrganization(ctx, id)
}

// ListTagOrganizations 提供持久化任务快照，客户端重连只恢复状态。
func (a *App) ListTagOrganizations(ctx context.Context) ([]contracts.TagOrganizationJobDTO, error) {
	return a.store.ListTagOrganizations(ctx, false)
}

// TagOrganizationStats reads persisted completion without invoking a model.
func (a *App) TagOrganizationStats(ctx context.Context) (contracts.TagOrganizationStatsDTO, error) {
	return a.store.TagOrganizationStats(ctx)
}

// GetTagOrganization 返回任务进度。
func (a *App) GetTagOrganization(ctx context.Context, id string) (contracts.TagOrganizationJobDTO, error) {
	return a.store.GetTagOrganization(ctx, id)
}

// TagOrganizationItems 分页读取结果。
func (a *App) TagOrganizationItems(ctx context.Context, id string, limit, offset int) ([]contracts.TagOrganizationItemDTO, error) {
	return a.store.TagOrganizationItems(ctx, id, false, limit, offset)
}

// CancelTagOrganization 先持久化取消；应用事务必须再次核对该状态。
func (a *App) CancelTagOrganization(ctx context.Context, id string) error {
	if err := a.store.UpdateTagOrganization(ctx, id, "cancelled", "stopped", ""); err != nil {
		return err
	}
	a.tagRT.mu.Lock()
	if a.tagRT.runningID == id && a.tagRT.cancelJob != nil {
		a.tagRT.cancelJob()
	}
	a.tagRT.mu.Unlock()
	a.publishTagOrganization(id)
	return nil
}

// RetryTagOrganization 只恢复可重试项，重新校验当前权限。
func (a *App) RetryTagOrganization(ctx context.Context, id string) error {
	if err := a.aiPermission(true); err != nil {
		return err
	}
	if err := a.store.RetryTagOrganization(ctx, id); err != nil {
		return err
	}
	a.wakeTagOrganization()
	return nil
}

// UndoTagOrganization 恢复用户标签，不授予源标签修改权限。
func (a *App) UndoTagOrganization(ctx context.Context, id string) (contracts.TagOrganizationUndoDTO, error) {
	if err := a.aiPermission(true); err != nil {
		return contracts.TagOrganizationUndoDTO{}, err
	}
	return a.store.UndoTopicOrganization(ctx, id)
}

// startTagOrganizationWorker 恢复持久化任务，不依赖页面在线。
func (a *App) startTagOrganizationWorker() {
	a.tagRT.mu.Lock()
	defer a.tagRT.mu.Unlock()
	if a.tagRT.cancel != nil {
		return
	}
	parent := a.appCtx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	a.tagRT.cancel = cancel
	a.tagRT.done = make(chan struct{})
	a.tagRT.wake = make(chan struct{}, 1)
	go func() {
		// 单 worker 串行处理所有活动任务；数据库唯一索引阻止第二个活动 job。
		defer close(a.tagRT.done)
		for {
			jobs, err := a.store.ListTagOrganizations(ctx, true)
			if err == nil {
				for _, j := range jobs {
					if ctx.Err() != nil {
						return
					}
					a.tagRT.mu.Lock()
					jobCtx, cancelJob := context.WithCancel(ctx)
					a.tagRT.runningID, a.tagRT.cancelJob = j.ID, cancelJob
					a.tagRT.mu.Unlock()
					a.runTagOrganization(jobCtx, j.ID)
					cancelJob()
					a.tagRT.mu.Lock()
					a.tagRT.runningID, a.tagRT.cancelJob = "", nil
					a.tagRT.mu.Unlock()
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-a.tagRT.wake:
			}
		}
	}()
}

// wakeTagOrganization 唤醒 worker，重复唤醒合并。
func (a *App) wakeTagOrganization() {
	a.startTagOrganizationWorker()
	select {
	case a.tagRT.wake <- struct{}{}:
	default:
	}
}

// topicCompleteAttempt 复用配置、代理与用量统计；每次请求超时有界。
func (a *App) topicCompleteAttempt(ctx context.Context, prompt prompts.Definition, data any) (raw string, retErr error) {
	if err := a.aiPermission(true); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	ctx, observation, finish := a.beginAIRun(ctx, "action", "organize_user_tags")
	defer func() { finish(retErr) }()
	observation.row.PromptVersion = prompt.Version
	requestedConfig := a.currentAIProviderConfig()
	requestedPolicy := a.AIGovernanceSettings()
	cfg, err := normalizeAIProviderConfig(requestedConfig)
	if err != nil {
		return "", err
	}
	observation.row.Model = cfg.Model
	client, err := newAIHTTPClient(a.currentProxyConfig(), 0)
	if err != nil {
		return "", err
	}
	c := llm.NewClient(llm.ClientConfig{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model}, client)
	c.Observe = observation.observe
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	budget := run.BudgetForContext(cfg.ContextWindow)
	messages := []llm.ChatMessage{{Role: "system", Content: prompt.Text}, {Role: "user", Content: "<source>" + string(encoded) + "</source>"}}
	serialized, _ := json.Marshal(messages)
	if len(serialized) > budget.Input {
		return "", &core.ToolError{Code: "AI_CONTEXT_TOO_LARGE", Message: "Organization input exceeds model context budget"}
	}
	raw, retErr = c.Complete(ctx, messages, budget.Output)
	if retErr == nil && (a.currentAIProviderConfig() != requestedConfig || a.AIGovernanceSettings() != requestedPolicy) {
		return "", &core.ToolError{Code: "AI_SETTINGS_CHANGED", Message: "AI settings changed during organization"}
	}
	return raw, retErr
}

// newTopicApplyGateway 使用独立内部注册表，后台写工具不会被 chat/MCP 发现。
func (a *App) newTopicApplyGateway() *core.Gateway {
	reg := core.NewRegistry()
	_ = reg.Register(core.ToolDefinition{Name: core.UserTagTaskApplyName, Permission: core.PermissionUserTagTask, Domain: core.DomainUserWrite,
		ParamsSchema: core.Schema{Type: "object", Required: []string{"movieId", "fingerprint", "revision", "names"}, Properties: map[string]core.Schema{"movieId": {Type: "string"}, "fingerprint": {Type: "string"}, "revision": {Type: "integer"}, "names": {Type: "array", MaxItems: 12, Items: &core.Schema{Type: "string"}}}},
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			// Args 由 worker 构造；存储层还会核对真实 job 范围和版本。
			var args struct {
				MovieID     string   `json:"movieId"`
				Fingerprint string   `json:"fingerprint"`
				Revision    int64    `json:"revision"`
				Names       []string `json:"names"`
			}
			if err := json.Unmarshal(call.Args, &args); err != nil {
				return core.Result{}, err
			}
			err := a.store.ApplyMovieTopics(ctx, call.SessionID, storage.TopicMovieInput{MovieID: args.MovieID, Fingerprint: args.Fingerprint, Revision: args.Revision}, args.Names)
			if err != nil && !errors.Is(err, storage.ErrAIWriteConflict) && !errors.Is(err, storage.ErrInvalidUserTags) && !errors.Is(err, sql.ErrNoRows) {
				return core.Result{Error: &core.ToolError{Code: "CHECKPOINT_FAILED", Message: "Cannot commit organization checkpoint"}}, nil
			}
			return core.Result{OK: err == nil}, err
		}})
	return a.ensureAgentGateway().WithRegistry(reg)
}

// runTagOrganization 执行固定范围并在每片提交后保存检查点。
func (a *App) runTagOrganization(ctx context.Context, id string) {
	defer a.publishTagOrganization(id)
	// Unexpected storage/checkpoint exits must not strand an active job while the
	// worker sleeps. Shutdown keeps the active checkpoint for startup recovery.
	defer func() {
		if ctx.Err() == nil {
			_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "stopped", "CHECKPOINT_FAILED")
		}
	}()
	jobState, stateErr := a.store.GetTagOrganization(ctx, id)
	if stateErr != nil || (jobState.Status != "queued" && jobState.Status != "running") {
		return
	}
	if err := a.aiPermission(true); err != nil {
		_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "preparing", "AI_PERMISSION_REQUIRED")
		return
	}
	if err := a.store.UpdateTagOrganization(ctx, id, "running", "vocabulary", ""); err != nil {
		return
	}
	a.publishTagOrganization(id)
	defs, err := a.store.GetTagOrganizationVocabulary(ctx, id)
	if err == nil && !jobState.VocabularyReady {
		defs, err = a.buildTopicVocabulary(ctx, id)
	}
	if err != nil {
		if ctx.Err() == nil {
			_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "vocabulary", topicOrganizationErrorCode(err))
		}
		return
	}
	if err = a.aiPermission(true); err != nil {
		_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "vocabulary", "AI_PERMISSION_REQUIRED")
		return
	}
	job, err := a.store.GetTagOrganization(ctx, id)
	if err != nil || job.Status != "running" {
		return
	}
	a.agentRT.applyMu.Lock()
	err = a.aiPermission(true)
	if err == nil {
		err = a.store.SaveTopicVocabularyForJob(ctx, id, defs)
	}
	a.agentRT.applyMu.Unlock()
	if err != nil {
		_ = a.store.UpdateTagOrganization(ctx, id, "failed", "vocabulary", "AI_VOCABULARY_INVALID")
		return
	}
	if err = a.store.SetTagOrganizationVocabulary(ctx, id, defs); err != nil {
		return
	}
	gw := a.newTopicApplyGateway()
	batchSize := 5
	for ctx.Err() == nil {
		job, err = a.store.GetTagOrganization(ctx, id)
		if err != nil || job.Status != "running" {
			return
		}
		if err = a.aiPermission(true); err != nil {
			_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "classifying", "AI_PERMISSION_REQUIRED")
			return
		}
		items, err := a.store.TagOrganizationItems(ctx, id, true, batchSize, 0)
		if err != nil {
			return
		}
		if len(items) == 0 {
			break
		}
		inputs := []storage.TopicMovieInput{}
		for _, item := range items {
			input, e := a.store.TopicMovieInput(ctx, item.MovieID)
			if e != nil {
				if !errors.Is(e, sql.ErrNoRows) {
					return
				}
				if err := a.store.SetTagOrganizationItem(ctx, id, item.MovieID, "failed", "MOVIE_UNAVAILABLE"); err != nil {
					return
				}
				continue
			}
			if len(input.Title)+len(input.Summary) > 24000 {
				if err := a.store.SetTagOrganizationItem(ctx, id, item.MovieID, "unresolved", "SOURCE_TOO_LONG"); err != nil {
					return
				}
				continue
			}
			inputs = append(inputs, input)
		}
		if len(inputs) == 0 {
			continue
		}
		if err = a.store.UpdateTagOrganization(ctx, id, "running", "classifying", ""); err != nil {
			return
		}
		job.Stage = "classifying"
		decisions, matches, e := a.classifyTopicBatch(ctx, defs, inputs)
		if ctx.Err() != nil {
			return
		}
		current, stateErr := a.store.GetTagOrganization(ctx, id)
		if stateErr != nil || current.Status != "running" {
			return
		}
		if a.aiPermission(true) != nil {
			_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "classifying", "AI_PERMISSION_REQUIRED")
			return
		}
		if e != nil {
			if !splittableTopicClassification(e) {
				// A provider outage/configuration failure affects the job, not every
				// remaining movie. Preserve pending items for an explicit retry.
				_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "classifying", topicOrganizationErrorCode(e))
				return
			}
			if len(inputs) > 1 {
				batchSize = max(1, len(inputs)/2)
				continue
			}
			for _, input := range inputs {
				if err := a.store.SetTagOrganizationItem(ctx, id, input.MovieID, "failed", topicOrganizationErrorCode(e)); err != nil {
					return
				}
			}
			a.publishTagOrganization(id)
			continue
		}
		for _, input := range inputs {
			if ctx.Err() != nil {
				return
			}
			if a.aiPermission(true) != nil {
				_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "applying", "AI_PERMISSION_REQUIRED")
				return
			}
			evidence := []contracts.TopicEvidenceDTO{}
			for _, decision := range decisions {
				if decision.MovieID != input.MovieID {
					continue
				}
				for _, match := range decision.Matches {
					evidence = append(evidence, contracts.TopicEvidenceDTO{Topic: match.Topic, Field: match.Field, Quote: match.Quote})
				}
			}
			if err := a.store.SetTopicEvidence(ctx, id, input.MovieID, input.Fingerprint, evidence); err != nil {
				_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "applying", "CHECKPOINT_FAILED")
				return
			}
			if len(matches[input.MovieID]) == 0 {
				if err := a.store.CompleteUnresolvedTopic(ctx, id, input); err != nil {
					if !errors.Is(err, storage.ErrAIWriteConflict) && !errors.Is(err, sql.ErrNoRows) {
						_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "applying", "CHECKPOINT_FAILED")
						return
					}
					if err = a.store.SetTagOrganizationItem(ctx, id, input.MovieID, "conflict", "TAG_WRITE_REJECTED"); err != nil {
						return
					}
				}
				continue
			}
			args, _ := json.Marshal(map[string]any{"movieId": input.MovieID, "fingerprint": input.Fingerprint, "revision": input.Revision, "names": matches[input.MovieID]})
			for {
				a.agentRT.applyMu.Lock()
				result := gw.Invoke(core.WithUserTagTaskGrant(ctx, id), core.Call{Name: core.UserTagTaskApplyName, SessionID: id, Channel: core.ChannelAction, Args: args})
				a.agentRT.applyMu.Unlock()
				if result.Error != nil && result.Error.Code == "AI_RATE_LIMITED" {
					if job.Stage != "waiting_quota" {
						if err = a.store.UpdateTagOrganization(ctx, id, "running", "waiting_quota", ""); err != nil {
							return
						}
						a.publishTagOrganization(id)
						job.Stage = "waiting_quota"
					}
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
					}
					job, err = a.store.GetTagOrganization(ctx, id)
					if err != nil || job.Status != "running" {
						return
					}
					continue
				}
				if job.Stage == "waiting_quota" {
					if err = a.store.UpdateTagOrganization(ctx, id, "running", "applying", ""); err != nil {
						return
					}
					job.Stage = "applying"
				}
				if result.Error != nil {
					if result.Error.Code == "CHECKPOINT_FAILED" || result.Error.Code == "AI_TOOL_PERMISSION_DENIED" {
						_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "applying", result.Error.Code)
						return
					}
					if err := a.store.SetTagOrganizationItem(ctx, id, input.MovieID, "conflict", "TAG_WRITE_REJECTED"); err != nil {
						return
					}
				}
				break
			}
		}
		batchSize = 5
		a.publishTagOrganization(id)
	}
	if ctx.Err() != nil {
		return
	}
	job, err = a.store.GetTagOrganization(ctx, id)
	if err != nil {
		return
	}
	status := "completed"
	if job.Failed > 0 {
		status = "partial_failed"
	}
	_ = a.store.UpdateTagOrganization(ctx, id, status, "finished", "")
	a.publishTagOrganization(id)
}

// buildTopicVocabulary adds only new definitions and resumes completed source pages after restart.
func (a *App) buildTopicVocabulary(ctx context.Context, id string) ([]storage.TopicDefinition, error) {
	job, err := a.store.GetTagOrganization(ctx, id)
	if err != nil {
		return nil, err
	}
	defs, err := a.store.GetTagOrganizationVocabulary(ctx, id)
	if err != nil {
		return nil, err
	}
	if job.VocabularyProcessed == 0 {
		defs, err = a.store.TopicVocabulary(ctx)
		if err != nil {
			return nil, err
		}
	}
	candidates, err := a.store.TopicReuseCatalog(ctx)
	if err != nil {
		return nil, err
	}
	batchSize := 50
	for offset := job.VocabularyProcessed; offset < job.Total; {
		current, err := a.store.GetTagOrganization(ctx, id)
		if err != nil {
			return nil, err
		}
		if current.Status != "running" {
			return nil, context.Canceled
		}
		items, err := a.store.TagOrganizationItems(ctx, id, false, batchSize, offset)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, fmt.Errorf("missing vocabulary page")
		}
		inputs := []topicVocabularySample{}
		for _, item := range items {
			input, e := a.store.TopicMovieInput(ctx, item.MovieID)
			if e != nil {
				if !errors.Is(e, sql.ErrNoRows) {
					return nil, e
				}
				continue
			}
			inputs = append(inputs, compactVocabularySample(input))
		}
		if len(inputs) > 0 {
			catalog := topicCatalog(defs, candidates)
			proposed, err := a.proposeTopicVocabulary(ctx, inputs, catalog, job.Locale)
			if err != nil {
				if topicOrganizationErrorCode(err) == "AI_CONTEXT_TOO_LARGE" && len(items) > 1 {
					batchSize = max(1, len(items)/2)
					continue
				}
				return nil, err
			}
			proposals, err := a.reuseTopicProposals(ctx, proposed, catalog)
			if err != nil {
				return nil, err
			}
			known := map[string]bool{}
			for _, d := range defs {
				known[strings.ToLower(d.Name)] = true
			}
			added := []storage.TopicDefinition{}
			for _, d := range proposals {
				if !known[strings.ToLower(d.Name)] {
					// Also compare proposals from the same response, so two new
					// synonymous names cannot bypass the existing-catalog check.
					if len(added) > 0 {
						reused, err := a.reuseTopicProposals(ctx, []storage.TopicDefinition{d}, added)
						if err != nil {
							return nil, err
						}
						d = reused[0]
					}
					if known[strings.ToLower(d.Name)] {
						continue
					}
					// Reserve established topic ownership, while allowing ordinary
					// user labels to remain valid aliases for normalization.
					cleaned, err := normalizeTopicProposals([]storage.TopicDefinition{d}, defs)
					if err != nil {
						return nil, err
					}
					d = cleaned[0]
					defs = append(defs, d)
					added = append(added, d)
					known[strings.ToLower(d.Name)] = true
				}
			}
			if err = validateTopicVocabulary(defs); err != nil {
				return nil, &core.ToolError{Code: "AI_ORGANIZATION_VOCABULARY_CONFLICT", Message: "Ambiguous topic vocabulary"}
			}
		}
		offset += len(items)
		if err = a.store.CheckpointTopicVocabulary(ctx, id, defs, offset, offset >= job.Total); err != nil {
			return nil, &core.ToolError{Code: "CHECKPOINT_FAILED", Message: "Cannot save vocabulary progress"}
		}
		a.publishTagOrganization(id)
	}
	return defs, nil
}

// topicOrganizationErrorCode exposes useful failure categories without returning source text or provider bodies.
func topicOrganizationErrorCode(err error) string {
	if llm.IsContextOverflow(err) {
		return "AI_CONTEXT_TOO_LARGE"
	}
	var toolErr *core.ToolError
	if errors.As(err, &toolErr) {
		return toolErr.Code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "AI_ORGANIZATION_TIMEOUT"
	}
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "truncated") || strings.Contains(text, "reasoning only") {
		return "AI_ORGANIZATION_OUTPUT_LIMIT"
	}
	return "AI_ORGANIZATION_PROVIDER_FAILED"
}

// publishTagOrganization 将持久进度投影到既有任务事件，不制造逐片 toast。
func (a *App) publishTagOrganization(id string) {
	if a.tasks == nil {
		return
	}
	j, err := a.store.GetTagOrganization(context.Background(), id)
	if err != nil {
		return
	}
	progress := 0
	if j.Total > 0 {
		progress = j.Processed * 100 / j.Total
	}
	a.tasks.Restore(contracts.TaskDTO{TaskID: id, Type: "ai.organize-tags", Status: contracts.TaskRunning, CreatedAt: j.CreatedAt})
	a.tasks.ProgressWithMetadata(id, progress, "", map[string]any{"organization": j})
	switch j.Status {
	case "cancelled":
		a.tasks.Cancel(id, "")
	case "completed":
		a.tasks.Complete(id, "")
	case "partial_failed":
		a.tasks.PartialFail(id, "TAG_PARTIAL_FAILED", "", nil)
	case "failed", "blocked":
		a.tasks.Fail(id, j.Error, "")
	}
}
