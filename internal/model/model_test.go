package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rm_ai_agent/internal/config"
)

func TestChatOpenAICompatible(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "deepseek-reasoner" {
			t.Fatalf("unexpected model: %s", req.Model)
		}
		if req.MaxTokens != 1234 {
			t.Fatalf("unexpected max_tokens: %d", req.MaxTokens)
		}
		if req.Temperature == nil || *req.Temperature != 0.3 {
			t.Fatalf("unexpected temperature: %#v", req.Temperature)
		}
		if len(req.Messages) != 2 {
			t.Fatalf("unexpected messages count: %d", len(req.Messages))
		}

		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			Choices: []struct {
				Message struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content          string `json:"content"`
						ReasoningContent string `json:"reasoning_content"`
					}{Content: "hello from deepseek"},
				},
			},
		})
	}))
	defer server.Close()

	client := NewFallbackClient(config.ModelsConfig{
		DefaultProvider: "deepseek",
		DefaultModel:    "deepseek-reasoner",
		ModelSettings: map[string]config.ModelSettings{
			"deepseek/deepseek-reasoner": {Temperature: 0.3, MaxTokens: 1234},
		},
		Providers: map[string]config.ProviderConfig{
			"deepseek": {
				Type:    "deepseek",
				BaseURL: server.URL,
				APIKey:  "test-key",
				Model:   "deepseek-reasoner",
			},
		},
	}, nil)
	client.http = server.Client()

	result, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "system"},
			{Role: RoleUser, Content: "hello"},
		},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	if result.Text != "hello from deepseek" {
		t.Fatalf("unexpected text: %q", result.Text)
	}
	if result.Model != "deepseek/deepseek-reasoner" {
		t.Fatalf("unexpected model: %s", result.Model)
	}
}

func TestChatExplicitStub(t *testing.T) {
	t.Parallel()

	client := NewFallbackClient(config.ModelsConfig{
		DefaultProvider: "broken",
		DefaultModel:    "deepseek-reasoner",
		Providers: map[string]config.ProviderConfig{
			"broken": {
				Type:    "deepseek",
				BaseURL: "https://api.deepseek.com",
			},
			"stub": {
				Type:  "stub",
				Model: "bootstrap",
			},
		},
	}, nil)

	result, err := client.Chat(context.Background(), ChatRequest{
		Model:    "stub/bootstrap",
		Messages: []Message{{Role: RoleUser, Content: "fallback please"}},
	}, nil)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	if result.Model != "stub/bootstrap" {
		t.Fatalf("unexpected model: %s", result.Model)
	}
}

func TestChatDoesNotFallbackToStubWhenRealProviderFails(t *testing.T) {
	t.Parallel()

	client := NewFallbackClient(config.ModelsConfig{
		DefaultProvider: "broken",
		DefaultModel:    "deepseek-reasoner",
		Providers: map[string]config.ProviderConfig{
			"broken": {
				Type:    "deepseek",
				BaseURL: "https://api.deepseek.com",
			},
			"stub": {
				Type:  "stub",
				Model: "bootstrap",
			},
		},
	}, nil)

	_, err := client.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: RoleUser, Content: "fallback please"}},
	}, nil)
	if err == nil {
		t.Fatal("expected error when real provider fails")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "missing api key") {
		t.Fatalf("unexpected error: %v", err)
	}
}
