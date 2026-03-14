package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"rm_ai_agent/internal/model"
	"rm_ai_agent/internal/planner"
	"rm_ai_agent/internal/protocol"
)

type scheduledReport struct {
	Summary  string                         `json:"summary"`
	Entities []protocol.ScheduledTaskEntity `json:"entities"`
}

func (s *Service) RunScheduledTask(ctx context.Context, task protocol.ScheduledTask) (protocol.ScheduledTaskExecution, error) {
	locale := "zh-CN"
	taskMemory, _ := s.store.GetScheduledTaskState(ctx, task.TaskID)
	knownEntities, _ := s.store.ListScheduledTaskEntities(ctx, task.TaskID, 100)
	plannerInput := buildScheduledPlannerInput(task, taskMemory, knownEntities)
	summary, _ := s.store.GetSessionSummary(ctx, task.SessionKey)
	recentTurns, _ := s.store.ListRecentTurns(ctx, task.SessionKey, minInt(maxInt(s.cfg.Session.MaxRecentTurns, 10), 16))
	memories, _ := s.memory.ListForContext(ctx, task.SessionKey)
	latestArtifact := protocol.Artifact{}
	if s.artifacts != nil {
		latestArtifact, _ = s.artifacts.GetLatest(ctx, task.SessionKey)
	}
	skillsPrompt := ""
	if s.prompt != nil {
		skillsPrompt = s.prompt.LoadSkillsPrompt()
	}
	plan, err := s.planner.BuildPlan(ctx, s.cfg.Scheduler.Model, plannerInput, "", summary, recentTurns, memories, latestArtifact, skillsPrompt)
	if err != nil {
		return protocol.ScheduledTaskExecution{}, err
	}
	toolResults := []string(nil)
	finalReply := strings.TrimSpace(plan.DirectReply)
	if len(plan.ToolCalls) > 0 {
		toolResults, finalReply, err = s.previewToolPlan(ctx, task.SessionKey, plannerInput, locale, plan.ToolCalls)
		if err != nil {
			return protocol.ScheduledTaskExecution{}, err
		}
	}
	report, err := s.buildScheduledReport(ctx, task, taskMemory, knownEntities, strings.Join(toolResults, "\n\n"), finalReply)
	if err != nil {
		report = scheduledReport{Summary: strings.TrimSpace(finalReply)}
	}
	notifyEntities, allEntities, stateJSON, summaryText := s.mergeScheduledEntities(task, report, knownEntities)
	if stateJSON != "" {
		_ = s.store.UpsertScheduledTaskState(ctx, task.TaskID, stateJSON)
	}
	run := protocol.ScheduledTaskRun{
		RunID:      fmt.Sprintf("tr-%d", time.Now().UnixNano()),
		TaskID:     task.TaskID,
		ScopeKey:   task.ScopeKey,
		SessionKey: task.SessionKey,
		Status:     "ok",
		Summary:    summaryText,
		Report:     report.Summary,
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
	}
	if body, err := json.Marshal(allEntities); err == nil {
		run.EntitiesJSON = string(body)
	}
	if err := s.store.SaveScheduledTaskRun(ctx, run); err != nil {
		return protocol.ScheduledTaskExecution{}, err
	}
	patch := protocol.ScheduledTaskPatch{LastSummary: summaryText, LastRunAt: run.FinishedAt, NextRunAt: run.FinishedAt.Add(time.Duration(task.IntervalSeconds) * time.Second)}
	if _, err := s.store.UpdateScheduledTask(ctx, task.ScopeKey, task.TaskID, patch); err != nil {
		return protocol.ScheduledTaskExecution{}, err
	}
	message := ""
	if len(notifyEntities) > 0 {
		message = s.buildScheduledTaskMessage(task, report.Summary, notifyEntities)
		_, _ = s.storeAssistantReply(ctx, task.SessionKey, message)
	}
	return protocol.ScheduledTaskExecution{Summary: summaryText, Message: message, Run: run, Entities: notifyEntities}, nil
}

func (s *Service) previewToolPlan(ctx context.Context, sessionKey string, userText string, locale string, calls []planner.ToolCall) ([]string, string, error) {
	if len(calls) == 0 {
		return nil, "", nil
	}
	var allToolResults []string
	currentCalls := calls
	finalDirectReply := ""
	seenSignatures := make(map[string]int)
	for {
		signature := toolCallsSignature(currentCalls)
		seenSignatures[signature]++
		if seenSignatures[signature] > 2 {
			break
		}
		stepResults, err := s.executeToolBatchPreview(ctx, sessionKey, locale, currentCalls)
		if err != nil {
			return nil, "", err
		}
		allToolResults = append(allToolResults, stepResults...)
		nextPlan, err := s.planner.BuildPlan(ctx, s.cfg.Scheduler.Model, userText, strings.Join(allToolResults, "\n\n"), "", nil, nil, protocol.Artifact{}, "")
		if err != nil {
			break
		}
		if len(nextPlan.ToolCalls) == 0 {
			finalDirectReply = nextPlan.DirectReply
			break
		}
		currentCalls = nextPlan.ToolCalls
	}
	return allToolResults, finalDirectReply, nil
}

func (s *Service) executeToolBatchPreview(ctx context.Context, sessionKey string, locale string, calls []planner.ToolCall) ([]string, error) {
	results := make([]string, 0, len(calls))
	for _, call := range calls {
		if strings.HasPrefix(call.Name, "scheduled_task_") || strings.HasPrefix(call.Name, "memory_") || call.Critical || isCriticalTool(call.Name) {
			results = append(results, s.msg(locale, "tool_error", map[string]string{"tool": call.Name, "error": "scheduled task skipped unsupported tool"}))
			continue
		}
		result, err := s.executeSingleTool(ctx, sessionKey, call, locale, nil)
		if err != nil {
			results = append(results, s.msg(locale, "tool_error", map[string]string{"tool": call.Name, "error": err.Error()}))
			continue
		}
		if strings.TrimSpace(result) != "" {
			results = append(results, result)
		}
	}
	return results, nil
}

func (s *Service) buildScheduledReport(ctx context.Context, task protocol.ScheduledTask, taskMemory string, knownEntities []protocol.ScheduledTaskEntity, toolContext string, finalReply string) (scheduledReport, error) {
	if s.model == nil {
		return scheduledReport{}, fmt.Errorf("model unavailable")
	}
	promptText := "你是一个定时任务子agent。你会拿到任务描述、任务记忆、历史已见对象和本轮真实工具结果。请只输出 JSON：{\"summary\":\"\",\"entities\":[{\"kind\":\"\",\"entity_id\":\"\",\"title\":\"\",\"host_name\":\"\",\"client_id\":\"\",\"severity\":\"\",\"last_summary\":\"\"}]}。只基于真实结果，不要编造。summary 用中文纯文本，简洁说明本轮发现。entities 只放本轮值得追踪的对象。"
	if s.prompt != nil {
		if loaded := s.prompt.LoadPrompt("scheduled_task_reporter"); strings.TrimSpace(loaded) != "" {
			promptText = loaded
		}
		promptText = s.prompt.ComposeSystemPrompt(promptText)
	}
	known := make([]string, 0, len(knownEntities))
	for _, item := range knownEntities {
		known = append(known, fmt.Sprintf("- %s | %s | %s | status=%s", item.Kind, item.EntityID, item.Title, item.Status))
	}
	result, err := s.model.Chat(ctx, model.ChatRequest{
		Model: s.cfg.Scheduler.Model,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: promptText},
			{Role: model.RoleUser, Content: "任务内容:\n" + task.Prompt + "\n\n任务记忆:\n" + firstNonEmptyString(taskMemory, "(空)") + "\n\n历史已见对象:\n" + firstNonEmptyString(strings.Join(known, "\n"), "(空)") + "\n\n本轮真实工具结果:\n" + firstNonEmptyString(toolContext, "(空)") + "\n\n本轮直接结论:\n" + firstNonEmptyString(finalReply, "(空)")},
		},
	}, nil)
	if err != nil {
		return scheduledReport{}, err
	}
	jsonText := extractJSONText(result.Text)
	if jsonText == "" {
		return scheduledReport{}, fmt.Errorf("scheduled report missing json")
	}
	var report scheduledReport
	if err := json.Unmarshal([]byte(jsonText), &report); err != nil {
		return scheduledReport{}, err
	}
	report.Summary = strings.TrimSpace(report.Summary)
	return report, nil
}

func (s *Service) mergeScheduledEntities(task protocol.ScheduledTask, report scheduledReport, existing []protocol.ScheduledTaskEntity) ([]protocol.ScheduledTaskEntity, []protocol.ScheduledTaskEntity, string, string) {
	now := time.Now().UTC()
	existingMap := make(map[string]protocol.ScheduledTaskEntity, len(existing))
	for _, item := range existing {
		existingMap[item.EntityKey] = item
	}
	all := make([]protocol.ScheduledTaskEntity, 0, len(report.Entities))
	notify := make([]protocol.ScheduledTaskEntity, 0, len(report.Entities))
	for _, item := range report.Entities {
		item.TaskID = task.TaskID
		item.Kind = strings.TrimSpace(item.Kind)
		item.EntityID = strings.TrimSpace(item.EntityID)
		item.Title = strings.TrimSpace(item.Title)
		item.HostName = strings.TrimSpace(item.HostName)
		item.ClientID = strings.TrimSpace(item.ClientID)
		item.Severity = strings.TrimSpace(item.Severity)
		item.EntityKey = scheduledEntityKey(item)
		item.LastSummary = scheduledEntitySignature(item)
		if item.EntityKey == "" {
			continue
		}
		old, ok := existingMap[item.EntityKey]
		if ok {
			item.FirstSeenAt = old.FirstSeenAt
			item.Status = old.Status
			item.Note = old.Note
			item.LastReportedAt = old.LastReportedAt
		} else {
			item.FirstSeenAt = now
			item.Status = "reported"
		}
		item.LastSeenAt = now
		shouldNotify := false
		if !ok {
			shouldNotify = true
			item.LastReportedAt = now
		} else if suppressScheduledEntity(old.Status) {
			shouldNotify = false
		} else if old.LastSummary != item.LastSummary {
			shouldNotify = true
			item.Status = "updated"
			item.LastReportedAt = now
		}
		all = append(all, item)
		_ = s.store.UpsertScheduledTaskEntity(context.Background(), item)
		if shouldNotify {
			notify = append(notify, item)
		}
	}
	state := map[string]any{"last_summary": report.Summary, "last_run_at": now.Format(time.RFC3339), "reported_entities": len(notify)}
	body, _ := json.Marshal(state)
	summary := report.Summary
	if summary == "" {
		if len(notify) == 0 {
			summary = "本轮没有发现需要新汇报的对象。"
		} else {
			summary = fmt.Sprintf("本轮发现 %d 个需要汇报的对象。", len(notify))
		}
	}
	return notify, all, string(body), summary
}

func (s *Service) buildScheduledTaskMessage(task protocol.ScheduledTask, summary string, entities []protocol.ScheduledTaskEntity) string {
	lines := []string{fmt.Sprintf("[定时任务 %s] %s", task.TaskID, task.Title)}
	if strings.TrimSpace(summary) != "" {
		lines = append(lines, strings.TrimSpace(summary))
	}
	for _, item := range entities {
		line := fmt.Sprintf("- %s", firstNonEmptyString(item.Title, item.EntityID))
		if item.HostName != "" || item.ClientID != "" {
			line += fmt.Sprintf(" host=%s client_id=%s", blankToDash(item.HostName), blankToDash(item.ClientID))
		}
		if item.Severity != "" {
			line += fmt.Sprintf(" severity=%s", item.Severity)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func buildScheduledPlannerInput(task protocol.ScheduledTask, taskMemory string, known []protocol.ScheduledTaskEntity) string {
	parts := []string{"这是一个定时任务子agent执行轮次。请直接执行，不要向用户追问。", "任务内容:\n" + strings.TrimSpace(task.Prompt)}
	if strings.TrimSpace(taskMemory) != "" {
		parts = append(parts, "任务专有记忆:\n"+strings.TrimSpace(taskMemory))
	}
	if len(known) > 0 {
		lines := make([]string, 0, len(known))
		for _, item := range known {
			lines = append(lines, fmt.Sprintf("- %s | %s | %s | status=%s", item.Kind, item.EntityID, item.Title, item.Status))
		}
		parts = append(parts, "已知对象:\n"+strings.Join(lines, "\n"))
	}
	parts = append(parts, "如果本轮只看到已经汇报过或已标记为误报/已处理的对象，就尽量不要重复扩线索。")
	return strings.Join(parts, "\n\n")
}

func scheduledEntityKey(item protocol.ScheduledTaskEntity) string {
	if strings.TrimSpace(item.Kind) == "" && strings.TrimSpace(item.EntityID) == "" && strings.TrimSpace(item.Title) == "" {
		return ""
	}
	if strings.TrimSpace(item.EntityID) != "" {
		return strings.ToLower(strings.TrimSpace(item.Kind) + ":" + strings.TrimSpace(item.EntityID))
	}
	return strings.ToLower(strings.TrimSpace(item.Kind) + ":" + strings.TrimSpace(item.ClientID) + ":" + strings.TrimSpace(item.Title))
}

func scheduledEntitySignature(item protocol.ScheduledTaskEntity) string {
	return strings.Join([]string{strings.TrimSpace(item.Title), strings.TrimSpace(item.HostName), strings.TrimSpace(item.ClientID), strings.TrimSpace(item.Severity)}, "|")
}

func suppressScheduledEntity(status string) bool {
	status = strings.TrimSpace(strings.ToLower(status))
	return status == "false_positive" || status == "resolved" || status == "ignored"
}

func decodeScheduledEntities(text string) ([]protocol.ScheduledTaskEntity, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	var items []protocol.ScheduledTaskEntity
	if err := json.Unmarshal([]byte(text), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func extractJSONText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return ""
	}
	return text[start : end+1]
}
