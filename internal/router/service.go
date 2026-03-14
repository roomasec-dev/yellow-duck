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
		"可选 action 只有：none, hosts, incidents, detections, logs, isolate, release。\n" +
		"如果是查询主机，尽量提取 hostname 或 client_ip。\n" +
		"如果是查事件、检出、日志，按最接近的 action 返回。\n" +
		"如果用户提到第几页、page、每页多少条，也尽量提取 page 和 page_size。\n" +
		"如果是高危写操作（隔离/恢复），needs_confirmation=true。\n" +
		"只输出 JSON，不要 markdown，不要解释。JSON 结构：{" +
		"\"action\":\"none|hosts|incidents|detections|logs|isolate|release\"," +
		"\"confidence\":0.0," +
		"\"hostname\":\"\"," +
		"\"client_id\":\"\"," +
		"\"client_ip\":\"\"," +
		"\"page\":0," +
		"\"page_size\":0," +
		"\"reason\":\"\"," +
		"\"needs_confirmation\":false}"
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
	case containsAny(plain, "主机", "机器", "终端", "host", "hostname"):
		decision.Action = "hosts"
		decision.Confidence = 0.6
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
