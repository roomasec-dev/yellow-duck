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
	Name                  string `json:"name"`
	Hostname              string `json:"hostname,omitempty"`
	ClientID              string `json:"client_id,omitempty"`
	ClientIP              string `json:"client_ip,omitempty"`
	OSType                string `json:"os_type,omitempty"`
	Operation             string `json:"operation,omitempty"`
	StartTime             string `json:"start_time,omitempty"`
	EndTime               string `json:"end_time,omitempty"`
	FilterField           string `json:"filter_field,omitempty"`
	FilterOp              string `json:"filter_operator,omitempty"`
	FilterValue           string `json:"filter_value,omitempty"`
	Page                  int    `json:"page,omitempty"`
	PageSize              int    `json:"page_size,omitempty"`
	IncidentID            string `json:"incident_id,omitempty"`
	DetectionID           string `json:"detection_id,omitempty"`
	ViewType              string `json:"view_type,omitempty"`
	ProcessUUID           string `json:"process_uuid,omitempty"`
	ArtifactID            string `json:"artifact_id,omitempty"`
	Query                 string `json:"query,omitempty"`
	StartLine             int    `json:"start_line,omitempty"`
	LineCount             int    `json:"line_count,omitempty"`
	MemoryKey             string `json:"memory_key,omitempty"`
	MemoryValue           string `json:"memory_value,omitempty"`
	TaskID                string `json:"task_id,omitempty"`
	TaskTitle             string `json:"task_title,omitempty"`
	TaskPrompt            string `json:"task_prompt,omitempty"`
	TaskAction            string `json:"task_action,omitempty"`
	TaskStatus            string `json:"task_status,omitempty"`
	TaskFeedback          string `json:"task_feedback,omitempty"`
	TaskIntervalMinutes   int    `json:"task_interval_minutes,omitempty"`
	InstructionName       string `json:"instruction_name,omitempty"`
	Path                  string `json:"path,omitempty"`
	KBTitle               string `json:"kb_title,omitempty"`
	KBQuery               string `json:"kb_query,omitempty"`
	KBContent             string `json:"kb_content,omitempty"`
	KBMode                string `json:"kb_mode,omitempty"`
	KBOldText             string `json:"kb_old_text,omitempty"`
	KBNewText             string `json:"kb_new_text,omitempty"`
	Reason                string `json:"reason,omitempty"`
	Critical              bool   `json:"critical,omitempty"`
	IOCAction             string `json:"ioc_action,omitempty"`
	IOCID                 string `json:"ioc_id,omitempty"`
	IOCHash               string `json:"ioc_hash,omitempty"`
	IOCDescription        string `json:"ioc_description,omitempty"`
	IOCExpirationDate     string `json:"ioc_expiration_date,omitempty"`
	IOCFileName           string `json:"ioc_file_name,omitempty"`
	IOCHostType           string `json:"ioc_host_type,omitempty"`
	IsolateFileGUIDs      string `json:"isolate_file_guids,omitempty"`
	IsolateFileAddExcl    bool   `json:"isolate_file_add_exclusion,omitempty"`
	IsolateFileReleaseAll bool   `json:"isolate_file_release_all,omitempty"`
	PlanName              string `json:"plan_name,omitempty"`
	ScanType              int    `json:"scan_type,omitempty"`
	PlanType              int    `json:"plan_type,omitempty"`
	Scope                 int    `json:"scope,omitempty"`
	RID                   string `json:"rid,omitempty"`
	Time                  int    `json:"time,omitempty"`
	Pid                   int    `json:"pid,omitempty"`
	Ids                   string `json:"ids,omitempty"`
	Allow                 bool   `json:"allow,omitempty"`
	Status                int    `json:"status,omitempty"`
	Scene                 string `json:"scene,omitempty"`
	Comment               string `json:"comment,omitempty"`
	Type                  string `json:"type,omitempty"`
}

type Plan struct {
	DirectReply string     `json:"direct_reply"`
	ToolCalls   []ToolCall `json:"tool_calls"`
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
		if plan.ToolCalls[i].Time < 0 {
			plan.ToolCalls[i].Time = 0
		}
		if plan.ToolCalls[i].Pid < 0 {
			plan.ToolCalls[i].Pid = 0
		}
	}
	plan.DirectReply = strings.TrimSpace(plan.DirectReply)
	s.logger.Info("planner parsed", "tool_count", len(plan.ToolCalls), "direct_reply_preview", preview(plan.DirectReply))
	return plan, nil
}

func buildPlannerUserInput(userText string, toolContext string, summary string, turnText string) string {
	userText = strings.TrimSpace(userText)
	toolContext = strings.TrimSpace(toolContext)
	summary = strings.TrimSpace(summary)
	turnText = strings.TrimSpace(turnText)
	parts := []string{"用户原始问题:\n" + userText}
	if summary != "" {
		parts = append(parts, "会话摘要:\n"+summary)
	}
	if turnText != "" {
		parts = append(parts, "最近几轮对话:\n"+turnText)
	}
	if toolContext != "" {
		parts = append(parts, "已经拿到的真实工具结果:\n"+toolContext)
	}
	parts = append(parts, "请基于这些上下文决定是否还需要继续调用工具。如果用户说“再来一次”“再试一次”“继续”“retry”“again”，优先把它理解为对上一轮相关工具链的延续，而不是重新开始一个泛化回答。")
	parts = append(parts, "如果已经拿到辅助分析结果，并且其中明确写了 enough_to_answer=true，就把它当作子agent已经完成取证。主agent应基于这份结果继续完成回答，而不是继续为了更完整去反复搜索。只有在 enough_to_answer=false，或者用户明确要求更深细节时，才继续规划下一步工具。")
	return strings.Join(parts, "\n\n")
}

func plannerPrompt(skillsPrompt string, memoryText string, latestArtifact protocol.Artifact) string {
	base := "你是一个工具规划器，只负责判断是否要调用工具，并输出 JSON。\n" +
		"可用工具：current_time, edr_hosts, edr_incidents, edr_detections, edr_event_log_alarms, edr_logs, edr_incident_view, edr_detection_view, edr_isolate, edr_release, edr_iocs, edr_ioc_add, edr_ioc_update, edr_ioc_delete, edr_isolate_files, edr_release_isolate_files, edr_tasks, edr_task_result, edr_send_instruction, edr_plan_list, edr_plan_task, edr_virus_by_host, edr_virus_by_hash, edr_virus_hash_hosts, edr_plan_add, edr_plan_edit, edr_plan_cancel, edr_ioas, edr_ioa_audit_log, edr_ioa_networks, edr_strategies, edr_strategy_single, edr_strategy_state, edr_host_offline, edr_host_offline_save, edr_add_host_blacklist, edr_remove_host, edr_batch_deal_incident, edr_incident_r2_summary, edr_detections_proxy, edr_instruction_policy_list, edr_instruction_policy_update, edr_instruction_policy_save_status, edr_instruction_policy_delete, edr_instruction_policy_sort, edr_instruction_policy_add, artifact_search, artifact_read, memory_upsert, memory_delete, scheduled_task_create, scheduled_task_list, scheduled_task_update, scheduled_task_delete, scheduled_task_feedback, knowledge_base_search, knowledge_base_write, knowledge_base_delete。\n" +
		"edr_isolate / edr_release / edr_ioc_add / edr_ioc_update / edr_ioc_delete / edr_delete_isolate_files / edr_release_isolate_files / edr_send_instruction / edr_plan_add / edr_plan_edit / edr_plan_cancel / edr_ioa_add / edr_ioa_update / edr_ioa_delete / edr_ioa_network_add / edr_ioa_network_update / edr_ioa_network_delete / edr_strategy_create / edr_strategy_update / edr_strategy_delete / edr_strategy_status / edr_host_offline_save / edr_add_host_blacklist / edr_remove_host / edr_batch_deal_incident / edr_instruction_policy_update / edr_instruction_policy_save_status / edr_instruction_policy_delete / edr_instruction_policy_sort / edr_instruction_policy_add 属于 critical=true。\n" +
		"优先原则：\n" +
		"1. 如果用户在问当前时间、现在几点、today/now/current time，就优先规划 current_time。\n" +
		"1.1 如果用户在创建、查看、修改、暂停、恢复、删除定时任务，优先规划 scheduled_task_* 工具。没有明确时间要求时，scheduled_task_create 默认 task_interval_minutes=5。\n" +
		"1.2 如果用户说‘这个是误报’‘这个已经处理了’‘别再报这个’，优先规划 scheduled_task_feedback。task_feedback 可用 false_positive、resolved、watch。\n" +
		"1.3 如果用户在查知识库、搜索文档、查已有经验、查手册，优先规划 knowledge_base_search。搜索范围就是 knowledge_base.path 下递归遍历到的 markdown 文件。\n" +
		"1.4 如果用户想新增、编辑、补充知识库，优先规划 knowledge_base_write。默认 kb_mode=upsert；如果明显是追加则用 append；如果明显是把旧内容改成新内容则用 replace_text。\n" +
		"1.5 knowledge_base_write / knowledge_base_delete 的 kb_title 可以直接填写知识库文件标题，也可以直接填写搜索结果里出现的相对路径，例如 runbook/linux/ssh.md。\n" +
		"1.6 如果用户要修改或删除某篇知识库，但目标文件还不够明确，先规划 knowledge_base_search，等看到候选文件后再继续写入或删除。\n" +
		"2. 如果用户在查询主机/事件/检出/日志，就优先规划 EDR 只读工具。\n" +
		"2.1 对 edr_incidents / edr_detections / edr_logs，如果用户明确提到第几页、page、下一页、每页多少条，要把 page 和 page_size 一起填进 tool_calls。\n" +
		"2.2 如果用户在做 hunting / 狩猎 / IOC 扩线 / 进程链排查，优先规划 edr_logs，并尽量提取 client_id、os_type、operation、start_time、end_time，以及一组最关键的 filter_field/filter_operator/filter_value。\n" +
		"2.3 对 edr_logs，filter_operator 优先用 is；如果用户已经给了明确进程名、操作名、系统类型、client_id 或哈希，优先用 is。contain 只用于路径片段、命令行片段、目录片段等模糊试探。\n" +
		"2.4 如果用户明确提到时间范围（最近1小时、今天、昨天、某个时间段），对 edr_logs 要尽量填写 start_time / end_time，格式优先用 YYYY-MM-DD HH:MM:SS。\n" +
		"2.5 如果用户查询某个事件关联的风险/检测/检出，例如'这个事件关联了哪些风险''该事件关联的风险'，优先规划 edr_detections_proxy 并传入 incident_id。incident_id 只能使用用户明确提供或真实工具结果里已经出现过的值。\n" +
		"3. 如果用户给了 incident_id 和 client_id，就优先规划 edr_incident_view。\n" +
		"4. 如果用户给了 detection_id 和 client_id，就优先规划 edr_detection_view；如果有 view_type 和 process_uuid 也一起带上。\n" +
		"4.1 incident_id / detection_id 只能使用用户明确提供或真实工具结果里已经出现过的值，绝对不要根据 host_name、incident_name、时间、样例或自然语言自行拼接猜测。\n" +
		"4.2 如果缺少真实 incident_id / detection_id，就先继续查列表或返回 direct_reply 说明还不能安全调用详情工具。\n" +
		"5. 如果用户在追问刚才那条超大 incident/detection 详情，优先规划 artifact_search；需要看一段连续原文时再规划 artifact_read。\n" +
		"6. 如果真实工具结果里已经出现辅助分析结果且 enough_to_answer=true，优先直接返回 direct_reply，不要继续规划 artifact_search / artifact_read。主agent应把辅助分析结果视为子agent回传的上下文，而不是新的待搜索对象。\n" +
		"7. 只有在辅助分析结果明确 enough_to_answer=false，或者用户明确要求更深的字段/片段/证据时，才继续规划 artifact_search / artifact_read。优先参考 next_queries。\n" +
		"8. artifact_search 需要 query；artifact_read 需要 artifact_id，可选 start_line 和 line_count。\n" +
		"9. 如果用户提供了稳定的长期偏好、资产映射、身份信息、主机别名、工作偏好，可以规划 memory_upsert。\n" +
		"10. 如果用户要求更正或删除旧记忆，可以规划 memory_delete。\n" +
		"11. 如果最近几轮里已经在查某个 incident、detection 或 artifact，而用户只说“再来一次”“继续”“retry”“again”，优先延续那条工具链。\n" +
		"12. 如果用户在查看、列表、搜索 IOC（威胁指标/hash/哈希），优先规划 edr_iocs；如果用户需要查某条 IOC 的详情，优先规划 edr_ioc_detail。\n" +
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
		"20. 如果不需要工具，就返回 direct_reply，tool_calls 为空。\n" +
		"只输出 JSON，不要 markdown。结构：{\"direct_reply\":\"\",\"tool_calls\":[{\"name\":\"\",\"hostname\":\"\",\"client_id\":\"\",\"client_ip\":\"\",\"os_type\":\"\",\"operation\":\"\",\"start_time\":\"\",\"end_time\":\"\",\"filter_field\":\"\",\"filter_operator\":\"\",\"filter_value\":\"\",\"page\":0,\"page_size\":0,\"incident_id\":\"\",\"detection_id\":\"\",\"view_type\":\"\",\"process_uuid\":\"\",\"artifact_id\":\"\",\"query\":\"\",\"start_line\":0,\"line_count\":0,\"memory_key\":\"\",\"memory_value\":\"\",\"task_id\":\"\",\"instruction_name\":\"\",\"path\":\"\",\"time\":0,\"pid\":0,\"task_title\":\"\",\"task_prompt\":\"\",\"task_action\":\"\",\"task_status\":\"\",\"status\":0,\"task_feedback\":\"\",\"task_interval_minutes\":0,\"kb_title\":\"\",\"kb_query\":\"\",\"kb_content\":\"\",\"kb_mode\":\"\",\"kb_old_text\":\"\",\"kb_new_text\":\"\",\"reason\":\"\",\"critical\":false,\"ioc_action\":\"\",\"ioc_hash\":\"\",\"ioc_id\":\"\",\"ioc_description\":\"\",\"ioc_expiration_date\":\"\",\"ioc_file_name\":\"\",\"ioc_host_type\":\"\",\"isolate_file_guids\":\"\",\"isolate_file_add_exclusion\":false,\"isolate_file_release_all\":false,\"type\":\"\"}]}"
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
