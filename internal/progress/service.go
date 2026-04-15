package progress

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/model"
	"rm_ai_agent/internal/prompt"
	"rm_ai_agent/internal/protocol"
)

type Sink interface {
	SendProgress(ctx context.Context, session protocol.SessionRef, text string) error
}

type Service struct {
	cfg    config.ProgressConfig
	model  model.Client
	logger *logx.Logger
	prompt *prompt.Service
}

type Reporter struct {
	service       *Service
	sink          Sink
	session       protocol.SessionRef
	sent          int
	toolSent      int
	lastSent      time.Time
	lastText      string
	lastStage     string
	lastToolStage string
	lastStageSent time.Time
}

func NewService(cfg config.ProgressConfig, modelClient model.Client, promptService *prompt.Service, logger *logx.Logger) *Service {
	return &Service{cfg: cfg, model: modelClient, prompt: promptService, logger: logger}
}

func (s *Service) NewReporter(session protocol.SessionRef, sink Sink) *Reporter {
	if sink == nil {
		return nil
	}
	return &Reporter{service: s, sink: sink, session: session}
}

func (r *Reporter) Step(ctx context.Context, detail string) {
	if r == nil || r.service == nil || r.sink == nil || !r.service.cfg.Enabled {
		return
	}
	if r.sent >= r.service.cfg.MaxUpdates && time.Since(r.lastSent) < 15*time.Second {
		return
	}

	text := r.service.render(ctx, detail)
	if strings.TrimSpace(text) == "" || strings.TrimSpace(text) == strings.TrimSpace(r.lastText) {
		return
	}
	if err := r.sink.SendProgress(ctx, r.session, text); err != nil {
		r.service.logger.Warn("send progress failed", "session_id", r.session.PublicID, "error", err)
		return
	}
	r.sent++
	r.lastSent = time.Now()
	r.lastText = text
}

func (r *Reporter) Stage(ctx context.Context, stage string, detail string) {
	if r == nil || r.service == nil || r.sink == nil || !r.service.cfg.Enabled {
		return
	}
	stage = normalizeStage(stage)
	if stage != "" && stage == r.lastStage && time.Since(r.lastStageSent) < 20*time.Second {
		return
	}
	if stage != "" {
		r.lastStage = stage
		r.lastStageSent = time.Now()
	}
	r.Step(ctx, detail)
}

func (r *Reporter) ToolStart(ctx context.Context, tool string, detail string) {
	stage := toolStage(tool)
	if stage != "" {
		r.Stage(ctx, stage, detail)
		return
	}
	r.Step(ctx, detail)
}

func (r *Reporter) ToolResult(ctx context.Context, tool string, result string) {
	if r == nil || r.service == nil || r.sink == nil || !r.service.cfg.Enabled {
		return
	}
	if stage := toolStage(tool); stage != "" && stage == r.lastToolStage && time.Since(r.lastSent) < 45*time.Second {
		return
	}
	if r.toolSent >= r.service.cfg.MaxToolUpdates && time.Since(r.lastSent) < 15*time.Second {
		return
	}

	text := r.service.renderToolResult(ctx, tool, result)
	if strings.TrimSpace(text) == "" || strings.TrimSpace(text) == strings.TrimSpace(r.lastText) {
		return
	}
	if err := r.sink.SendProgress(ctx, r.session, text); err != nil {
		r.service.logger.Warn("send tool progress failed", "session_id", r.session.PublicID, "tool", tool, "error", err)
		return
	}
	r.toolSent++
	r.lastSent = time.Now()
	r.lastText = text
	r.lastToolStage = toolStage(tool)
}

func normalizeStage(stage string) string {
	stage = strings.ToLower(strings.TrimSpace(stage))
	switch stage {
	case "overview", "pick_target", "drill_down", "answer":
		return stage
	default:
		return ""
	}
}

func toolStage(tool string) string {
	switch strings.TrimSpace(tool) {
	case "edr_incidents", "edr_detections", "edr_logs", "edr_event_log_alarms", "edr_hosts":
		return "overview"
	case "edr_incident_view", "edr_detection_view":
		return "pick_target"
	case "artifact_outline", "artifact_search", "artifact_read":
		return "drill_down"
	default:
		return ""
	}
}

func (s *Service) render(ctx context.Context, detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return "我在继续处理这件事。"
	}
	if s.model == nil || strings.TrimSpace(s.cfg.Model) == "" {
		return fallback(detail)
	}

	result, err := s.model.Chat(ctx, model.ChatRequest{
		Model: s.cfg.Model,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: s.progressPrompt()},
			{Role: model.RoleUser, Content: fmt.Sprintf("内部步骤：%s", detail)},
		},
	}, nil)
	if err != nil {
		s.logger.Warn("render progress with model failed", "error", err)
		return fallback(detail)
	}

	text := strings.TrimSpace(result.Text)
	if text == "" {
		return fallback(detail)
	}
	return text
}

func (s *Service) renderToolResult(ctx context.Context, tool string, result string) string {
	tool = strings.TrimSpace(tool)
	result = strings.TrimSpace(result)
	if result == "" {
		return ""
	}
	preview := progressPreview(result, 900)
	if s.model == nil || strings.TrimSpace(s.cfg.Model) == "" {
		return fallbackToolResult(tool, preview)
	}

	resultText, err := s.model.Chat(ctx, model.ChatRequest{
		Model: s.cfg.Model,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: s.toolResultPrompt()},
			{Role: model.RoleUser, Content: fmt.Sprintf("工具名称：%s\n真实结果：%s", tool, preview)},
		},
	}, nil)
	if err != nil {
		s.logger.Warn("render tool progress with model failed", "tool", tool, "error", err)
		return fallbackToolResult(tool, preview)
	}

	text := strings.TrimSpace(resultText.Text)
	if text == "" {
		return fallbackToolResult(tool, preview)
	}
	return text
}

func fallback(detail string) string {
	plain := strings.TrimSpace(detail)
	switch {
	case strings.HasPrefix(plain, "调查计划："), strings.HasPrefix(plain, "下一步计划："):
		return plain
	case strings.Contains(plain, "上下文"):
		return "我在整理一下上下文和关键线索。"
	case strings.Contains(plain, "EDR") || strings.Contains(plain, "主机"):
		return "我在跑一个小检查，先把目标信息拉回来。"
	case strings.Contains(plain, "模型") || strings.Contains(plain, "思考"):
		return "我在把信息串起来，准备给你一个直接可用的结论。"
	default:
		return "我在继续处理这件事，马上给你结果。"
	}
}

func fallbackToolResult(tool string, preview string) string {
	switch tool {
	case "edr_incidents":
		return "我已经拉到近期事件列表，正在挑最值得展开的一条。"
	case "edr_detections":
		return "我已经拿到近期检出结果，正在筛关键风险线索。"
	case "edr_logs":
		return "我已经拿到行为日志，正在提炼关键操作轨迹。"
	case "edr_incident_view":
		return "我已经拿到这条事件详情，正在整理主机、时间线和关键行为。"
	case "edr_detection_view":
		return "我已经拿到这条检出详情，正在整理主体和关键证据。"
	case "artifact_outline":
		return "我已经看清这份大对象的结构了，接下来会顺着重点字段继续深挖。"
	case "artifact_search":
		return "我已经在大对象详情里定位到关键片段，正在串联上下文。"
	case "artifact_read":
		return "我已经读到详情里的关键片段，正在把事件经过讲清楚。"
	default:
		if strings.Contains(preview, "共找到") {
			return "我已经拿到一批真实结果，正在压缩成最关键的结论。"
		}
		return "我已经拿到中间结果，正在继续整理重点。"
	}
}

func progressPreview(text string, limit int) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

func (s *Service) progressPrompt() string {
	base := "你是 AI 助手的进度播报员。请把内部操作步骤改写成发给终端用户的一句中文进度说明。要求：第一人称、18 到 40 字、友好自然、像靠谱分析员同步阶段进展、优先反映新的阶段变化、避免重复同义句、不要输出编号、不要泄露路径/密钥/API 细节、不要夸大结果、不要使用 markdown。可以偶尔带 1 个轻量 emoji（如🙂、🔎），但不要每句都带。只输出一句话。"
	if s.prompt != nil {
		if loaded := s.prompt.LoadPrompt("progress_reporter"); strings.TrimSpace(loaded) != "" {
			base = loaded
		}
		return s.prompt.ComposeSystemPrompt(base)
	}
	return base
}

func (s *Service) toolResultPrompt() string {
	base := "你是 AI 助手的进度播报员。现在某个工具已经返回了真实结果，请把最值得告诉终端用户的中间结论改写成一句中文进度说明。要求：第一人称、20 到 50 字、友好自然、像分析员汇报阶段性发现、只基于给定真实结果、不编造没有出现的字段、不输出编号或 markdown、不泄露路径/密钥/API 细节。如果只是同一阶段的重复小动作，就不要换着说法重复播报。只输出一句话。"
	if s.prompt != nil {
		if loaded := s.prompt.LoadPrompt("progress_tool_result"); strings.TrimSpace(loaded) != "" {
			base = loaded
		}
		return s.prompt.ComposeSystemPrompt(base)
	}
	return base
}
