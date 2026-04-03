package router

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/model"
	"rm_ai_agent/internal/prompt"
)

type Decision struct {
	Action            string  `json:"action"`
	Confidence        float64 `json:"confidence"`
	Hostname          string  `json:"hostname"`
	ClientID          string  `json:"client_id"`
	ClientIP          string  `json:"client_ip"`
	Page              int     `json:"page"`
	PageSize          int     `json:"page_size"`
	Reason            string  `json:"reason"`
	NeedsConfirmation bool    `json:"needs_confirmation"`
	TaskID            string  `json:"task_id"`
	InstructionName   string  `json:"instruction_name"`
	PlanName          string  `json:"plan_name"`
	ScanType          int     `json:"scan_type"`
	PlanType          int     `json:"plan_type"`
	Scope             int     `json:"scope"`
	RID               string  `json:"rid"`
	Path              string  `json:"path"` // for send_instruction batch_params
}

type Service struct {
	cfg    config.RoutingConfig
	model  model.Client
	prompt *prompt.Service
	logger *logx.Logger
}

func NewService(cfg config.RoutingConfig, modelClient model.Client, promptService *prompt.Service, logger *logx.Logger) *Service {
	return &Service{cfg: cfg, model: modelClient, prompt: promptService, logger: logger}
}

func (s *Service) Analyze(ctx context.Context, text string) (Decision, error) {
	if !s.cfg.Enabled {
		return Decision{}, nil
	}

	decision, err := s.analyzeByModel(ctx, text)
	if err == nil && s.valid(decision) {
		return decision, nil
	}
	if err != nil {
		s.logger.Warn("route analyze by model failed", "error", err)
	}
	return heuristicDecision(text), nil
}

func (s *Service) analyzeByModel(ctx context.Context, text string) (Decision, error) {
	if s.model == nil || strings.TrimSpace(s.cfg.Model) == "" {
		return Decision{}, fmt.Errorf("routing model is not configured")
	}

	systemPrompt := "你是 EDR 意图路由器。请把用户输入路由成结构化 JSON。\n" +
		"可选 action 只有：none, hosts, incidents, detections, logs, isolate, release, iocs, tasks, task_result, send_instruction, virus_by_host, virus_by_hash, virus_hash_hosts, virus_scan_record, ioa, ioa_network, strategy, host_offline, plan_list, plan_add, plan_edit, plan_cancel, instruction_policy_list, instruction_policy_update, instruction_policy_save_status, instruction_policy_delete, instruction_policy_sort, instruction_policy_add, remove_host。\n" +
		"如果是查询主机，尽量提取 hostname 或 client_ip。\n" +
		"如果是查事件、检出、日志，按最接近的 action 返回。\n" +
		"如果用户提到第几页、page、每页多少条，也尽量提取 page 和 page_size。\n" +
		"如果是高危写操作（隔离/恢复/删除主机/增删改IOA/增删改策略/新建计划/编辑计划/取消计划/增删改自动响应策略），needs_confirmation=true。\n" +
		"如果是 send_instruction 指令，且提到文件路径（path），必须提取到 path 字段中。\n" +
		"只输出 JSON，不要 markdown，不要解释。JSON 结构：{" +
		"\"action\":\"none|hosts|incidents|detections|logs|isolate|release|iocs|tasks|task_result|send_instruction|virus_by_host|virus_by_hash|virus_hash_hosts|virus_scan_record|ioa|ioa_network|strategy|host_offline|plan_list|plan_add|plan_edit|plan_cancel|instruction_policy_list|instruction_policy_update|instruction_policy_save_status|instruction_policy_delete|instruction_policy_sort|instruction_policy_add|remove_host\"," +
		"\"confidence\":0.0," +
		"\"hostname\":\"\"," +
		"\"client_id\":\"\"," +
		"\"client_ip\":\"\"," +
		"\"page\":0," +
		"\"page_size\":0," +
		"\"task_id\":\"\"," +
		"\"reason\":\"\"," +
		"\"needs_confirmation\":false," +
		"\"instruction_name\":\"\"," +
		"\"path\":\"\"}"
	if s.prompt != nil {
		systemPrompt = s.prompt.ComposeSystemPrompt(systemPrompt)
	}

	result, err := s.model.Chat(ctx, model.ChatRequest{
		Model: s.cfg.Model,
		Messages: []model.Message{
			{
				Role:    model.RoleSystem,
				Content: systemPrompt,
			},
			{Role: model.RoleUser, Content: text},
		},
	}, nil)
	if err != nil {
		return Decision{}, err
	}

	clean := extractJSONObject(result.Text)
	if clean == "" {
		return Decision{}, fmt.Errorf("router returned non-json content")
	}

	var decision Decision
	if err := json.Unmarshal([]byte(clean), &decision); err != nil {
		return Decision{}, fmt.Errorf("decode router json: %w", err)
	}
	return normalizeDecision(decision), nil
}

func (s *Service) valid(d Decision) bool {
	if d.Action == "" {
		return false
	}
	if d.Action == "none" {
		return true
	}
	return d.Confidence >= s.cfg.MinConfidence
}

func normalizeDecision(d Decision) Decision {
	d.Action = strings.ToLower(strings.TrimSpace(d.Action))
	d.Hostname = strings.TrimSpace(d.Hostname)
	d.ClientID = strings.TrimSpace(d.ClientID)
	d.ClientIP = strings.TrimSpace(d.ClientIP)
	d.TaskID = strings.TrimSpace(d.TaskID)
	d.InstructionName = strings.TrimSpace(d.InstructionName)
	d.PlanName = strings.TrimSpace(d.PlanName)
	d.RID = strings.TrimSpace(d.RID)
	if d.Page < 0 {
		d.Page = 0
	}
	if d.PageSize < 0 {
		d.PageSize = 0
	}
	d.Reason = strings.TrimSpace(d.Reason)
	return d
}

func extractJSONObject(text string) string {
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

func heuristicDecision(text string) Decision {
	plain := strings.ToLower(strings.TrimSpace(text))
	decision := Decision{Action: "none", Confidence: 0.2}
	decision.Page, decision.PageSize = parsePaginationHints(text)
	decision.TaskID = extractTaskID(text)
	switch {
	case containsAny(plain, "检出", "detection", "detections"):
		decision.Action = "detections"
		decision.Confidence = 0.7
	case containsAny(plain, "事件", "incident", "incidents", "告警"):
		decision.Action = "incidents"
		decision.Confidence = 0.7
	case containsAny(plain, "日志", "log", "logs", "进程记录"):
		decision.Action = "logs"
		decision.Confidence = 0.7
	case containsAny(plain, "隔离", "isolate"):
		decision.Action = "isolate"
		decision.Confidence = 0.7
		decision.NeedsConfirmation = true
	case containsAny(plain, "恢复主机", "解除隔离", "release"):
		decision.Action = "release"
		decision.Confidence = 0.7
		decision.NeedsConfirmation = true
	case containsAny(plain, "删除主机", "移除主机", "注销主机", "remove host", "delete host"):
		decision.Action = "remove_host"
		decision.Confidence = 0.8
		decision.NeedsConfirmation = true
	case containsAny(plain, "主机", "机器", "终端", "host", "hostname"):
		decision.Action = "hosts"
		decision.Confidence = 0.6
	case containsAny(plain, "ioc", "威胁指标", "hash", "哈希"):
		decision.Action = "iocs"
		decision.Confidence = 0.7
	case containsAny(plain, "任务结果", "task_result", "任务详情"):
		decision.Action = "task_result"
		decision.Confidence = 0.7
	case containsAny(plain, "发送指令", "下发指令"):
		decision.Action = "send_instruction"
		decision.Confidence = 0.9
		decision.InstructionName = extractInstructionName(text)
	case containsAny(plain, "任务", "tasks", "指令任务", "查询任务"):
		decision.Action = "tasks"
		decision.Confidence = 0.7
	case containsAny(plain, "计划列表", "计划记录", "查看计划", "查询计划"):
		decision.Action = "plan_list"
		decision.Confidence = 0.8
	case containsAny(plain, "新建计划", "新增计划", "创建计划", "添加计划"):
		decision.Action = "plan_add"
		decision.Confidence = 0.9
		decision.NeedsConfirmation = true
		decision.PlanName = extractPlanName(text)
		decision.ScanType = extractScanType(text)
		decision.PlanType = extractPlanType(text)
		decision.Scope = extractScope(text)
	case containsAny(plain, "编辑计划", "修改计划", "更新计划"):
		decision.Action = "plan_edit"
		decision.Confidence = 0.9
		decision.NeedsConfirmation = true
		decision.RID = extractRID(text)
		decision.PlanName = extractPlanName(text)
		decision.ScanType = extractScanType(text)
	case containsAny(plain, "取消计划", "删除计划", "停止计划"):
		decision.Action = "plan_cancel"
		decision.Confidence = 0.9
		decision.NeedsConfirmation = true
		decision.RID = extractRID(text)
	case containsAny(plain, "病毒扫描记录", "扫描记录", "virus_scan_record", "scan_record"):
		decision.Action = "virus_scan_record"
		decision.Confidence = 0.8
	case containsAny(plain, "主机病毒", "染毒主机", "中毒主机", "病毒主机"):
		decision.Action = "virus_by_host"
		decision.Confidence = 0.8
	case containsAny(plain, "病毒hash", "病毒哈希", "病毒md5", "病毒sha1"):
		decision.Action = "virus_by_hash"
		decision.Confidence = 0.8
	case containsAny(plain, "hash关联主机", "哈希主机", "md5主机", "sha1主机"):
		decision.Action = "virus_hash_hosts"
		decision.Confidence = 0.8
	case containsAny(plain, "自动响应策略列表", "自动响应列表", "instruction_policy_list", "响应策略列表"):
		decision.Action = "instruction_policy_list"
		decision.Confidence = 0.8
	case containsAny(plain, "新建自动响应策略", "新增自动响应策略", "添加自动响应策略", "instruction_policy_add"):
		decision.Action = "instruction_policy_add"
		decision.Confidence = 0.9
		decision.NeedsConfirmation = true
	case containsAny(plain, "编辑自动响应策略", "修改自动响应策略", "更新自动响应策略", "instruction_policy_update"):
		decision.Action = "instruction_policy_update"
		decision.Confidence = 0.9
		decision.NeedsConfirmation = true
	case containsAny(plain, "删除自动响应策略", "移除自动响应策略", "instruction_policy_delete"):
		decision.Action = "instruction_policy_delete"
		decision.Confidence = 0.9
		decision.NeedsConfirmation = true
	case containsAny(plain, "启用自动响应策略", "禁用自动响应策略", "启停自动响应策略", "instruction_policy_save_status"):
		decision.Action = "instruction_policy_save_status"
		decision.Confidence = 0.9
		decision.NeedsConfirmation = true
	case containsAny(plain, "排序自动响应策略", "自动响应策略排序", "instruction_policy_sort"):
		decision.Action = "instruction_policy_sort"
		decision.Confidence = 0.8
	}
	return decision
}

func parsePaginationHints(text string) (int, int) {
	page := firstIntMatch(text,
		`第\s*(\d+)\s*页`,
		`[Pp]age\s*(\d+)`,
		`页码\s*(\d+)`,
	)
	pageSize := firstIntMatch(text,
		`每页\s*(\d+)\s*条`,
		`每页\s*(\d+)`,
		`page[_\s-]*size\s*(\d+)`,
		`每次\s*(\d+)\s*条`,
	)
	return page, pageSize
}

func extractTaskID(text string) string {
	taskIDPatterns := []string{
		`(?:task[_\s-]?id|task_id|任务ID|任务\s*ID)[:\s=]*([a-zA-Z0-9\-_]+)`,
		`(?:任务|task)\s+([a-zA-Z0-9\-_]+)\s*(?:结果|的\s*结果)`,
		`(?:结果|result)\s+(?:task[_\s]?id\s*)?([a-zA-Z0-9\-_]+)`,
	}
	for _, pattern := range taskIDPatterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(text)
		if len(match) >= 2 {
			taskID := strings.TrimSpace(match[1])
			if taskID != "" {
				return taskID
			}
		}
	}
	return ""
}

func extractInstructionName(text string) string {
	patterns := []string{
		`(?:发送指令|下发指令)\s+([a-zA-Z_][a-zA-Z0-9_]*)`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(text)
		if len(match) >= 2 {
			name := strings.TrimSpace(match[1])
			if name != "" {
				return name
			}
		}
	}
	return ""
}

func firstIntMatch(text string, patterns ...string) int {
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(text)
		if len(match) < 2 {
			continue
		}
		value, err := strconv.Atoi(strings.TrimSpace(match[1]))
		if err == nil && value > 0 {
			return value
		}
	}
	return 0
}

func containsAny(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func extractPlanName(text string) string {
	patterns := []string{
		`(?:计划名|plan_name|计划名称)[=:]\s*([^\s,，]+)`,
		`(?:新建|创建|添加)[^\s]*\s+(\S+)(?:计划|扫描)`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(text)
		if len(match) >= 2 {
			name := strings.TrimSpace(match[1])
			if name != "" {
				return name
			}
		}
	}
	return ""
}

func extractScanType(text string) int {
	plain := strings.ToLower(text)
	switch {
	case containsAny(plain, "快速扫描", "快速", "quick"):
		return 1
	case containsAny(plain, "全盘扫描", "全盘", "full"):
		return 2
	case containsAny(plain, "自定义扫描", "自定义路径", "custom", "自定义路径扫描"):
		return 3
	case containsAny(plain, "漏洞修复", "leak_repair"):
		return 4
	case containsAny(plain, "安装软件", "distribute_software"):
		return 5
	case containsAny(plain, "卸载软件"):
		return 6
	case containsAny(plain, "更新软件"):
		return 7
	case containsAny(plain, "发送文件", "distribute_file"):
		return 8
	}
	return 0
}

func extractPlanType(text string) int {
	plain := strings.ToLower(text)
	switch {
	case containsAny(plain, "立即执行", "立即", "立刻"):
		return 1
	case containsAny(plain, "定时执行", "定时", "schedule"):
		return 2
	case containsAny(plain, "周期执行", "周期", "循环"):
		return 3
	}
	return 0
}

func extractScope(text string) int {
	plain := strings.ToLower(text)
	switch {
	case containsAny(plain, "特定主机", "单台", "单主机", "指定主机"):
		return 1
	case containsAny(plain, "主机组", "分组", "group"):
		return 2
	case containsAny(plain, "全网", "全部主机", "所有主机"):
		return 3
	}
	return 0
}

func extractRID(text string) string {
	patterns := []string{
		`(?:rid|plan_id|计划ID)[=:]\s*([a-zA-Z0-9\-_]+)`,
		`(?:计划)\s*ID[:\s=]*([a-zA-Z0-9\-_]+)`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(text)
		if len(match) >= 2 {
			rid := strings.TrimSpace(match[1])
			if rid != "" {
				return rid
			}
		}
	}
	return ""
}
