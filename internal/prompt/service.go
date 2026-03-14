package prompt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ReplyStyle struct {
	DefaultLocale   string   `json:"default_locale"`
	AllowMarkdown   bool     `json:"allow_markdown"`
	AllowSystemTags bool     `json:"allow_system_tags"`
	EmojiPolicy     string   `json:"emoji_policy"`
	Chinese         []string `json:"chinese"`
	English         []string `json:"english"`
}

type Service struct {
	root string
}

func NewService(root string) *Service {
	return &Service{root: root}
}

func (s *Service) LoadAgentsPrompt() string {
	path := filepath.Join(s.root, "prompts", "AGENTS.md")
	body, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func (s *Service) LoadSkillsPrompt() string {
	paths, err := filepath.Glob(filepath.Join(s.root, "skills", "*.md"))
	if err != nil || len(paths) == 0 {
		return ""
	}
	sort.Strings(paths)
	parts := make([]string, 0, len(paths))
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("[%s]\n%s", filepath.Base(path), strings.TrimSpace(string(body))))
	}
	return strings.Join(parts, "\n\n")
}

func (s *Service) LoadReplyStylePrompt() string {
	style, err := s.LoadReplyStyle()
	if err != nil {
		return ""
	}
	parts := []string{
		fmt.Sprintf("default_locale=%s", style.DefaultLocale),
		fmt.Sprintf("allow_markdown=%t", style.AllowMarkdown),
		fmt.Sprintf("allow_system_tags=%t", style.AllowSystemTags),
		fmt.Sprintf("emoji_policy=%s", style.EmojiPolicy),
	}
	if len(style.Chinese) > 0 {
		parts = append(parts, "中文规则:\n- "+strings.Join(style.Chinese, "\n- "))
	}
	if len(style.English) > 0 {
		parts = append(parts, "English rules:\n- "+strings.Join(style.English, "\n- "))
	}
	return strings.Join(parts, "\n")
}

func (s *Service) LoadReplyStyle() (ReplyStyle, error) {
	body, err := os.ReadFile(filepath.Join(s.root, "configs", "reply_style.json"))
	if err != nil {
		return ReplyStyle{}, err
	}
	var style ReplyStyle
	if err := json.Unmarshal(body, &style); err != nil {
		return ReplyStyle{}, err
	}
	return style, nil
}

func (s *Service) LoadPrompt(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	body, err := os.ReadFile(filepath.Join(s.root, "prompts", name+".md"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

func (s *Service) ComposeSystemPrompt(parts ...string) string {
	items := make([]string, 0, len(parts)+1)
	if agentsPrompt := s.LoadAgentsPrompt(); agentsPrompt != "" {
		items = append(items, agentsPrompt)
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return strings.Join(items, "\n\n")
}
