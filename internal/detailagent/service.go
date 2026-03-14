package detailagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/model"
	"rm_ai_agent/internal/prompt"
	"rm_ai_agent/internal/protocol"
)

type Service struct {
	cfg    config.DetailAgentConfig
	model  model.Client
	prompt *prompt.Service
	logger *logx.Logger
}

type Report struct {
	EnoughToAnswer bool     `json:"enough_to_answer"`
	Summary        string   `json:"summary"`
	Evidence       []string `json:"evidence"`
	Gaps           []string `json:"gaps,omitempty"`
	NextQueries    []string `json:"next_queries,omitempty"`
}

func NewService(cfg config.DetailAgentConfig, modelClient model.Client, promptService *prompt.Service, logger *logx.Logger) *Service {
	return &Service{cfg: cfg, model: modelClient, prompt: promptService, logger: logger}
}

func (s *Service) Enabled() bool {
	return s != nil && s.cfg.Enabled && s.model != nil && strings.TrimSpace(s.cfg.Model) != ""
}

func (s *Service) DirectMaxBytes() int {
	if s == nil || s.cfg.DirectMaxBytes <= 0 {
		return 12 * 1024
	}
	return s.cfg.DirectMaxBytes
}

func (s *Service) MaxInputBytes() int {
	if s == nil || s.cfg.MaxInputBytes <= 0 {
		return 64 * 1024
	}
	return s.cfg.MaxInputBytes
}

func (s *Service) SummarizeDetail(ctx context.Context, item protocol.Artifact, overview string, payloadText string, focus string) (Report, error) {
	content, truncated := capText(payloadText, s.MaxInputBytes())
	task := fmt.Sprintf("你在阅读一条较大的 %s 详情。请为主agent产出一份可继续推理的中文摘要。", item.Kind)
	extra := fmt.Sprintf("artifact_id=%s\n标题=%s\n顶层概览=%s\n关注点=%s", item.ArtifactID, item.Title, strings.TrimSpace(overview), strings.TrimSpace(focus))
	return s.summarize(ctx, task, extra, content, truncated, "请判断当前信息是否已经足够让主agent直接回答用户。如果够了，enough_to_answer=true，并给出精炼 summary 和 evidence；如果不够，enough_to_answer=false，并在 gaps / next_queries 里说明还缺什么、下一步该搜什么。")
}

func (s *Service) SummarizeSearch(ctx context.Context, item protocol.Artifact, query string, result string) (Report, error) {
	content, truncated := capText(result, s.MaxInputBytes())
	task := fmt.Sprintf("你在阅读 %s 详情中的关键词搜索结果。请告诉主agent这些命中说明了什么。", item.Kind)
	extra := fmt.Sprintf("artifact_id=%s\n标题=%s\n搜索词=%s", item.ArtifactID, item.Title, strings.TrimSpace(query))
	return s.summarize(ctx, task, extra, content, truncated, "请判断这些命中是否已经足够回答用户当前问题。如果够了，enough_to_answer=true；如果还不够，明确 gaps 和 next_queries。evidence 里保留关键行号。")
}

func (s *Service) SummarizeRead(ctx context.Context, item protocol.Artifact, startLine int, lineCount int, chunk string) (Report, error) {
	content, truncated := capText(chunk, s.MaxInputBytes())
	task := fmt.Sprintf("你在阅读 %s 详情中的局部片段。请把这段片段解释给主agent。", item.Kind)
	extra := fmt.Sprintf("artifact_id=%s\n标题=%s\n读取范围=start:%d line_count:%d", item.ArtifactID, item.Title, startLine, lineCount)
	return s.summarize(ctx, task, extra, content, truncated, "请判断这段片段是否已经足够回答用户当前问题。如果够了，enough_to_answer=true；如果还不够，明确 gaps 和 next_queries。evidence 里保留关键行号。")
}

func (s *Service) summarize(ctx context.Context, task string, extra string, content string, truncated bool, outputHint string) (Report, error) {
	if !s.Enabled() {
		return Report{}, fmt.Errorf("detail agent disabled")
	}
	partial := "否"
	if truncated {
		partial = "是，只基于截取片段"
	}
	systemPrompt := "你是主agent的辅助分析agent，专门阅读超大的 incident/detection 详情。你的输出不是直接发给用户，而是发给主agent继续推理。请只基于给定真实内容输出 JSON，不要额外解释，不要 markdown。JSON 结构固定为：{\"enough_to_answer\":true,\"summary\":\"\",\"evidence\":[\"\"],\"gaps\":[\"\"],\"next_queries\":[\"\"]}。当信息已经足够支撑主agent直接回答当前问题时，enough_to_answer=true，并尽量不要再给 next_queries；只有在确实还缺关键信息时，才返回 enough_to_answer=false。"
	if s.prompt != nil {
		systemPrompt = s.prompt.ComposeSystemPrompt(systemPrompt)
	}
	result, err := s.model.Chat(ctx, model.ChatRequest{
		Model: s.cfg.Model,
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: systemPrompt},
			{Role: model.RoleUser, Content: task + "\n\n附加信息：\n" + strings.TrimSpace(extra) + "\n\n当前内容是否截断：" + partial + "\n\n真实内容：\n" + strings.TrimSpace(content) + "\n\n输出要求：\n" + outputHint},
		},
	}, nil)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("detail agent summarize failed", "error", err)
		}
		return Report{}, err
	}
	report, err := parseReport(result.Text)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("detail agent parse failed", "error", err, "preview", preview(result.Text))
		}
		return Report{}, err
	}
	return report, nil
}

func parseReport(text string) (Report, error) {
	jsonText := extractJSON(text)
	if jsonText == "" {
		return Report{}, fmt.Errorf("detail agent returned non-json content")
	}
	var report Report
	if err := json.Unmarshal([]byte(jsonText), &report); err != nil {
		return Report{}, err
	}
	report.Summary = strings.TrimSpace(report.Summary)
	report.Evidence = cleanList(report.Evidence)
	report.Gaps = cleanList(report.Gaps)
	report.NextQueries = cleanList(report.NextQueries)
	if report.Summary == "" {
		return Report{}, fmt.Errorf("detail agent returned empty summary")
	}
	return report, nil
}

func cleanList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
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

func capText(text string, maxBytes int) (string, bool) {
	text = strings.TrimSpace(text)
	if maxBytes <= 0 || len([]byte(text)) <= maxBytes {
		return text, false
	}
	trimmed := strings.TrimSpace(text)
	for len([]byte(trimmed)) > maxBytes && len(trimmed) > 0 {
		trimmed = trimmed[:len(trimmed)-1]
	}
	if utf8.ValidString(trimmed) {
		return trimmed, true
	}
	for len(trimmed) > 0 && !utf8.ValidString(trimmed) {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return strings.TrimSpace(trimmed), true
}
