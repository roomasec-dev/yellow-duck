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
		// t.Logf("tasks result %+v", result)
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("instructions_tasks raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("instructions tasks failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("send_instruction", func(t *testing.T) {
		// 先获取一台在线主机的 client_id
		hosts, err := client.ListHosts(ctx, ListHostsRequest{Page: 1, Limit: 10})
		if err != nil {
			t.Fatalf("list hosts failed: %v", err)
		}
		var clientID string
		for _, h := range hosts.Hosts {
			if h.Status == "online" {
				clientID = h.ClientID
				break
			}
		}
		if clientID == "" {
			t.Skip("no online host to send instruction")
		}
		t.Logf("sending instruction to client_id=%s", clientID)

		result, err := client.SendInstruction(ctx, clientID, "quarantine_network", "integration test 隔离网络")
		t.Logf("send_instruction result %+v", result)
		if err != nil {
			t.Fatalf("send instruction failed: %v", err)
		}
		if result.TaskID == "" {
			t.Fatalf("empty task_id in result: %+v", result)
		}
		t.Logf("send_instruction done: task_id=%s host_name=%s", result.TaskID, result.HostName)
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

	// Virus Scan tests
	t.Run("virus_scan_list", func(t *testing.T) {
		result, err := client.ListVirusScans(ctx, ListVirusScansRequest{Page: 1, Limit: 10})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("virus_scan_list raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("virus scan list failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("virus_scan_scan_record", func(t *testing.T) {
		result, err := client.ListVirusScanRecords(ctx, ListVirusScanRecordsRequest{Page: 1, Limit: 10})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("virus_scan_scan_record raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("virus scan records failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	// Virus Statistics tests
	t.Run("virus_host_list", func(t *testing.T) {
		result, err := client.ListVirusByHost(ctx, ListVirusByHostRequest{Page: 1, Limit: 10})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("virus_host_list raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("virus host list failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("virus_hash_list", func(t *testing.T) {
		result, err := client.ListVirusByHash(ctx, ListVirusByHashRequest{Page: 1, Limit: 10})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("virus_hash_list raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("virus hash list failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	t.Run("virus_hash_host_list", func(t *testing.T) {
		result, err := client.ListVirusHashHosts(ctx, ListVirusHashHostsRequest{Page: 1, Limit: 10})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("virus_hash_host_list raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("virus hash host list failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
	})

	// NGAV Settings tests
	t.Run("settings_get_ngav_conf", func(t *testing.T) {
		result, err := client.GetNGAVConf(ctx)
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("settings_get_ngav_conf raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("get ngav conf failed: %v", err)
		}
		if result == nil {
			t.Fatal("ngav conf returned nil")
		}
	})

	// Virus Scan write operations tests
	t.Run("virus_scan_add", func(t *testing.T) {
		// 先获取一台在线主机的 client_id
		hosts, err := client.ListHosts(ctx, ListHostsRequest{Page: 1, Limit: 10})
		if err != nil {
			t.Fatalf("list hosts failed: %v", err)
		}
		var clientID string
		for _, h := range hosts.Hosts {
			if h.Status == "online" {
				clientID = h.ClientID
				break
			}
		}
		if clientID == "" {
			t.Skip("no online host to create virus scan")
		}

		// 创建快速扫描计划
		err = client.AddVirusScan(ctx, AddVirusScanRequest{
			ScanType: 1, // 1 快速扫描
			PlanName: "integration test scan",
			PlanType: 1, // 1 立即执行
			Scope:    1, // 1 特定主机
			ClientID: clientID,
		})
		if err != nil {
			t.Fatalf("add virus scan failed: %v", err)
		}
		t.Logf("virus_scan_add done: client_id=%s", clientID)
	})

	t.Run("virus_scan_update", func(t *testing.T) {
		// 先获取一个现有的扫描计划
		result, err := client.ListVirusScans(ctx, ListVirusScansRequest{Page: 1, Limit: 1})
		if err != nil {
			t.Fatalf("list virus scans failed: %v", err)
		}
		if len(result.Results) == 0 {
			t.Skip("no virus scan to update")
		}
		rid := result.Results[0].RID
		t.Logf("updating virus scan: rid=%s", rid)

		// 更新扫描计划名称
		err = client.UpdateVirusScan(ctx, UpdateVirusScanRequest{
			RID:      rid,
			PlanName: "updated integration test scan",
		})
		if err != nil {
			t.Fatalf("update virus scan failed: %v", err)
		}
		t.Logf("virus_scan_update done: rid=%s", rid)
	})

	t.Run("virus_scan_cancel", func(t *testing.T) {
		// 先获取一个现有的扫描计划
		result, err := client.ListVirusScans(ctx, ListVirusScansRequest{Page: 1, Limit: 10})
		if err != nil {
			t.Fatalf("list virus scans failed: %v", err)
		}
		var rid string
		for _, scan := range result.Results {
			if scan.Status == 0 || scan.Status == 1 {
				rid = scan.RID
				break
			}
		}
		if rid == "" {
			t.Skip("no virus scan to cancel")
		}
		t.Logf("canceling virus scan: rid=%s", rid)

		err = client.CancelVirusScan(ctx, rid)
		if err != nil {
			t.Fatalf("cancel virus scan failed: %v", err)
		}
		t.Logf("virus_scan_cancel done: rid=%s", rid)
	})

	t.Run("settings_switch_ngav_status", func(t *testing.T) {
		// 先获取当前 NGAV 配置
		conf, err := client.GetNGAVConf(ctx)
		if err != nil {
			t.Fatalf("get ngav conf failed: %v", err)
		}
		t.Logf("current ngav conf: %+v", conf)

		// 切换状态
		err = client.SwitchNGAVStatus(ctx, "off")
		if err != nil {
			t.Fatalf("switch ngav status failed: %v", err)
		}
		t.Logf("settings_switch_ngav_status done")
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
