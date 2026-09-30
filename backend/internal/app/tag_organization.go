package app

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"curated-backend/internal/agent/core"
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
	ids := []string{}
	switch req.Scope {
	case "all":
		for offset := 0; ; {
			page, err := a.store.ListMovies(ctx, contracts.ListMoviesRequest{Limit: 100, Offset: offset})
			if err != nil {
				return contracts.TagOrganizationJobDTO{}, err
			}
			for _, m := range page.Items {
				ids = append(ids, m.ID)
			}
			offset += len(page.Items)
			if offset >= page.Total || len(page.Items) == 0 {
				break
			}
		}
	case "selected":
		if len(req.MovieIDs) == 0 || len(req.MovieIDs) > 600 {
			return contracts.TagOrganizationJobDTO{}, &core.ToolError{Code: "BAD_REQUEST", Message: "select 1 to 600 movies"}
		}
		ids = req.MovieIDs
	default:
		return contracts.TagOrganizationJobDTO{}, &core.ToolError{Code: "BAD_REQUEST", Message: "invalid scope"}
	}
	if len(ids) == 0 {
		return contracts.TagOrganizationJobDTO{}, &core.ToolError{Code: "BAD_REQUEST", Message: "no movies to organize"}
	}
	id, err := a.store.CreateTagOrganization(ctx, newAgentID("tag-org-"), req.RequestID, "manual", ids)
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

// topicComplete 复用配置、代理与用量统计；每次请求超时有界。
func (a *App) topicComplete(ctx context.Context, prompt string, data any) (raw string, retErr error) {
	if err := a.aiPermission(true); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	ctx, observation, finish := a.beginAIRun(ctx, "action", "organize_user_tags")
	defer func() { finish(retErr) }()
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
	messages := []llm.ChatMessage{{Role: "system", Content: prompt}, {Role: "user", Content: "<source>" + string(encoded) + "</source>"}}
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
			return core.Result{OK: err == nil}, err
		}})
	return a.ensureAgentGateway().WithRegistry(reg)
}

// runTagOrganization 执行固定范围并在每片提交后保存检查点。
func (a *App) runTagOrganization(ctx context.Context, id string) {
	defer a.publishTagOrganization(id)
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
	if err == nil && len(defs) == 0 {
		defs, err = a.buildTopicVocabulary(ctx, id)
	}
	if err != nil {
		if ctx.Err() == nil {
			_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "vocabulary", "AI_VOCABULARY_FAILED")
		}
		return
	}
	if err = a.aiPermission(true); err != nil {
		_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "vocabulary", "AI_PERMISSION_REQUIRED")
		return
	}
	job, _ := a.store.GetTagOrganization(ctx, id)
	if job.Status == "cancelled" {
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
	_ = a.store.SetTagOrganizationVocabulary(ctx, id, defs)
	gw := a.newTopicApplyGateway()
	for ctx.Err() == nil {
		job, err = a.store.GetTagOrganization(ctx, id)
		if err != nil || job.Status == "cancelled" {
			return
		}
		if err = a.aiPermission(true); err != nil {
			_ = a.store.UpdateTagOrganization(ctx, id, "blocked", "classifying", "AI_PERMISSION_REQUIRED")
			return
		}
		items, err := a.store.TagOrganizationItems(ctx, id, true, 5, 0)
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
				_ = a.store.SetTagOrganizationItem(ctx, id, item.MovieID, "failed", "MOVIE_UNAVAILABLE")
				continue
			}
			if len(input.Title)+len(input.Summary) > 24000 {
				_ = a.store.SetTagOrganizationItem(ctx, id, item.MovieID, "unresolved", "SOURCE_TOO_LONG")
				continue
			}
			inputs = append(inputs, input)
		}
		if len(inputs) == 0 {
			continue
		}
		_ = a.store.UpdateTagOrganization(ctx, id, "running", "classifying", "")
		var result struct {
			Movies []topicClassification `json:"movies"`
		}
		raw, e := a.topicComplete(ctx, topicClassificationPrompt, map[string]any{"vocabulary": defs, "movies": inputs})
		if e == nil {
			e = decodeTopicJSON(raw, &result)
		}
		var matches map[string][]string
		if e == nil {
			matches, e = validateTopicClassification(result.Movies, inputs, defs)
		}
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
			for _, input := range inputs {
				_ = a.store.SetTagOrganizationItem(ctx, id, input.MovieID, "failed", "AI_CLASSIFICATION_FAILED")
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
			for _, decision := range result.Movies {
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
				_ = a.store.SetTagOrganizationItem(ctx, id, input.MovieID, "unresolved", "INSUFFICIENT_EVIDENCE")
				continue
			}
			args, _ := json.Marshal(map[string]any{"movieId": input.MovieID, "fingerprint": input.Fingerprint, "revision": input.Revision, "names": matches[input.MovieID]})
			for {
				a.agentRT.applyMu.Lock()
				result := gw.Invoke(core.WithUserTagTaskGrant(ctx, id), core.Call{Name: core.UserTagTaskApplyName, SessionID: id, Channel: core.ChannelAction, Args: args})
				a.agentRT.applyMu.Unlock()
				if result.Error != nil && result.Error.Code == "AI_RATE_LIMITED" {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
					}
					job, _ = a.store.GetTagOrganization(ctx, id)
					if job.Status == "cancelled" {
						return
					}
					continue
				}
				if result.Error != nil {
					_ = a.store.SetTagOrganizationItem(ctx, id, input.MovieID, "conflict", "TAG_WRITE_REJECTED")
				}
				break
			}
		}
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

// buildTopicVocabulary 分页归纳词汇，以有界资料批次覆盖全部选中影片。
func (a *App) buildTopicVocabulary(ctx context.Context, id string) ([]storage.TopicDefinition, error) {
	defs, err := a.store.TopicVocabulary(ctx)
	if err != nil {
		return nil, err
	}
	for offset := 0; ; offset += 20 {
		job, err := a.store.GetTagOrganization(ctx, id)
		if err != nil {
			return nil, err
		}
		if job.Status == "cancelled" {
			return nil, context.Canceled
		}
		items, err := a.store.TagOrganizationItems(ctx, id, false, 20, offset)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		inputs := []storage.TopicMovieInput{}
		for _, item := range items {
			v, e := a.store.TopicMovieInput(ctx, item.MovieID)
			if e != nil {
				continue
			}
			if len(v.Title)+len(v.Summary) > 8000 {
				v.Summary = ""
			}
			inputs = append(inputs, v)
		}
		raw, err := a.topicComplete(ctx, topicVocabularyPrompt, map[string]any{"existing": defs, "movies": inputs})
		if err != nil {
			return nil, err
		}
		var result struct {
			Topics []storage.TopicDefinition `json:"topics"`
		}
		if err = decodeTopicJSON(raw, &result); err != nil {
			return nil, err
		}
		if err = validateTopicVocabulary(result.Topics); err != nil {
			return nil, err
		}
		known := map[string]bool{}
		for _, d := range defs {
			known[strings.ToLower(d.Name)] = true
		}
		for _, d := range result.Topics {
			if !known[strings.ToLower(d.Name)] {
				defs = append(defs, d)
				known[strings.ToLower(d.Name)] = true
			}
		}
		if err := validateTopicVocabulary(defs); err != nil {
			return nil, err
		}
		a.publishTagOrganization(id)
	}
	return defs, nil
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
