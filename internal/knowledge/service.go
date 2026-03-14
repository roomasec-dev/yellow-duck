package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"rm_ai_agent/internal/config"
)

type Match struct {
	Title   string
	Path    string
	RelPath string
	Snippet string
	Score   int
}

type fileEntry struct {
	Title   string
	Path    string
	RelPath string
	Content string
}

type Service struct {
	cfg config.KnowledgeBaseConfig
}

func NewService(cfg config.KnowledgeBaseConfig) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Enabled() bool {
	return s != nil && s.cfg.Enabled
}

func (s *Service) Search(query string) ([]Match, error) {
	if !s.Enabled() {
		return nil, nil
	}
	entries, err := s.markdownEntries()
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	tokens := searchTokens(query)
	matches := make([]Match, 0)
	for _, entry := range entries {
		score, snippet := scoreContent(entry.Title, entry.RelPath, entry.Content, query, tokens, s.cfg.SnippetLength)
		if score <= 0 && query != "" {
			continue
		}
		if query == "" {
			snippet = trimSnippet(entry.Content, s.cfg.SnippetLength)
		}
		matches = append(matches, Match{Title: entry.Title, Path: entry.Path, RelPath: entry.RelPath, Snippet: snippet, Score: score})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].RelPath < matches[j].RelPath
		}
		return matches[i].Score > matches[j].Score
	})
	if len(matches) > s.cfg.SearchLimit {
		matches = matches[:s.cfg.SearchLimit]
	}
	return matches, nil
}

func (s *Service) Upsert(title string, content string, mode string, oldText string, newText string) (Match, error) {
	if !s.Enabled() {
		return Match{}, nil
	}
	root, err := s.rootPath()
	if err != nil {
		return Match{}, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Match{}, fmt.Errorf("mkdir knowledge base: %w", err)
	}
	path, found, err := s.resolveFile(title)
	if err != nil {
		return Match{}, err
	}
	if !found {
		path, err = s.newFilePath(title)
		if err != nil {
			return Match{}, err
		}
	}
	existing, _ := os.ReadFile(path)
	current := string(existing)
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = "upsert"
	}
	var next string
	switch mode {
	case "append":
		if strings.TrimSpace(content) == "" {
			return Match{}, fmt.Errorf("append mode requires content")
		}
		if strings.TrimSpace(current) == "" {
			next = strings.TrimSpace(content)
		} else {
			next = strings.TrimRight(current, "\r\n") + "\n\n" + strings.TrimSpace(content)
		}
	case "replace_text":
		if strings.TrimSpace(oldText) == "" {
			return Match{}, fmt.Errorf("replace_text mode requires old_text")
		}
		if !found {
			return Match{}, fmt.Errorf("knowledge base file not found")
		}
		if !strings.Contains(current, oldText) {
			return Match{}, fmt.Errorf("target text not found in knowledge base file")
		}
		next = strings.ReplaceAll(current, oldText, newText)
	case "upsert":
		fallthrough
	default:
		if strings.TrimSpace(content) == "" {
			return Match{}, fmt.Errorf("knowledge base content is empty")
		}
		next = strings.TrimSpace(content)
	}
	if strings.TrimSpace(next) == "" {
		return Match{}, fmt.Errorf("knowledge base content is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Match{}, fmt.Errorf("mkdir knowledge base subdir: %w", err)
	}
	if err := os.WriteFile(path, []byte(next+"\n"), 0o644); err != nil {
		return Match{}, fmt.Errorf("write knowledge base file: %w", err)
	}
	relPath, _ := filepath.Rel(root, path)
	return Match{Title: titleFromContent(path, next), Path: path, RelPath: filepath.ToSlash(relPath), Snippet: trimSnippet(next, s.cfg.SnippetLength)}, nil
}

func (s *Service) Delete(title string) error {
	if !s.Enabled() {
		return nil
	}
	path, found, err := s.resolveFile(title)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("knowledge base file not found")
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("delete knowledge base file: %w", err)
	}
	return nil
}

func (s *Service) markdownEntries() ([]fileEntry, error) {
	root, err := s.rootPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir knowledge base: %w", err)
	}
	entries := make([]fileEntry, 0)
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".md") {
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			relPath, relErr := filepath.Rel(root, path)
			if relErr != nil {
				relPath = filepath.Base(path)
			}
			content := string(body)
			entries = append(entries, fileEntry{
				Title:   titleFromContent(path, content),
				Path:    path,
				RelPath: filepath.ToSlash(relPath),
				Content: content,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk knowledge base: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RelPath < entries[j].RelPath
	})
	return entries, nil
}

func (s *Service) rootPath() (string, error) {
	root := strings.TrimSpace(s.cfg.Path)
	if root == "" {
		root = "knowledge_base"
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve knowledge base path: %w", err)
	}
	return filepath.Clean(abs), nil
}

func (s *Service) resolveFile(ref string) (string, bool, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", false, nil
	}
	entries, err := s.markdownEntries()
	if err != nil {
		return "", false, err
	}
	bestPath := ""
	bestRelPath := ""
	bestScore := 0
	lookup := normalizeLookup(ref)
	lookupStem := trimMDExt(lookup)
	tokens := searchTokens(ref)
	for _, entry := range entries {
		score := matchFileScore(entry, lookup, lookupStem, ref, tokens, s.cfg.SnippetLength)
		if score > bestScore || (score == bestScore && score > 0 && (bestRelPath == "" || entry.RelPath < bestRelPath)) {
			bestScore = score
			bestPath = entry.Path
			bestRelPath = entry.RelPath
		}
	}
	if bestScore == 0 {
		return "", false, nil
	}
	return bestPath, true, nil
}

func (s *Service) newFilePath(ref string) (string, error) {
	root, err := s.rootPath()
	if err != nil {
		return "", err
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = "knowledge"
	}
	if strings.ContainsAny(ref, `/\\`) || strings.HasSuffix(strings.ToLower(ref), ".md") {
		candidate := ref
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(root, candidate)
		}
		candidate = filepath.Clean(candidate)
		ext := strings.ToLower(filepath.Ext(candidate))
		switch ext {
		case "":
			candidate += ".md"
		case ".md":
		default:
			return "", fmt.Errorf("knowledge base file must be markdown")
		}
		if !withinRoot(root, candidate) {
			return "", fmt.Errorf("knowledge base file must stay under configured path")
		}
		return candidate, nil
	}
	return filepath.Join(root, slugify(ref)+".md"), nil
}

func scoreContent(title string, relPath string, content string, query string, tokens []string, snippetLength int) (int, string) {
	lowerTitle := strings.ToLower(title)
	lowerPath := strings.ToLower(relPath)
	lowerContent := strings.ToLower(content)
	if query == "" {
		return 1, trimSnippet(content, snippetLength)
	}
	score := 0
	for _, token := range tokens {
		if strings.Contains(lowerTitle, token) {
			score += 5
		}
		if strings.Contains(lowerPath, token) {
			score += 4
		}
		if strings.Contains(lowerContent, token) {
			score += 2
		}
	}
	if strings.Contains(lowerTitle, strings.ToLower(query)) {
		score += 8
	}
	if strings.Contains(lowerPath, strings.ToLower(query)) {
		score += 6
	}
	if strings.Contains(lowerContent, strings.ToLower(query)) {
		score += 4
	}
	return score, bestSnippet(content, tokens, snippetLength)
}

func matchFileScore(entry fileEntry, lookup string, lookupStem string, rawQuery string, tokens []string, snippetLength int) int {
	pathNorm := normalizeLookup(entry.Path)
	relNorm := normalizeLookup(entry.RelPath)
	relStem := trimMDExt(relNorm)
	baseNorm := normalizeLookup(strings.TrimSuffix(filepath.Base(entry.RelPath), filepath.Ext(entry.RelPath)))
	titleNorm := normalizeLookup(entry.Title)
	score := 0
	switch {
	case lookup != "" && pathNorm == lookup:
		score += 200
	case lookup != "" && relNorm == lookup:
		score += 180
	case lookupStem != "" && relStem == lookupStem:
		score += 170
	case lookupStem != "" && baseNorm == lookupStem:
		score += 160
	case lookup != "" && titleNorm == lookup:
		score += 160
	}
	contentScore, _ := scoreContent(entry.Title, entry.RelPath, entry.Content, rawQuery, tokens, snippetLength)
	score += contentScore
	return score
}

func bestSnippet(content string, tokens []string, snippetLength int) string {
	if len(tokens) == 0 {
		return trimSnippet(content, snippetLength)
	}
	lower := strings.ToLower(content)
	idx := -1
	for _, token := range tokens {
		pos := strings.Index(lower, token)
		if pos >= 0 && (idx < 0 || pos < idx) {
			idx = pos
		}
	}
	if idx < 0 {
		return trimSnippet(content, snippetLength)
	}
	runes := []rune(content)
	runeIdx := utf8.RuneCountInString(content[:idx])
	start := runeIdx - snippetLength/3
	if start < 0 {
		start = 0
	}
	end := start + snippetLength
	if end > len(runes) {
		end = len(runes)
	}
	return strings.TrimSpace(string(runes[start:end]))
}

func trimSnippet(content string, snippetLength int) string {
	content = strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	runes := []rune(content)
	if len(runes) <= snippetLength {
		return content
	}
	return string(runes[:snippetLength]) + "..."
}

func searchTokens(query string) []string {
	parts := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(",.;:()[]{}<>/\\_-，。；：（）【】、", r)
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func titleFromContent(path string, content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			title := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if title != "" {
				return title
			}
		}
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func normalizeLookup(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	text = strings.ReplaceAll(text, `\\`, "/")
	text = strings.TrimPrefix(text, "./")
	text = strings.Trim(text, "/")
	return text
}

func trimMDExt(text string) string {
	return strings.TrimSuffix(text, ".md")
}

func withinRoot(root string, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func slugify(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return "knowledge"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || strings.ContainsRune("-_/\\", r):
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "knowledge"
	}
	return result
}
