package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/logx"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role
	Content string
}

type ChatRequest struct {
	SessionKey string
	Model      string
	Messages   []Message
}

type ChatChunk struct {
	Content string
	Done    bool
}

type ChatResult struct {
	Text  string
	Model string
}

type Client interface {
	Chat(ctx context.Context, req ChatRequest, onChunk func(ChatChunk) error) (ChatResult, error)
}

type FallbackClient struct {
	cfg    config.ModelsConfig
	http   *http.Client
	logger *logx.Logger
}

func NewFallbackClient(cfg config.ModelsConfig, logger *logx.Logger) *FallbackClient {
	return &FallbackClient{
		cfg: cfg,
		http: &http.Client{
			Timeout: 90 * time.Second,
		},
		logger: logger,
	}
}

func (c *FallbackClient) Chat(ctx context.Context, req ChatRequest, onChunk func(ChatChunk) error) (ChatResult, error) {
	candidates := c.resolveCandidates(req)
	var errs []string
	for _, candidate := range candidates {
		result, err := c.chatOnce(ctx, candidate, req, onChunk)
		if err == nil {
			return result, nil
		}
		if c.logger != nil {
			c.logger.Warn("model candidate failed", "candidate", candidate.provider+"/"+candidate.model, "error", err)
		}
		errs = append(errs, candidate.provider+"/"+candidate.model+": "+err.Error())
	}

	if len(errs) == 0 {
		return ChatResult{}, fmt.Errorf("no model candidate configured")
	}
	return ChatResult{}, fmt.Errorf("all model candidates failed: %s", strings.Join(errs, " | "))
}

type candidate struct {
	provider string
	model    string
}

func (c *FallbackClient) resolveCandidates(req ChatRequest) []candidate {
	seen := make(map[string]struct{})
	push := func(provider string, model string, list *[]candidate) {
		provider = strings.TrimSpace(provider)
		model = strings.TrimSpace(model)
		if provider == "" || model == "" {
			return
		}
		key := provider + "/" + model
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		*list = append(*list, candidate{provider: provider, model: model})
	}

	var out []candidate
	if strings.TrimSpace(req.Model) != "" {
		provider, model := c.parseModelRef(req.Model)
		push(provider, model, &out)
	} else {
		push(c.cfg.DefaultProvider, c.cfg.DefaultModel, &out)
	}
	if len(out) > 0 && strings.TrimSpace(out[0].provider) != "stub" {
		out = filterNonStubCandidates(out)
	}
	return out
}

func filterNonStubCandidates(candidates []candidate) []candidate {
	hasNonStub := false
	for _, candidate := range candidates {
		if candidate.provider != "stub" {
			hasNonStub = true
			break
		}
	}
	if !hasNonStub {
		return candidates
	}
	filtered := make([]candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.provider == "stub" {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func (c *FallbackClient) parseModelRef(ref string) (string, string) {
	if strings.Contains(ref, "/") {
		parts := strings.SplitN(ref, "/", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return c.cfg.DefaultProvider, strings.TrimSpace(ref)
}

func (c *FallbackClient) chatOnce(ctx context.Context, target candidate, req ChatRequest, onChunk func(ChatChunk) error) (ChatResult, error) {
	providerCfg, ok := c.cfg.Providers[target.provider]
	if !ok {
		return ChatResult{}, fmt.Errorf("provider not found")
	}

	switch providerType(providerCfg.Type) {
	case providerTypeStub:
		return c.chatStub(ctx, req, target, onChunk)
	case providerTypeOpenAICompatible, providerTypeDeepSeek:
		return c.chatOpenAICompatible(ctx, req, target, providerCfg, onChunk)
	default:
		return ChatResult{}, fmt.Errorf("unsupported provider type %q", providerCfg.Type)
	}
}

type providerType string

const (
	providerTypeStub             providerType = "stub"
	providerTypeOpenAICompatible providerType = "openai_compatible"
	providerTypeDeepSeek         providerType = "deepseek"
)

func (c *FallbackClient) chatStub(ctx context.Context, req ChatRequest, target candidate, onChunk func(ChatChunk) error) (ChatResult, error) {
	_ = ctx
	lastUserMessage := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == RoleUser {
			lastUserMessage = req.Messages[i].Content
			break
		}
	}

	text := fmt.Sprintf("已收到你的消息：%s\n\n当前是脚手架模式：主 Chat、会话记忆、SQLite 存储、飞书 webhook 和 EDR 工具入口已经接好。下一步可以把真实模型 provider 配进去，让主 Chat 从自然语言自动编排 EDR 操作。", strings.TrimSpace(lastUserMessage))
	if onChunk != nil {
		if err := onChunk(ChatChunk{Content: text, Done: false}); err != nil {
			return ChatResult{}, err
		}
		if err := onChunk(ChatChunk{Done: true}); err != nil {
			return ChatResult{}, err
		}
	}

	return ChatResult{Text: text, Model: target.provider + "/" + target.model}, nil
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature *float64            `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *FallbackClient) chatOpenAICompatible(ctx context.Context, req ChatRequest, target candidate, providerCfg config.ProviderConfig, onChunk func(ChatChunk) error) (ChatResult, error) {
	apiKey := strings.TrimSpace(providerCfg.APIKey)
	if apiKey == "" && providerCfg.APIKeyEnv != "" {
		apiKey = strings.TrimSpace(os.Getenv(providerCfg.APIKeyEnv))
	}
	if apiKey == "" {
		return ChatResult{}, fmt.Errorf("missing api key for provider %q", target.provider)
	}

	baseURL := strings.TrimRight(strings.TrimSpace(providerCfg.BaseURL), "/")
	if baseURL == "" {
		return ChatResult{}, fmt.Errorf("missing base_url for provider %q", target.provider)
	}

	requestBody := openAIChatRequest{
		Model:    target.model,
		Messages: make([]openAIChatMessage, 0, len(req.Messages)),
		Stream:   false,
	}
	settings := c.resolveModelSettings(target, providerCfg)
	if settings.Temperature > 0 {
		temp := settings.Temperature
		requestBody.Temperature = &temp
	}
	if settings.MaxTokens > 0 {
		requestBody.MaxTokens = normalizeMaxTokens(providerType(providerCfg.Type), settings.MaxTokens)
	}
	for _, msg := range req.Messages {
		requestBody.Messages = append(requestBody.Messages, openAIChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return ChatResult{}, fmt.Errorf("marshal chat request: %w", err)
	}

	url := baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ChatResult{}, fmt.Errorf("create chat request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return ChatResult{}, fmt.Errorf("send chat request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return ChatResult{}, fmt.Errorf("read chat response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return ChatResult{}, fmt.Errorf("chat api http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var response openAIChatResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return ChatResult{}, fmt.Errorf("decode chat response: %w", err)
	}
	if response.Error != nil && response.Error.Message != "" {
		return ChatResult{}, fmt.Errorf("chat api error: %s", response.Error.Message)
	}
	if len(response.Choices) == 0 {
		return ChatResult{}, fmt.Errorf("chat api returned no choices")
	}

	text := strings.TrimSpace(response.Choices[0].Message.Content)
	if text == "" {
		return ChatResult{}, fmt.Errorf("chat api returned empty content")
	}
	if onChunk != nil {
		if err := onChunk(ChatChunk{Content: text, Done: false}); err != nil {
			return ChatResult{}, err
		}
		if err := onChunk(ChatChunk{Done: true}); err != nil {
			return ChatResult{}, err
		}
	}

	return ChatResult{Text: text, Model: target.provider + "/" + target.model}, nil
}

func normalizeMaxTokens(kind providerType, value int) int {
	if value <= 0 {
		return 0
	}
	if kind == providerTypeDeepSeek && value > 8192 {
		return 8192
	}
	return value
}

func (c *FallbackClient) resolveModelSettings(target candidate, providerCfg config.ProviderConfig) config.ModelSettings {
	settings := config.ModelSettings{
		Temperature: providerCfg.Temperature,
		MaxTokens:   providerCfg.MaxTokens,
	}
	if override, ok := c.cfg.ModelSettings[target.provider+"/"+target.model]; ok {
		if override.Temperature > 0 {
			settings.Temperature = override.Temperature
		}
		if override.MaxTokens > 0 {
			settings.MaxTokens = override.MaxTokens
		}
	}
	return settings
}
