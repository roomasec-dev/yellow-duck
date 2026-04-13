package config

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server        ServerConfig        `toml:"server"`
	Storage       StorageConfig       `toml:"storage"`
	Channel       ChannelConfig       `toml:"channel"`
	Models        ModelsConfig        `toml:"models"`
	Session       SessionConfig       `toml:"session"`
	Routing       RoutingConfig       `toml:"routing"`
	Memory        MemoryConfig        `toml:"memory"`
	Compression   CompressionConfig   `toml:"compression"`
	Progress      ProgressConfig      `toml:"progress"`
	DetailAgent   DetailAgentConfig   `toml:"detail_agent"`
	Scheduler     SchedulerConfig     `toml:"scheduler"`
	KnowledgeBase KnowledgeBaseConfig `toml:"knowledge_base"`
	EDR           EDRConfig           `toml:"edr"`
	Policy        PolicyConfig        `toml:"policy"`
}

type ServerConfig struct {
	Address  string   `toml:"address"`
	LogLevel LogLevel `toml:"log_level"`
	LogFile  string   `toml:"log_file"`
}

type StorageConfig struct {
	Path string `toml:"path"`
}

type ChannelConfig struct {
	Feishu  FeishuConfig  `toml:"feishu"`
	Dingtalk DingtalkConfig `toml:"dingtalk"`
}

type FeishuConfig struct {
	Enabled           bool   `toml:"enabled"`
	Mode              string `toml:"mode"`
	AppID             string `toml:"app_id"`
	AppSecret         string `toml:"app_secret"`
	VerificationToken string `toml:"verification_token"`
	EncryptKey        string `toml:"encrypt_key"`
	BaseURL           string `toml:"base_url"`
	ReplyMode         string `toml:"reply_mode"`
	WebhookPath       string `toml:"webhook_path"`
}

type DingtalkConfig struct {
	Enabled         bool   `toml:"enabled"`
	Mode            string `toml:"mode"`
	ClientID        string `toml:"client_id"`
	ClientSecret    string `toml:"client_secret"`
	BaseURL         string `toml:"base_url"`
	WebhookPath     string `toml:"webhook_path"`
	ReplyMode       string `toml:"reply_mode"`
	EncryptKey      string `toml:"encrypt_key"`
	VerificationToken string `toml:"verification_token"`
}

type ModelsConfig struct {
	DefaultProvider string                    `toml:"default_provider"`
	DefaultModel    string                    `toml:"default_model"`
	ModelSettings   map[string]ModelSettings  `toml:"model_settings"`
	Providers       map[string]ProviderConfig `toml:"providers"`
}

type ModelSettings struct {
	Temperature float64 `toml:"temperature"`
	MaxTokens   int     `toml:"max_tokens"`
}

type ProviderConfig struct {
	Type        string  `toml:"type"`
	BaseURL     string  `toml:"base_url"`
	APIKey      string  `toml:"api_key"`
	APIKeyEnv   string  `toml:"api_key_env"`
	Model       string  `toml:"model"`
	Temperature float64 `toml:"temperature"`
	MaxTokens   int     `toml:"max_tokens"`
}

type SessionConfig struct {
	RequireMentionInGroup bool `toml:"require_mention_in_group"`
	UseThreadInGroup      bool `toml:"use_thread_in_group"`
	MaxRecentTurns        int  `toml:"max_recent_turns"`
}

type RoutingConfig struct {
	Enabled           bool    `toml:"enabled"`
	Model             string  `toml:"model"`
	MinConfidence     float64 `toml:"min_confidence"`
	AllowWriteActions bool    `toml:"allow_write_actions"`
}

type MemoryConfig struct {
	Enabled        bool `toml:"enabled"`
	MaxEntries     int  `toml:"max_entries"`
	ContextEntries int  `toml:"context_entries"`
}

type CompressionConfig struct {
	Enabled             bool    `toml:"enabled"`
	MaxTurnsBeforeRun   int     `toml:"max_turns_before_run"`
	SummaryMaxBytes     int     `toml:"summary_max_bytes"`
	Model               string  `toml:"model"`
	ContextWindowTokens int     `toml:"context_window_tokens"`
	TriggerRatio        float64 `toml:"trigger_ratio"`
	RecentTurnsToKeep   int     `toml:"recent_turns_to_keep"`
}

type ProgressConfig struct {
	Enabled        bool   `toml:"enabled"`
	Model          string `toml:"model"`
	MaxUpdates     int    `toml:"max_updates"`
	MaxToolUpdates int    `toml:"max_tool_updates"`
	SystemPrompt   string `toml:"system_prompt"`
}

type DetailAgentConfig struct {
	Enabled        bool   `toml:"enabled"`
	Model          string `toml:"model"`
	DirectMaxBytes int    `toml:"direct_max_bytes"`
	MaxInputBytes  int    `toml:"max_input_bytes"`
}

type SchedulerConfig struct {
	Enabled          bool   `toml:"enabled"`
	PollSeconds      int    `toml:"poll_seconds"`
	Model            string `toml:"model"`
	DefaultIntervalM int    `toml:"default_interval_minutes"`
	ScopeKey         string `toml:"scope_key"`
}

type KnowledgeBaseConfig struct {
	Enabled       bool   `toml:"enabled"`
	Path          string `toml:"path"`
	SearchLimit   int    `toml:"search_limit"`
	SnippetLength int    `toml:"snippet_length"`
}

type EDRConfig struct {
	BaseURL          string            `toml:"base_url"`
	TimeoutSeconds   int               `toml:"timeout_seconds"`
	AuthToken        string            `toml:"auth_token"`
	Headers          map[string]string `toml:"headers"`
	DefaultConnectIP string            `toml:"default_connect_ip"`
	AllowActions     []string          `toml:"allow_actions"`
	DefaultPageSize  int               `toml:"default_page_size"`
	Platform         PlatformAPIConfig `toml:"platform"`
}

type PlatformAPIConfig struct {
	Enabled      bool   `toml:"enabled"`
	BaseURL      string `toml:"base_url"`
	AppKey       string `toml:"app_key"`
	AppSecret    string `toml:"app_secret"`
	AppKeyEnv    string `toml:"app_key_env"`
	AppSecretEnv string `toml:"app_secret_env"`
}

type PolicyConfig struct {
	DangerousActionKeywords []string `toml:"dangerous_action_keywords"`
	ApprovedUsers           []string `toml:"approved_users"`
}

type LogLevel string

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l LogLevel) Level() Level {
	switch strings.ToLower(string(l)) {
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func Load(path string) (Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode %s: %w", path, err)
	}

	applyDefaults(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Address == "" {
		cfg.Server.Address = ":8080"
	}
	if cfg.Server.LogFile == "" {
		cfg.Server.LogFile = "data/rm-ai-agent.log"
	}
	if cfg.Storage.Path == "" {
		cfg.Storage.Path = "data/rm-ai-agent.sqlite"
	}
	if cfg.Channel.Feishu.BaseURL == "" {
		cfg.Channel.Feishu.BaseURL = "https://open.feishu.cn"
	}
	if cfg.Channel.Feishu.Mode == "" {
		cfg.Channel.Feishu.Mode = "longconn"
	}
	if cfg.Channel.Feishu.ReplyMode == "" {
		cfg.Channel.Feishu.ReplyMode = "reply"
	}
	if cfg.Channel.Feishu.WebhookPath == "" {
		cfg.Channel.Feishu.WebhookPath = "/webhook/feishu/event"
	}
	if cfg.Channel.Dingtalk.BaseURL == "" {
		cfg.Channel.Dingtalk.BaseURL = "https://api.dingtalk.com"
	}
	if cfg.Channel.Dingtalk.Mode == "" {
		cfg.Channel.Dingtalk.Mode = "webhook"
	}
	if cfg.Channel.Dingtalk.ReplyMode == "" {
		cfg.Channel.Dingtalk.ReplyMode = "reply"
	}
	if cfg.Channel.Dingtalk.WebhookPath == "" {
		cfg.Channel.Dingtalk.WebhookPath = "/webhook/dingtalk/event"
	}
	for name, provider := range cfg.Models.Providers {
		if provider.Type == "" {
			provider.Type = "openai_compatible"
		}
		if provider.Model == "" && name == cfg.Models.DefaultProvider {
			provider.Model = cfg.Models.DefaultModel
		}
		cfg.Models.Providers[name] = provider
	}
	if cfg.Session.MaxRecentTurns <= 0 {
		cfg.Session.MaxRecentTurns = 12
	}
	if cfg.Routing.MinConfidence <= 0 {
		cfg.Routing.MinConfidence = 0.6
	}
	if cfg.Memory.MaxEntries <= 0 {
		cfg.Memory.MaxEntries = 32
	}
	if cfg.Memory.ContextEntries <= 0 {
		cfg.Memory.ContextEntries = 12
	}
	if cfg.Compression.MaxTurnsBeforeRun <= 0 {
		cfg.Compression.MaxTurnsBeforeRun = 30
	}
	if cfg.Compression.SummaryMaxBytes <= 0 {
		cfg.Compression.SummaryMaxBytes = 4000
	}
	if cfg.Compression.ContextWindowTokens <= 0 {
		cfg.Compression.ContextWindowTokens = 128000
	}
	if cfg.Compression.TriggerRatio <= 0 {
		cfg.Compression.TriggerRatio = 0.8
	}
	if cfg.Compression.RecentTurnsToKeep <= 0 {
		cfg.Compression.RecentTurnsToKeep = 24
	}
	if cfg.Progress.MaxUpdates <= 0 {
		cfg.Progress.MaxUpdates = 6
	}
	if cfg.Progress.MaxToolUpdates <= 0 {
		cfg.Progress.MaxToolUpdates = 4
	}
	if cfg.Progress.SystemPrompt == "" {
		cfg.Progress.SystemPrompt = "你是 AI 助手的进度播报员。请把内部操作步骤改写成发给终端用户的一句中文进度说明。要求：第一人称、18 到 40 字、友好自然、不要输出编号、不要泄露路径/密钥/API 细节、不要夸大结果、不要使用 markdown。只输出一句话。"
	}
	if cfg.DetailAgent.Model == "" {
		switch {
		case cfg.Progress.Model != "":
			cfg.DetailAgent.Model = cfg.Progress.Model
		case cfg.Routing.Model != "":
			cfg.DetailAgent.Model = cfg.Routing.Model
		default:
			cfg.DetailAgent.Model = "deepseek/deepseek-chat"
		}
	}
	if cfg.DetailAgent.DirectMaxBytes <= 0 {
		cfg.DetailAgent.DirectMaxBytes = 12 * 1024
	}
	if cfg.DetailAgent.MaxInputBytes <= 0 {
		cfg.DetailAgent.MaxInputBytes = 64 * 1024
	}
	if cfg.Scheduler.PollSeconds <= 0 {
		cfg.Scheduler.PollSeconds = 30
	}
	if cfg.Scheduler.DefaultIntervalM <= 0 {
		cfg.Scheduler.DefaultIntervalM = 5
	}
	if cfg.Scheduler.Model == "" {
		if cfg.Routing.Model != "" {
			cfg.Scheduler.Model = cfg.Routing.Model
		} else {
			cfg.Scheduler.Model = cfg.Models.DefaultProvider + "/" + cfg.Models.DefaultModel
		}
	}
	if cfg.KnowledgeBase.Path == "" {
		cfg.KnowledgeBase.Path = "knowledge_base"
	}
	if cfg.KnowledgeBase.SearchLimit <= 0 {
		cfg.KnowledgeBase.SearchLimit = 5
	}
	if cfg.KnowledgeBase.SnippetLength <= 0 {
		cfg.KnowledgeBase.SnippetLength = 240
	}
	if cfg.EDR.TimeoutSeconds <= 0 {
		cfg.EDR.TimeoutSeconds = 15
	}
	if cfg.EDR.Platform.BaseURL == "" {
		cfg.EDR.Platform.BaseURL = "https://qax-openapi.zboundary.com/sase/open_api/rm/v1"
	}
	if cfg.EDR.DefaultPageSize <= 0 {
		cfg.EDR.DefaultPageSize = 10
	}
}

func (c Config) Validate() error {
	if c.Server.Address == "" {
		return fmt.Errorf("server.address is required")
	}
	if c.Storage.Path == "" {
		return fmt.Errorf("storage.path is required")
	}
	if c.Models.DefaultProvider == "" {
		return fmt.Errorf("models.default_provider is required")
	}
	if _, ok := c.Models.Providers[c.Models.DefaultProvider]; !ok {
		return fmt.Errorf("models.default_provider %q not found in models.providers", c.Models.DefaultProvider)
	}
	if c.Channel.Feishu.Enabled {
		if c.Channel.Feishu.AppID == "" || c.Channel.Feishu.AppSecret == "" {
			return fmt.Errorf("channel.feishu app_id and app_secret are required when enabled")
		}
		switch strings.ToLower(c.Channel.Feishu.Mode) {
		case "webhook", "longconn", "both":
		default:
			return fmt.Errorf("channel.feishu.mode must be one of webhook, longconn, both")
		}
	}
	if c.Channel.Dingtalk.Enabled {
		if c.Channel.Dingtalk.ClientID == "" || c.Channel.Dingtalk.ClientSecret == "" {
			return fmt.Errorf("channel.dingtalk client_id and client_secret are required when enabled")
		}
		switch strings.ToLower(c.Channel.Dingtalk.Mode) {
		case "webhook", "longconn", "both":
		default:
			return fmt.Errorf("channel.dingtalk.mode must be one of webhook, longconn, both")
		}
	}
	return nil
}
