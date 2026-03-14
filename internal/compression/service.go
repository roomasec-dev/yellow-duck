package compression

import (
	"context"
	"fmt"
	"strings"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/model"
	"rm_ai_agent/internal/prompt"
	"rm_ai_agent/internal/store"
)

type Service struct {
	cfg    config.CompressionConfig
	store  store.Store
	model  model.Client
	prompt *prompt.Service
}

func NewService(cfg config.CompressionConfig, store store.Store, modelClient model.Client, promptService *prompt.Service) *Service {
	return &Service{cfg: cfg, store: store, model: modelClient, prompt: promptService}
}

func (s *Service) MaybeCompact(ctx context.Context, sessionKey string) error {
	if !s.cfg.Enabled {
		return nil
	}

	count, err := s.store.CountTurns(ctx, sessionKey)
	if err != nil {
		return err
	}
	if count < s.cfg.MaxTurnsBeforeRun {
		return nil
	}

	turns, err := s.store.ListTurns(ctx, sessionKey, 200)
	if err != nil {
		return err
	}
	if len(turns) <= s.cfg.RecentTurnsToKeep {
		return nil
	}

	summary, err := s.store.GetSessionSummary(ctx, sessionKey)
	if err != nil {
		return err
	}
	approx := estimateTokens(summary)
	for _, turn := range turns {
		approx += estimateTokens(turn.Role + ":" + turn.Content)
	}
	if approx < int(float64(s.cfg.ContextWindowTokens)*s.cfg.TriggerRatio) {
		return nil
	}

	olderTurns := turns[:len(turns)-s.cfg.RecentTurnsToKeep]
	parts := make([]string, 0, len(olderTurns)+1)
	if strings.TrimSpace(summary) != "" {
		parts = append(parts, "已有摘要:\n"+strings.TrimSpace(summary))
	}
	for _, turn := range olderTurns {
		parts = append(parts, fmt.Sprintf("%s: %s", turn.Role, strings.TrimSpace(turn.Content)))
	}
	newSummary := strings.Join(parts, "\n")
	if s.model != nil && strings.TrimSpace(s.cfg.Model) != "" {
		systemPrompt := "You are a context compressor. Summarize stable facts, user preferences, asset mappings, important conclusions, and unfinished tasks from the conversation. Remove repetition and filler. Keep only the information that is useful for future turns. Output plain text only."
		if s.prompt != nil {
			systemPrompt = s.prompt.ComposeSystemPrompt(systemPrompt)
		}
		result, err := s.model.Chat(ctx, model.ChatRequest{
			Model: s.cfg.Model,
			Messages: []model.Message{
				{Role: model.RoleSystem, Content: systemPrompt},
				{Role: model.RoleUser, Content: newSummary},
			},
		}, nil)
		if err == nil && strings.TrimSpace(result.Text) != "" {
			newSummary = result.Text
		}
	}
	if len(newSummary) > s.cfg.SummaryMaxBytes {
		newSummary = newSummary[:s.cfg.SummaryMaxBytes]
	}

	return s.store.UpsertSessionSummary(ctx, sessionKey, newSummary)
}

func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	return len([]rune(text))/4 + 1
}
