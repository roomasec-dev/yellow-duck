package edr

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
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
		t.Logf("detections_list result %+v", result)
		if err != nil {
			t.Fatalf("list detections failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("logs_list", func(t *testing.T) {
		result, err := client.ListLogs(ctx, ListLogsRequest{Page: 1, PageSize: 1})
		t.Logf("logs_list result: %+v", result)
		if err != nil {
			t.Fatalf("list logs failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("ioc_list", func(t *testing.T) {
		result, err := client.ListIOCs(ctx, ListIOCsRequest{Page: 1, Limit: 3})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("ioc_list raw json:\n%s", string(raw))
		// t.Logf("ioc_list result: %+v", result)
		if err != nil {
			t.Fatalf("ioc list failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("ioc_delete", func(t *testing.T) {
		// 先添加一条 IOC，再删除它
		testHash := strings.ToUpper("382919d25113457f96e6428548e492033253aad2")
		addReq := AddIOCRequest{
			Action:      "Allow",
			Hash:        testHash,
			Description: "integration test ioc",
			FileName:    "test.exe",
			HostType:    "ALL",
		}
		if err := client.AddIOC(ctx, addReq); err != nil {
			t.Logf("add ioc failed: %v", err)
		}
		t.Logf("ioc_add done: hash=%s", testHash)

		// 查找该 IOC 的 id
		// listResult, err := client.ListIOCs(ctx, ListIOCsRequest{Hash: testHash, Limit: 100})
		// if err != nil {
		// 	t.Fatalf("list iocs after add failed: %v", err)
		// }
		// var iocID string
		// for _, i := range listResult.Results {
		// 	if i.Hash == testHash {
		// 		iocID = i.ExclusionID.(string)
		// 		break
		// 	}
		// }
		// if iocID == "" {
		// 	t.Fatalf("ioc not found after add: hash=%s", testHash)
		// }
		// t.Logf("found ioc id: %s", iocID)

		var iocID = "1ac26fb991ee46448031e17f21d99304"

		if err := client.DeleteIOC(ctx, iocID); err != nil {
			t.Fatalf("delete ioc failed: %v", err)
		}
		t.Logf("ioc_delete done: id=%s", iocID)
	})

	t.Run("ioc_update", func(t *testing.T) {
		// 先添加一条 IOC，再更新它，最后删除它
		testHash := strings.ToUpper("382919d25113457f96e6428548e492033253aad2")
		addReq := AddIOCRequest{
			Action:      "Allow",
			Hash:        testHash,
			Description: "integration test ioc",
			FileName:    "test.exe",
			HostType:    "ALL",
		}
		if err := client.AddIOC(ctx, addReq); err != nil {
			t.Logf("add ioc failed: %v", err)
		}
		t.Logf("ioc_add done: hash=%s", testHash)

		// // 查找该 IOC 的 id
		// listResult, err := client.ListIOCs(ctx, ListIOCsRequest{Hash: testHash, Limit: 100})
		// if err != nil {
		// 	t.Fatalf("list iocs after add failed: %v", err)
		// }
		// var iocID string
		// for _, i := range listResult.Results {
		// 	if i.Hash == testHash {
		// 		iocID = i.ExclusionID.(string)
		// 		break
		// 	}
		// }
		// if iocID == "" {
		// 	t.Fatalf("ioc not found after add: hash=%s", testHash)
		// }
		// t.Logf("found ioc id: %s", iocID)

		var iocID = "8616adda9b9047b8abd8466ff533c02c"

		// 更新 IOC
		updateReq := UpdateIOCRequest{
			ID:          iocID,
			Description: "updated integration test ioc lalala",
		}
		if err := client.UpdateIOC(ctx, updateReq); err != nil {
			t.Fatalf("update ioc failed: %v", err)
		}
		t.Logf("ioc_update done: id=%s", iocID)
	})

	t.Run("isolate_file_get_list", func(t *testing.T) {
		result, err := client.ListIsolateFiles(ctx, ListIsolateFilesRequest{Page: 1, Limit: 1})
		if err != nil {
			t.Fatalf("isolate file list failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("isolate_file_release", func(t *testing.T) {
		// 先获取一条隔离文件的 GUID
		result, err := client.ListIsolateFiles(ctx, ListIsolateFilesRequest{Page: 1, Limit: 1})
		t.Logf("isolate_file_release result %+v", result)
		if err != nil {
			t.Fatalf("list isolate files failed: %v", err)
		}
		if len(result.Results) == 0 {
			t.Skip("no isolate files to release")
		}
		guid := result.Results[0].GUID
		t.Logf("releasing isolate file: guid=%s", guid)
		if err := client.ReleaseIsolateFiles(ctx, ReleaseIsolateFilesRequest{GUIDs: []string{guid}}); err != nil {
			t.Fatalf("release isolate file failed: %v", err)
		}
		t.Logf("isolate_file_release done: guid=%s", guid)
	})

	t.Run("isolate_file_delete", func(t *testing.T) {
		// 先获取一条隔离文件的 GUID
		result, err := client.ListIsolateFiles(ctx, ListIsolateFilesRequest{Page: 1, Limit: 1})
		t.Logf("isolate_file_delete result %+v", result)
		if err != nil {
			t.Fatalf("list isolate files failed: %v", err)
		}
		if len(result.Results) == 0 {
			t.Skip("no isolate files to release")
		}
		guid := result.Results[0].GUID
		t.Logf("deleting isolate file: guid=%s", guid)
		if err := client.DeleteIsolateFiles(ctx, []string{guid}); err != nil {
			t.Fatalf("delete isolate file failed: %v", err)
		}
		t.Logf("isolate_file_delete done: guid=%s", guid)
	})

	t.Run("instructions_tasks", func(t *testing.T) {
		result, err := client.ListTasks(ctx, ListTasksRequest{Page: 1, Limit: 1})
		if err != nil {
			t.Fatalf("instructions tasks failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("incident_view", func(t *testing.T) {
		result, err := client.ViewIncident(ctx, IncidentViewRequest{
			IncidentID: "89be88c5911e42acbcedcaf6f64ac0b6-20260323161549",
			ClientID:   "89be88c5911e42acbcedcaf6f64ac0b6",
		})
		// t.Logf("incident_view result %+v", result)
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
		t.Logf("detection_view result %+v", result)
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
