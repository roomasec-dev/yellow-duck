package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/model"
	"rm_ai_agent/internal/prompt"
	"rm_ai_agent/internal/protocol"
)

type ToolCall struct {
	Name                         string `json:"name"`
	Hostname                     string `json:"hostname,omitempty"`
	ClientID                     string `json:"client_id,omitempty"`
	ClientIP                     string `json:"client_ip,omitempty"`
	OSType                       string `json:"os_type,omitempty"`
	Operation                    string `json:"operation,omitempty"`
	StartTime                    string `json:"start_time,omitempty"`
	EndTime                      string `json:"end_time,omitempty"`
	FilterField                  string `json:"filter_field,omitempty"`
	FilterOp                     string `json:"filter_operator,omitempty"`
	FilterValue                  string `json:"filter_value,omitempty"`
	Page                         int    `json:"page,omitempty"`
	PageSize                     int    `json:"page_size,omitempty"`
	IncidentID                   string `json:"incident_id,omitempty"`
	DetectionID                  string `json:"detection_id,omitempty"`
	ViewType                     string `json:"view_type,omitempty"`
	ProcessUUID                  string `json:"process_uuid,omitempty"`
	ArtifactID                   string `json:"artifact_id,omitempty"`
	Query                        string `json:"query,omitempty"`
	StartLine                    int    `json:"start_line,omitempty"`
	LineCount                    int    `json:"line_count,omitempty"`
	MemoryKey                    string `json:"memory_key,omitempty"`
	MemoryValue                  string `json:"memory_value,omitempty"`
	TaskID                       string `json:"task_id,omitempty"`
	TaskTitle                    string `json:"task_title,omitempty"`
	TaskPrompt                   string `json:"task_prompt,omitempty"`
	TaskAction                   string `json:"task_action,omitempty"`
	TaskStatus                   string `json:"task_status,omitempty"`
	TaskFeedback                 string `json:"task_feedback,omitempty"`
	TaskIntervalMinutes          int    `json:"task_interval_minutes,omitempty"`
	InstructionName              string `json:"instruction_name,omitempty"`
	Path                         string `json:"path,omitempty"`
	KBTitle                      string `json:"kb_title,omitempty"`
	KBQuery                      string `json:"kb_query,omitempty"`
	KBContent                    string `json:"kb_content,omitempty"`
	KBMode                       string `json:"kb_mode,omitempty"`
	KBOldText                    string `json:"kb_old_text,omitempty"`
	KBNewText                    string `json:"kb_new_text,omitempty"`
	Reason                       string `json:"reason,omitempty"`
	Critical                     bool   `json:"critical,omitempty"`
	IOCAction                    string `json:"ioc_action,omitempty"`
	IOCID                        string `json:"ioc_id,omitempty"`
	IOCHash                      string `json:"ioc_hash,omitempty"`
	IOCDescription               string `json:"ioc_description,omitempty"`
	IOCExpirationDate            string `json:"ioc_expiration_date,omitempty"`
	IOCFileName                  string `json:"ioc_file_name,omitempty"`
	IOCHostType                  string `json:"ioc_host_type,omitempty"`
	IsolateFileGUIDs             string `json:"isolate_file_guids,omitempty"`
	IsolateFileAddExcl           bool   `json:"isolate_file_add_exclusion,omitempty"`
	IsolateFileReleaseAll        bool   `json:"isolate_file_release_all,omitempty"`
	PlanName                     string `json:"plan_name,omitempty"`
	ScanType                     int    `json:"scan_type,omitempty"`
	PlanType                     int    `json:"plan_type,omitempty"`
	Scope                        int    `json:"scope,omitempty"`
	RID                          string `json:"rid,omitempty"`
	Time                         int    `json:"time,omitempty"`
	Pid                          int    `json:"pid,omitempty"`
	Ids                          string `json:"ids,omitempty"`
	Allow                        bool   `json:"allow,omitempty"`
	Status                       int    `json:"status,omitempty"`
	Scene                        string `json:"scene,omitempty"`
	Comment                      string `json:"comment,omitempty"`
	Type                         string `json:"type,omitempty"`
	ScanFileScope                string `json:"scan_file_scope,omitempty"`
	StartupScanMode              string `json:"startup_scan_mode,omitempty"`
	ArchiveSizeLimitEnabled      *bool  `json:"archive_size_limit_enabled,omitempty"`
	ArchiveSizeLimit             int    `json:"archive_size_limit,omitempty"`
	RealtimeMemCacheTechEnabled  *bool  `json:"realtime_mem_cache_tech_enabled,omitempty"`
	DynamicCpuMonitorEnabled     *bool  `json:"dynamic_cpu_monitor_enabled,omitempty"`
	DynamicCpuHighPercent        int    `json:"dynamic_cpu_high_percent,omitempty"`
	StopRealtimeOnCpuHighEnabled *bool  `json:"stop_realtime_on_cpu_high_enabled,omitempty"`
	StopRealtimeCpuHighPercent   int    `json:"stop_realtime_cpu_high_percent,omitempty"`
	OwlOnRealtimeEnabled         *bool  `json:"owl_on_realtime_enabled,omitempty"`
	RealtimeScanArchiveEnabled   *bool  `json:"realtime_scan_archive_enabled,omitempty"`
	RuntimeMaxFileSizeMb         int    `json:"runtime_max_file_size_mb,omitempty"`
	CustomMaxFileSizeMb          int    `json:"custom_max_file_size_mb,omitempty"`
	StrategyID                   string `json:"strategy_id,omitempty"`
	VerifyCode                   string `json:"verify_code,omitempty"`
}

type Plan struct {
	TaskMode           string     `json:"task_mode"`
	Phase              string     `json:"phase"`
	IntentSummary      string     `json:"intent_summary"`
	DoneWhen           string     `json:"done_when"`
	NeedClarification  bool       `json:"need_clarification"`
	ClarifyingQuestion string     `json:"clarifying_question"`
	PlanPreview        string     `json:"plan_preview"`
	DirectReply        string     `json:"direct_reply"`
	ToolCalls          []ToolCall `json:"tool_calls"`
}

type Service struct {
	model  model.Client
	prompt *prompt.Service
	logger *logx.Logger
}

func NewService(modelClient model.Client, promptService *prompt.Service, logger *logx.Logger) *Service {
	return &Service{model: modelClient, prompt: promptService, logger: logger}
}

func (s *Service) BuildPlan(ctx context.Context, modelRef string, userText string, toolContext string, summary string, recentTurns []protocol.Turn, memories []protocol.MemoryEntry, latestArtifact protocol.Artifact, skillsPrompt string) (Plan, error) {
	if s.model == nil || strings.TrimSpace(modelRef) == "" {
		return Plan{}, nil
	}
	s.logger.Info("planner start", "model", modelRef, "user_preview", preview(userText), "tool_context_len", len(toolContext), "memory_count", len(memories), "latest_artifact", latestArtifact.ArtifactID)
	memoryText := formatMemories(memories)
	turnText := formatTurns(recentTurns)
	systemPrompt := plannerPrompt(skillsPrompt, memoryText, latestArtifact)
	if s.prompt != nil {
		systemPrompt = s.prompt.ComposeSystemPrompt(systemPrompt)
	}
	result, err := s.model.Chat(ctx, model.ChatRequest{
		Model: modelRef,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: systemPrompt},
			{Role: model.RoleUser, Content: buildPlannerUserInput(userText, toolContext, summary, turnText)},
		},
	}, nil)
	if err != nil {
		s.logger.Warn("planner model call failed", "model", modelRef, "error", err)
		return Plan{}, err
	}
	// s.logger.Info("planner raw output", "preview", preview(result.Text))
	jsonText := extractJSON(result.Text)
	// s.logger.Info("planner json extracted", "json", jsonText)
	if jsonText == "" {
		s.logger.Warn("planner output missing json", "preview", preview(result.Text))
		return Plan{}, fmt.Errorf("planner did not return json")
	}
	var plan Plan
	if err := json.Unmarshal([]byte(jsonText), &plan); err != nil {
		s.logger.Warn("planner json decode failed", "json_preview", preview(jsonText), "error", err)
		return Plan{}, err
	}
	if len(plan.ToolCalls) > 0 {
		s.logger.Info("planner first tool_call", "name", plan.ToolCalls[0].Name, "client_id", plan.ToolCalls[0].ClientID, "instruction_name", plan.ToolCalls[0].InstructionName, "path", plan.ToolCalls[0].Path)
	}
	for i := range plan.ToolCalls {
		plan.ToolCalls[i].Name = strings.ToLower(strings.TrimSpace(plan.ToolCalls[i].Name))
		plan.ToolCalls[i].Hostname = strings.TrimSpace(plan.ToolCalls[i].Hostname)
		plan.ToolCalls[i].ClientID = strings.TrimSpace(plan.ToolCalls[i].ClientID)
		plan.ToolCalls[i].ClientIP = strings.TrimSpace(plan.ToolCalls[i].ClientIP)
		plan.ToolCalls[i].OSType = strings.TrimSpace(plan.ToolCalls[i].OSType)
		plan.ToolCalls[i].Operation = strings.TrimSpace(plan.ToolCalls[i].Operation)
		plan.ToolCalls[i].StartTime = strings.TrimSpace(plan.ToolCalls[i].StartTime)
		plan.ToolCalls[i].EndTime = strings.TrimSpace(plan.ToolCalls[i].EndTime)
		plan.ToolCalls[i].FilterField = strings.TrimSpace(plan.ToolCalls[i].FilterField)
		plan.ToolCalls[i].FilterOp = strings.TrimSpace(plan.ToolCalls[i].FilterOp)
		plan.ToolCalls[i].FilterValue = strings.TrimSpace(plan.ToolCalls[i].FilterValue)
		if plan.ToolCalls[i].Page < 0 {
			plan.ToolCalls[i].Page = 0
		}
		if plan.ToolCalls[i].PageSize < 0 {
			plan.ToolCalls[i].PageSize = 0
		}
		plan.ToolCalls[i].IncidentID = strings.TrimSpace(plan.ToolCalls[i].IncidentID)
		plan.ToolCalls[i].DetectionID = strings.TrimSpace(plan.ToolCalls[i].DetectionID)
		plan.ToolCalls[i].ViewType = strings.TrimSpace(plan.ToolCalls[i].ViewType)
		plan.ToolCalls[i].ProcessUUID = strings.TrimSpace(plan.ToolCalls[i].ProcessUUID)
		plan.ToolCalls[i].ArtifactID = strings.TrimSpace(plan.ToolCalls[i].ArtifactID)
		plan.ToolCalls[i].Query = strings.TrimSpace(plan.ToolCalls[i].Query)
		plan.ToolCalls[i].MemoryKey = strings.TrimSpace(plan.ToolCalls[i].MemoryKey)
		plan.ToolCalls[i].MemoryValue = strings.TrimSpace(plan.ToolCalls[i].MemoryValue)
		plan.ToolCalls[i].TaskID = strings.TrimSpace(plan.ToolCalls[i].TaskID)
		plan.ToolCalls[i].TaskTitle = strings.TrimSpace(plan.ToolCalls[i].TaskTitle)
		plan.ToolCalls[i].TaskPrompt = strings.TrimSpace(plan.ToolCalls[i].TaskPrompt)
		plan.ToolCalls[i].TaskAction = strings.TrimSpace(plan.ToolCalls[i].TaskAction)
		plan.ToolCalls[i].TaskStatus = strings.TrimSpace(plan.ToolCalls[i].TaskStatus)
		plan.ToolCalls[i].TaskFeedback = strings.TrimSpace(plan.ToolCalls[i].TaskFeedback)
		if plan.ToolCalls[i].TaskIntervalMinutes < 0 {
			plan.ToolCalls[i].TaskIntervalMinutes = 0
		}
		plan.ToolCalls[i].KBTitle = strings.TrimSpace(plan.ToolCalls[i].KBTitle)
		plan.ToolCalls[i].KBQuery = strings.TrimSpace(plan.ToolCalls[i].KBQuery)
		plan.ToolCalls[i].KBContent = strings.TrimSpace(plan.ToolCalls[i].KBContent)
		plan.ToolCalls[i].KBMode = strings.TrimSpace(plan.ToolCalls[i].KBMode)
		plan.ToolCalls[i].KBOldText = strings.TrimSpace(plan.ToolCalls[i].KBOldText)
		plan.ToolCalls[i].KBNewText = strings.TrimSpace(plan.ToolCalls[i].KBNewText)
		plan.ToolCalls[i].IOCAction = strings.TrimSpace(plan.ToolCalls[i].IOCAction)
		plan.ToolCalls[i].IOCHash = strings.TrimSpace(plan.ToolCalls[i].IOCHash)
		plan.ToolCalls[i].IOCID = strings.TrimSpace(plan.ToolCalls[i].IOCID)
		plan.ToolCalls[i].IOCDescription = strings.TrimSpace(plan.ToolCalls[i].IOCDescription)
		plan.ToolCalls[i].IOCExpirationDate = strings.TrimSpace(plan.ToolCalls[i].IOCExpirationDate)
		plan.ToolCalls[i].IOCFileName = strings.TrimSpace(plan.ToolCalls[i].IOCFileName)
		plan.ToolCalls[i].IOCHostType = strings.TrimSpace(plan.ToolCalls[i].IOCHostType)
		plan.ToolCalls[i].IsolateFileGUIDs = strings.TrimSpace(plan.ToolCalls[i].IsolateFileGUIDs)
		plan.ToolCalls[i].Path = strings.TrimSpace(plan.ToolCalls[i].Path)
		plan.ToolCalls[i].ScanFileScope = strings.TrimSpace(plan.ToolCalls[i].ScanFileScope)
		plan.ToolCalls[i].StartupScanMode = strings.TrimSpace(plan.ToolCalls[i].StartupScanMode)
		if plan.ToolCalls[i].Time < 0 {
			plan.ToolCalls[i].Time = 0
		}
		if plan.ToolCalls[i].Pid < 0 {
			plan.ToolCalls[i].Pid = 0
		}
		if plan.ToolCalls[i].ArchiveSizeLimit < 0 {
			plan.ToolCalls[i].ArchiveSizeLimit = 0
		}
		if plan.ToolCalls[i].DynamicCpuHighPercent < 0 {
			plan.ToolCalls[i].DynamicCpuHighPercent = 0
		}
		if plan.ToolCalls[i].StopRealtimeCpuHighPercent < 0 {
			plan.ToolCalls[i].StopRealtimeCpuHighPercent = 0
		}
		if plan.ToolCalls[i].RuntimeMaxFileSizeMb < 0 {
			plan.ToolCalls[i].RuntimeMaxFileSizeMb = 0
		}
		if plan.ToolCalls[i].CustomMaxFileSizeMb < 0 {
			plan.ToolCalls[i].CustomMaxFileSizeMb = 0
		}
	}
	plan.TaskMode = normalizeTaskMode(plan.TaskMode, userText)
	plan.Phase = normalizePhase(plan.Phase, plan.ToolCalls, plan.DirectReply)
	plan.IntentSummary = strings.TrimSpace(plan.IntentSummary)
	plan.DoneWhen = strings.TrimSpace(plan.DoneWhen)
	plan.ClarifyingQuestion = strings.TrimSpace(plan.ClarifyingQuestion)
	if plan.NeedClarification && plan.ClarifyingQuestion == "" {
		plan.NeedClarification = false
	}
	plan.PlanPreview = strings.TrimSpace(plan.PlanPreview)
	plan.DirectReply = strings.TrimSpace(plan.DirectReply)
	s.logger.Info("planner parsed", "tool_count", len(plan.ToolCalls), "task_mode", plan.TaskMode, "phase", plan.Phase, "intent", preview(plan.IntentSummary), "direct_reply_preview", preview(plan.DirectReply))
	return plan, nil
}

func normalizeTaskMode(mode string, userText string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "overview", "overview_drilldown", "drill_down", "exhaustive", "action", "general":
		return mode
	}
	text := strings.ToLower(strings.TrimSpace(userText))
	switch {
	case containsAny(text, "全部", "所有", "全量", "每一", "翻页", "多看几页", "再看几页", "多翻几页", "继续往后", "往后翻", "继续看更多"):
		return "exhaustive"
	case strings.Contains(text, "详细") || strings.Contains(text, "原因") || strings.Contains(text, "为什么") || strings.Contains(text, "深挖"):
		if strings.Contains(text, "有什么") || strings.Contains(text, "哪些") || strings.Contains(text, "最近") {
			return "overview_drilldown"
		}
		return "drill_down"
	case strings.Contains(text, "处置") || strings.Contains(text, "建议") || strings.Contains(text, "怎么办"):
		return "action"
	case strings.Contains(text, "有什么") || strings.Contains(text, "哪些") || strings.Contains(text, "最近") || strings.Contains(text, "概览"):
		return "overview"
	default:
		return "general"
	}
}

func normalizePhase(phase string, calls []ToolCall, directReply string) string {
	phase = strings.ToLower(strings.TrimSpace(phase))
	switch phase {
	case "overview", "scan_pages", "collect_candidates", "pick_target", "compare_candidates", "hypothesis_check", "drill_down", "answer":
		return phase
	}
	if strings.TrimSpace(directReply) != "" && len(calls) == 0 {
		return "answer"
	}
	if allThreatListToolCalls(calls) {
		for _, call := range calls {
			if positiveOr(call.Page, 1) > 1 {
				return "scan_pages"
			}
		}
		return "overview"
	}
	for _, call := range calls {
		switch call.Name {
		case "edr_incident_view", "edr_detection_view", "artifact_outline", "artifact_search", "artifact_read":
			return "drill_down"
		}
	}
	return "overview"
}

func allThreatListToolCalls(calls []ToolCall) bool {
	if len(calls) == 0 {
		return false
	}
	for _, call := range calls {
		switch call.Name {
		case "edr_incidents", "edr_detections", "edr_logs", "edr_event_log_alarms":
		default:
			return false
		}
	}
	return true
}

func positiveOr(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func buildPlannerUserInput(userText string, toolContext string, summary string, turnText string) string {
	userText = strings.TrimSpace(userText)
	toolContext = strings.TrimSpace(toolContext)
	summary = strings.TrimSpace(summary)
	turnText = strings.TrimSpace(turnText)
	parts := []string{"用户原始问题:\n" + userText}
	if card := buildTaskFocusCard(userText, toolContext); card != "" {
		parts = append(parts, "当前任务卡片（优先级高于历史上下文）：\n"+card)
	}
	if summary != "" {
		parts = append(parts, "会话摘要:\n"+summary)
	}
	if turnText != "" {
		parts = append(parts, "最近几轮对话:\n"+turnText)
	}
	if toolContext != "" {
		parts = append(parts, "已经拿到的真实工具结果:\n"+toolContext)
	}
	parts = append(parts, "请先对齐当前任务目标，再决定是否还需要继续调用工具。如果用户说“再来一次”“再试一次”“继续”“retry”“again”，优先把它理解为对上一轮相关工具链的延续，而不是重新开始一个泛化回答。")
	parts = append(parts, "如果当前任务目标不明确且存在会导致不同工具路线的分叉，可以返回 need_clarification=true 并问一个简短问题；如果能合理推断，就不要问，直接行动并在 intent_summary 里写明你的理解。")
	parts = append(parts, "如果已经拿到 artifact 或较大的真实结果，默认先继续做有限的浏览、搜索、局部阅读，再决定是否可以收口。只有用户明确要快速总结，或者已经拿到足够证据时，才直接收口。")
	return strings.Join(parts, "\n\n")
}

func buildTaskFocusCard(userText string, toolContext string) string {
	plain := strings.ToLower(strings.TrimSpace(userText))
	if plain == "" {
		return ""
	}
	var lines []string
	lines = append(lines, "- 当前目标："+inferGoal(userText))
	lines = append(lines, "- 建议完成标准："+inferDoneWhen(userText))
	if focus := inferFocus(userText); focus != "" {
		lines = append(lines, "- 用户关注点："+focus)
	}
	if phase := inferPreferredPhase(userText, toolContext); phase != "" {
		lines = append(lines, "- 建议阶段："+phase)
	}
	if ambiguity := inferAmbiguity(userText); ambiguity != "" {
		lines = append(lines, "- 潜在分叉："+ambiguity)
	}
	return strings.Join(lines, "\n")
}

func inferGoal(userText string) string {
	plain := strings.ToLower(strings.TrimSpace(userText))
	switch {
	case containsAny(plaintoChineseFriendly(plain), "多看几页", "再看几页", "往后", "继续看", "更多"):
		return "继续探索更多结果，先扩大候选范围，再决定是否深入某一条"
	case containsAny(plain, "全部", "所有", "全量", "每一"):
		return "尽量完整地排查和列出结果"
	case containsAny(plaintoChineseFriendly(plain), "原因", "为什么", "详细", "深挖"):
		return "围绕重点对象查清原因并给出判断"
	case containsAny(plaintoChineseFriendly(plain), "有什么", "哪些", "概览", "最近"):
		return "快速了解当前风险概况并挑出重点"
	default:
		return "理解用户当前问题并选择最直接的安全调查动作"
	}
}

func inferDoneWhen(userText string) string {
	plain := strings.ToLower(strings.TrimSpace(userText))
	switch {
	case containsAny(plaintoChineseFriendly(plain), "多看几页", "再看几页", "往后", "继续看", "更多"):
		return "已经扫过若干页、找到有代表性的候选，或连续结果没有明显新增时收口"
	case containsAny(plain, "全部", "所有", "全量", "每一"):
		return "达到合理页数/时间预算后给出完整度说明和未覆盖范围"
	case containsAny(plaintoChineseFriendly(plain), "原因", "为什么", "详细", "深挖"):
		return "拿到主体、关键行为、风险判断和 2-3 条证据后收口"
	case containsAny(plaintoChineseFriendly(plain), "有什么", "哪些", "概览", "最近"):
		return "基于当前页或当前结果给出概览和最值得关注的条目"
	default:
		return "足以直接回答用户当前问题，且继续查不会明显改变结论"
	}
}

func inferFocus(userText string) string {
	plain := strings.ToLower(strings.TrimSpace(userText))
	var items []string
	if containsAny(plaintoChineseFriendly(plain), "其他", "别的", "不同") {
		items = append(items, "用户想看不同于当前主线的对象或类型")
	}
	if containsAny(plaintoChineseFriendly(plain), "主机", "机器", "终端", "host") {
		items = append(items, "主机/终端维度")
	}
	if containsAny(plaintoChineseFriendly(plain), "事件", "incident", "告警") {
		items = append(items, "事件/告警维度")
	}
	if containsAny(plaintoChineseFriendly(plain), "风险", "检测", "检出", "detection") {
		items = append(items, "风险/检测维度")
	}
	return strings.Join(items, "；")
}

func inferPreferredPhase(userText string, toolContext string) string {
	plain := strings.ToLower(strings.TrimSpace(userText))
	switch {
	case containsAny(plaintoChineseFriendly(plain), "多看几页", "再看几页", "往后", "继续看", "更多"):
		return "scan_pages/collect_candidates：先扩大观察范围，不要过早钻单条详情"
	case strings.TrimSpace(toolContext) != "" && containsAny(plaintoChineseFriendly(plain), "这条", "详细", "原因", "为什么"):
		return "drill_down：围绕已选对象查关键证据"
	default:
		return "按当前目标选择 overview、scan_pages、drill_down 或 answer"
	}
}

func inferAmbiguity(userText string) string {
	plain := strings.ToLower(strings.TrimSpace(userText))
	if containsAny(plaintoChineseFriendly(plain), "看看", "处理一下", "弄一下") && !containsAny(plaintoChineseFriendly(plain), "事件", "风险", "日志", "主机", "详情", "原因", "全部", "更多") {
		return "用户目标可能过宽；如果无法从历史上下文推断，需要简短确认目标"
	}
	return ""
}

func plaintoChineseFriendly(text string) string {
	return strings.ToLower(strings.TrimSpace(text))
}

func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if keyword != "" && strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func plannerPrompt(skillsPrompt string, memoryText string, latestArtifact protocol.Artifact) string {
	base := "你是驻留在 IM 里的安全分析员兼工具规划器。你的目标不是把工具用完，而是像人类分析员一样判断用户真正要什么、查到什么程度够了、什么时候该收口，并输出 JSON。\n" +
		"可用工具：current_time, edr_hosts, edr_incidents, edr_detections, edr_event_log_alarms, edr_logs, edr_incident_view, edr_detection_view, edr_isolate, edr_release, edr_iocs, edr_ioc_add, edr_ioc_update, edr_ioc_delete, edr_isolate_files, edr_release_isolate_files, edr_tasks, edr_task_result, edr_send_instruction, edr_plan_list, edr_plan_task, edr_virus_by_host, edr_virus_by_hash, edr_virus_hash_hosts, edr_plan_add, edr_plan_edit, edr_plan_cancel, edr_ioas, edr_ioa_audit_log, edr_ioa_networks, edr_strategies, edr_strategy_single, edr_strategy_state, edr_host_offline, edr_host_offline_save, edr_add_host_blacklist, edr_remove_host, edr_batch_deal_incident, edr_incident_r2_summary, edr_instruction_policy_list, edr_instruction_policy_update, edr_instruction_policy_save_status, edr_instruction_policy_delete, edr_instruction_policy_sort, edr_instruction_policy_add, artifact_outline, artifact_search, artifact_read, memory_upsert, memory_delete, scheduled_task_create, scheduled_task_list, scheduled_task_update, scheduled_task_delete, scheduled_task_feedback, knowledge_base_search, knowledge_base_write, knowledge_base_delete。\n" +
		"edr_isolate / edr_release / edr_ioc_add / edr_ioc_update / edr_ioc_delete / edr_delete_isolate_files / edr_release_isolate_files / edr_send_instruction / edr_plan_add / edr_plan_edit / edr_plan_cancel / edr_ioa_add / edr_ioa_update / edr_ioa_delete / edr_ioa_network_add / edr_ioa_network_update / edr_ioa_network_delete / edr_strategy_create / edr_strategy_update / edr_strategy_delete / edr_strategy_status / edr_host_offline_save / edr_add_host_blacklist / edr_remove_host / edr_batch_deal_incident / edr_instruction_policy_update / edr_instruction_policy_save_status / edr_instruction_policy_delete / edr_instruction_policy_sort / edr_instruction_policy_add 属于 critical=true。\n" +
		"优先原则：\n" +
		"0. 先给本轮任务定性：task_mode 只能是 overview、overview_drilldown、drill_down、exhaustive、action、general；phase 只能是 overview、scan_pages、collect_candidates、pick_target、compare_candidates、hypothesis_check、drill_down、answer。\n" +
		"0.0 在决定工具之前，先用 intent_summary 用一句中文写出你对用户当前目标的理解，再用 done_when 写出什么情况下算完成。\n" +
		"0.0.1 如果用户当前目标存在明显分叉，且不同理解会导致完全不同的工具路线，可以返回 need_clarification=true 和 clarifying_question；问题必须简短、只问一个点、能直接帮助对齐目标。能推断时不要问。\n" +
		"0.1 overview=快速看整体，不默认穷尽；overview_drilldown=先概览，再只挑最危险/最相关的 1 条做有限深挖；drill_down=围绕单个目标深挖；exhaustive=用户明确要全量/所有/继续翻页时才使用；action=给处置建议或执行动作；general=其他。\n" +
		"0.1.1 如果用户明显想扩大观察范围（如继续看更多、多看几页、看看别的/其他），优先进入 scan_pages 或 collect_candidates，不要太早跳去单条详情。\n" +
		"0.2 你必须有完成意识：如果已经能回答用户问题，phase=answer、tool_calls=[]，直接给 direct_reply；不要因为还能搜就继续搜。\n" +
		"0.3 对安全调查，足够回答通常意味着已经拿到主体、风险名称/类型、关键行为或证据、下一步建议；不追求完美穷尽。\n" +
		"1. 如果用户在问当前时间、现在几点、today/now/current time，就优先规划 current_time。\n" +
		"1.1 如果用户在创建、查看、修改、暂停、恢复、删除定时任务，优先规划 scheduled_task_* 工具。没有明确时间要求时，scheduled_task_create 默认 task_interval_minutes=5。\n" +
		"1.2 如果用户说‘这个是误报’‘这个已经处理了’‘别再报这个’，优先规划 scheduled_task_feedback。task_feedback 可用 false_positive、resolved、watch。\n" +
		"1.3 如果用户在查知识库、搜索文档、查已有经验、查手册，优先规划 knowledge_base_search。搜索范围就是 knowledge_base.path 下递归遍历到的 markdown 文件。\n" +
		"1.4 如果用户想新增、编辑、补充知识库，优先规划 knowledge_base_write。默认 kb_mode=upsert；如果明显是追加则用 append；如果明显是把旧内容改成新内容则用 replace_text。\n" +
		"1.5 knowledge_base_write / knowledge_base_delete 的 kb_title 可以直接填写知识库文件标题，也可以直接填写搜索结果里出现的相对路径，例如 runbook/linux/ssh.md。\n" +
		"1.6 如果用户要修改或删除某篇知识库，但目标文件还不够明确，先规划 knowledge_base_search，等看到候选文件后再继续写入或删除。\n" +
		"2. 如果用户在查询主机/风险/检测/检出/事件/日志，就优先规划 EDR 只读工具。关键词对应：'风险''检测''检出'对应 edr_detections；'事件'对应 edr_incidents。注意：detection 结构体有 incident_id 字段；incident 结构体有 id 字段。\n" +
		"2.1 对 edr_detections / edr_incidents / edr_logs，如果用户明确提到第几页、page、下一页、继续翻页、每页多少条、全部列出，要把 page 和 page_size 一起填进 tool_calls。\n" +
		"2.1.1 如果用户只是问“有什么威胁”“最近有哪些风险/事件/检出”这类概览问题，默认只查第一页并基于当前页先回答，不要因为 has_more=true 就自动翻页。概览类问题默认先总结当前页前 6 条；只有用户明确要求继续、更多、全部、下一页时才翻页。\n" +
		"2.2 如果用户在做 hunting / 狩猎 / IOC 扩线 / 进程链排查，优先规划 edr_logs，并尽量提取 client_id、os_type、operation、start_time、end_time，以及一组最关键的 filter_field/filter_operator/filter_value。\n" +
		"2.3 对 edr_logs，filter_operator 优先用 is；如果用户已经给了明确进程名、操作名、系统类型、client_id 或哈希，优先用 is。contain 只用于路径片段、命令行片段、目录片段等模糊试探。\n" +
		"2.4 如果用户明确提到时间范围（最近1小时、今天、昨天、某个时间段），对 edr_logs 要尽量填写 start_time / end_time，格式优先用 YYYY-MM-DD HH:MM:SS。\n" +
		"2.5 如果用户明确查询'这个事件有哪些风险''该事件关联的检测/检出''风险列表'，即需要用 incident_id 查关联风险时，优先规划 edr_detections。注意：如果用户要查的是事件本身（如'查看相关事件'），应走 2.6 规划 edr_incident_r2_summary，不要走 edr_detections。incident_id 只能使用用户明确提供或真实工具结果里已经出现过的值。\n" +
		"2.6 如果用户提供 incident_id 并说'查看事件''查看相关事件''查看这个事件''事件详情'，或者在查看某条风险后追问'这个风险的事件''该风险关联的事件'，优先规划 edr_incident_r2_summary 并传入 incident_id。注意：'查看相关事件'查的是事件本身，不是风险，不要调用 edr_detections。incident_id 只能使用用户明确提供或真实工具结果里已经出现过的值。\n" +
		"3. 如果用户给了 incident_id 和 client_id，要查看事件详情，就优先规划 edr_incident_view。\n" +
		"4. 如果用户给了 detection_id 和 client_id，要查看风险详情，就优先规划 edr_detection_view；如果有 view_type 和 process_uuid 也一起带上。\n" +
		"4.1 incident_id / detection_id 只能使用用户明确提供或真实工具结果里已经出现过的值，绝对不要根据 host_name、incident_name、时间、样例或自然语言自行拼接猜测。\n" +
		"4.2 如果缺少真实 incident_id / detection_id，就先继续查列表或返回 direct_reply 说明还不能安全调用详情工具。\n" +
		"5. 如果用户在追问刚才那条超大 incident/detection 详情，优先先规划 artifact_outline 看结构，再规划 artifact_search；需要看一段连续原文时再规划 artifact_read。只有已经完成候选筛选后，才进入这类深挖。\n" +
		"6. 对 artifact 调查，默认流程是 outline -> search -> read -> answer，但最多做有限探索。拿到 2-3 个能解释原因的关键证据后必须收口，不要反复换说法继续查同一件事。\n" +
		"7. artifact_outline 需要 artifact_id；artifact_search 需要 query；artifact_read 需要 artifact_id，可选 start_line 和 line_count。artifact_read 的 start_line 尽量跟随前一轮搜索命中的 line/window。\n" +
		"8. 如果用户提供了稳定的长期偏好、资产映射、身份信息、主机别名、工作偏好，可以规划 memory_upsert。\n" +
		"9. 如果用户要求更正或删除旧记忆，可以规划 memory_delete。\n" +
		"10. 如果最近几轮里已经在查某个 incident、detection 或 artifact，而用户只说“再来一次”“继续”“retry”“again”，优先延续那条工具链。\n" +
		"11. 如果用户在查看、列表、搜索 IOC（威胁指标/hash/哈希），优先规划 edr_iocs；如果用户需要查某条 IOC 的详情，优先规划 edr_ioc_detail。\n" +
		"12.1 对 edr_iocs，用户说第几页、每页多少条时要把 page 和 page_size 填进 tool_calls。在回复 IOC 列表结果时，优先引用每条记录的 id 字段（而不是 hash 字段），格式如“id=xxx”。\n" +
		"12.2 如果用户想新增 IOC（加黑名单/加白名单），优先规划 edr_ioc_add，ioc_action 填 block 或 allow，ioc_hash 填 MD5/SHA1，ioc_host_type 填 ALL 或具体客户端 ID，ioc_file_name 填文件名（如有）。\n" +
		"12.3 如果用户想修改已有 IOC，优先规划 edr_ioc_update，需要同时填 ioc_id（必填）和 hash（必填），以及要改的字段（ioc_action、ioc_host_type、ioc_description、ioc_expiration_date 等）。\n" +
		"12.4 如果用户想删除 IOC，优先规划 edr_ioc_delete，需要填 ioc_id。\n" +
		"13. 如果用户在查看隔离文件列表，优先规划 edr_isolate_files。\n" +
		"14. 如果用户在放行隔离文件（解除隔离/恢复文件），优先规划 edr_release_isolate_files，需要填 isolate_file_guids（多个用英文逗号分隔）；如果同时要把 hash 加排除名单，isolate_file_add_exclusion=true。\n" +
		"14.1 如果用户在删除隔离文件记录（彻底删除），优先规划 edr_delete_isolate_files，需要填 isolate_file_guids。注意：删除是彻底移除记录，放行是解除隔离让文件恢复正常使用，两者完全不同。\n" +
		"15. 如果用户在查看指令任务列表，优先规划 edr_tasks。\n" +
		"16. 如果用户在查看任务结果/详情，优先规划 edr_task_result，需要提取 task_id。\n" +
		"17. 如果用户要删除主机、移除主机、注销主机，优先规划 edr_remove_host，需要填 client_id。\n" +
		"18. 如果用户要求发送指令到主机，优先规划 edr_send_instruction，需要同时填 client_id 和 instruction_name；如果提到文件路径（path=\\xxx 或\"文件路径是 xxx\"），必须提取到 path 字段中；涉及可疑文件、批量隔离、批量结束进程时 path 通常不能为空；如果提到隔离时间（如\"1小时\"、\"30分钟\"），必须提取到 time 字段并转换成秒为单位（如1小时=3600）；如果提到 pid 或进程 id，必须提取到 pid 字段中。\n" +
		"19. 如果用户要开启/关闭离线终端管理、保存主机离线配置，优先规划 edr_host_offline_save，需要填 status（开启=1，关闭=2）和 time（离线超时天数，如18天则 time=18）。\n" +
		"20. 如果用户在查看策略配置、查杀设置、扫描设置，优先规划 edr_strategy_single，type 填 virus_scan_settings；如果查资产登记策略，type 填 asset_registration。\n" +
		"20.1 如果用户在修改查杀设置、修改扫描策略、修改扫描配置（查杀范围、启动模式、压缩包限制、CPU避让、实时防护文件大小等），优先规划 edr_strategy_update，需要填 rid 和需要修改的字段（scan_file_scope、startup_scan_mode、archive_size_limit_enabled、archive_size_limit、realtime_mem_cache_tech_enabled、dynamic_cpu_monitor_enabled、dynamic_cpu_high_percent、stop_realtime_on_cpu_high_enabled、stop_realtime_cpu_high_percent、owl_on_realtime_enabled、realtime_scan_archive_enabled、runtime_max_file_size_mb、custom_max_file_size_mb 等）。\n" +
		"21. 无论是否调用工具，都额外给一个面向用户的简短 plan_preview，说明你准备怎么查或为什么准备收口，限制在 18 到 40 个字，不能泄露内部术语。\n" +
		"22. 如果不需要工具，就返回 direct_reply，tool_calls 为空。\n" +
		"只输出 JSON，不要 markdown。结构：{\"task_mode\":\"overview\",\"phase\":\"overview\",\"intent_summary\":\"\",\"done_when\":\"\",\"need_clarification\":false,\"clarifying_question\":\"\",\"plan_preview\":\"\",\"direct_reply\":\"\",\"tool_calls\":[{\"name\":\"\",\"hostname\":\"\",\"client_id\":\"\",\"client_ip\":\"\",\"os_type\":\"\",\"operation\":\"\",\"start_time\":\"\",\"end_time\":\"\",\"filter_field\":\"\",\"filter_operator\":\"\",\"filter_value\":\"\",\"page\":0,\"page_size\":0,\"incident_id\":\"\",\"detection_id\":\"\",\"view_type\":\"\",\"process_uuid\":\"\",\"artifact_id\":\"\",\"query\":\"\",\"start_line\":0,\"line_count\":0,\"memory_key\":\"\",\"memory_value\":\"\",\"task_id\":\"\",\"instruction_name\":\"\",\"path\":\"\",\"time\":0,\"pid\":0,\"task_title\":\"\",\"task_prompt\":\"\",\"task_action\":\"\",\"task_status\":\"\",\"status\":0,\"task_feedback\":\"\",\"task_interval_minutes\":0,\"kb_title\":\"\",\"kb_query\":\"\",\"kb_content\":\"\",\"kb_mode\":\"\",\"kb_old_text\":\"\",\"kb_new_text\":\"\",\"reason\":\"\",\"critical\":false,\"ioc_action\":\"\",\"ioc_hash\":\"\",\"ioc_id\":\"\",\"ioc_description\":\"\",\"ioc_expiration_date\":\"\",\"ioc_file_name\":\"\",\"ioc_host_type\":\"\",\"isolate_file_guids\":\"\",\"isolate_file_add_exclusion\":false,\"isolate_file_release_all\":false,\"type\":\"\"}]}"
	if memoryText != "" {
		base += "\n\n当前已有记忆：\n" + memoryText
	}
	if strings.TrimSpace(skillsPrompt) != "" {
		base += "\n\n工具技能说明：\n" + strings.TrimSpace(skillsPrompt)
	}
	if latestArtifact.ArtifactID != "" {
		base += fmt.Sprintf("\n\n最近一次大对象 artifact：id=%s kind=%s title=%s", latestArtifact.ArtifactID, latestArtifact.Kind, latestArtifact.Title)
	}
	return base
}

func formatMemories(memories []protocol.MemoryEntry) string {
	if len(memories) == 0 {
		return ""
	}
	parts := make([]string, 0, len(memories))
	for _, item := range memories {
		parts = append(parts, fmt.Sprintf("- %s=%s", item.Key, item.Value))
	}
	return strings.Join(parts, "\n")
}

func formatTurns(turns []protocol.Turn) string {
	if len(turns) == 0 {
		return ""
	}
	parts := make([]string, 0, len(turns))
	for _, turn := range turns {
		parts = append(parts, fmt.Sprintf("- %s: %s", turn.Role, strings.TrimSpace(turn.Content)))
	}
	return strings.Join(parts, "\n")
}

func extractJSON(text string) string {
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

func preview(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 200 {
		return text
	}
	return text[:200] + "..."
}
