package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Catalog struct {
	DefaultLocale string                       `json:"default_locale"`
	Messages      map[string]map[string]string `json:"messages"`
}

type Service struct {
	defaultLocale string
	messages      map[string]map[string]string
}

func New(root string) *Service {
	path := filepath.Join(root, "configs", "messages.json")
	body, err := os.ReadFile(path)
	if err != nil {
		return &Service{defaultLocale: "zh-CN", messages: map[string]map[string]string{}}
	}
	var cat Catalog
	if err := json.Unmarshal(body, &cat); err != nil {
		return &Service{defaultLocale: "zh-CN", messages: map[string]map[string]string{}}
	}
	if cat.DefaultLocale == "" {
		cat.DefaultLocale = "zh-CN"
	}
	if cat.Messages == nil {
		cat.Messages = map[string]map[string]string{}
	}
	return &Service{defaultLocale: cat.DefaultLocale, messages: cat.Messages}
}

func (s *Service) DetectLocale(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return s.defaultLocale
	}
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return "zh-CN"
		}
	}
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return "en-US"
		}
	}
	return s.defaultLocale
}

func (s *Service) T(locale string, key string, vars map[string]string) string {
	msg := s.lookup(locale, key)
	if msg == "" {
		msg = s.lookup(s.defaultLocale, key)
	}
	for k, v := range vars {
		msg = strings.ReplaceAll(msg, "{{"+k+"}}", v)
	}
	return msg
}

func (s *Service) lookup(locale string, key string) string {
	if locale == "" {
		locale = s.defaultLocale
	}
	if bucket, ok := s.messages[locale]; ok {
		return bucket[key]
	}
	return ""
}
