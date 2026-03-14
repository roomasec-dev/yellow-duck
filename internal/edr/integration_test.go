package edr

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"rm_ai_agent/internal/config"
)

func TestIntegrationEDRReadOnlyAPIs(t *testing.T) {
	if testing.Short() {
		t.Skip("skip integration test in short mode")
	}

	cfg := mustLoadLocalConfig(t)
	client := NewClient(cfg.EDR)
	ctx := context.Background()

	t.Run("get_open_api_token", func(t *testing.T) {
		token, err := client.platformTokenValue(ctx)
		if err != nil {
			t.Fatalf("get token failed: %v", err)
		}
		if token == "" {
			t.Fatal("empty token")
		}
	})

	t.Run("hosts_globalization_list", func(t *testing.T) {
		result, err := client.ListHosts(ctx, ListHostsRequest{Page: 1, Limit: 1})
		if err != nil {
			t.Fatalf("list hosts failed: %v", err)
		}
		if result.Pages <= 0 {
			t.Fatalf("unexpected pages: %+v", result)
		}
	})

	t.Run("incidents_list", func(t *testing.T) {
		result, err := client.ListIncidents(ctx, ListIncidentsRequest{Page: 1, PageSize: 1})
		if err != nil {
			t.Fatalf("list incidents failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("detections_list", func(t *testing.T) {
		result, err := client.ListDetections(ctx, ListDetectionsRequest{Page: 1, PageSize: 1})
		if err != nil {
			t.Fatalf("list detections failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("logs_list", func(t *testing.T) {
		result, err := client.ListLogs(ctx, ListLogsRequest{Page: 1, PageSize: 1})
		if err != nil {
			t.Fatalf("list logs failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("ioc_list", func(t *testing.T) {
		payload := map[string]any{"page": 1, "limit": 1}
		var envelope apiEnvelope[struct {
			Total   int               `json:"total"`
			Results []json.RawMessage `json:"results"`
		}]
		if err := client.postPlatform(ctx, "/configure/ioc/list", payload, &envelope); err != nil {
			t.Fatalf("ioc list failed: %v", err)
		}
		if envelope.Error != 0 {
			t.Fatalf("ioc list business error: %d %s", envelope.Error, envelope.Message)
		}
	})

	t.Run("isolate_file_get_list", func(t *testing.T) {
		payload := map[string]any{"page": 1, "limit": 1}
		var envelope apiEnvelope[struct {
			Total   int               `json:"total"`
			Results []json.RawMessage `json:"results"`
		}]
		if err := client.postPlatform(ctx, "/isolate_file/get_list", payload, &envelope); err != nil {
			t.Fatalf("isolate file list failed: %v", err)
		}
		if envelope.Error != 0 {
			t.Fatalf("isolate file list business error: %d %s", envelope.Error, envelope.Message)
		}
	})

	t.Run("instructions_tasks", func(t *testing.T) {
		payload := map[string]any{"page": 1, "limit": 1}
		var envelope apiEnvelope[struct {
			Total   int               `json:"total"`
			Results []json.RawMessage `json:"results"`
		}]
		if err := client.postPlatform(ctx, "/instructions/tasks", payload, &envelope); err != nil {
			t.Fatalf("instructions tasks failed: %v", err)
		}
		if envelope.Error != 0 {
			t.Fatalf("instructions tasks business error: %d %s", envelope.Error, envelope.Message)
		}
	})

	t.Run("incident_view", func(t *testing.T) {
		result, err := client.ViewIncident(ctx, IncidentViewRequest{
			IncidentID: "2a1443b624944ca6a628ec8d0c42c2d9-20260312220845",
			ClientID:   "2a1443b624944ca6a628ec8d0c42c2d9",
		})
		if err != nil {
			t.Fatalf("incident view failed: %v", err)
		}
		if len(result) == 0 {
			t.Fatal("incident view returned empty result")
		}
	})

	t.Run("detection_view", func(t *testing.T) {
		result, err := client.ViewDetection(ctx, DetectionViewRequest{
			DetectionID: "2a1443b624944ca6a628ec8d0c42c2d9-{f79084fb-1e1b-11f1-8b8f-000c2973d451}-20260312220845",
			ClientID:    "2a1443b624944ca6a628ec8d0c42c2d9",
			ViewType:    "process",
			ProcessUUID: "{f79085c3-1e1b-11f1-8b8f-000c2973d451}",
		})
		if err != nil {
			t.Fatalf("detection view failed: %v", err)
		}
		if len(result) == 0 {
			t.Fatal("detection view returned empty result")
		}
	})
}

func mustLoadLocalConfig(t *testing.T) config.Config {
	t.Helper()
	_, currentFile, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	cfg, err := config.Load(filepath.Join(root, "configs", "config.local.toml"))
	if err != nil {
		t.Fatalf("load local config: %v", err)
	}
	return cfg
}
