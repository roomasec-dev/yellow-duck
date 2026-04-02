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
		testHash := strings.ToUpper("382919d25113457f96e6428548e492033253aad3")
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
		listResult, err := client.ListIOCs(ctx, ListIOCsRequest{Hash: testHash, Limit: 100})
		if err != nil {
			t.Fatalf("list iocs after add failed: %v", err)
		}
		var iocID string
		for _, i := range listResult.Results {
			if i.Hash == testHash {
				iocID = i.ExclusionID.(string)
				break
			}
		}
		if iocID == "" {
			t.Fatalf("ioc not found after add: hash=%s", testHash)
		}
		t.Logf("found ioc id: %s", iocID)

		// 更新 IOC
		updateReq := UpdateIOCRequest{
			ID:          iocID,
			Hash:        testHash,
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
		result, err := client.ListIsolateFiles(ctx, ListIsolateFilesRequest{Page: 1, Limit: 3})
		// t.Logf("isolate_file_delete result %+v", result)
		// raw, _ := json.MarshalIndent(result, "", "  ")
		// t.Logf("isolate_file_delete raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("list isolate files failed: %v", err)
		}
		if len(result.Results) == 0 {
			t.Skip("no isolate files to release")
		}
		// 循环查找第一个 recoverStatus=1 的隔离文件
		var guid string
		for _, f := range result.Results {
			if f.RecoverStatus == 1 {
				guid = f.GUID
				break
			}
		}
		if guid == "" {
			t.Skip("no isolate file with recoverStatus=1 to delete")
		}
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

	t.Run("incident_r2_summary", func(t *testing.T) {
		// 先获取一个事件 ID
		incidents, err := client.ListIncidents(ctx, ListIncidentsRequest{Page: 1, PageSize: 1})
		if err != nil {
			t.Fatalf("list incidents failed: %v", err)
		}
		if len(incidents.Incidents) == 0 {
			t.Skip("no incident to get r2 summary")
		}
		incidentID := incidents.Incidents[0].IncidentID
		clientID := incidents.Incidents[0].ClientID
		fullIncidentID := incidentID
		if clientID != "" {
			fullIncidentID = clientID + "-" + incidentID
		}
		t.Logf("getting incident r2 summary: id=%s", fullIncidentID)

		result, err := client.IncidentR2Summary(ctx, fullIncidentID)
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("incident_r2_summary raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("incident r2 summary failed: %v", err)
		}
		t.Logf("incident_r2_summary done: id=%s", result.ID)
	})

	t.Run("batch_deal_incident", func(t *testing.T) {
		// 先获取一个事件
		incidents, err := client.ListIncidents(ctx, ListIncidentsRequest{Page: 1, PageSize: 1})
		if err != nil {
			t.Fatalf("list incidents failed: %v", err)
		}
		if len(incidents.Incidents) == 0 {
			t.Skip("no incident to batch deal")
		}
		incidentID := incidents.Incidents[0].IncidentID
		clientID := incidents.Incidents[0].ClientID
		fullIncidentID := incidentID
		if clientID != "" {
			fullIncidentID = clientID + "-" + incidentID
		}
		t.Logf("batch dealing incident: id=%s", fullIncidentID)

		result, err := client.BatchDealIncident(ctx, BatchDealIncidentRequest{
			IDs:    []string{fullIncidentID},
			Allow:  false,
			Status: 2, // 2 = 已处理
			Scene:  "manual",
		})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("batch_deal_incident raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("batch deal incident failed: %v", err)
		}
		t.Logf("batch_deal_incident done: total=%d", result.TotalIncident)
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

	t.Run("detections_proxy_list", func(t *testing.T) {
		result, err := client.ListDetectionsProxy(ctx, ListDetectionsProxyRequest{Page: 1, Limit: 10})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("detections_proxy_list raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("detections proxy list failed: %v", err)
		}
		t.Logf("detections_proxy_list done: total=%d", result.Total)
	})

	t.Run("update_detection_status", func(t *testing.T) {
		// 先获取一个检测记录
		result, err := client.ListDetectionsProxy(ctx, ListDetectionsProxyRequest{Page: 1, Limit: 1})
		if err != nil {
			t.Fatalf("list detections proxy failed: %v", err)
		}
		if len(result.Results) == 0 {
			t.Skip("no detection to update status")
		}
		// 获取第一个检测的 id
		detectionID, ok := result.Results[0]["id"].(string)
		if !ok || detectionID == "" {
			t.Skip("no detection id found")
		}
		t.Logf("updating detection status: id=%s", detectionID)

		err = client.UpdateDetectionStatus(ctx, UpdateDetectionStatusRequest{
			IDs:        []string{detectionID},
			DealStatus: 1, // 1 = 已处理
		})
		if err != nil {
			t.Fatalf("update detection status failed: %v", err)
		}
		t.Logf("update_detection_status done: id=%s", detectionID)
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

	// Client Setting (Host Offline) tests
	t.Run("client_setting_get_host_offline", func(t *testing.T) {
		result, err := client.GetHostOfflineConf(ctx)
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("client_setting_get_host_offline raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("get host offline conf failed: %v", err)
		}
		t.Logf("client_setting_get_host_offline done: status=%d", result.Status)
	})

	t.Run("client_setting_save_host_offline", func(t *testing.T) {
		err := client.SaveHostOfflineConf(ctx, SaveHostOfflineConfRequest{
			Status: 1,
			Setting: HostOfflineSetting{
				Timeout: "10m",
			},
		})
		if err != nil {
			t.Fatalf("save host offline conf failed: %v", err)
		}
		t.Logf("client_setting_save_host_offline done")
	})

	// IOA Configuration tests
	t.Run("ioa_list", func(t *testing.T) {
		result, err := client.ListIOAs(ctx, ListIOAsRequest{Page: 1, Limit: 10})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("ioa_list raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("list ioas failed: %v", err)
		}
		if result.Total < 0 {
			t.Fatalf("unexpected total: %+v", result)
		}
		t.Logf("ioa_list done: total=%d", result.Total)
	})

	t.Run("ioa_add", func(t *testing.T) {
		err := client.AddIOA(ctx, AddIOARequest{
			CommandLine:   "test_cmd",
			Description:   "integration test ioa",
			ExclusionName: "test_ioa",
			FileName:      "test.exe",
			HostType:      "ALL",
			Severity:      "high",
		})
		if err != nil {
			t.Logf("add ioa failed (may already exist): %v", err)
		}
		t.Logf("ioa_add done")
	})

	t.Run("ioa_update", func(t *testing.T) {
		// 先获取一个 IOA
		result, err := client.ListIOAs(ctx, ListIOAsRequest{Page: 1, Limit: 1})
		if err != nil {
			t.Fatalf("list ioas failed: %v", err)
		}
		if len(result.Results) == 0 {
			t.Skip("no ioa to update")
		}
		ioaID := result.Results[0].ExclusionID
		t.Logf("updating ioa: id=%s", ioaID)

		err = client.UpdateIOA(ctx, UpdateIOARequest{
			ID:          ioaID,
			Description: "updated integration test ioa",
		})
		if err != nil {
			t.Fatalf("update ioa failed: %v", err)
		}
		t.Logf("ioa_update done: id=%s", ioaID)
	})

	t.Run("ioa_delete", func(t *testing.T) {
		// 先获取一个 IOA
		result, err := client.ListIOAs(ctx, ListIOAsRequest{Page: 1, Limit: 1})
		if err != nil {
			t.Fatalf("list ioas failed: %v", err)
		}
		if len(result.Results) == 0 {
			t.Skip("no ioa to delete")
		}
		ioaID := result.Results[0].ExclusionID
		t.Logf("deleting ioa: id=%s", ioaID)

		err = client.DeleteIOA(ctx, ioaID)
		if err != nil {
			t.Fatalf("delete ioa failed: %v", err)
		}
		t.Logf("ioa_delete done: id=%s", ioaID)
	})

	t.Run("ioa_audit_log", func(t *testing.T) {
		result, err := client.ListIOAAuditLogs(ctx, ListIOAAuditLogsRequest{Page: 1, Limit: 10})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("ioa_audit_log raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("list ioa audit logs failed: %v", err)
		}
		t.Logf("ioa_audit_log done: total=%d", result.Total)
	})

	// IOA Network Exclusion tests
	t.Run("ioa_network_list", func(t *testing.T) {
		result, err := client.ListIOANetworks(ctx, ListIOANetworksRequest{Page: 1, Limit: 10})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("ioa_network_list raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("list ioa networks failed: %v", err)
		}
		t.Logf("ioa_network_list done: total=%d", result.Total)
	})

	t.Run("ioa_network_add", func(t *testing.T) {
		err := client.AddIOANetwork(ctx, AddIOANetworkRequest{
			ExclusionName: "test_network",
			HostType:      "ALL",
			IP:            "192.168.1.1",
		})
		if err != nil {
			t.Logf("add ioa network failed (may already exist): %v", err)
		}
		t.Logf("ioa_network_add done")
	})

	t.Run("ioa_network_update", func(t *testing.T) {
		// 先获取一个 IOA Network
		result, err := client.ListIOANetworks(ctx, ListIOANetworksRequest{Page: 1, Limit: 1})
		if err != nil {
			t.Fatalf("list ioa networks failed: %v", err)
		}
		if len(result.Results) == 0 {
			t.Skip("no ioa network to update")
		}
		networkID := result.Results[0].ID
		t.Logf("updating ioa network: id=%s", networkID)

		err = client.UpdateIOANetwork(ctx, UpdateIOANetworkRequest{
			ID:            networkID,
			ExclusionName: "updated_network",
		})
		if err != nil {
			t.Fatalf("update ioa network failed: %v", err)
		}
		t.Logf("ioa_network_update done: id=%s", networkID)
	})

	t.Run("ioa_network_delete", func(t *testing.T) {
		// 先获取一个 IOA Network
		result, err := client.ListIOANetworks(ctx, ListIOANetworksRequest{Page: 1, Limit: 1})
		if err != nil {
			t.Fatalf("list ioa networks failed: %v", err)
		}
		if len(result.Results) == 0 {
			t.Skip("no ioa network to delete")
		}
		networkID := result.Results[0].ID
		t.Logf("deleting ioa network: id=%s", networkID)

		err = client.DeleteIOANetwork(ctx, networkID)
		if err != nil {
			t.Fatalf("delete ioa network failed: %v", err)
		}
		t.Logf("ioa_network_delete done: id=%s", networkID)
	})

	// Strategy Management tests
	t.Run("strategy_state", func(t *testing.T) {
		result, err := client.GetStrategyState(ctx)
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("strategy_state raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("get strategy state failed: %v", err)
		}
		t.Logf("strategy_state done: all_strategy=%d active_strategy=%d", result.AllStrategy, result.ActiveStrategy)
	})

	t.Run("strategy_list", func(t *testing.T) {
		result, err := client.ListStrategies(ctx, ListStrategiesRequest{Page: 1, Limit: 10, Type: "virus_scan_settings"})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("strategy_list raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("list strategies failed: %v", err)
		}
		t.Logf("strategy_list done: total=%d", result.Total)
	})

	t.Run("strategy_single", func(t *testing.T) {
		result, err := client.GetStrategySingle(ctx, "virus_scan_settings")
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("strategy_single raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("get strategy single failed: %v", err)
		}
		t.Logf("strategy_single done: name=%s", result.Name)
	})

	t.Run("strategy_detail", func(t *testing.T) {
		// 先获取一个策略
		listResult, err := client.ListStrategies(ctx, ListStrategiesRequest{Page: 1, Limit: 1, Type: "virus_scan_settings"})
		if err != nil {
			t.Fatalf("list strategies failed: %v", err)
		}
		if len(listResult.Items) == 0 {
			t.Skip("no strategy to get detail")
		}
		strategyID := listResult.Items[0].StrategyID
		strategyType := listResult.Items[0].Type
		t.Logf("getting strategy detail: id=%s type=%s", strategyID, strategyType)

		result, err := client.GetStrategyDetail(ctx, GetStrategyDetailRequest{
			StrategyID: strategyID,
			Type:       strategyType,
		})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("strategy_detail raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("get strategy detail failed: %v", err)
		}
		t.Logf("strategy_detail done: name=%s", result.Name)
	})

	t.Run("strategy_create", func(t *testing.T) {
		result, err := client.CreateStrategy(ctx, CreateStrategyRequest{
			Name:      "integration test strategy",
			Type:      "virus_scan_settings",
			RangeType: 1,
			Status:    1,
		})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("strategy_create raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("create strategy failed: %v", err)
		}
		t.Logf("strategy_create done: strategy_id=%s", result.StrategyID)
	})

	t.Run("strategy_update", func(t *testing.T) {
		// 先获取一个策略
		listResult, err := client.ListStrategies(ctx, ListStrategiesRequest{Page: 1, Limit: 1, Type: "virus_scan_settings"})
		if err != nil {
			t.Fatalf("list strategies failed: %v", err)
		}
		if len(listResult.Items) == 0 {
			t.Skip("no strategy to update")
		}
		strategyID := listResult.Items[0].StrategyID
		t.Logf("updating strategy: id=%s", strategyID)

		err = client.UpdateStrategy(ctx, UpdateStrategyRequest{
			StrategyID: strategyID,
			Name:       "updated integration test strategy",
		})
		if err != nil {
			t.Fatalf("update strategy failed: %v", err)
		}
		t.Logf("strategy_update done: id=%s", strategyID)
	})

	t.Run("strategy_delete", func(t *testing.T) {
		// 先创建一个策略再删除
		createResult, err := client.CreateStrategy(ctx, CreateStrategyRequest{
			Name:      "integration test strategy to delete",
			Type:      "virus_scan_settings",
			RangeType: 1,
			Status:    1,
		})
		if err != nil {
			t.Fatalf("create strategy failed: %v", err)
		}
		t.Logf("created strategy: id=%s", createResult.StrategyID)

		err = client.DeleteStrategy(ctx, createResult.StrategyID, "virus_scan_settings")
		if err != nil {
			t.Fatalf("delete strategy failed: %v", err)
		}
		t.Logf("strategy_delete done: id=%s", createResult.StrategyID)
	})

	t.Run("strategy_sort", func(t *testing.T) {
		// 先获取策略列表
		listResult, err := client.ListStrategies(ctx, ListStrategiesRequest{Page: 1, Limit: 10, Type: "virus_scan_settings"})
		if err != nil {
			t.Fatalf("list strategies failed: %v", err)
		}
		if len(listResult.Items) == 0 {
			t.Skip("no strategy to sort")
		}

		var sortIDs []string
		for _, item := range listResult.Items {
			sortIDs = append(sortIDs, item.StrategyID)
		}

		err = client.SortStrategies(ctx, sortIDs, "virus_scan_settings")
		if err != nil {
			t.Fatalf("sort strategies failed: %v", err)
		}
		t.Logf("strategy_sort done")
	})

	t.Run("strategy_status", func(t *testing.T) {
		// 先获取一个策略
		listResult, err := client.ListStrategies(ctx, ListStrategiesRequest{Page: 1, Limit: 1, Type: "virus_scan_settings"})
		if err != nil {
			t.Fatalf("list strategies failed: %v", err)
		}
		if len(listResult.Items) == 0 {
			t.Skip("no strategy to update status")
		}
		strategyID := listResult.Items[0].StrategyID
		t.Logf("updating strategy status: id=%s", strategyID)

		err = client.UpdateStrategyStatus(ctx, UpdateStrategyStatusRequest{
			StrategyID: strategyID,
			Type:       "virus_scan_settings",
			Status:     1,
		})
		if err != nil {
			t.Fatalf("update strategy status failed: %v", err)
		}
		t.Logf("strategy_status done: id=%s", strategyID)
	})

	t.Run("strategy_get_default", func(t *testing.T) {
		// 先获取一个策略
		listResult, err := client.ListStrategies(ctx, ListStrategiesRequest{Page: 1, Limit: 1, Type: "virus_scan_settings"})
		if err != nil {
			t.Fatalf("list strategies failed: %v", err)
		}
		if len(listResult.Items) == 0 {
			t.Skip("no strategy to get default")
		}
		strategyID := listResult.Items[0].StrategyID
		t.Logf("getting default strategy: id=%s", strategyID)

		result, err := client.GetDefaultStrategy(ctx, GetDefaultStrategyRequest{
			StrategyID: strategyID,
			Type:       "virus_scan_settings",
		})
		raw, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("strategy_get_default raw json:\n%s", string(raw))
		if err != nil {
			t.Fatalf("get default strategy failed: %v", err)
		}
		t.Logf("strategy_get_default done: name=%s", result.Name)
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
