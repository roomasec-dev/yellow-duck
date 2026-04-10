package session

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"rm_ai_agent/internal/artifact"
	"rm_ai_agent/internal/compression"
	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/detailagent"
	"rm_ai_agent/internal/edr"
	"rm_ai_agent/internal/i18n"
	"rm_ai_agent/internal/knowledge"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/memory"
	"rm_ai_agent/internal/model"
	"rm_ai_agent/internal/planner"
	"rm_ai_agent/internal/progress"
	"rm_ai_agent/internal/prompt"
	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/router"
	"rm_ai_agent/internal/store"
	"rm_ai_agent/internal/textutil"
)

type Service struct {
	cfg       config.Config
	store     store.Store
	model     model.Client
	compactor *compression.Service
	progress  *progress.Service
	detailer  *detailagent.Service
	router    *router.Service
	planner   *planner.Service
	memory    *memory.Service
	artifacts *artifact.Service
	i18n      *i18n.Service
	knowledge *knowledge.Service
	prompt    *prompt.Service
	edr       edr.Client
	logger    *logx.Logger
	runMu     sync.Mutex
	runs      map[string]runState
	dedupCache *toolDedupCache
}

type runState struct {
	id     string
	cancel context.CancelFunc
}

type dedupEntry struct {
	result string
	err    error
	doneAt time.Time
	ttl    time.Duration
}

func (e *dedupEntry) isFresh() bool {
	return e.ttl > 0 && time.Since(e.doneAt) < e.ttl
}

type toolDedupCache struct {
	mu    sync.Mutex
	items map[string]*dedupEntry
	ttl   time.Duration
}

func newToolDedupCache(ttl time.Duration) *toolDedupCache {
	return &toolDedupCache{items: make(map[string]*dedupEntry), ttl: ttl}
}

func (c *toolDedupCache) GetOrSubmit(key string) (string, error, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.items[key]; ok && entry.isFresh() {
		return entry.result, entry.err, true
	}
	c.items[key] = &dedupEntry{ttl: c.ttl}
	return "", nil, false
}

func (c *toolDedupCache) Done(key string, result string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.items[key]; ok {
		entry.result = result
		entry.err = err
		entry.doneAt = time.Now()
	}
}

func NewService(cfg config.Config, store store.Store, modelClient model.Client, compactor *compression.Service, progressService *progress.Service, detailAgentService *detailagent.Service, routerService *router.Service, plannerService *planner.Service, memoryService *memory.Service, artifactService *artifact.Service, i18nService *i18n.Service, knowledgeService *knowledge.Service, promptService *prompt.Service, edrClient edr.Client, logger *logx.Logger) *Service {
	return &Service{
		cfg:       cfg,
		store:     store,
		model:     modelClient,
		compactor: compactor,
		progress:  progressService,
		detailer:  detailAgentService,
		router:    routerService,
		planner:   plannerService,
		memory:    memoryService,
		artifacts: artifactService,
		i18n:      i18nService,
		knowledge: knowledgeService,
		prompt:    promptService,
		edr:       edrClient,
		logger:    logger,
		runs:      make(map[string]runState),
		dedupCache: newToolDedupCache(30 * time.Second),
	}
}

func (s *Service) HandleInbound(ctx context.Context, msg protocol.InboundMessage, sink progress.Sink) (string, error) {
	scopeKey := buildScopeKey(msg, s.cfg.Session)
	locale := s.detectLocale(msg.Text)
	if response, ok, err := s.handleSessionCommand(ctx, scopeKey, strings.TrimSpace(msg.Text), locale); ok || err != nil {
		return response, err
	}
	if response, ok := s.handleInterrupt(scopeKey, strings.TrimSpace(msg.Text), locale); ok {
		return response, nil
	}
	sessionRef, err := s.resolveConversationSession(ctx, scopeKey, msg, locale)
	if err != nil {
		return "", err
	}
	sessionKey := sessionRef.Key
	ctx, finishRun := s.startRun(ctx, sessionKey)
	defer finishRun()
	reporter := s.progress.NewReporter(sessionRef, sink)
	reporter.Step(ctx, "收到用户新消息，正在整理会话上下文并判断该走通用问答还是 EDR 操作。")

	if err := s.store.AppendTurn(ctx, sessionKey, string(model.RoleUser), msg.Text); err != nil {
		return "", err
	}

	if response, ok, err := s.handlePendingConfirmation(ctx, sessionKey, strings.TrimSpace(msg.Text), locale, reporter); ok || err != nil {
		return response, err
	}

	if response, ok, err := s.handleEDRCommand(ctx, sessionKey, msg.Text, locale, reporter); ok || err != nil {
		return response, err
	}

	if response, ok, err := s.handlePlannedTools(ctx, sessionKey, msg.Text, locale, reporter); ok || err != nil {
		return response, err
	}

	if response, ok, err := s.handleNaturalLanguageEDR(ctx, sessionKey, msg.Text, locale, reporter); ok || err != nil {
		return response, err
	}

	if response, ok := s.guardUnsupportedEDRAction(msg.Text, locale); ok {
		return s.storeAssistantReply(ctx, sessionKey, response)
	}

	reporter.Step(ctx, "正在读取历史摘要和最近几轮消息，把上下文串起来。")

	messages, err := s.buildBaseMessages(ctx, sessionKey)
	if err != nil {
		return "", err
	}

	reporter.Step(ctx, "主模型开始思考了，我在把关键信息收束成一个直接可用的答复。")

	result, err := s.model.Chat(ctx, model.ChatRequest{
		SessionKey: sessionKey,
		Messages:   messages,
	}, nil)
	if err != nil {
		return "", fmt.Errorf("chat with model: %w", err)
	}

	result.Text = sanitizeReply(result.Text)
	if _, err := s.storeAssistantReply(ctx, sessionKey, result.Text); err != nil {
		return "", err
	}
	if err := s.compactor.MaybeCompact(ctx, sessionKey); err != nil {
		s.logger.Warn("compact session failed", "session_key", sessionKey, "error", err)
	}

	return result.Text, nil
}

func (s *Service) handlePendingConfirmation(ctx context.Context, sessionKey string, userText string, locale string, reporter *progress.Reporter) (string, bool, error) {
	pending, err := s.store.GetPendingAction(ctx, sessionKey)
	if err != nil || pending.ActionType == "" {
		return "", false, err
	}
	trimmed := strings.TrimSpace(strings.ToLower(userText))
	switch trimmed {
	case "确认", "确认执行", "确认继续", "confirm", "yes", "proceed":
		reporter.Step(ctx, "我收到确认了，正在执行刚才挂起的高危动作。")
		response, execErr := s.executePendingAction(ctx, sessionKey, pending, locale, reporter)
		if delErr := s.store.DeletePendingAction(ctx, sessionKey); delErr != nil && execErr == nil {
			execErr = delErr
		}
		return response, true, execErr
	case "取消", "不用了", "先取消", "cancel", "stop", "no":
		if err := s.store.DeletePendingAction(ctx, sessionKey); err != nil {
			return "", true, err
		}
		response := s.msg(locale, "cancel_pending", nil)
		response, err = s.storeAssistantReply(ctx, sessionKey, response)
		return response, true, err
	default:
		// 尝试补全 plan_add / plan_edit 的参数（scan_type、plan_type、scope）
		if pending.ActionType == "edr_plan_add" || pending.ActionType == "edr_plan_edit" {
			if val, err := strconv.Atoi(strings.TrimSpace(userText)); err == nil && val >= 1 && val <= 8 {
				var call planner.ToolCall
				if err := json.Unmarshal([]byte(pending.Payload), &call); err != nil {
					return "", true, err
				}
				// 依次填充第一个缺失的字段
				if call.ScanType == 0 {
					call.ScanType = val
				} else if call.PlanType == 0 {
					call.PlanType = val
				} else if call.Scope == 0 {
					call.Scope = val
				} else {
					// 所有字段都已填充，直接执行
				}
				reporter.Step(ctx, "我收到参数了，正在执行计划操作。")
				response, execErr := s.executeConfirmedTool(ctx, call, locale, reporter)
				if delErr := s.store.DeletePendingAction(ctx, sessionKey); delErr != nil && execErr == nil {
					execErr = delErr
				}
				if execErr == nil {
					reply, storeErr := s.storeAssistantReply(ctx, sessionKey, response)
					return reply, true, storeErr
				}
				return "", true, execErr
			}
		}
		return "", false, nil
	}
}

func (s *Service) handleSessionCommand(ctx context.Context, scopeKey string, text string, locale string) (string, bool, error) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 || fields[0] != "/session" {
		return "", false, nil
	}
	if len(fields) == 1 {
		return s.msg(locale, "session_help", nil), true, nil
	}
	switch fields[1] {
	case "current":
		items, err := s.store.ListSessions(ctx, scopeKey, 20)
		if err != nil {
			return "", true, err
		}
		var item protocol.SessionRef
		for _, candidate := range items {
			if candidate.Active {
				item = candidate
				break
			}
		}
		if item.Key == "" {
			return s.msg(locale, "session_none", nil), true, nil
		}
		return s.msg(locale, "session_current", map[string]string{"session_id": item.PublicID, "title": item.Title, "status": item.Status}), true, nil
	case "list":
		items, err := s.store.ListSessions(ctx, scopeKey, 20)
		if err != nil {
			return "", true, err
		}
		if len(items) == 0 {
			return s.msg(locale, "session_none", nil), true, nil
		}
		lines := []string{s.msg(locale, "session_list_head", nil)}
		for _, item := range items {
			active := "false"
			if item.Active {
				active = "true"
			}
			lines = append(lines, s.msg(locale, "session_list_item", map[string]string{"session_id": item.PublicID, "title": item.Title, "status": item.Status, "active": active}))
		}
		return strings.Join(lines, "\n"), true, nil
	case "new":
		title := s.msg(locale, "session_title_fallback", nil)
		if len(fields) > 2 {
			title = strings.TrimSpace(strings.Join(fields[2:], " "))
		}
		item, err := s.store.CreateSession(ctx, scopeKey, title)
		if err != nil {
			return "", true, err
		}
		return s.msg(locale, "session_new_done", map[string]string{"session_id": item.PublicID}), true, nil
	case "use":
		if len(fields) < 3 {
			return s.msg(locale, "session_help", nil), true, nil
		}
		item, err := s.store.SetActiveSession(ctx, scopeKey, fields[2])
		if err != nil {
			return "", true, err
		}
		if item.Key == "" {
			return s.msg(locale, "session_not_found", map[string]string{"session_id": fields[2]}), true, nil
		}
		return s.msg(locale, "session_use_done", map[string]string{"session_id": item.PublicID}), true, nil
	case "close":
		item, err := s.store.CloseActiveSession(ctx, scopeKey)
		if err != nil {
			return "", true, err
		}
		if item.Key == "" {
			return s.msg(locale, "session_none", nil), true, nil
		}
		return s.msg(locale, "session_close_done", map[string]string{"session_id": item.PublicID}), true, nil
	case "delete":
		if len(fields) < 3 {
			return s.msg(locale, "session_help", nil), true, nil
		}
		items, err := s.store.ListSessions(ctx, scopeKey, 100)
		if err != nil {
			return "", true, err
		}
		found := false
		for _, item := range items {
			if item.PublicID == fields[2] {
				found = true
				break
			}
		}
		if !found {
			return s.msg(locale, "session_not_found", map[string]string{"session_id": fields[2]}), true, nil
		}
		if err := s.store.DeleteSession(ctx, scopeKey, fields[2]); err != nil {
			return "", true, err
		}
		return s.msg(locale, "session_delete_done", map[string]string{"session_id": fields[2]}), true, nil
	default:
		return s.msg(locale, "session_help", nil), true, nil
	}
}

func (s *Service) handleInterrupt(scopeKey string, text string, locale string) (string, bool) {
	plain := strings.TrimSpace(strings.ToLower(text))
	switch plain {
	case "/stop", "停止", "停止当前任务", "取消当前任务", "cancel", "stop":
		if s.cancelRun(scopeKey) {
			return s.msg(locale, "run_interrupted", nil), true
		}
		return s.msg(locale, "run_not_found", nil), true
	default:
		return "", false
	}
}

func (s *Service) handlePlannedTools(ctx context.Context, sessionKey string, text string, locale string, reporter *progress.Reporter) (string, bool, error) {
	if s.planner == nil {
		return "", false, nil
	}
	summary, _ := s.store.GetSessionSummary(ctx, sessionKey)
	recentTurns, _ := s.store.ListRecentTurns(ctx, sessionKey, minInt(maxInt(s.cfg.Session.MaxRecentTurns, 10), 16))
	s.logger.Info("planner context loaded", "session_key", sessionKey, "summary_len", len(summary), "recent_turns", len(recentTurns))
	memories, _ := s.memory.ListForContext(ctx, sessionKey)
	latestArtifact := protocol.Artifact{}
	if s.artifacts != nil {
		latestArtifact, _ = s.artifacts.GetLatest(ctx, sessionKey)
	}
	skillsPrompt := ""
	if s.prompt != nil {
		skillsPrompt = s.prompt.LoadSkillsPrompt()
	}
	plan, err := s.planner.BuildPlan(ctx, s.cfg.Routing.Model, text, "", summary, recentTurns, memories, latestArtifact, skillsPrompt)
	if err != nil {
		s.logger.Warn("planner failed", "error", err)
		return "", false, nil
	}
	if len(plan.ToolCalls) == 0 {
		if plan.DirectReply == "" {
			return "", false, nil
		}
		plan.DirectReply, err = s.storeAssistantReply(ctx, sessionKey, plan.DirectReply)
		return plan.DirectReply, true, err
	}

	response, err := s.executeToolPlan(ctx, sessionKey, text, locale, plan.ToolCalls, reporter)
	if err != nil {
		return "", true, err
	}
	return response, true, nil
}

func buildScopeKey(msg protocol.InboundMessage, cfg config.SessionConfig) string {
	parts := []string{string(msg.Channel), msg.TenantKey, msg.ChatID}
	if cfg.UseThreadInGroup && msg.ChatType == "group" && msg.ThreadID != "" {
		parts = append(parts, msg.ThreadID)
	}
	return strings.Join(parts, ":")
}

func scopeKeyFromSessionKey(sessionKey string) string {
	idx := strings.LastIndex(sessionKey, "::")
	if idx < 0 {
		return sessionKey
	}
	return sessionKey[:idx]
}

func parseScopeMetadata(scopeKey string) (string, string, string, string) {
	parts := strings.Split(scopeKey, ":")
	channel := "feishu"
	tenantKey := ""
	chatID := ""
	threadID := ""
	if len(parts) > 0 {
		channel = parts[0]
	}
	if len(parts) > 1 {
		tenantKey = parts[1]
	}
	if len(parts) > 2 {
		chatID = parts[2]
	}
	if len(parts) > 3 {
		threadID = strings.Join(parts[3:], ":")
	}
	return channel, tenantKey, chatID, threadID
}

func (s *Service) resolveConversationSession(ctx context.Context, scopeKey string, msg protocol.InboundMessage, locale string) (protocol.SessionRef, error) {
	if msg.ChatType == "group" && strings.TrimSpace(msg.ThreadID) == "" {
		title := s.msg(locale, "session_title_fallback", nil)
		return s.store.CreateSession(ctx, scopeKey, title)
	}
	return s.store.EnsureActiveSession(ctx, scopeKey)
}

func sanitizeReply(text string) string {
	return textutil.SanitizeReply(text)
}

func (s *Service) createScheduledTask(ctx context.Context, sessionKey string, call planner.ToolCall) string {
	scopeKey := scopeKeyFromSessionKey(sessionKey)
	channel, tenantKey, chatID, threadID := parseScopeMetadata(scopeKey)
	intervalMinutes := call.TaskIntervalMinutes
	if intervalMinutes <= 0 {
		intervalMinutes = positiveOr(s.cfg.Scheduler.DefaultIntervalM, 5)
	}
	if strings.TrimSpace(call.TaskPrompt) == "" {
		return "创建定时任务时需要明确任务内容。"
	}
	title := strings.TrimSpace(call.TaskTitle)
	if title == "" {
		title = taskTitleFromPrompt(call.TaskPrompt)
	}
	task := protocol.ScheduledTask{
		ScopeKey:        scopeKey,
		SessionKey:      sessionKey,
		Channel:         protocol.Channel(channel),
		TenantKey:       tenantKey,
		ChatID:          chatID,
		ThreadID:        threadID,
		Title:           title,
		Prompt:          strings.TrimSpace(call.TaskPrompt),
		IntervalSeconds: intervalMinutes * 60,
		Status:          firstNonEmptyString(call.TaskStatus, "active"),
	}
	stored, err := s.store.CreateScheduledTask(ctx, task)
	if err != nil {
		return "创建定时任务失败：" + err.Error()
	}
	return fmt.Sprintf("已创建定时任务 %s，每 %d 分钟执行一次。任务内容：%s。下次执行时间：%s。", stored.TaskID, stored.IntervalSeconds/60, stored.Prompt, stored.NextRunAt.Local().Format("2006-01-02 15:04:05"))
}

func (s *Service) listScheduledTasks(ctx context.Context, sessionKey string) string {
	items, err := s.store.ListScheduledTasks(ctx, scopeKeyFromSessionKey(sessionKey), 50)
	if err != nil {
		return "查看定时任务失败：" + err.Error()
	}
	if len(items) == 0 {
		return "当前还没有定时任务。"
	}
	lines := []string{"当前定时任务如下："}
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- %s | status=%s | 每 %d 分钟 | 下次=%s | 内容=%s", item.TaskID, item.Status, maxInt(item.IntervalSeconds/60, 1), item.NextRunAt.Local().Format("2006-01-02 15:04:05"), item.Prompt))
	}
	return strings.Join(lines, "\n")
}

func (s *Service) updateScheduledTask(ctx context.Context, sessionKey string, call planner.ToolCall) string {
	if strings.TrimSpace(call.TaskID) == "" {
		return "要修改定时任务的话，先告诉我任务 id。"
	}
	patch := protocol.ScheduledTaskPatch{
		Title:       strings.TrimSpace(call.TaskTitle),
		Prompt:      strings.TrimSpace(call.TaskPrompt),
		Status:      strings.TrimSpace(call.TaskStatus),
		LastSummary: "",
	}
	if call.TaskIntervalMinutes > 0 {
		patch.IntervalSeconds = call.TaskIntervalMinutes * 60
		patch.NextRunAt = time.Now().UTC().Add(time.Duration(patch.IntervalSeconds) * time.Second)
	}
	item, err := s.store.UpdateScheduledTask(ctx, scopeKeyFromSessionKey(sessionKey), call.TaskID, patch)
	if err != nil {
		return "修改定时任务失败：" + err.Error()
	}
	if item.TaskID == "" {
		return "没有找到这个定时任务。"
	}
	return fmt.Sprintf("已更新定时任务 %s，当前 status=%s，每 %d 分钟执行一次，内容=%s。", item.TaskID, item.Status, maxInt(item.IntervalSeconds/60, 1), item.Prompt)
}

func (s *Service) deleteScheduledTask(ctx context.Context, sessionKey string, call planner.ToolCall) string {
	if strings.TrimSpace(call.TaskID) == "" {
		return "要删除定时任务的话，先告诉我任务 id。"
	}
	if err := s.store.DeleteScheduledTask(ctx, scopeKeyFromSessionKey(sessionKey), call.TaskID); err != nil {
		return "删除定时任务失败：" + err.Error()
	}
	return fmt.Sprintf("已删除定时任务 %s。", call.TaskID)
}

func (s *Service) feedbackScheduledTask(ctx context.Context, sessionKey string, call planner.ToolCall) string {
	scopeKey := scopeKeyFromSessionKey(sessionKey)
	run, err := s.store.GetLatestScheduledTaskRun(ctx, scopeKey)
	if err != nil {
		return "记录任务反馈失败：" + err.Error()
	}
	if run.RunID == "" {
		return "当前没有可回写反馈的定时任务结果。"
	}
	entities, err := decodeScheduledEntities(run.EntitiesJSON)
	if err != nil || len(entities) == 0 {
		return "最近一轮定时任务没有可标记的对象。"
	}
	feedback := strings.TrimSpace(call.TaskFeedback)
	if feedback == "" {
		feedback = "false_positive"
	}
	now := time.Now().UTC()
	for _, entity := range entities {
		entity.TaskID = run.TaskID
		entity.Status = feedback
		entity.Note = "user_feedback"
		entity.LastSeenAt = now
		if entity.FirstSeenAt.IsZero() {
			entity.FirstSeenAt = now
		}
		if err := s.store.UpsertScheduledTaskEntity(ctx, entity); err != nil {
			return "记录任务反馈失败：" + err.Error()
		}
	}
	state := map[string]any{"last_feedback": feedback, "updated_at": now.Format(time.RFC3339), "entity_count": len(entities)}
	body, _ := json.Marshal(state)
	_ = s.store.UpsertScheduledTaskState(ctx, run.TaskID, string(body))
	return fmt.Sprintf("已把最近一轮定时任务里的 %d 个对象标记为 %s，后续会优先按这个结论处理。", len(entities), feedback)
}

func taskTitleFromPrompt(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "定时任务"
	}
	if len(text) <= 24 {
		return text
	}
	return text[:24] + "..."
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func (s *Service) searchKnowledgeBase(call planner.ToolCall) string {
	if s.knowledge == nil || !s.knowledge.Enabled() {
		return "知识库当前未启用。"
	}
	query := firstNonEmptyString(call.KBQuery, call.Query, call.KBTitle)
	items, err := s.knowledge.Search(query)
	if err != nil {
		return "搜索知识库失败：" + err.Error()
	}
	if len(items) == 0 {
		if strings.TrimSpace(query) == "" {
			return "知识库目前还是空的。"
		}
		return fmt.Sprintf("我在知识库里按“%s”搜了一下，暂时没有命中相关内容。", query)
	}
	lines := []string{fmt.Sprintf("我在知识库里按“%s”找到了这些内容：", blankToDash(query))}
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- %s (%s)\n  %s", item.Title, item.RelPath, item.Snippet))
	}
	return strings.Join(lines, "\n")
}

func (s *Service) writeKnowledgeBase(call planner.ToolCall) string {
	if s.knowledge == nil || !s.knowledge.Enabled() {
		return "知识库当前未启用。"
	}
	title := firstNonEmptyString(call.KBTitle, call.KBQuery)
	if strings.TrimSpace(title) == "" {
		return "写入知识库时需要明确标题。"
	}
	item, err := s.knowledge.Upsert(title, call.KBContent, call.KBMode, call.KBOldText, call.KBNewText)
	if err != nil {
		return "写入知识库失败：" + err.Error()
	}
	return fmt.Sprintf("知识库已更新：%s (%s)。\n%s", item.Title, item.RelPath, item.Snippet)
}

func (s *Service) deleteKnowledgeBase(call planner.ToolCall) string {
	if s.knowledge == nil || !s.knowledge.Enabled() {
		return "知识库当前未启用。"
	}
	title := firstNonEmptyString(call.KBTitle, call.KBQuery)
	if strings.TrimSpace(title) == "" {
		return "删除知识库条目时需要明确标题。"
	}
	if err := s.knowledge.Delete(title); err != nil {
		return "删除知识库失败：" + err.Error()
	}
	return fmt.Sprintf("知识库条目已删除：%s。", title)
}

func (s *Service) startRun(parent context.Context, sessionKey string) (context.Context, func()) {
	runCtx, cancel := context.WithCancel(parent)
	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	s.runMu.Lock()
	if old, ok := s.runs[sessionKey]; ok {
		old.cancel()
	}
	s.runs[sessionKey] = runState{id: runID, cancel: cancel}
	s.runMu.Unlock()
	return runCtx, func() {
		s.runMu.Lock()
		if current, ok := s.runs[sessionKey]; ok && current.id == runID {
			delete(s.runs, sessionKey)
		}
		s.runMu.Unlock()
		cancel()
	}
}

func (s *Service) cancelRun(scopeKey string) bool {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	for key, cancel := range s.runs {
		if strings.HasPrefix(key, scopeKey+"::") || key == scopeKey {
			cancel.cancel()
			delete(s.runs, key)
			return true
		}
	}
	return false
}

func (s *Service) detectLocale(text string) string {
	if s.i18n == nil {
		return "zh-CN"
	}
	return s.i18n.DetectLocale(text)
}

func shortResult(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 120 {
		return text
	}
	return text[:120] + "..."
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Service) msg(locale string, key string, vars map[string]string) string {
	if s.i18n == nil {
		return ""
	}
	return s.i18n.T(locale, key, vars)
}

func (s *Service) storeAssistantReply(ctx context.Context, sessionKey string, text string) (string, error) {
	original := text
	text = sanitizeReply(text)
	if original != text {
		s.logger.Info("assistant reply sanitized", "session_key", sessionKey, "before_preview", shortResult(original), "after_preview", shortResult(text))
	}
	s.logger.Info("store assistant reply", "session_key", sessionKey, "preview", shortResult(text))
	if err := s.store.AppendTurn(ctx, sessionKey, string(model.RoleAssistant), text); err != nil {
		return "", err
	}
	return text, nil
}

func (s *Service) executeToolPlan(ctx context.Context, sessionKey string, userText string, locale string, calls []planner.ToolCall, reporter *progress.Reporter) (string, error) {
	var allToolResults []string
	currentCalls := calls
	finalDirectReply := ""
	seenSignatures := make(map[string]int)
	hasGroundedEDRTool := false

	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		signature := toolCallsSignature(currentCalls)
		seenSignatures[signature]++
		if seenSignatures[signature] > 3 {
			s.logger.Warn("tool chain stopped by repeated signature", "session_key", sessionKey, "signature", signature, "count", seenSignatures[signature])
			reporter.Step(ctx, "我发现后续步骤开始重复了，先基于已拿到的真实结果整理结论。")
			break
		}
		s.logger.Info("execute tool round", "session_key", sessionKey, "tool_count", len(currentCalls))
		stepResults, earlyReply, err := s.executeToolBatch(ctx, sessionKey, locale, currentCalls, reporter)
		if err != nil {
			return "", err
		}
		if earlyReply != "" {
			return earlyReply, nil
		}
		if len(stepResults) > 0 {
			allToolResults = append(allToolResults, stepResults...)
		}
		for _, call := range currentCalls {
			if callNeedsGroundedEDRAnswer(call.Name) {
				hasGroundedEDRTool = true
				break
			}
		}

		toolContext := strings.Join(allToolResults, "\n\n")
		s.logger.Info("tool round context ready", "session_key", sessionKey, "result_count", len(allToolResults), "context_preview", shortResult(toolContext))
		memories, _ := s.memory.ListForContext(ctx, sessionKey)
		summary, _ := s.store.GetSessionSummary(ctx, sessionKey)
		recentTurns, _ := s.store.ListRecentTurns(ctx, sessionKey, minInt(maxInt(s.cfg.Session.MaxRecentTurns, 10), 16))
		latestArtifact := protocol.Artifact{}
		if s.artifacts != nil {
			latestArtifact, _ = s.artifacts.GetLatest(ctx, sessionKey)
		}
		skillsPrompt := ""
		if s.prompt != nil {
			skillsPrompt = s.prompt.LoadSkillsPrompt()
		}
		nextPlan, err := s.planner.BuildPlan(ctx, s.cfg.Routing.Model, userText, toolContext, summary, recentTurns, memories, latestArtifact, skillsPrompt)
		if err != nil {
			s.logger.Warn("follow-up planner failed", "error", err)
			break
		}
		if len(nextPlan.ToolCalls) == 0 {
			finalDirectReply = nextPlan.DirectReply
			s.logger.Info("tool chain finished planning", "session_key", sessionKey, "final_reply_preview", shortResult(finalDirectReply))
			break
		}
		s.logger.Info("tool chain continues", "session_key", sessionKey, "next_tool_count", len(nextPlan.ToolCalls))
		currentCalls = nextPlan.ToolCalls
	}

	if len(allToolResults) == 0 && finalDirectReply == "" {
		return "", nil
	}
	reporter.Step(ctx, "我拿到真实工具结果了，正在整理成更自然的回复。")
	response := finalDirectReply
	if strings.TrimSpace(response) == "" {
		if hasGroundedEDRTool {
			var err error
			response, err = s.answerGroundedByEDR(ctx, sessionKey, userText, strings.Join(allToolResults, "\n\n"))
			if err != nil {
				response = strings.Join(allToolResults, "\n\n")
			}
		} else {
			response = strings.Join(allToolResults, "\n\n")
		}
	}
	response, err := s.storeAssistantReply(ctx, sessionKey, response)
	if err != nil {
		return "", err
	}
	if err := s.compactor.MaybeCompact(ctx, sessionKey); err != nil {
		s.logger.Warn("compact session failed", "session_key", sessionKey, "error", err)
	}
	return response, nil
}

func toolCallsSignature(calls []planner.ToolCall) string {
	parts := make([]string, 0, len(calls))
	for _, call := range calls {
		parts = append(parts, strings.Join([]string{
			call.Name,
			call.ClientID,
			call.Hostname,
			call.ClientIP,
			call.OSType,
			call.Operation,
			call.StartTime,
			call.EndTime,
			call.FilterField,
			call.FilterOp,
			call.FilterValue,
			fmt.Sprintf("p=%d", call.Page),
			fmt.Sprintf("ps=%d", call.PageSize),
			call.IncidentID,
			call.DetectionID,
			call.ViewType,
			call.ProcessUUID,
			call.ArtifactID,
			call.Query,
			fmt.Sprintf("sl=%d", call.StartLine),
			fmt.Sprintf("lc=%d", call.LineCount),
			call.MemoryKey,
			call.MemoryValue,
			call.TaskID,
			call.TaskTitle,
			call.TaskPrompt,
			call.TaskAction,
			call.TaskStatus,
			call.TaskFeedback,
			fmt.Sprintf("ti=%d", call.TaskIntervalMinutes),
			call.InstructionName,
			call.Path,
			call.KBTitle,
			call.KBQuery,
			call.KBContent,
			call.KBMode,
			call.KBOldText,
			call.KBNewText,
			call.Reason,
			fmt.Sprintf("critical=%t", call.Critical),
			call.IOCAction,
			call.IOCID,
			call.IOCHash,
			call.IOCDescription,
			call.IOCExpirationDate,
			call.IOCFileName,
			call.IOCHostType,
			call.IsolateFileGUIDs,
			fmt.Sprintf("add_excl=%t", call.IsolateFileAddExcl),
			fmt.Sprintf("release_all=%t", call.IsolateFileReleaseAll),
			call.PlanName,
			fmt.Sprintf("st=%d", call.ScanType),
			fmt.Sprintf("pt=%d", call.PlanType),
			fmt.Sprintf("scope=%d", call.Scope),
			call.RID,
			fmt.Sprintf("time=%d", call.Time),
			fmt.Sprintf("pid=%d", call.Pid),
			call.Ids,
			fmt.Sprintf("allow=%t", call.Allow),
			fmt.Sprintf("status=%d", call.Status),
			call.Scene,
			call.Comment,
			call.Type,
		}, "|"))
	}
	return strings.Join(parts, "||")
}

func toolCallCacheKey(call planner.ToolCall) string {
	data := strings.Join([]string{
		call.Name,
		call.ClientID,
		call.Hostname,
		call.ClientIP,
		call.OSType,
		call.Operation,
		call.StartTime,
		call.EndTime,
		call.FilterField,
		call.FilterOp,
		call.FilterValue,
		fmt.Sprintf("p=%d", call.Page),
		fmt.Sprintf("ps=%d", call.PageSize),
		call.IncidentID,
		call.DetectionID,
		call.ViewType,
		call.ProcessUUID,
		call.ArtifactID,
		call.Query,
		fmt.Sprintf("sl=%d", call.StartLine),
		fmt.Sprintf("lc=%d", call.LineCount),
		call.MemoryKey,
		call.MemoryValue,
		call.TaskID,
		call.TaskTitle,
		call.TaskPrompt,
		call.TaskAction,
		call.TaskStatus,
		call.TaskFeedback,
		fmt.Sprintf("ti=%d", call.TaskIntervalMinutes),
		call.InstructionName,
		call.Path,
		call.KBTitle,
		call.KBQuery,
		call.KBContent,
		call.KBMode,
		call.KBOldText,
		call.KBNewText,
		call.Reason,
		call.IOCAction,
		call.IOCID,
		call.IOCHash,
		call.IOCDescription,
		call.IOCExpirationDate,
		call.IOCFileName,
		call.IOCHostType,
		call.IsolateFileGUIDs,
		call.PlanName,
		fmt.Sprintf("st=%d", call.ScanType),
		fmt.Sprintf("pt=%d", call.PlanType),
		fmt.Sprintf("scope=%d", call.Scope),
		call.RID,
		fmt.Sprintf("time=%d", call.Time),
		fmt.Sprintf("pid=%d", call.Pid),
		call.Ids,
		fmt.Sprintf("allow=%t", call.Allow),
		fmt.Sprintf("status=%d", call.Status),
		call.Scene,
		call.Comment,
		call.Type,
	}, "|")
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func callNeedsGroundedEDRAnswer(name string) bool {
	switch name {
	case "edr_hosts", "edr_incidents", "edr_detections", "edr_logs", "edr_incident_view", "edr_detection_view", "artifact_search", "artifact_read", "edr_iocs", "edr_isolate_files", "edr_tasks", "edr_task_result", "edr_plan_list", "edr_virus_by_host", "edr_virus_by_hash", "edr_virus_hash_hosts", "edr_virus_scan_record", "edr_instruction_policy_list":
		return true
	case "edr_isolate", "edr_release", "edr_ioc_add", "edr_ioc_update", "edr_ioc_delete", "edr_delete_isolate_files", "edr_release_isolate_files", "edr_send_instruction", "edr_plan_add", "edr_plan_edit", "edr_plan_cancel", "edr_instruction_policy_update", "edr_instruction_policy_save_status", "edr_instruction_policy_delete", "edr_instruction_policy_sort", "edr_instruction_policy_add":
		return false
	default:
		return false
	}
}

func (s *Service) executeToolBatch(ctx context.Context, sessionKey string, locale string, calls []planner.ToolCall, reporter *progress.Reporter) ([]string, string, error) {
	var results []string
	for _, call := range calls {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		s.logger.Info("start tool call", "session_key", sessionKey, "tool", call.Name, "client_id", call.ClientID, "os_type", call.OSType, "operation", call.Operation, "start_time", call.StartTime, "end_time", call.EndTime, "filter_field", call.FilterField, "filter_operator", call.FilterOp, "filter_value", call.FilterValue, "page", call.Page, "page_size", call.PageSize, "incident_id", call.IncidentID, "detection_id", call.DetectionID, "artifact_id", call.ArtifactID, "query", call.Query, "kb_title", call.KBTitle, "kb_query", call.KBQuery, "kb_mode", call.KBMode, "instruction_name", call.InstructionName, "path", call.Path, "status", call.Status, "allow", call.Allow, "scene", call.Scene)
		if call.Critical || isCriticalTool(call.Name) {
			payload, _ := json.Marshal(call)
			summary := call.Name
			if call.Ids != "" {
				summary += " ids=" + call.Ids
			}
			if call.ClientID != "" {
				summary += ", client_id=" + call.ClientID
			}
			if call.InstructionName != "" {
				summary += ", instruction_name=" + call.InstructionName
			}
			if call.Path != "" {
				summary += ", path=" + call.Path
			}
			if call.Time > 0 {
				summary += fmt.Sprintf(", time=%d", call.Time)
			}
			if call.Pid > 0 {
				summary += fmt.Sprintf(", pid=%d", call.Pid)
			}
			if call.TaskID != "" {
				summary += ", task_id=" + call.TaskID
			}
			if call.IncidentID != "" {
				summary += ", incident_id=" + call.IncidentID
			}
			if call.DetectionID != "" {
				summary += ", detection_id=" + call.DetectionID
			}
			if call.IOCAction != "" {
				summary += ", ioc_action=" + call.IOCAction
			}
			if call.IOCHash != "" {
				summary += ", ioc_hash=" + call.IOCHash
			}
			if call.IOCID != "" {
				summary += ", ioc_id=" + call.IOCID
			}
			if call.IsolateFileGUIDs != "" {
				summary += ", isolate_file_guids=" + call.IsolateFileGUIDs
			}
			if call.PlanName != "" {
				summary += ", plan_name=" + call.PlanName
			}
			if call.ScanType > 0 {
				summary += fmt.Sprintf(", scan_type=%d", call.ScanType)
			}
			if call.Status > 0 {
				summary += fmt.Sprintf(", status=%d", call.Status)
			}
			if call.Scope > 0 {
				summary += fmt.Sprintf(", scope=%d", call.Scope)
			}
			if call.PlanType > 0 {
				summary += fmt.Sprintf(", plan_type=%d", call.PlanType)
			}
			if call.Type != "" {
				summary += ", type=" + call.Type
			}
			if call.RID != "" {
				summary += ", rid=" + call.RID
			}
			if call.Hostname != "" {
				summary += ", hostname=" + call.Hostname
			}
			if call.Reason != "" {
				summary += ", reason=" + call.Reason
			}
			if call.StrategyID != "" {
				summary += ", strategy_id=" + call.StrategyID
				if call.ScanFileScope != "" {
					summary += ", scan_file_scope=" + call.ScanFileScope
				}
				if call.StartupScanMode != "" {
					summary += ", startup_scan_mode=" + call.StartupScanMode
				}
				if call.ArchiveSizeLimitEnabled != nil && *call.ArchiveSizeLimitEnabled {
					summary += fmt.Sprintf(", archive_size_limit=%d", call.ArchiveSizeLimit)
				}
				if call.RealtimeMemCacheTechEnabled != nil && *call.RealtimeMemCacheTechEnabled {
					summary += ", realtime_mem_cache_tech=true"
				}
				if call.DynamicCpuMonitorEnabled != nil && *call.DynamicCpuMonitorEnabled {
					summary += fmt.Sprintf(", dynamic_cpu_high_percent=%d", call.DynamicCpuHighPercent)
				}
				if call.StopRealtimeOnCpuHighEnabled != nil && *call.StopRealtimeOnCpuHighEnabled {
					summary += fmt.Sprintf(", stop_realtime_on_cpu_high=%d", call.StopRealtimeCpuHighPercent)
				}
				if call.OwlOnRealtimeEnabled != nil && *call.OwlOnRealtimeEnabled {
					summary += ", owl_on_realtime=true"
				}
				if call.RealtimeScanArchiveEnabled != nil && *call.RealtimeScanArchiveEnabled {
					summary += ", realtime_scan_archive=true"
				}
				if call.RuntimeMaxFileSizeMb > 0 {
					summary += fmt.Sprintf(", runtime_max_file_size_mb=%d", call.RuntimeMaxFileSizeMb)
				}
				if call.CustomMaxFileSizeMb > 0 {
					summary += fmt.Sprintf(", custom_max_file_size_mb=%d", call.CustomMaxFileSizeMb)
				}
			}

			if err := s.store.SavePendingAction(ctx, sessionKey, call.Name, string(payload), summary); err != nil {
				return nil, "", err
			}
			response := s.msg(locale, "pending_high_risk", map[string]string{"summary": summary})
			stored, err := s.storeAssistantReply(ctx, sessionKey, response)
			return nil, stored, err
		}
		result, err := s.executeSingleTool(ctx, sessionKey, call, locale, reporter)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, "", err
			}
			s.logger.Warn("tool call failed", "session_key", sessionKey, "tool", call.Name, "error", err)
			results = append(results, s.msg(locale, "tool_error", map[string]string{"tool": call.Name, "error": err.Error()}))
			continue
		}
		s.logger.Info("tool call finished", "session_key", sessionKey, "tool", call.Name, "result_preview", shortResult(result))
		reporter.ToolResult(ctx, call.Name, result)
		if strings.TrimSpace(result) != "" {
			results = append(results, result)
		}
	}
	return results, "", nil
}

func (s *Service) executeSingleTool(ctx context.Context, sessionKey string, call planner.ToolCall, locale string, reporter *progress.Reporter) (string, error) {
	if !call.Critical && !isCriticalTool(call.Name) {
		key := toolCallCacheKey(call)
		if result, err, hit := s.dedupCache.GetOrSubmit(key); hit {
			s.logger.Info("tool call dedup hit", "session_key", sessionKey, "tool", call.Name, "key", key)
			return result, err
		}
		result, err := s.executeToolImpl(ctx, sessionKey, call, locale, reporter)
		s.dedupCache.Done(key, result, err)
		return result, err
	}
	return s.executeToolImpl(ctx, sessionKey, call, locale, reporter)
}

func (s *Service) executeToolImpl(ctx context.Context, sessionKey string, call planner.ToolCall, locale string, reporter *progress.Reporter) (string, error) {
	switch call.Name {
	case "memory_upsert":
		if err := s.memory.Upsert(ctx, sessionKey, call.MemoryKey, call.MemoryValue); err != nil {
			return "", err
		}
		return s.msg(locale, "memory_upsert", map[string]string{"key": call.MemoryKey, "value": call.MemoryValue}), nil
	case "current_time":
		now := time.Now()
		return s.msg(locale, "current_time", map[string]string{
			"local": now.Format("2006-01-02 15:04:05 MST"),
			"utc":   now.UTC().Format(time.RFC3339),
		}), nil
	case "memory_delete":
		if err := s.memory.Delete(ctx, sessionKey, call.MemoryKey); err != nil {
			return "", err
		}
		return s.msg(locale, "memory_delete", map[string]string{"key": call.MemoryKey}), nil
	case "scheduled_task_create":
		return s.createScheduledTask(ctx, sessionKey, call), nil
	case "scheduled_task_list":
		return s.listScheduledTasks(ctx, sessionKey), nil
	case "scheduled_task_update":
		return s.updateScheduledTask(ctx, sessionKey, call), nil
	case "scheduled_task_delete":
		return s.deleteScheduledTask(ctx, sessionKey, call), nil
	case "scheduled_task_feedback":
		return s.feedbackScheduledTask(ctx, sessionKey, call), nil
	case "knowledge_base_search":
		return s.searchKnowledgeBase(call), nil
	case "knowledge_base_write":
		return s.writeKnowledgeBase(call), nil
	case "knowledge_base_delete":
		return s.deleteKnowledgeBase(call), nil
	case "edr_hosts":
		reporter.Step(ctx, "我在查主机状态和基础信息。")
		result, err := s.edr.ListHosts(ctx, edr.ListHostsRequest{Hostname: call.Hostname, ClientIP: call.ClientIP})
		if err != nil {
			return "", err
		}
		return formatHosts(result), nil
	case "edr_incidents":
		reporter.Step(ctx, "我在拉取近期事件。")
		result, err := s.edr.ListIncidents(ctx, edr.ListIncidentsRequest{ClientID: call.ClientID, Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return formatIncidents(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_batch_deal_incident":
		reporter.Step(ctx, "我正在批量处置事件。")
		result, err := s.edr.BatchDealIncident(ctx, edr.BatchDealIncidentRequest{
			IDs:    strings.Split(strings.TrimSpace(call.IncidentID), ","),
			Allow:  call.Allow,
			Status: call.Status,
			Scene:  call.Scene,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("批量处置完成：%d 个事件已处理", result.TotalIncident), nil
	case "edr_incident_r2_summary":
		reporter.Step(ctx, "我在拉取事件 R2 摘要。")
		result, err := s.edr.IncidentR2Summary(ctx, call.IncidentID)
		if err != nil {
			return "", err
		}
		return formatIncidentR2Summary(result), nil
	case "edr_detections":
		reporter.Step(ctx, "我在拉取近期行为检出。")
		result, err := s.edr.ListDetections(ctx, edr.ListDetectionsRequest{
			Page:       positiveOr(call.Page, 1),
			Limit:      positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize),
			IncidentID: call.IncidentID})
		if err != nil {
			return "", err
		}
		return formatDetections(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_event_log_alarms":
		reporter.Step(ctx, "我在拉取事件日志告警列表。")
		result, err := s.edr.ListEventLogAlarms(ctx, edr.ListEventLogAlarmsRequest{Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return formatEventLogAlarms(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_logs":
		reporter.Step(ctx, "我在拉取行为日志。")
		result, err := s.edr.ListLogs(ctx, edr.ListLogsRequest{ClientID: call.ClientID, OSType: call.OSType, Operation: call.Operation, StartTime: call.StartTime, EndTime: call.EndTime, FilterField: call.FilterField, FilterOperator: call.FilterOp, FilterValue: call.FilterValue, Page: positiveOr(call.Page, 1), PageSize: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return s.formatLogs(ctx, result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize), call), nil
	case "edr_incident_view":
		reporter.Step(ctx, "我在拉取这条 incident 的详情。")
		result, err := s.edr.ViewIncident(ctx, edr.IncidentViewRequest{IncidentID: call.IncidentID, ClientID: call.ClientID})
		if err != nil {
			return "", err
		}
		return s.prepareDetailContext(ctx, sessionKey, locale, "incident", call.IncidentID, userHint(call), result, reporter)
	case "edr_detection_view":
		reporter.Step(ctx, "我在拉取这条 detection 的详情。")
		result, err := s.edr.ViewDetection(ctx, edr.DetectionViewRequest{DetectionID: call.DetectionID, ClientID: call.ClientID, ViewType: call.ViewType, ProcessUUID: call.ProcessUUID})
		if err != nil {
			return "", err
		}
		return s.prepareDetailContext(ctx, sessionKey, locale, "detection", call.DetectionID, userHint(call), result, reporter)
	case "artifact_search":
		if s.artifacts == nil {
			return "", nil
		}
		reporter.Step(ctx, "我在大对象详情里做定向搜索。")
		item, matches, err := s.artifacts.Search(ctx, sessionKey, call.ArtifactID, call.Query, 8)
		if err != nil {
			return "", err
		}
		return s.formatArtifactSearch(ctx, locale, item, call.Query, matches, reporter), nil
	case "artifact_read":
		if s.artifacts == nil {
			return "", nil
		}
		reporter.Step(ctx, "我在读取大对象详情里的指定片段。")
		item, chunk, err := s.artifacts.Read(ctx, sessionKey, call.ArtifactID, call.StartLine, call.LineCount)
		if err != nil {
			return "", err
		}
		return s.formatArtifactRead(ctx, locale, item, call.StartLine, call.LineCount, chunk, reporter), nil
	case "edr_iocs":
		reporter.Step(ctx, "我在拉取 IOC 列表。")
		result, err := s.edr.ListIOCs(ctx, edr.ListIOCsRequest{Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return s.formatIOCs(ctx, result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_isolate_files":
		reporter.Step(ctx, "我在拉取隔离文件列表。")
		result, err := s.edr.ListIsolateFiles(ctx, edr.ListIsolateFilesRequest{Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return s.formatIsolateFiles(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_tasks":
		reporter.Step(ctx, "我在拉取指令任务列表。")
		result, err := s.edr.ListTasks(ctx, edr.ListTasksRequest{Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return s.formatTasks(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_task_result":
		reporter.Step(ctx, "我在拉取任务结果。")
		result, err := s.edr.GetTaskResult(ctx, call.TaskID)
		if err != nil {
			return "", err
		}
		return formatTaskResult(result), nil
	case "edr_plan_list":
		reporter.Step(ctx, "我在拉取计划列表。")
		result, err := s.edr.ListPlans(ctx, edr.ListPlansRequest{Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize), Type: call.Type})
		if err != nil {
			return "", err
		}
		return formatPlans(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_virus_scan_record":
		reporter.Step(ctx, "我在拉取病毒扫描记录。")
		result, err := s.edr.ListVirusScanRecords(ctx, edr.ListVirusScanRecordsRequest{HostName: call.Hostname, ClientID: call.ClientID, Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return formatVirusScanRecords(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_virus_by_host":
		reporter.Step(ctx, "我在按主机查询病毒信息。")
		result, err := s.edr.ListVirusByHost(ctx, edr.ListVirusByHostRequest{HostName: call.Hostname, ClientID: call.ClientID, Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return formatVirusByHost(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_virus_by_hash":
		reporter.Step(ctx, "我在按哈希查询病毒信息。")
		result, err := s.edr.ListVirusByHash(ctx, edr.ListVirusByHashRequest{Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return formatVirusByHash(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_virus_hash_hosts":
		reporter.Step(ctx, "我在按哈希查询关联主机。")
		result, err := s.edr.ListVirusHashHosts(ctx, edr.ListVirusHashHostsRequest{HostName: call.Hostname, ClientID: call.ClientID, Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return formatVirusHashHosts(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_instruction_policy_list":
		reporter.Step(ctx, "我在拉取自动响应策略列表。")
		result, err := s.edr.ListInstructionPolicies(ctx, edr.ListInstructionPoliciesRequest{PolicyType: call.Page, Status: call.PageSize})
		if err != nil {
			return "", err
		}
		return formatInstructionPolicies(result), nil
	case "edr_ioas":
		reporter.Step(ctx, "我在拉取 IOA 列表。")
		result, err := s.edr.ListIOAs(ctx, edr.ListIOAsRequest{Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return formatIOAs(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_ioa_audit_log":
		reporter.Step(ctx, "我在拉取 IOA 活动记录。")
		result, err := s.edr.ListIOAAuditLogs(ctx, edr.ListIOAAuditLogsRequest{Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return formatIOAAuditLogs(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_ioa_networks":
		reporter.Step(ctx, "我在拉取 IOA 网络排除列表。")
		result, err := s.edr.ListIOANetworks(ctx, edr.ListIOANetworksRequest{Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)})
		if err != nil {
			return "", err
		}
		return formatIOANetworks(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_strategies":
		reporter.Step(ctx, "我在拉取策略列表。")
		result, err := s.edr.ListStrategies(ctx, edr.ListStrategiesRequest{Page: positiveOr(call.Page, 1), Limit: positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize), Type: call.InstructionName})
		if err != nil {
			return "", err
		}
		return formatStrategies(result, positiveOr(call.Page, 1), positiveOr(call.PageSize, s.cfg.EDR.DefaultPageSize)), nil
	case "edr_strategy_single":
		reporter.Step(ctx, "我在拉取单个策略。")
		result, err := s.edr.GetStrategySingle(ctx, call.Type)
		if err != nil {
			return "", err
		}
		return formatStrategySingle(result), nil
	case "edr_strategy_state":
		reporter.Step(ctx, "我在拉取策略状态统计。")
		result, err := s.edr.GetStrategyState(ctx)
		if err != nil {
			return "", err
		}
		return formatStrategyState(result), nil
	case "edr_host_offline":
		reporter.Step(ctx, "我在拉取主机离线配置。")
		result, err := s.edr.GetHostOfflineConf(ctx)
		if err != nil {
			return "", err
		}
		return formatHostOfflineConf(result), nil
	default:
		return "", nil
	}
}

func (s *Service) executePendingAction(ctx context.Context, sessionKey string, pending protocol.PendingAction, locale string, reporter *progress.Reporter) (string, error) {
	var call planner.ToolCall
	if err := json.Unmarshal([]byte(pending.Payload), &call); err != nil {
		return "", err
	}
	result, err := s.executeConfirmedTool(ctx, call, locale, reporter)
	if err != nil {
		return "", err
	}
	return s.storeAssistantReply(ctx, sessionKey, result)
}

func (s *Service) executeConfirmedTool(ctx context.Context, call planner.ToolCall, locale string, reporter *progress.Reporter) (string, error) {
	switch call.Name {
	case "edr_isolate":
		reporter.Step(ctx, "我在下发隔离动作，并等待任务回执。")
		result, err := s.edr.IsolateHost(ctx, call.ClientID, call.Time)
		if err != nil {
			return "", err
		}
		return s.msg(locale, "confirm_isolate_done", map[string]string{"task_id": result.TaskID, "host": result.HostName, "repeat": strconv.FormatBool(result.Repeat)}), nil
	case "edr_release":
		reporter.Step(ctx, "我在下发恢复动作，并等待任务回执。")
		result, err := s.edr.ReleaseHost(ctx, call.ClientID)
		if err != nil {
			return "", err
		}
		return s.msg(locale, "confirm_release_done", map[string]string{"task_id": result.TaskID, "host": result.HostName, "repeat": strconv.FormatBool(result.Repeat)}), nil
	case "edr_ioc_add":
		reporter.Step(ctx, "我在添加 IOC。")
		if err := s.edr.AddIOC(ctx, edr.AddIOCRequest{Action: call.IOCAction, Hash: call.IOCHash, Description: call.IOCDescription, ExpirationDate: call.IOCExpirationDate, FileName: call.IOCFileName, HostType: call.IOCHostType}); err != nil {
			return "", err
		}
		return fmt.Sprintf("IOC %s 已添加完成。", call.IOCHash), nil
	case "edr_ioc_update":
		reporter.Step(ctx, "我在更新 IOC。")
		if err := s.edr.UpdateIOC(ctx, edr.UpdateIOCRequest{ID: call.IOCID, Action: call.IOCAction, Hash: call.IOCHash, Description: call.IOCDescription, ExpirationDate: call.IOCExpirationDate, HostType: call.IOCHostType}); err != nil {
			return "", err
		}
		return fmt.Sprintf("IOC %s 已更新完成。", call.IOCID), nil
	case "edr_ioc_delete":
		reporter.Step(ctx, "我在删除 IOC。")
		if err := s.edr.DeleteIOC(ctx, call.IOCID); err != nil {
			return "", err
		}
		return fmt.Sprintf("IOC %s 已删除。", call.IOCID), nil
	case "edr_delete_isolate_files":
		reporter.Step(ctx, "我在删除隔离文件记录。")
		guids := strings.Split(strings.TrimSpace(call.IsolateFileGUIDs), ",")
		cleaned := make([]string, 0, len(guids))
		for _, g := range guids {
			g = strings.TrimSpace(g)
			if g != "" {
				cleaned = append(cleaned, g)
			}
		}
		if err := s.edr.DeleteIsolateFiles(ctx, cleaned); err != nil {
			return "", err
		}
		return fmt.Sprintf("已删除 %d 条隔离文件记录。", len(cleaned)), nil
	case "edr_release_isolate_files":
		reporter.Step(ctx, "我在放行隔离文件。")
		guids := strings.Split(strings.TrimSpace(call.IsolateFileGUIDs), ",")
		cleaned := make([]string, 0, len(guids))
		for _, g := range guids {
			g = strings.TrimSpace(g)
			if g != "" {
				cleaned = append(cleaned, g)
			}
		}
		if err := s.edr.ReleaseIsolateFiles(ctx, edr.ReleaseIsolateFilesRequest{GUIDs: cleaned, IsAddExclusion: call.IsolateFileAddExcl, ReleaseAllHash: call.IsolateFileReleaseAll}); err != nil {
			return "", err
		}
		return fmt.Sprintf("已放行 %d 个隔离文件。", len(cleaned)), nil
	case "edr_send_instruction":
		log.Printf("edr_send_instruction call: %+v", call)
		if call.ClientID == "" {
			return "", fmt.Errorf("发送指令需要提供 client_id")
		}
		if call.InstructionName == "" {
			return "", fmt.Errorf("发送指令需要提供 instruction_name")
		}
		reporter.Step(ctx, "我正在下发指令到目标主机。")
		req := edr.SendInstructionRequest{
			ClientID:        call.ClientID,
			InstructionName: call.InstructionName,
		}
		switch call.InstructionName {
		case "list_ps":
			req.IsOnline = 1
		case "get_suspicious_file", "batch_quarantine_file", "batch_kill_ps":
			req.IsBatch = 1
			bp := edr.BatchParam{}
			if call.Path != "" {
				bp.Path = call.Path
			}
			if call.Pid != 0 {
				bp.Pid = call.Pid
			}
			if bp.Path != "" || bp.Pid != 0 {
				req.BatchParams = []edr.BatchParam{bp}
			}
		}
		result, err := s.edr.SendInstruction(ctx, req)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("指令已下发成功，任务ID: %s，主机: %s，重复: %t", result.TaskID, result.HostName, result.Repeat), nil
	case "edr_plan_add":
		if call.Type == "" {
			return "", fmt.Errorf("type（业务类型）未指定，请补充：kill_plan/leak_repair/distribute_software/distribute_file")
		}
		if call.ScanType == 0 {
			return "", fmt.Errorf("scan_type（扫描类型）未指定，请补充：1-快速扫描 2-全盘扫描 3-自定义路径扫描 4-漏洞修复 5-安装软件 6-卸载软件 7-更新软件 8-发送文件")
		}
		if call.PlanType == 0 {
			return "", fmt.Errorf("plan_type（执行方式）未指定，请补充：1-立即执行 2-定时执行 3-周期执行")
		}
		if call.Scope == 0 {
			return "", fmt.Errorf("scope（范围）未指定，请补充：1-特定主机 2-主机组 3-全网主机")
		}
		reporter.Step(ctx, "我正在创建计划。")
		if err := s.edr.AddPlan(ctx, edr.AddPlanRequest{
			PlanName: call.PlanName,
			ScanType: call.ScanType,
			PlanType: call.PlanType,
			Scope:    call.Scope,
			Type:     call.Type,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("计划「%s」创建成功", call.PlanName), nil
	case "edr_plan_edit":
		if call.Type == "" {
			return "", fmt.Errorf("type（业务类型）未指定，请补充：kill_plan/leak_repair/distribute_software/distribute_file")
		}
		if call.ScanType == 0 {
			return "", fmt.Errorf("scan_type（扫描类型）未指定，请补充：1-快速扫描 2-全盘扫描 3-自定义路径扫描 4-漏洞修复 5-安装软件 6-卸载软件 7-更新软件 8-发送文件")
		}
		if call.PlanType == 0 {
			return "", fmt.Errorf("plan_type（执行方式）未指定，请补充：1-立即执行 2-定时执行 3-周期执行")
		}
		if call.Scope == 0 {
			return "", fmt.Errorf("scope（范围）未指定，请补充：1-特定主机 2-主机组 3-全网主机")
		}
		reporter.Step(ctx, "我正在编辑计划。")
		if err := s.edr.EditPlan(ctx, edr.EditPlanRequest{
			RID:      call.RID,
			PlanName: call.PlanName,
			ScanType: call.ScanType,
			PlanType: call.PlanType,
			Scope:    call.Scope,
			Type:     call.Type,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("计划 %s 编辑成功", call.RID), nil
	case "edr_plan_cancel":
		reporter.Step(ctx, "我正在取消计划。")
		if err := s.edr.CancelPlan(ctx, call.RID); err != nil {
			return "", err
		}
		return fmt.Sprintf("计划 %s 已取消", call.RID), nil
	case "edr_instruction_policy_add":
		reporter.Step(ctx, "我正在添加自动响应策略。")
		result, err := s.edr.AddInstructionPolicy(ctx, edr.AddInstructionPolicyRequest{
			Name: call.PlanName,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("自动响应策略 %s 创建成功", result.RID), nil
	case "edr_instruction_policy_update":
		reporter.Step(ctx, "我正在更新自动响应策略。")
		if err := s.edr.UpdateInstructionPolicy(ctx, edr.UpdateInstructionPolicyRequest{
			RID:  call.RID,
			Name: call.PlanName,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("自动响应策略 %s 更新成功", call.RID), nil
	case "edr_instruction_policy_delete":
		reporter.Step(ctx, "我正在删除自动响应策略。")
		if _, err := s.edr.DeleteInstructionPolicy(ctx, call.RID); err != nil {
			return "", err
		}
		return fmt.Sprintf("自动响应策略 %s 已删除", call.RID), nil
	case "edr_instruction_policy_save_status":
		reporter.Step(ctx, "我正在更新自动响应策略状态。")
		if _, err := s.edr.SaveInstructionPolicyStatus(ctx, edr.SaveInstructionPolicyStatusRequest{
			RID: call.RID,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("自动响应策略 %s 状态已更新", call.RID), nil
	case "edr_instruction_policy_sort":
		reporter.Step(ctx, "我正在排序自动响应策略。")
		// 需要解析rids，这里暂时用空的
		if err := s.edr.SortInstructionPolicies(ctx, nil); err != nil {
			return "", err
		}
		return "自动响应策略排序已保存", nil
	case "edr_ioa_add":
		reporter.Step(ctx, "我正在添加 IOA。")
		if err := s.edr.AddIOA(ctx, edr.AddIOARequest{
			CommandLine: call.Operation,
			Description: call.Reason,
			FileName:    call.IOCFileName,
			HostType:    call.IOCHostType,
			Severity:    call.KBQuery,
		}); err != nil {
			return "", err
		}
		return "IOA 添加成功", nil
	case "edr_ioa_update":
		reporter.Step(ctx, "我正在更新 IOA。")
		if err := s.edr.UpdateIOA(ctx, edr.UpdateIOARequest{
			ID:          call.IOCID,
			Description: call.Reason,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("IOA %s 更新成功", call.IOCID), nil
	case "edr_ioa_delete":
		reporter.Step(ctx, "我正在删除 IOA。")
		if err := s.edr.DeleteIOA(ctx, call.IOCID); err != nil {
			return "", err
		}
		return fmt.Sprintf("IOA %s 已删除", call.IOCID), nil
	case "edr_ioa_network_add":
		reporter.Step(ctx, "我正在添加 IOA 网络排除。")
		if err := s.edr.AddIOANetwork(ctx, edr.AddIOANetworkRequest{
			ExclusionName: call.PlanName,
			IP:            call.ClientIP,
			HostType:      call.IOCHostType,
		}); err != nil {
			return "", err
		}
		return "IOA 网络排除添加成功", nil
	case "edr_ioa_network_update":
		reporter.Step(ctx, "我正在更新 IOA 网络排除。")
		if err := s.edr.UpdateIOANetwork(ctx, edr.UpdateIOANetworkRequest{
			ID:            call.IOCID,
			ExclusionName: call.PlanName,
			IP:            call.ClientIP,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("IOA 网络排除 %s 更新成功", call.IOCID), nil
	case "edr_ioa_network_delete":
		reporter.Step(ctx, "我正在删除 IOA 网络排除。")
		if err := s.edr.DeleteIOANetwork(ctx, call.IOCID); err != nil {
			return "", err
		}
		return fmt.Sprintf("IOA 网络排除 %s 已删除", call.IOCID), nil
	case "edr_strategy_create":
		reporter.Step(ctx, "我正在创建策略。")
		result, err := s.edr.CreateStrategy(ctx, edr.CreateStrategyRequest{
			Name:      call.PlanName,
			Type:      call.Type,
			RangeType: call.Scope,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("策略创建成功，ID: %s", result.StrategyID), nil
	case "edr_strategy_update":
		reporter.Step(ctx, "我正在更新策略。")
		configContent := make(map[string]any)
		if call.ScanFileScope != "" {
			configContent["scan_file_scope"] = call.ScanFileScope
		}
		if call.StartupScanMode != "" {
			configContent["startup_scan_mode"] = call.StartupScanMode
		}
		if call.ArchiveSizeLimitEnabled != nil {
			configContent["archive_size_limit_enabled"] = *call.ArchiveSizeLimitEnabled
		}
		if call.ArchiveSizeLimit > 0 {
			configContent["archive_size_limit"] = call.ArchiveSizeLimit
		}
		if call.RealtimeMemCacheTechEnabled != nil {
			configContent["realtime_mem_cache_tech_enabled"] = *call.RealtimeMemCacheTechEnabled
		}
		if call.DynamicCpuMonitorEnabled != nil {
			configContent["dynamic_cpu_monitor_enabled"] = *call.DynamicCpuMonitorEnabled
		}
		if call.DynamicCpuHighPercent > 0 {
			configContent["dynamic_cpu_high_percent"] = call.DynamicCpuHighPercent
		}
		if call.StopRealtimeOnCpuHighEnabled != nil {
			configContent["stop_realtime_on_cpu_high_enabled"] = *call.StopRealtimeOnCpuHighEnabled
		}
		if call.StopRealtimeCpuHighPercent > 0 {
			configContent["stop_realtime_cpu_high_percent"] = call.StopRealtimeCpuHighPercent
		}
		if call.OwlOnRealtimeEnabled != nil {
			configContent["owl_on_realtime_enabled"] = *call.OwlOnRealtimeEnabled
		}
		if call.RealtimeScanArchiveEnabled != nil {
			configContent["realtime_scan_archive_enabled"] = *call.RealtimeScanArchiveEnabled
		}
		if call.RuntimeMaxFileSizeMb > 0 {
			configContent["runtime_max_file_size_mb"] = call.RuntimeMaxFileSizeMb
		}
		if call.CustomMaxFileSizeMb > 0 {
			configContent["custom_max_file_size_mb"] = call.CustomMaxFileSizeMb
		}
		configJSON, _ := json.Marshal(configContent)
		// fmt.Printf("=== configJSON is: %s", configJSON)
		if err := s.edr.UpdateStrategy(ctx, edr.UpdateStrategyRequest{
			StrategyID:    call.StrategyID,
			Name:          call.Name,
			Type:          call.Type,
			ConfigContent: string(configJSON),
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("策略 %s 更新成功", call.StrategyID), nil
	case "edr_strategy_delete":
		reporter.Step(ctx, "我正在删除策略。")
		if err := s.edr.DeleteStrategy(ctx, call.StrategyID, call.Type); err != nil {
			return "", err
		}
		return fmt.Sprintf("策略 %s 已删除", call.StrategyID), nil
	case "edr_strategy_status":
		reporter.Step(ctx, "我正在更新策略状态。")
		if err := s.edr.UpdateStrategyStatus(ctx, edr.UpdateStrategyStatusRequest{
			StrategyID: call.StrategyID,
			Type:       call.Type,
			Status:     call.Status,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("策略 %s 状态已更新", call.StrategyID), nil
	case "edr_host_offline_save":
		reporter.Step(ctx, "我正在保存主机离线配置。")
		if err := s.edr.SaveHostOfflineConf(ctx, edr.SaveHostOfflineConfRequest{
			Status: call.Status,
			Setting: edr.HostOfflineSetting{
				Timeout: call.Time,
			},
		}); err != nil {
			return "", err
		}
		return "主机离线配置已保存", nil
	case "edr_add_host_blacklist":
		reporter.Step(ctx, "我正在将主机加入黑名单。")
		clientIDs := strings.Split(strings.TrimSpace(call.ClientID), ",")
		cleaned := make([]string, 0, len(clientIDs))
		for _, id := range clientIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				cleaned = append(cleaned, id)
			}
		}
		if err := s.edr.AddHostBlacklist(ctx, cleaned, call.Reason); err != nil {
			return "", err
		}
		return fmt.Sprintf("已成功将 %d 台主机加入黑名单。", len(cleaned)), nil
	case "edr_remove_host":
		reporter.Step(ctx, "我正在从管控中移除主机。")
		clientIDs := strings.Split(strings.TrimSpace(call.ClientID), ",")
		cleaned := make([]string, 0, len(clientIDs))
		for _, id := range clientIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				cleaned = append(cleaned, id)
			}
		}
		if err := s.edr.RemoveHost(ctx, cleaned); err != nil {
			return "", err
		}
		return fmt.Sprintf("已成功从管控中移除 %d 台主机。", len(cleaned)), nil
	case "edr_update_detection_status":
		reporter.Step(ctx, "我正在更新检测状态。")
		if err := s.edr.UpdateDetectionStatus(ctx, edr.UpdateDetectionStatusRequest{
			IDs:        strings.Split(strings.TrimSpace(call.DetectionID), ","),
			DealStatus: call.Status,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("检测状态已更新，共 %d 个检测", len(strings.Split(strings.TrimSpace(call.DetectionID), ","))), nil
	case "edr_batch_deal_incident":
		reporter.Step(ctx, "我正在批量处置事件。")
		result, err := s.edr.BatchDealIncident(ctx, edr.BatchDealIncidentRequest{
			IDs:    strings.Split(strings.TrimSpace(call.IncidentID), ","),
			Allow:  call.Allow,
			Status: call.Status,
			Scene:  call.Scene,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("批量处置完成：%d 个事件已处理", result.TotalIncident), nil
	default:
		return "", fmt.Errorf("unsupported confirmed action: %s", call.Name)
	}
}

func isCriticalTool(name string) bool {
	switch name {
	case "edr_isolate", "edr_release", "edr_ioc_add", "edr_ioc_update", "edr_ioc_delete", "edr_delete_isolate_files", "edr_release_isolate_files", "edr_send_instruction", "edr_plan_add", "edr_plan_edit", "edr_plan_cancel", "edr_ioa_add", "edr_ioa_update", "edr_ioa_delete", "edr_ioa_network_add", "edr_ioa_network_update", "edr_ioa_network_delete", "edr_strategy_create", "edr_strategy_update", "edr_strategy_delete", "edr_strategy_status", "edr_host_offline_save", "edr_add_host_blacklist", "edr_remove_host", "edr_update_detection_status", "edr_batch_deal_incident":
		return true
	default:
		return false
	}
}

func (s *Service) handleNaturalLanguageEDR(ctx context.Context, sessionKey string, text string, locale string, reporter *progress.Reporter) (string, bool, error) {
	if s.router == nil {
		return "", false, nil
	}

	decision, err := s.router.Analyze(ctx, text)
	if err != nil {
		return "", false, nil
	}
	if decision.Action == "" || decision.Action == "none" {
		return "", false, nil
	}

	if decision.NeedsConfirmation {
		response := s.msg(locale, "write_action_hint", nil)
		response, err = s.storeAssistantReply(ctx, sessionKey, response)
		return response, true, err
	}

	response, err := s.executeNaturalLanguageEDR(ctx, sessionKey, text, decision, reporter)
	if err != nil {
		return "", true, err
	}
	return response, true, nil
}

func (s *Service) handleEDRCommand(ctx context.Context, sessionKey string, text string, locale string, reporter *progress.Reporter) (string, bool, error) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 || fields[0] != "/edr" {
		return "", false, nil
	}
	if len(fields) < 2 {
		return sanitizeReply(s.msg(locale, "commands_help", nil)), true, nil
	}

	var response string
	var err error

	switch fields[1] {
	case "hosts":
		reporter.Step(ctx, "我在通过 EDR 查这台主机的当前状态和基础信息。")
		keyword := ""
		if len(fields) > 2 {
			keyword = strings.Join(fields[2:], " ")
		}
		var result edr.ListHostsResponse
		result, err = s.edr.ListHosts(ctx, edr.ListHostsRequest{Hostname: keyword})
		if err == nil {
			response = formatHosts(result)
		}
	case "isolate":
		if len(fields) < 3 {
			response = s.msg(locale, "usage_isolate", nil)
			break
		}
		payload, _ := json.Marshal(planner.ToolCall{Name: "edr_isolate", ClientID: fields[2], Critical: true})
		err = s.store.SavePendingAction(ctx, sessionKey, "edr_isolate", string(payload), "edr_isolate client_id="+fields[2])
		if err == nil {
			response = s.msg(locale, "confirm_isolate", map[string]string{"client_id": fields[2]})
		}
	case "release":
		if len(fields) < 3 {
			response = s.msg(locale, "usage_release", nil)
			break
		}
		payload, _ := json.Marshal(planner.ToolCall{Name: "edr_release", ClientID: fields[2], Critical: true})
		err = s.store.SavePendingAction(ctx, sessionKey, "edr_release", string(payload), "edr_release client_id="+fields[2])
		if err == nil {
			response = s.msg(locale, "confirm_release", map[string]string{"client_id": fields[2]})
		}
	case "incidents":
		reporter.Step(ctx, "我在从平台 API 拉取近期事件，整理威胁和主机状态。")
		clientID, page, pageSize := parseIncidentListArgs(fields[2:], s.cfg.EDR.DefaultPageSize)
		var result edr.ListIncidentsResponse
		result, err = s.edr.ListIncidents(ctx, edr.ListIncidentsRequest{ClientID: clientID, Page: page, Limit: pageSize})
		if err == nil {
			response = formatIncidents(result, page, pageSize)
		}
	case "detections":
		reporter.Step(ctx, "我在从平台 API 拉取近期行为检出，看看最近有哪些高风险动作。")
		page, pageSize := parsePagedArgs(fields[2:], s.cfg.EDR.DefaultPageSize)
		var result edr.ListDetectionsResponse
		result, err = s.edr.ListDetections(ctx, edr.ListDetectionsRequest{Page: page, Limit: pageSize})
		if err == nil {
			response = formatDetections(result, page, pageSize)
		}
	case "logs":
		reporter.Step(ctx, "我在从平台 API 拉取行为日志，先把关键操作线索整理出来。")
		clientID, page, pageSize := parseIncidentListArgs(fields[2:], s.cfg.EDR.DefaultPageSize)
		var result edr.ListLogsResponse
		result, err = s.edr.ListLogs(ctx, edr.ListLogsRequest{ClientID: clientID, Page: page, PageSize: pageSize})
		if err == nil {
			response = s.formatLogs(ctx, result, page, pageSize, planner.ToolCall{ClientID: clientID, Page: page, PageSize: pageSize})
		}
	case "incident-view":
		reporter.Step(ctx, "我在拉取这条 incident 的详情。")
		if len(fields) < 4 {
			response = s.msg(locale, "usage_incident_view", nil)
			break
		}
		var result map[string]any
		result, err = s.edr.ViewIncident(ctx, edr.IncidentViewRequest{IncidentID: fields[2], ClientID: fields[3]})
		if err == nil {
			response, err = s.prepareDetailContext(ctx, sessionKey, locale, "incident", fields[2], fields[2], result, reporter)
		}
	case "detection-view":
		reporter.Step(ctx, "我在拉取这条 detection 的详情。")
		if len(fields) < 4 {
			response = s.msg(locale, "usage_detection_view", nil)
			break
		}
		viewType := ""
		processUUID := ""
		if len(fields) > 4 {
			viewType = fields[4]
		}
		if len(fields) > 5 {
			processUUID = fields[5]
		}
		var result map[string]any
		result, err = s.edr.ViewDetection(ctx, edr.DetectionViewRequest{DetectionID: fields[2], ClientID: fields[3], ViewType: viewType, ProcessUUID: processUUID})
		if err == nil {
			response, err = s.prepareDetailContext(ctx, sessionKey, locale, "detection", fields[2], fields[2], result, reporter)
		}
	default:
		response = s.msg(locale, "unknown_command", nil)
	}

	if err != nil {
		return "", true, err
	}
	reporter.Step(ctx, "这一步已经处理好了，我在把结果整理成更好读的说明。")
	response, err = s.storeAssistantReply(ctx, sessionKey, response)
	if err != nil {
		return "", true, err
	}
	if compactErr := s.compactor.MaybeCompact(ctx, sessionKey); compactErr != nil {
		s.logger.Warn("compact session failed", "session_key", sessionKey, "error", compactErr)
	}
	return response, true, nil
}

func (s *Service) executeNaturalLanguageEDR(ctx context.Context, sessionKey string, userText string, decision router.Decision, reporter *progress.Reporter) (string, error) {
	var toolResult string
	var err error

	switch decision.Action {
	case "hosts":
		reporter.Step(ctx, "我在按自然语言线索查主机状态和基础信息。")
		result, callErr := s.edr.ListHosts(ctx, edr.ListHostsRequest{Hostname: decision.Hostname, ClientIP: decision.ClientIP})
		err = callErr
		if err == nil {
			toolResult = formatHosts(result)
		}
	case "incidents":
		reporter.Step(ctx, "我在按你的问题拉取近期事件，并整理关键信息。")
		page := positiveOr(decision.Page, 1)
		pageSize := positiveOr(decision.PageSize, s.cfg.EDR.DefaultPageSize)
		result, callErr := s.edr.ListIncidents(ctx, edr.ListIncidentsRequest{ClientID: decision.ClientID, Page: page, Limit: pageSize})
		err = callErr
		if err == nil {
			toolResult = formatIncidents(result, page, pageSize)
		}
	case "detections":
		reporter.Step(ctx, "我在拉取近期行为检出，看看有哪些直接相关的风险线索。")
		page := positiveOr(decision.Page, 1)
		pageSize := positiveOr(decision.PageSize, s.cfg.EDR.DefaultPageSize)
		result, callErr := s.edr.ListDetections(ctx, edr.ListDetectionsRequest{Page: page, Limit: pageSize})
		err = callErr
		if err == nil {
			toolResult = formatDetections(result, page, pageSize)
		}
	case "logs":
		reporter.Step(ctx, "我在拉取行为日志，先把和你问题最相关的记录整理出来。")
		page := positiveOr(decision.Page, 1)
		pageSize := positiveOr(decision.PageSize, s.cfg.EDR.DefaultPageSize)
		result, callErr := s.edr.ListLogs(ctx, edr.ListLogsRequest{ClientID: decision.ClientID, Page: page, PageSize: pageSize})
		err = callErr
		if err == nil {
			toolResult = s.formatLogs(ctx, result, page, pageSize, planner.ToolCall{ClientID: decision.ClientID, Page: page, PageSize: pageSize})
		}
	case "tasks":
		reporter.Step(ctx, "我在拉取指令任务列表。")
		page := positiveOr(decision.Page, 1)
		pageSize := positiveOr(decision.PageSize, s.cfg.EDR.DefaultPageSize)
		result, callErr := s.edr.ListTasks(ctx, edr.ListTasksRequest{Page: page, Limit: pageSize})
		err = callErr
		if err == nil {
			toolResult = s.formatTasks(result, page, pageSize)
		}
	case "task_result":
		if decision.TaskID == "" {
			return "", fmt.Errorf("查询任务结果需要提供 task_id，请使用类似「查看任务结果 12345」或「task_id=12345」的格式")
		}
		reporter.Step(ctx, "我在拉取任务结果。")
		result, callErr := s.edr.GetTaskResult(ctx, decision.TaskID)
		err = callErr
		if err == nil {
			toolResult = formatTaskResult(result)
		}
	case "send_instruction":
		if decision.ClientID == "" {
			return "", fmt.Errorf("发送指令需要提供 client_id，请使用类似「发送指令 list_ps client_id=xxx」或「发送到 hostname=xxx」的格式")
		}
		if decision.InstructionName == "" {
			return "", fmt.Errorf("发送指令需要提供指令名称，如「发送指令 list_ps」")
		}
		reporter.Step(ctx, "我正在下发指令到目标主机。")
		req := edr.SendInstructionRequest{
			ClientID:        decision.ClientID,
			InstructionName: decision.InstructionName,
		}
		switch decision.InstructionName {
		case "list_ps":
			req.IsOnline = 1
		case "get_suspicious_file", "batch_quarantine_file", "batch_kill_ps":
			req.IsBatch = 1
			if decision.Path != "" {
				req.BatchParams = []edr.BatchParam{{Path: decision.Path}}
			}
		}
		result, callErr := s.edr.SendInstruction(ctx, req)
		err = callErr
		if err == nil {
			toolResult = fmt.Sprintf("指令已下发成功，任务ID: %s，主机: %s，重复: %t", result.TaskID, result.HostName, result.Repeat)
		}
	case "plan_list":
		reporter.Step(ctx, "我在拉取计划列表。")
		page := positiveOr(decision.Page, 1)
		pageSize := positiveOr(decision.PageSize, s.cfg.EDR.DefaultPageSize)
		result, callErr := s.edr.ListPlans(ctx, edr.ListPlansRequest{Page: page, Limit: pageSize, Type: "kill_plan"})
		err = callErr
		if err == nil {
			toolResult = formatPlans(result, page, pageSize)
		}
	case "virus_scan_record":
		reporter.Step(ctx, "我在拉取病毒扫描记录。")
		page := positiveOr(decision.Page, 1)
		pageSize := positiveOr(decision.PageSize, s.cfg.EDR.DefaultPageSize)
		result, callErr := s.edr.ListVirusScanRecords(ctx, edr.ListVirusScanRecordsRequest{HostName: decision.Hostname, ClientID: decision.ClientID, Page: page, Limit: pageSize})
		err = callErr
		if err == nil {
			toolResult = formatVirusScanRecords(result, page, pageSize)
		}
	case "virus_by_host":
		reporter.Step(ctx, "我在按主机查询病毒信息。")
		page := positiveOr(decision.Page, 1)
		pageSize := positiveOr(decision.PageSize, s.cfg.EDR.DefaultPageSize)
		result, callErr := s.edr.ListVirusByHost(ctx, edr.ListVirusByHostRequest{HostName: decision.Hostname, ClientID: decision.ClientID, Page: page, Limit: pageSize})
		err = callErr
		if err == nil {
			toolResult = formatVirusByHost(result, page, pageSize)
		}
	case "virus_by_hash":
		reporter.Step(ctx, "我在按哈希查询病毒信息。")
		page := positiveOr(decision.Page, 1)
		pageSize := positiveOr(decision.PageSize, s.cfg.EDR.DefaultPageSize)
		result, callErr := s.edr.ListVirusByHash(ctx, edr.ListVirusByHashRequest{Page: page, Limit: pageSize})
		err = callErr
		if err == nil {
			toolResult = formatVirusByHash(result, page, pageSize)
		}
	case "virus_hash_hosts":
		reporter.Step(ctx, "我在按哈希查询关联主机。")
		page := positiveOr(decision.Page, 1)
		pageSize := positiveOr(decision.PageSize, s.cfg.EDR.DefaultPageSize)
		result, callErr := s.edr.ListVirusHashHosts(ctx, edr.ListVirusHashHostsRequest{HostName: decision.Hostname, ClientID: decision.ClientID, Page: page, Limit: pageSize})
		err = callErr
		if err == nil {
			toolResult = formatVirusHashHosts(result, page, pageSize)
		}
	case "plan_add":
		if decision.PlanName == "" {
			return "", fmt.Errorf("创建计划需要提供计划名称（plan_name），请使用类似「创建计划 plan_name=xxx」的格式")
		}
		if decision.ScanType == 0 {
			return "", fmt.Errorf("创建计划需要提供操作类型（scan_type）：1-快速扫描 2-全盘扫描 3-自定义路径扫描 4-漏洞修复 5-安装软件 6-卸载软件 7-更新软件 8-发送文件，请使用类似「创建计划 plan_name=xxx scan_type=1」的格式")
		}
		if decision.PlanType == 0 {
			return "", fmt.Errorf("创建计划需要提供执行方式（plan_type）：1-立即执行 2-定时执行 3-周期执行，请使用类似「创建计划 plan_name=xxx scan_type=1 plan_type=1」的格式")
		}
		if decision.Scope == 0 {
			return "", fmt.Errorf("创建计划需要提供范围（scope）：1-特定主机 2-主机组 3-全网主机，请使用类似「创建计划 plan_name=xxx scan_type=1 plan_type=1 scope=1」的格式")
		}
		reporter.Step(ctx, "我正在创建计划。")
		callErr := s.edr.AddPlan(ctx, edr.AddPlanRequest{
			PlanName: decision.PlanName,
			ScanType: decision.ScanType,
			PlanType: decision.PlanType,
			Scope:    decision.Scope,
			Type:     "kill_plan",
		})
		err = callErr
		if err == nil {
			toolResult = fmt.Sprintf("计划「%s」创建成功", decision.PlanName)
		}
	case "plan_edit":
		if decision.RID == "" {
			return "", fmt.Errorf("编辑计划需要提供计划ID（rid），请使用类似「编辑计划 rid=xxx」的格式")
		}
		if decision.ScanType == 0 {
			return "", fmt.Errorf("编辑计划需要提供操作类型（scan_type）：1-快速扫描 2-全盘扫描 3-自定义路径扫描 4-漏洞修复 5-安装软件 6-卸载软件 7-更新软件 8-发送文件，请使用类似「编辑计划 rid=xxx scan_type=1」的格式")
		}
		if decision.PlanType == 0 {
			return "", fmt.Errorf("编辑计划需要提供执行方式（plan_type）：1-立即执行 2-定时执行 3-周期执行，请使用类似「编辑计划 rid=xxx scan_type=1 plan_type=1」的格式")
		}
		if decision.Scope == 0 {
			return "", fmt.Errorf("编辑计划需要提供范围（scope）：1-特定主机 2-主机组 3-全网主机，请使用类似「编辑计划 rid=xxx scan_type=1 plan_type=1 scope=1」的格式")
		}
		if decision.Type == "" {
			return "", fmt.Errorf("编辑计划需要提供业务类型（type）：kill_plan/leak_repair/distribute_software/distribute_file，请使用类似「编辑计划 rid=xxx scan_type=1 plan_type=1 scope=1 type=kill_plan」的格式")
		}
		reporter.Step(ctx, "我正在编辑计划。")
		callErr := s.edr.EditPlan(ctx, edr.EditPlanRequest{
			RID:      decision.RID,
			PlanName: decision.PlanName,
			ScanType: decision.ScanType,
			PlanType: decision.PlanType,
			Scope:    decision.Scope,
			Type:     decision.Type,
		})
		err = callErr
		if err == nil {
			toolResult = fmt.Sprintf("计划 %s 编辑成功", decision.RID)
		}
	case "plan_cancel":
		if decision.RID == "" {
			return "", fmt.Errorf("取消计划需要提供计划ID（rid），请使用类似「取消计划 rid=xxx」的格式")
		}
		reporter.Step(ctx, "我正在取消计划。")
		callErr := s.edr.CancelPlan(ctx, decision.RID)
		err = callErr
		if err == nil {
			toolResult = fmt.Sprintf("计划 %s 已取消", decision.RID)
		}
	default:
		return "", fmt.Errorf("unsupported routed edr action: %s", decision.Action)
	}
	if err != nil {
		return "", err
	}

	reporter.Step(ctx, "我拿到真实 EDR 数据了，正在整理成更贴近你问题的回答。")
	response, err := s.answerGroundedByEDR(ctx, sessionKey, userText, toolResult)
	if err != nil {
		response = toolResult
	}
	response, err = s.storeAssistantReply(ctx, sessionKey, response)
	if err != nil {
		return "", err
	}
	if compactErr := s.compactor.MaybeCompact(ctx, sessionKey); compactErr != nil {
		s.logger.Warn("compact session failed", "session_key", sessionKey, "error", compactErr)
	}
	return response, nil
}

func (s *Service) answerGroundedByEDR(ctx context.Context, sessionKey string, userText string, toolResult string) (string, error) {
	messages, err := s.buildBaseMessages(ctx, sessionKey)
	if err != nil {
		return "", err
	}
	groundedPrompt := "You will receive real EDR tool output. Answer strictly based on that real output. Limited interpretation is allowed, but do not invent missing fields, API results, audit details, execution steps, or system states. If the data is insufficient, say so clearly. Use plain text only. Do not use markdown headings, bold markers, tables, fenced code blocks, or report-style sections. Follow the reply-style rules already given in the conversation context."
	if s.prompt != nil {
		if loaded := s.prompt.LoadPrompt("grounded_edr_answer"); strings.TrimSpace(loaded) != "" {
			groundedPrompt = loaded
		}
		groundedPrompt = s.prompt.ComposeSystemPrompt(groundedPrompt)
	}
	messages = append(messages,
		model.Message{Role: model.RoleSystem, Content: groundedPrompt},
		model.Message{Role: model.RoleUser, Content: "Current user question:\n" + strings.TrimSpace(userText) + "\n\nReal EDR result:\n" + strings.TrimSpace(toolResult) + "\n\nAnswer based only on the real result above."},
	)
	result, err := s.model.Chat(ctx, model.ChatRequest{SessionKey: sessionKey, Messages: messages}, nil)
	if err != nil {
		return "", err
	}
	s.logger.Info("grounded answer generated", "session_key", sessionKey, "preview", shortResult(result.Text))
	return result.Text, nil
}

func (s *Service) buildBaseMessages(ctx context.Context, sessionKey string) ([]model.Message, error) {
	summary, err := s.store.GetSessionSummary(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	turns, err := s.store.ListRecentTurns(ctx, sessionKey, s.cfg.Session.MaxRecentTurns)
	if err != nil {
		return nil, err
	}
	memories, err := s.memory.ListForContext(ctx, sessionKey)
	if err != nil {
		return nil, err
	}
	skillsPrompt := ""
	replyStylePrompt := ""
	if s.prompt != nil {
		skillsPrompt = s.prompt.LoadSkillsPrompt()
		replyStylePrompt = s.prompt.LoadReplyStylePrompt()
	}
	memoryParts := make([]string, 0, len(memories))
	for _, item := range memories {
		memoryParts = append(memoryParts, fmt.Sprintf("- %s=%s", item.Key, item.Value))
	}
	now := time.Now()
	mainPrompt := strings.TrimSpace("You are the main chat controller for EDR operations. Follow the reply-style JSON to decide whether the answer should be in Chinese or English, and keep the tone natural. Use plain text only, without markdown, tables, fenced blocks, or pseudo system tags.\n\nCurrent time:\n- local: " + now.Format("2006-01-02 15:04:05 MST") + "\n- utc: " + now.UTC().Format(time.RFC3339) + "\n\nHard rule: you may describe API results, task receipts, audit details, execution steps, or system state only when the system has actually executed a tool and returned real data. Never invent interface responses, JSON payloads, audit ids, risk levels, or execution logs.\n\nReply style JSON:\n" + replyStylePrompt + "\n\nSkills:\n" + skillsPrompt + "\n\nLong-term memory:\n" + strings.Join(memoryParts, "\n") + "\n\nConversation summary:\n" + summary)
	if s.prompt != nil {
		mainPrompt = s.prompt.ComposeSystemPrompt(mainPrompt)
	}
	messages := []model.Message{{
		Role:    model.RoleSystem,
		Content: mainPrompt,
	}}
	for _, turn := range turns {
		messages = append(messages, model.Message{Role: model.Role(turn.Role), Content: turn.Content})
	}
	return messages, nil
}

func formatHosts(result edr.ListHostsResponse) string {
	if len(result.Hosts) == 0 {
		return "没有查到匹配主机。"
	}

	lines := []string{fmt.Sprintf("共找到 %d 台主机，展示前 %d 台：", result.Total, len(result.Hosts))}
	for _, host := range result.Hosts {
		lines = append(lines, fmt.Sprintf("- %s client_id=%s ip=%s status=%s user=%s", host.Hostname, host.ClientID, host.ClientIP, host.Status, host.Username))
	}
	return strings.Join(lines, "\n")
}

func formatIncidents(result edr.ListIncidentsResponse, page int, pageSize int) string {
	if len(result.Incidents) == 0 {
		return "近期没有查到匹配事件。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Incidents))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条事件，当前第 %d/%d 页，本页 %d 条（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Incidents), pageSize, hasMore)}
	for _, incident := range result.Incidents {
		lines = append(lines, fmt.Sprintf("- %s incident_id=%s score=%.1f host=%s client_id=%s status=%d", incident.IncidentName, incident.IncidentID, incident.Score, incident.HostName, incident.ClientID, incident.Status))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr incidents [%s]<page> [page_size]，例如 /edr incidents %d %d", formatOptionalClientPrefix(result.Incidents[0].ClientID), minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatIncidentR2Summary(result edr.IncidentR2SummaryResponse) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("事件 R2 摘要 - %s (ID: %s)", result.IncidentName, result.IncidentID))
	lines = append(lines, fmt.Sprintf("状态: %d | 评分: %.1f | 主机: %s", result.Status, result.Score, result.HostName))
	lines = append(lines, fmt.Sprintf("客户端: %s | IP: %s | 用户: %s", result.ClientID, result.ExternalIP, result.Username))
	lines = append(lines, fmt.Sprintf("操作系统: %s | 客户端版本: %s | 隔离状态: %d", result.OperatingSystem, result.ClientVersion, result.Isolation))
	if len(result.TNames) > 0 {
		lines = append(lines, fmt.Sprintf("威胁名称: %s", strings.Join(result.TNames, ", ")))
	}
	if len(result.Tags) > 0 {
		lines = append(lines, fmt.Sprintf("标签: %s", strings.Join(result.Tags, ", ")))
	}
	return strings.Join(lines, "\n")
}

func formatDetections(result edr.ListDetectionsResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到检测记录。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条检测记录，当前第 %d/%d 页，本页 %d 条（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Results), pageSize, hasMore)}
	for i, item := range result.Results {
		if i >= 10 {
			lines = append(lines, fmt.Sprintf("... 还有 %d 条记录", len(result.Results)-10))
			break
		}
		lines = append(lines, fmt.Sprintf("- host=%s detection_id=%s level=%s status=%d rootname=%s client_id=%s incident_id=%s", item.HostName, item.DetectionID, item.ThreatLevel, item.DealStatus, item.RootName, item.ClientID, item.IncidentID))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr detections %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatEventLogAlarms(result edr.ListEventLogAlarmsResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到事件日志告警。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条事件日志告警，当前第 %d/%d 页，本页 %d 条（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Results), pageSize, hasMore)}
	for _, alarm := range result.Results {
		lines = append(lines, fmt.Sprintf("- id=%s name=%s risk=%s client_id=%s log_num=%d", alarm.ID, alarm.Name, alarm.RiskLevel, alarm.ClientID, alarm.LogNum))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr event_log_alarms %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func (*Service) formatIOCs(ctx context.Context, result edr.ListIOCsResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到 IOC。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条 IOC，当前第 %d/%d 页，本页 %d 条（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Results), pageSize, hasMore)}
	for _, ioc := range result.Results {
		lines = append(lines, fmt.Sprintf("- id=%s hash=%s action=%s filename=%s desc=%s", ioc.ExclusionID, ioc.Hash, ioc.Action, ioc.FileName, ioc.Description))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr iocs %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func (*Service) formatIsolateFiles(result edr.ListIsolateFilesResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到隔离文件。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条隔离文件，当前第 %d/%d 页，本页 %d 条（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Results), pageSize, hasMore)}
	for _, f := range result.Results {
		status := "未知"
		if f.RecoverStatus == 1 {
			status = "已隔离"
		} else if f.RecoverStatus == 2 {
			status = "已释放"
		} else if f.RecoverStatus == 3 {
			status = "已清除"
		}
		lines = append(lines, fmt.Sprintf("- GUID=%s 主机=%s 文件=%s MD5=%s SHA1=%s 状态=%s ClientID=%s 组织=%s", f.GUID, f.Hostname, f.FileName, f.MD5, f.SHA1, status, f.ClientID, f.OrgName))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr isolate_files %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatPlans(result edr.ListPlansResponse, page int, pageSize int) string {
	if len(result.Items) == 0 {
		return "当前没有查到计划。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Items))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 个计划，当前第 %d/%d 页，本页 %d 个（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Items), pageSize, hasMore)}
	for _, plan := range result.Items {
		statusStr := "未知"
		switch plan.Status {
		case 0:
			statusStr = "未执行"
		case 1:
			statusStr = "执行中"
		case 2:
			statusStr = "已完成"
		case 3:
			statusStr = "已取消"
		}
		scanTypeStr := "未知"
		switch plan.ScanType {
		case 1:
			scanTypeStr = "快速扫描"
		case 2:
			scanTypeStr = "全盘扫描"
		case 3:
			scanTypeStr = "自定义路径扫描"
		case 4:
			scanTypeStr = "漏洞修复"
		case 5:
			scanTypeStr = "安装软件"
		case 6:
			scanTypeStr = "卸载软件"
		case 7:
			scanTypeStr = "更新软件"
		case 8:
			scanTypeStr = "发送文件"
		}
		lines = append(lines, fmt.Sprintf("- rid=%s name=%s type=%s scan_type=%s scope=%d status=%d(%s) user=%s", plan.RID, plan.PlanName, plan.Type, scanTypeStr, plan.Scope, plan.Status, statusStr, plan.OperationUser))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr plan_list %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatVirusScanRecords(result edr.ListVirusScanRecordsResponse, page, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到病毒扫描记录。"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条病毒扫描记录（第 %d 页，每页 %d 条）：", result.Total, page, pageSize)}
	for _, record := range result.Results {
		statusStr := "未知"
		switch record.Status {
		case 0:
			statusStr = "待执行"
		case 1:
			statusStr = "执行中"
		case 2:
			statusStr = "执行完成"
		case 3:
			statusStr = "执行失败"
		}
		lines = append(lines, fmt.Sprintf("- task_id=%s hostname=%s scan_type=%s status=%d(%s) virus_file_num=%d memory_virus_num=%d", record.TaskID, record.HostName, record.ScanType, record.Status, statusStr, record.VirusFileNum, record.MemoryVirusNum))
	}
	return strings.Join(lines, "\n")
}

func formatInstructionPolicies(result edr.ListInstructionPoliciesResponse) string {
	if len(result.Result) == 0 {
		return "当前没有查到自动响应策略。"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条自动响应策略：", len(result.Result))}
	for _, policy := range result.Result {
		statusStr := "未知"
		switch policy.Status {
		case 1:
			statusStr = "启用"
		case 2:
			statusStr = "禁用"
		}
		policyTypeStr := "内置策略"
		if policy.PolicyType == 2 {
			policyTypeStr = "自定义策略"
		}
		scopeStr := "未知"
		switch policy.Scope {
		case 1:
			scopeStr = "特定主机"
		case 2:
			scopeStr = "主机组"
		case 3:
			scopeStr = "全网"
		}
		lines = append(lines, fmt.Sprintf("- rid=%s name=%s type=%s status=%d(%s) scope=%d(%s) user=%s", policy.RID, policy.Name, policyTypeStr, policy.Status, statusStr, policy.Scope, scopeStr, policy.OperationUser))
	}
	return strings.Join(lines, "\n")
}

func formatVirusByHost(result edr.ListVirusByHostResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到染毒主机。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 台染毒主机，当前第 %d/%d 页，本页 %d 台（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Results), pageSize, hasMore)}
	for _, host := range result.Results {
		statusStr := "未知"
		switch host.Status {
		case 0:
			statusStr = "未处理"
		case 1:
			statusStr = "已处理"
		case 2:
			statusStr = "已忽略"
		}
		lines = append(lines, fmt.Sprintf("- hostname=%s client_id=%s virus_file=%d virus_mem=%d status=%d(%s) last_check=%d", host.HostName, host.ClientID, host.VirusFileCount, host.VirusMemoryCount, host.Status, statusStr, host.LastCheckedTime))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr virus_by_host %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatVirusByHash(result edr.ListVirusByHashResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到病毒哈希信息。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条病毒哈希，当前第 %d/%d 页，本页 %d 条（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Results), pageSize, hasMore)}
	for _, hash := range result.Results {
		lines = append(lines, fmt.Sprintf("- id=%s name=%s md5=%s sha1=%s host_count=%d", hash.ID, hash.Name, hash.MD5, hash.SHA1, hash.HostCount))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr virus_by_hash %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatVirusHashHosts(result edr.ListVirusHashHostsResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到哈希关联的主机。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 台关联主机，当前第 %d/%d 页，本页 %d 台（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Results), pageSize, hasMore)}
	for _, host := range result.Results {
		lines = append(lines, fmt.Sprintf("- hostname=%s client_id=%s sha1=%s path=%s virus_file=%d virus_mem=%d", host.HostName, host.ClientID, host.SHA1, host.Path, host.VirusFileCount, host.VirusMemoryCount))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr virus_hash_hosts %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func (*Service) formatTasks(result edr.ListTasksResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到指令任务。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条指令任务，当前第 %d/%d 页，本页 %d 条（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Results), pageSize, hasMore)}
	for _, task := range result.Results {
		statusStr := "未知"
		switch task.Status {
		case 0:
			statusStr = "下发中"
		case 1:
			statusStr = "执行中"
		case 2:
			statusStr = "已完成"
		case 3:
			statusStr = "失败"
		case 4:
			statusStr = "超时"
		}
		lines = append(lines, fmt.Sprintf("- id=%s hostname=%s instruction=%s status=%d(%s) user=%s", task.ID, task.HostName, task.InstructionName, task.Status, statusStr, task.OperationUser))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr tasks %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatTaskResult(result edr.TaskResult) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("任务结果 - hostname=%s instruction=%s", result.HostName, result.InstructionName))
	if result.Message != "" {
		lines = append(lines, fmt.Sprintf("消息：%s", result.Message))
	}
	if len(result.Process) > 0 {
		lines = append(lines, fmt.Sprintf("进程数量：%d", len(result.Process)))
		for _, p := range result.Process {
			sig := "无签名"
			if p.Signature != "" {
				sig = p.Signature
			}
			lines = append(lines, fmt.Sprintf("  - pid=%d name=%s path=%s sha1=%s signature=%s", p.PID, p.PName, p.Path, p.SHA1, sig))
		}
	}
	if len(result.ImageDetail) > 0 {
		lines = append(lines, fmt.Sprintf("镜像数量：%d", len(result.ImageDetail)))
		for _, img := range result.ImageDetail {
			sys := "否"
			if img.IsSystem == 1 {
				sys = "是"
			}
			lines = append(lines, fmt.Sprintf("  - path=%s sha1=%s system=%s", img.ImagePath, img.ImageSHA1, sys))
		}
	}
	if len(result.ProcessDetail) > 0 {
		lines = append(lines, fmt.Sprintf("进程详情数量：%d", len(result.ProcessDetail)))
	}
	if len(lines) == 1 {
		return "任务结果为空"
	}
	return strings.Join(lines, "\n")
}

func formatIOAs(result edr.ListIOAsResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到 IOA。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条 IOA，当前第 %d/%d 页，本页 %d 条（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), len(result.Results), pageSize, hasMore)}
	for _, ioa := range result.Results {
		lines = append(lines, fmt.Sprintf("- id=%s name=%s severity=%s file=%s cmd=%s", ioa.ExclusionID, ioa.IOAName, ioa.Severity, ioa.FileName, ioa.CommandLine))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr ioa_list %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatIOAAuditLogs(result edr.ListIOAAuditLogsResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到 IOA 活动记录。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条 IOA 活动记录，当前第 %d/%d 页（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), pageSize, hasMore)}
	for _, log := range result.Results {
		lines = append(lines, fmt.Sprintf("- id=%s ioa=%s hostname=%s file=%s cmd=%s time=%d", log.ID, log.IOAName, log.HostName, log.FileName, log.CommandLine, log.EventTime))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr ioa_audit_log %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatIOANetworks(result edr.ListIOANetworksResponse, page int, pageSize int) string {
	if len(result.Results) == 0 {
		return "当前没有查到 IOA 网络排除。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Results))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条 IOA 网络排除，当前第 %d/%d 页（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), pageSize, hasMore)}
	for _, net := range result.Results {
		lines = append(lines, fmt.Sprintf("- id=%s name=%s ip=%s host_type=%s", net.ID, net.ExclusionName, net.IP, net.HostType))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr ioa_network_list %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatStrategies(result edr.ListStrategiesResponse, page int, pageSize int) string {
	if len(result.Items) == 0 {
		return "当前没有查到策略。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Items))
	totalPages := calcTotalPages(result.Total, pageSize)
	hasMore := "否"
	if totalPages > 0 && page < totalPages {
		hasMore = "是"
	}
	lines := []string{fmt.Sprintf("共找到 %d 条策略，当前第 %d/%d 页（page_size=%d，has_more=%s）：", result.Total, page, maxInt(totalPages, 1), pageSize, hasMore)}
	for _, strategy := range result.Items {
		statusStr := "未知"
		if strategy.Status == 1 {
			statusStr = "启用"
		} else if strategy.Status == 0 {
			statusStr = "禁用"
		}
		lines = append(lines, fmt.Sprintf("- id=%s name=%s type=%s status=%d(%s) range_type=%d", strategy.StrategyID, strategy.Name, strategy.Type, strategy.Status, statusStr, strategy.RangeType))
	}
	if totalPages > 1 {
		lines = append(lines, fmt.Sprintf("翻页示例：/edr strategy_list %d %d", minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func formatStrategySingle(result edr.Strategy) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("策略详情 - id=%s name=%s type=%s", result.StrategyID, result.Name, result.Type))
	if result.Status == 1 {
		lines = append(lines, "状态：启用")
	} else if result.Status == 0 {
		lines = append(lines, "状态：禁用")
	}
	if result.Content != "" {
		lines = append(lines, fmt.Sprintf("内容：%s", result.Content))
	}
	if result.ConfigContent != "" {
		lines = append(lines, fmt.Sprintf("配置：%s", result.ConfigContent))
	}
	return strings.Join(lines, "\n")
}

func formatStrategyState(result edr.StrategyState) string {
	var lines []string
	lines = append(lines, "策略状态统计：")
	lines = append(lines, fmt.Sprintf("- 总策略数：%d", result.AllStrategy))
	lines = append(lines, fmt.Sprintf("- 活跃策略数：%d", result.ActiveStrategy))
	lines = append(lines, fmt.Sprintf("- 告警终端数：%d", result.AlarmTerminalCount))
	lines = append(lines, fmt.Sprintf("- 封禁互联网访问数：%d", result.BanInternetAccess))
	lines = append(lines, fmt.Sprintf("- 检测周期：%d", result.DetectionPeriod))
	return strings.Join(lines, "\n")
}

func formatHostOfflineConf(result edr.HostOfflineConf) string {
	var lines []string
	lines = append(lines, "主机离线配置：")
	if result.Status == 1 {
		lines = append(lines, "状态：开启")
	} else if result.Status == 2 {
		lines = append(lines, "状态：关闭")
	}
	if result.Setting.Timeout > 0 {
		lines = append(lines, fmt.Sprintf("离线超时时间：%d 天", result.Setting.Timeout))
	}
	lines = append(lines, fmt.Sprintf("组织：%s", result.OrgName))
	return strings.Join(lines, "\n")
}

func (s *Service) formatLogs(ctx context.Context, result edr.ListLogsResponse, page int, pageSize int, call planner.ToolCall) string {
	if len(result.Logs) == 0 {
		return "近期没有查到匹配行为日志。"
	}
	page = positiveOr(page, 1)
	pageSize = positiveOr(pageSize, len(result.Logs))
	totalPages := calcTotalPages(result.Total, pageSize)
	head := fmt.Sprintf("共找到 %d 条行为日志，当前第 %d/%d 页，本页 %d 条：", result.Total, page, maxInt(totalPages, 1), len(result.Logs))
	filters := describeLogFilters(call)
	if filters != "" {
		head += "\n筛选条件：" + filters
	}
	hostMap := s.resolveHostnamesForLogs(ctx, result.Logs)
	lines := []string{head}
	for _, item := range result.Logs {
		operation := stringifyLogValue(item["operation"])
		processName := firstNonEmptyStringify(item["process_name"], item["process"], item["command_line"])
		newProcess := firstNonEmptyStringify(item["new_process_name"], item["newprocess"], item["newcommandline"])
		clientID := stringifyLogValue(item["client_id"])
		hostName := firstNonEmptyStringify(item["host_name"], item["hostname"], hostMap[clientID])
		osType := stringifyLogValue(item["os_type"])
		timestamp := firstNonEmptyStringify(item["timestamp"], item["time"])
		line := fmt.Sprintf("- os=%s operation=%s process=%s", blankToDash(osType), blankToDash(operation), blankToDash(processName))
		if strings.TrimSpace(newProcess) != "" {
			line += fmt.Sprintf(" -> %s", newProcess)
		}
		if strings.TrimSpace(hostName) != "" {
			line += fmt.Sprintf(" host=%s client_id=%s", hostName, blankToDash(clientID))
		} else {
			line += fmt.Sprintf(" host=- client_id=%s", blankToDash(clientID))
		}
		line += fmt.Sprintf(" time=%s", blankToDash(timestamp))
		lines = append(lines, line)
	}
	if totalPages > 1 {
		clientPrefix := formatOptionalClientPrefix(call.ClientID)
		lines = append(lines, fmt.Sprintf("翻页示例：/edr logs [%s]%d %d", clientPrefix, minInt(page+1, totalPages), pageSize))
	}
	return strings.Join(lines, "\n")
}

func (s *Service) resolveHostnamesForLogs(ctx context.Context, logs []map[string]any) map[string]string {
	result := make(map[string]string)
	if s == nil || s.edr == nil || len(logs) == 0 {
		return result
	}
	need := make(map[string]struct{})
	for _, item := range logs {
		clientID := strings.TrimSpace(stringifyLogValue(item["client_id"]))
		if clientID == "" {
			continue
		}
		if firstNonEmptyStringify(item["host_name"], item["hostname"]) != "" {
			continue
		}
		need[clientID] = struct{}{}
	}
	if len(need) == 0 {
		return result
	}
	page := 1
	limit := 100
	for attempts := 0; attempts < 20 && len(need) > 0; attempts++ {
		hosts, err := s.edr.ListHosts(ctx, edr.ListHostsRequest{Page: page, Limit: limit})
		if err != nil {
			return result
		}
		for _, host := range hosts.Hosts {
			if _, ok := need[host.ClientID]; !ok {
				continue
			}
			result[host.ClientID] = strings.TrimSpace(host.Hostname)
			delete(need, host.ClientID)
		}
		if hosts.Pages <= 0 || page >= hosts.Pages {
			break
		}
		page++
	}
	return result
}

func describeLogFilters(call planner.ToolCall) string {
	parts := make([]string, 0, 5)
	if strings.TrimSpace(call.ClientID) != "" {
		parts = append(parts, "client_id="+call.ClientID)
	}
	if strings.TrimSpace(call.OSType) != "" {
		parts = append(parts, "os_type="+call.OSType)
	}
	if strings.TrimSpace(call.Operation) != "" {
		parts = append(parts, "operation="+call.Operation)
	}
	if strings.TrimSpace(call.StartTime) != "" {
		parts = append(parts, "start_time="+call.StartTime)
	}
	if strings.TrimSpace(call.EndTime) != "" {
		parts = append(parts, "end_time="+call.EndTime)
	}
	if strings.TrimSpace(call.FilterField) != "" && strings.TrimSpace(call.FilterValue) != "" {
		op := strings.TrimSpace(call.FilterOp)
		if op == "" {
			op = "is"
		}
		parts = append(parts, fmt.Sprintf("%s %s %s", call.FilterField, op, call.FilterValue))
	}
	return strings.Join(parts, "; ")
}

func stringifyLogValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		if value == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func firstNonEmptyStringify(values ...any) string {
	for _, value := range values {
		text := stringifyLogValue(value)
		if strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

func (s *Service) prepareDetailContext(ctx context.Context, sessionKey string, locale string, kind string, title string, query string, payload map[string]any, reporter *progress.Reporter) (string, error) {
	if len(payload) == 0 {
		return s.msg(locale, "detail_empty", map[string]string{"kind": kind}), nil
	}
	if s.artifacts == nil {
		return s.formatViewDetail(locale, kind, payload), nil
	}
	item, payloadText, err := s.artifacts.SaveJSON(ctx, sessionKey, kind, title, payload)
	if err != nil {
		return s.formatViewDetail(locale, kind, payload), nil
	}
	if s.canInlineArtifact(ctx, sessionKey, item, payloadText) {
		return s.formatViewDetail(locale, kind, payload), nil
	}
	if s.detailer != nil && s.detailer.Enabled() {
		reporter.Step(ctx, "这条详情比较大，我先让辅助分析器提炼一版摘要给主流程继续看。")
		overview := artifact.BuildOverview(payload)
		report, summaryErr := s.detailer.SummarizeDetail(ctx, item, overview, payloadText, query)
		if summaryErr == nil && strings.TrimSpace(report.Summary) != "" {
			return formatDetailAgentReport(item, report, "detail"), nil
		}
	}
	overview := artifact.BuildOverview(payload)
	excerpt := artifact.BuildSelectiveContext(payloadText, query, 60)
	return s.msg(locale, "detail_large", map[string]string{"kind": kind, "artifact_id": item.ArtifactID, "overview": overview, "excerpt": excerpt}), nil
}

func (s *Service) formatViewDetail(locale string, kind string, payload map[string]any) string {
	if len(payload) == 0 {
		return s.msg(locale, "detail_empty", map[string]string{"kind": kind})
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return s.msg(locale, "detail_raw_fail", map[string]string{"kind": kind})
	}
	return s.msg(locale, "detail_raw_head", map[string]string{"kind": kind}) + "\n" + string(body)
}

func userHint(call planner.ToolCall) string {
	parts := []string{call.IncidentID, call.DetectionID, call.ClientID, call.ViewType, call.ProcessUUID}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func parseIncidentListArgs(args []string, defaultPageSize int) (string, int, int) {
	if len(args) == 0 {
		return "", 1, positiveOr(defaultPageSize, 10)
	}
	clientID := ""
	if _, err := strconv.Atoi(strings.TrimSpace(args[0])); err != nil {
		clientID = strings.TrimSpace(args[0])
		args = args[1:]
	}
	page, pageSize := parsePagedArgs(args, defaultPageSize)
	return clientID, page, pageSize
}

func parsePagedArgs(args []string, defaultPageSize int) (int, int) {
	page := 1
	pageSize := positiveOr(defaultPageSize, 10)
	if len(args) > 0 {
		if value, err := strconv.Atoi(strings.TrimSpace(args[0])); err == nil && value > 0 {
			page = value
		}
	}
	if len(args) > 1 {
		if value, err := strconv.Atoi(strings.TrimSpace(args[1])); err == nil && value > 0 {
			pageSize = value
		}
	}
	return page, pageSize
}

func positiveOr(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func calcTotalPages(total int, pageSize int) int {
	pageSize = positiveOr(pageSize, 1)
	if total <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

func formatOptionalClientPrefix(clientID string) string {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return ""
	}
	return clientID + " "
}

func (s *Service) formatArtifactSearch(ctx context.Context, locale string, item protocol.Artifact, query string, matches []protocol.ArtifactMatch, reporter *progress.Reporter) string {
	if item.ArtifactID == "" {
		return s.msg(locale, "artifact_none", nil)
	}
	if len(matches) == 0 {
		return s.msg(locale, "artifact_search_none", map[string]string{"artifact_id": item.ArtifactID, "query": query})
	}
	if s.shouldUseDetailAgentForArtifact(ctx, item) && s.detailer != nil && s.detailer.Enabled() {
		reporter.Step(ctx, "这份大详情我先交给辅助分析器解释命中的线索。")
		raw := s.renderArtifactSearchMatches(locale, item, query, matches)
		report, err := s.detailer.SummarizeSearch(ctx, item, query, raw)
		if err == nil && strings.TrimSpace(report.Summary) != "" {
			return formatDetailAgentReport(item, report, "search")
		}
	}
	return s.renderArtifactSearchMatches(locale, item, query, matches)
}

func (s *Service) renderArtifactSearchMatches(locale string, item protocol.Artifact, query string, matches []protocol.ArtifactMatch) string {
	lines := []string{s.msg(locale, "artifact_search_head", map[string]string{"artifact_id": item.ArtifactID, "query": query})}
	for _, match := range matches {
		lines = append(lines, s.msg(locale, "artifact_search_line", map[string]string{"line": fmt.Sprintf("%d", match.Line), "snippet": match.Snippet}))
	}
	return strings.Join(lines, "\n")
}

func (s *Service) formatArtifactRead(ctx context.Context, locale string, item protocol.Artifact, startLine int, lineCount int, chunk string, reporter *progress.Reporter) string {
	if item.ArtifactID == "" {
		return s.msg(locale, "artifact_none", nil)
	}
	if strings.TrimSpace(chunk) == "" {
		return s.msg(locale, "artifact_read_none", map[string]string{"artifact_id": item.ArtifactID})
	}
	if s.shouldUseDetailAgentForArtifact(ctx, item) && s.detailer != nil && s.detailer.Enabled() {
		reporter.Step(ctx, "这份大详情我先交给辅助分析器解释当前片段。")
		report, err := s.detailer.SummarizeRead(ctx, item, startLine, lineCount, chunk)
		if err == nil && strings.TrimSpace(report.Summary) != "" {
			return formatDetailAgentReport(item, report, "read")
		}
	}
	return s.msg(locale, "artifact_read_head", map[string]string{"artifact_id": item.ArtifactID}) + "\n" + chunk
}

func formatDetailAgentReport(item protocol.Artifact, report detailagent.Report, mode string) string {
	status := "false"
	if report.EnoughToAnswer {
		status = "true"
	}
	lines := []string{
		fmt.Sprintf("辅助分析结果（artifact_id=%s kind=%s mode=%s）", item.ArtifactID, item.Kind, mode),
		"enough_to_answer=" + status,
		"summary=" + report.Summary,
	}
	if len(report.Evidence) > 0 {
		lines = append(lines, "evidence:")
		for _, item := range report.Evidence {
			lines = append(lines, "- "+item)
		}
	}
	if len(report.Gaps) > 0 {
		lines = append(lines, "gaps:")
		for _, item := range report.Gaps {
			lines = append(lines, "- "+item)
		}
	}
	if len(report.NextQueries) > 0 {
		lines = append(lines, "next_queries:")
		for _, item := range report.NextQueries {
			lines = append(lines, "- "+item)
		}
	}
	return strings.Join(lines, "\n")
}

func (s *Service) shouldUseDetailAgentForArtifact(ctx context.Context, item protocol.Artifact) bool {
	if s.detailer == nil || !s.detailer.Enabled() {
		return false
	}
	if item.ArtifactID == "" {
		return false
	}
	if item.Kind != "incident" && item.Kind != "detection" {
		return false
	}
	return !s.canInlineArtifact(ctx, item.SessionKey, item, item.Content)
}

func (s *Service) canInlineArtifact(ctx context.Context, sessionKey string, item protocol.Artifact, content string) bool {
	if item.Kind != "incident" && item.Kind != "detection" {
		threshold := s.cfg.Compression.ContextWindowTokens / 2
		if threshold <= 0 {
			threshold = 32000
		}
		return artifact.EstimateTokens(content) <= threshold
	}
	directMaxBytes := s.cfg.DetailAgent.DirectMaxBytes
	if directMaxBytes <= 0 {
		directMaxBytes = 12 * 1024
	}
	if len([]byte(content)) > directMaxBytes {
		return false
	}
	available := s.availableContextTokens(ctx, sessionKey)
	if available <= 0 {
		return false
	}
	return artifact.EstimateTokens(content) <= available
}

func (s *Service) availableContextTokens(ctx context.Context, sessionKey string) int {
	window := s.cfg.Compression.ContextWindowTokens
	if window <= 0 {
		window = 128000
	}
	reserve := 4096
	if reserve > window/4 {
		reserve = window / 4
	}
	messages, err := s.buildBaseMessages(ctx, sessionKey)
	if err != nil {
		available := window - reserve
		if available < 0 {
			return 0
		}
		return available
	}
	used := 0
	for _, msg := range messages {
		used += artifact.EstimateTokens(msg.Content)
	}
	available := window - used - reserve
	if available < 0 {
		return 0
	}
	return available
}

func blankToDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func (s *Service) guardUnsupportedEDRAction(text string, locale string) (string, bool) {
	plain := strings.ToLower(strings.TrimSpace(text))
	if plain == "" {
		return "", false
	}

	actionHints := []string{
		"edr", "api", "接口", "隔离", "恢复", "终止进程", "kill process", "kill", "ping", "检测", "事件", "日志", "执行", "调用", "测试接口",
	}
	hit := false
	for _, item := range actionHints {
		if strings.Contains(plain, item) {
			hit = true
			break
		}
	}
	if !hit {
		return "", false
	}

	return sanitizeReply(s.msg(locale, "unsupported_query", nil)), true
}
