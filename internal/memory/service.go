package memory

import (
	"context"
	"fmt"
	"strings"

	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/store"
)

type Service struct {
	cfg   config.MemoryConfig
	store store.Store
}

func NewService(cfg config.MemoryConfig, dataStore store.Store) *Service {
	return &Service{cfg: cfg, store: dataStore}
}

func (s *Service) ListForContext(ctx context.Context, sessionKey string) ([]protocol.MemoryEntry, error) {
	if !s.cfg.Enabled {
		return nil, nil
	}
	return s.store.ListMemories(ctx, sessionKey, s.cfg.ContextEntries)
}

func (s *Service) Upsert(ctx context.Context, sessionKey string, key string, value string) error {
	if !s.cfg.Enabled {
		return nil
	}
	key = sanitize(key)
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return fmt.Errorf("memory key and value are required")
	}
	if err := s.store.UpsertMemory(ctx, sessionKey, key, value); err != nil {
		return err
	}
	return s.pruneIfNeeded(ctx, sessionKey)
}

func (s *Service) Delete(ctx context.Context, sessionKey string, key string) error {
	if !s.cfg.Enabled {
		return nil
	}
	return s.store.DeleteMemory(ctx, sessionKey, sanitize(key))
}

func (s *Service) pruneIfNeeded(ctx context.Context, sessionKey string) error {
	count, err := s.store.CountMemories(ctx, sessionKey)
	if err != nil || count <= s.cfg.MaxEntries {
		return err
	}
	items, err := s.store.ListMemories(ctx, sessionKey, count)
	if err != nil {
		return err
	}
	for i := s.cfg.MaxEntries; i < len(items); i++ {
		if err := s.store.DeleteMemory(ctx, sessionKey, items[i].Key); err != nil {
			return err
		}
	}
	return nil
}

func sanitize(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	key = strings.ReplaceAll(key, " ", "_")
	return key
}
