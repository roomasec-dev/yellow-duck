package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/store"
)

type Service struct {
	store store.Store
}

func NewService(dataStore store.Store) *Service {
	return &Service{store: dataStore}
}

func (s *Service) SaveJSON(ctx context.Context, sessionKey string, kind string, title string, payload map[string]any) (protocol.Artifact, string, error) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return protocol.Artifact{}, "", err
	}
	text := string(body)
	item, err := s.store.SaveArtifact(ctx, sessionKey, kind, title, text)
	if err != nil {
		return protocol.Artifact{}, "", err
	}
	return item, text, nil
}

func (s *Service) GetLatest(ctx context.Context, sessionKey string) (protocol.Artifact, error) {
	return s.store.GetLatestArtifact(ctx, sessionKey)
}

func (s *Service) Get(ctx context.Context, sessionKey string, artifactID string) (protocol.Artifact, error) {
	if strings.TrimSpace(artifactID) == "" {
		return s.GetLatest(ctx, sessionKey)
	}
	return s.store.GetArtifact(ctx, sessionKey, strings.TrimSpace(artifactID))
}

func (s *Service) Search(ctx context.Context, sessionKey string, artifactID string, query string, maxMatches int) (protocol.Artifact, []protocol.ArtifactMatch, error) {
	item, err := s.Get(ctx, sessionKey, artifactID)
	if err != nil {
		return protocol.Artifact{}, nil, err
	}
	if item.ArtifactID == "" {
		return protocol.Artifact{}, nil, nil
	}
	if maxMatches <= 0 {
		maxMatches = 8
	}
	lines := strings.Split(item.Content, "\n")
	keywords := extractKeywords(query)
	if len(keywords) == 0 {
		keywords = []string{strings.ToLower(strings.TrimSpace(query))}
	}
	var matches []protocol.ArtifactMatch
	for i, line := range lines {
		plain := strings.ToLower(line)
		for _, keyword := range keywords {
			if keyword != "" && strings.Contains(plain, keyword) {
				matches = append(matches, protocol.ArtifactMatch{Line: i + 1, Snippet: strings.TrimSpace(line)})
				break
			}
		}
		if len(matches) >= maxMatches {
			break
		}
	}
	return item, matches, nil
}

func (s *Service) Read(ctx context.Context, sessionKey string, artifactID string, startLine int, lineCount int) (protocol.Artifact, string, error) {
	item, err := s.Get(ctx, sessionKey, artifactID)
	if err != nil {
		return protocol.Artifact{}, "", err
	}
	if item.ArtifactID == "" {
		return protocol.Artifact{}, "", nil
	}
	lines := strings.Split(item.Content, "\n")
	if startLine <= 0 {
		startLine = 1
	}
	if lineCount <= 0 {
		lineCount = 60
	}
	startIdx := min(len(lines), startLine-1)
	endIdx := min(len(lines), startIdx+lineCount)
	parts := make([]string, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		parts = append(parts, fmt.Sprintf("%d: %s", i+1, lines[i]))
	}
	return item, strings.Join(parts, "\n"), nil
}

func BuildSelectiveContext(payloadText string, query string, maxLines int) string {
	lines := strings.Split(payloadText, "\n")
	keywords := extractKeywords(query)
	matched := make(map[int]struct{})
	for i, line := range lines {
		plain := strings.ToLower(line)
		for _, keyword := range keywords {
			if strings.Contains(plain, keyword) {
				for j := max(0, i-2); j <= min(len(lines)-1, i+2); j++ {
					matched[j] = struct{}{}
				}
			}
		}
	}
	if len(matched) == 0 {
		limit := min(maxLines, len(lines))
		return strings.Join(lines[:limit], "\n")
	}
	indexes := make([]int, 0, len(matched))
	for idx := range matched {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	parts := make([]string, 0, min(maxLines, len(indexes)))
	for _, idx := range indexes {
		parts = append(parts, lines[idx])
		if len(parts) >= maxLines {
			break
		}
	}
	return strings.Join(parts, "\n")
}

func BuildOverview(payload map[string]any) string {
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "无顶层字段。"
	}
	if len(keys) > 20 {
		keys = keys[:20]
	}
	return fmt.Sprintf("顶层字段：%s", strings.Join(keys, ", "))
}

func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	return len([]rune(text))/4 + 1
}

func extractKeywords(query string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	query = strings.NewReplacer("，", " ", "。", " ", ",", " ", ":", " ", "：", " ", "\n", " ").Replace(query)
	parts := strings.Fields(query)
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len([]rune(part)) < 2 {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
