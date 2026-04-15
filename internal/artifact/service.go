package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/store"
)

type Service struct {
	store store.Store
}

type Outline struct {
	TopLevelKeys []string
	Arrays       []ArrayOutline
	SignalPaths  []string
	SampleFields []string
	TotalLines   int
	TotalBytes   int
}

type ArrayOutline struct {
	Path   string
	Count  int
	Fields []string
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

func (s *Service) Outline(ctx context.Context, sessionKey string, artifactID string) (protocol.Artifact, Outline, error) {
	item, err := s.Get(ctx, sessionKey, artifactID)
	if err != nil {
		return protocol.Artifact{}, Outline{}, err
	}
	if item.ArtifactID == "" {
		return protocol.Artifact{}, Outline{}, nil
	}
	outline := Outline{
		TotalLines: len(strings.Split(item.Content, "\n")),
		TotalBytes: len([]byte(item.Content)),
	}
	var root any
	if err := json.Unmarshal([]byte(item.Content), &root); err != nil {
		outline.TopLevelKeys = []string{"非 JSON 或无法解析，建议用 artifact_search / artifact_read 按文本探索"}
		return item, outline, nil
	}
	outline.TopLevelKeys = topLevelKeys(root)
	outline.Arrays = collectArrayOutlines(root, "", 12)
	outline.SignalPaths = collectSignalPaths(root, "", 24)
	outline.SampleFields = collectSampleFields(root, 24)
	return item, outline, nil
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
	if matches := searchJSON(item.Content, query, maxMatches); len(matches) > 0 {
		return item, matches, nil
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
				lineNo := i + 1
				matches = append(matches, protocol.ArtifactMatch{Line: lineNo, StartLine: max(1, lineNo-6), EndLine: min(len(lines), lineNo+10), Snippet: strings.TrimSpace(line)})
				break
			}
		}
		if len(matches) >= maxMatches {
			break
		}
	}
	return item, matches, nil
}

func searchJSON(content string, query string, maxMatches int) []protocol.ArtifactMatch {
	keywords := extractKeywords(query)
	if len(keywords) == 0 {
		return nil
	}
	var root any
	if err := json.Unmarshal([]byte(content), &root); err != nil {
		return nil
	}
	lines := strings.Split(content, "\n")
	seen := make(map[string]struct{})
	var matches []protocol.ArtifactMatch
	walkJSON(root, "", nil, func(path string, value any, container any) bool {
		if !matchesQuery(path, value, keywords) {
			return true
		}
		contextValue := compactContext(container)
		if contextValue == "" {
			contextValue = compactContext(value)
		}
		key := path + "\x00" + contextValue
		if _, ok := seen[key]; ok {
			return true
		}
		seen[key] = struct{}{}
		line := locateLine(lines, path, value)
		if line <= 0 {
			line = 1
		}
		matches = append(matches, protocol.ArtifactMatch{
			Line:      line,
			StartLine: max(1, line-8),
			EndLine:   min(len(lines), line+16),
			Path:      path,
			Snippet:   contextValue,
		})
		return len(matches) < maxMatches
	})
	return matches
}

func walkJSON(value any, path string, container any, visit func(path string, value any, container any) bool) bool {
	if !visit(path, value, container) {
		return false
	}
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			childPath := joinPath(path, key)
			if !walkJSON(typed[key], childPath, typed, visit) {
				return false
			}
		}
	case []any:
		for i, item := range typed {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if path == "" {
				childPath = fmt.Sprintf("[%d]", i)
			}
			if !walkJSON(item, childPath, item, visit) {
				return false
			}
		}
	}
	return true
}

func matchesQuery(path string, value any, keywords []string) bool {
	pathText := strings.ToLower(path)
	valueText := strings.ToLower(scalarText(value))
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		if strings.Contains(pathText, keyword) || strings.Contains(valueText, keyword) {
			return true
		}
	}
	return false
}

func scalarText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64, bool, nil:
		return fmt.Sprint(typed)
	default:
		return ""
	}
}

func compactContext(value any) string {
	if value == nil {
		return ""
	}
	if reflect.TypeOf(value).Kind() != reflect.Map && reflect.TypeOf(value).Kind() != reflect.Slice {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	body, err := json.Marshal(value)
	if err != nil {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	text := string(body)
	if len([]rune(text)) > 1200 {
		return string([]rune(text)[:1200]) + "..."
	}
	return text
}

func locateLine(lines []string, path string, value any) int {
	candidates := []string{lastPathToken(path), scalarText(value)}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || candidate == "<nil>" {
			continue
		}
		for i, line := range lines {
			if strings.Contains(line, candidate) {
				return i + 1
			}
		}
	}
	return 1
}

func lastPathToken(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if idx := strings.LastIndex(path, "."); idx >= 0 && idx+1 < len(path) {
		path = path[idx+1:]
	}
	if idx := strings.Index(path, "["); idx >= 0 {
		path = path[:idx]
	}
	return strings.TrimSpace(path)
}

func topLevelKeys(root any) []string {
	object, ok := root.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("root=%T", root)}
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 40 {
		keys = keys[:40]
	}
	return keys
}

func collectArrayOutlines(root any, basePath string, limit int) []ArrayOutline {
	var out []ArrayOutline
	walkJSON(root, basePath, nil, func(path string, value any, container any) bool {
		items, ok := value.([]any)
		if !ok {
			return true
		}
		fields := sampleObjectFields(items, 16)
		out = append(out, ArrayOutline{Path: blankToRoot(path), Count: len(items), Fields: fields})
		return len(out) < limit
	})
	return out
}

func collectSignalPaths(root any, basePath string, limit int) []string {
	signalKeys := []string{"process", "command", "cmd", "md5", "sha1", "sha256", "hash", "file", "path", "registry", "reg", "task", "service", "network", "ip", "domain", "url", "parent", "child", "uuid", "time"}
	seen := make(map[string]struct{})
	var out []string
	walkJSON(root, basePath, nil, func(path string, value any, container any) bool {
		lower := strings.ToLower(path)
		for _, key := range signalKeys {
			if strings.Contains(lower, key) {
				if _, ok := seen[path]; !ok {
					seen[path] = struct{}{}
					out = append(out, path)
				}
				break
			}
		}
		return len(out) < limit
	})
	return out
}

func collectSampleFields(root any, limit int) []string {
	seen := make(map[string]struct{})
	var out []string
	walkJSON(root, "", nil, func(path string, value any, container any) bool {
		if path == "" || strings.Contains(path, "[") {
			return true
		}
		key := lastPathToken(path)
		if key == "" {
			return true
		}
		if _, ok := seen[key]; ok {
			return true
		}
		seen[key] = struct{}{}
		out = append(out, key)
		return len(out) < limit
	})
	sort.Strings(out)
	return out
}

func sampleObjectFields(items []any, limit int) []string {
	seen := make(map[string]struct{})
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		for key := range object {
			seen[key] = struct{}{}
		}
		break
	}
	fields := make([]string, 0, len(seen))
	for key := range seen {
		fields = append(fields, key)
	}
	sort.Strings(fields)
	if len(fields) > limit {
		fields = fields[:limit]
	}
	return fields
}

func joinPath(parent string, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

func blankToRoot(path string) string {
	if strings.TrimSpace(path) == "" {
		return "root"
	}
	return path
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
