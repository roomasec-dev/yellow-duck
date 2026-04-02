# open_api Service Swagger

**Version**: 2026-03-27

按当前 open_api 路由映射和 ../ratp-console/services/console 下游实现生成；open_api 依赖 X-UserId/X-UserName/X-OrgName 头，而不是 Bearer Token。

---

## Authentication

This API uses the following headers:

- `X-UserId`: User ID header
- `X-UserName`: User Name header
- `X-OrgName`: Organization Name header

---

## Rm Native

### RM 检测列表

**POST** `/open_api/rm/v1/detections/list`

open_api 原生接口；上下文来自 X-UserId/X-UserName/X-OrgName。

**Request Body**:

Content-Type: `application/json`

```json
  - **page**:     - **cur_page**: integer (required) 
    - **page_size**: integer (required)  
  - **sort**: array<OpenApiSort> 
  - **start_time**: integer 
  - **end_time**: integer 
  - **threat_severity**: array<integer> 
  - **process_result**: array<integer> 
  - **threat_phase_list**: array<OpenApiThreatPhase> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **data**: array<OpenApiDetectionVO> (required) 
    - **total**: integer (required) 

---

### RM 事件列表

**POST** `/open_api/rm/v1/incidents/list`

open_api 原生接口；上下文来自 X-UserId/X-UserName/X-OrgName。

**Request Body**:

Content-Type: `application/json`

```json
  - **org_name**: string 可选，会被 X-OrgName 覆盖
  - **page**:     - **cur_page**: integer (required) 
    - **page_size**: integer (required)  
  - **sort**: array<OpenApiSort> 
  - **start_time**: integer 
  - **end_time**: integer 
  - **params**: array<OpenApiIncidentParam> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **data**: array<OpenApiIncidentVO> (required) 
    - **total**: integer (required) 

---

### RM 日志列表

**POST** `/open_api/rm/v1/logs/list`

open_api 原生接口；上下文来自 X-UserId/X-UserName/X-OrgName。 日志项为动态结构。

**Request Body**:

Content-Type: `application/json`

```json
  - **page**:     - **cur_page**: integer (required) 
    - **page_size**: integer (required)  
  - **sort**: array<OpenApiSort> 
  - **start_time**: integer 
  - **end_time**: integer 
  - **filter**: array<OpenApiLogFilterItem> 
  - **sql**: string 
  - **search_key**: array<string> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **data**: array<OpenApiLogInfoEntry> (required) 
    - **total**: integer (required) 

---

## Ts Native

### TS 检测列表

**POST** `/open_api/ts/v1/detections/list`

open_api 原生接口；上下文来自 X-UserId/X-UserName/X-OrgName。

**Request Body**:

Content-Type: `application/json`

```json
  - **page**:     - **cur_page**: integer (required) 
    - **page_size**: integer (required)  
  - **sort**: array<OpenApiSort> 
  - **start_time**: integer 
  - **end_time**: integer 
  - **threat_severity**: array<integer> 
  - **process_result**: array<integer> 
  - **threat_phase_list**: array<OpenApiThreatPhase> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **data**: array<OpenApiDetectionVO> (required) 
    - **total**: integer (required) 

---

### TS 事件列表

**POST** `/open_api/ts/v1/incidents/list`

open_api 原生接口；上下文来自 X-UserId/X-UserName/X-OrgName。

**Request Body**:

Content-Type: `application/json`

```json
  - **org_name**: string 可选，会被 X-OrgName 覆盖
  - **page**:     - **cur_page**: integer (required) 
    - **page_size**: integer (required)  
  - **sort**: array<OpenApiSort> 
  - **start_time**: integer 
  - **end_time**: integer 
  - **params**: array<OpenApiIncidentParam> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **data**: array<OpenApiIncidentVO> (required) 
    - **total**: integer (required) 

---

## Proxy Hosts

### 全球化主机列表

**POST** `/open_api/rm/v1/hosts/globalization/list`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/hosts/globalization/list

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **page**: integer 
  - **limit**: integer 
  - **hostname**: string 
  - **client_id**: string 
  - **client_ids**: array<string> 
  - **status**: string 
  - **importance**: integer 
  - **os_version**: string 
  - **os_type**: integer 
  - **rmconnectip**: string 
  - **orgconnectip**: string 
  - **client_ip**: string 
  - **mac_address**: string 
  - **win_version**: string 
  - **client_version**: string 
  - **username**: string 
  - **remarks**: string 
  - **isolate**: boolean 
  - **business_type**: integer 
  - **is_export**: integer 
  - **gids**: array<integer> 
  - **platform**: string 
  - **last_logon_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
  - **first_seen_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
  - **last_seen_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **hosts**: array<object> 
    - **total**: integer 
    - **pages**: integer 
    - **current_page**: integer 

---

## Proxy Isolate File

### 删除隔离文件记录

**POST** `/open_api/rm/v1/isolate_file/delete`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/isolate_file/delete

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **guids**: array<string> (required) 
  - **is_add_exclusion**: boolean 下游 DTO 仅识别该字段
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 隔离文件列表

**POST** `/open_api/rm/v1/isolate_file/get_list`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/isolate_file/get_list

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **page**: integer 
  - **limit**: integer 
  - **recover_status**: string 
  - **path**: string 
  - **md5**: string 
  - **sha1**: string 
  - **file_name**: string 
  - **hostname**: string 
  - **username**: string 
  - **task_id**: string 
  - **last_quarantine_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **total**: integer 
    - **results**: array<object> 

---

### 释放隔离文件

**POST** `/open_api/rm/v1/isolate_file/release`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/isolate_file/release

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **guids**: array<string> (required) 
  - **is_add_exclusion**: boolean 下游 DTO 仅识别该字段
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

## Proxy Configure

### 新增 IOA 规则

**POST** `/open_api/rm/v1/configure/ioa/add`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/configure/ioa/add

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **ioa_id**: string IOA 规则 ID
  - **ta_id**: string 战术 ID
  - **t_id**: string 技术 ID
  - **ioa_name**: string IOA 名称
  - **description**: string 描述
  - **severity**: string 严重程度
  - **file_name**: string 文件名称
  - **command_line**: string 命令行
  - **host_type**: string 主机类型
  - **exclusion_name**: string 排除名称
  - **group_ids**: array<integer> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **exclusion_id**: string 唯一 ID
    - **ioa_id**: string 
    - **ta_name**: string 战术名称
    - **t_name**: string 技术名称
    - **ioa_name**: string 
    - **description**: string 
    - **severity**: string 
    - **file_name**: string 
    - **command_line**: string 
    - **host_type**: string 
    - **exclusion_name**: string 
    - **p_uuid**: string 进程唯一 ID
    - **create_time**: integer 
    - **update_time**: integer 
    - **modified_by_id**: string 
    - **operate_user**: string 
    - **group_ids**: array<integer> 

---

### 新增 IOC

**POST** `/open_api/rm/v1/configure/ioc/add`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/configure/ioc/add

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **guid**: string 
  - **ioc_id**: string 
  - **hash**: string (required) 
  - **action**: string (required) 
  - **host_type**: string (required) 
  - **description**: string 
  - **expiration_date**: string 
  - **file_name**: string 
  - **group_ids**: array<integer> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **exclusion_id**: string 
    - **ioc_id**: string 
    - **hash**: string 
    - **action**: string 
    - **date_added**: integer 
    - **last_seen**: integer 
    - **last_modified**: integer 
    - **host_type**: string 
    - **expiration_date**: integer 
    - **detection_count**: integer 
    - **description**: string 
    - **file_name**: string 
    - **level**: integer 
    - **org_name**: string 
    - **group_ids**: array<integer> 

---

### 删除 IOC

**POST** `/open_api/rm/v1/configure/ioc/delete`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/configure/ioc/delete

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **id**: string (required) 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **exclusion_id**: string 
    - **ioc_id**: string 
    - **hash**: string 
    - **action**: string 
    - **date_added**: integer 
    - **last_seen**: integer 
    - **last_modified**: integer 
    - **host_type**: string 
    - **expiration_date**: integer 
    - **detection_count**: integer 
    - **description**: string 
    - **file_name**: string 
    - **level**: integer 
    - **org_name**: string 
    - **group_ids**: array<integer> 

---

### IOC 详情（按 hash）

**POST** `/open_api/rm/v1/configure/ioc/detail`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。 detail 当前入参是 hash。

> Proxy Route: /api/v1/overseas/configure/ioc/detail

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **hash**: string (required) 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### IOC 列表

**POST** `/open_api/rm/v1/configure/ioc/list`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/configure/ioc/list

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **page**: integer 
  - **limit**: integer 
  - **hash**: string 
  - **action**: string 
  - **date_add**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
  - **last_modified**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
  - **host_type**: string 
  - **group_ids**: array<integer> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **results**: array<ConsoleIocDetailResponse> 
    - **total**: integer 

---

### 更新 IOC

**POST** `/open_api/rm/v1/configure/ioc/update`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/configure/ioc/update

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **id**: string (required) 
  - **hash**: string (required) 
  - **action**: string 
  - **host_type**: string 
  - **description**: string 
  - **expiration_date**: string 
  - **file_name**: string 
  - **group_ids**: array<integer> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **exclusion_id**: string 
    - **ioc_id**: string 
    - **hash**: string 
    - **action**: string 
    - **date_added**: integer 
    - **last_seen**: integer 
    - **last_modified**: integer 
    - **host_type**: string 
    - **expiration_date**: integer 
    - **detection_count**: integer 
    - **description**: string 
    - **file_name**: string 
    - **level**: integer 
    - **org_name**: string 
    - **group_ids**: array<integer> 

---

## Proxy Instructions

### 发送指令

**POST** `/open_api/rm/v1/instructions/send_instruction`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v2/instructions/send_instruction

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **client_id**: string 
  - **instruction_name**: string 
  - **process**: array<object> 
  - **params**: object 
  - **batch_params**: array<object> 
  - **is_batch**: integer 
  - **instruction_type**: integer 
  - **incident_id**: string 
  - **task_name**: string 
  - **is_online**: integer 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **task_id**: string 
    - **host_name**: string 
    - **repeat**: boolean 
    - **category**: integer 

---

### 查询指令任务结果

**POST** `/open_api/rm/v1/instructions/task_result`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/instructions/task_result

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **task_id**: string (required) 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **instruction_name**: string 
    - **status**: integer 
    - **message**: string 
    - **host_name**: string 
    - **host_status**: string 
    - **collect_time**: integer 
    - **process**: array<object> 
    - **process_detail**: array<object> 
    - **image_detail**: array<object> 

---

### 查询指令任务列表

**POST** `/open_api/rm/v1/instructions/tasks`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/instructions/tasks

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **page**: integer 
  - **limit**: integer 
  - **client_id**: string 
  - **host_name**: string 
  - **id**: string 
  - **instruction_name**: string 
  - **content**: string 
  - **status**: string 
  - **user**: string 
  - **is_export**: integer 
  - **instruction_type**: integer 
  - **policy_name**: string 
  - **policy_id**: string 
  - **create_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
  - **update_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **results**: array<object> 
    - **total**: integer 

---

## Proxy Incident

### 批量处置事件

**POST** `/open_api/rm/v1/incident/batch_deal`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/incident/batch_deal

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **ids**: array<string> (required) 
  - **allow**: boolean (required) 
  - **status**: integer (required) 
  - **scene**: string (enum: batch, alone) (enum: batch, alone) (required) 
  - **comment**: string 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **incident_name**: string 
    - **status**: integer 
    - **total_detection**: integer 
    - **total_incident**: integer 
    - **incident_names**: array<string> 

---

### 事件摘要（R2）

**POST** `/open_api/rm/v1/incident/r2/summary`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/incident/r2/summary

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **incident_id**: string (required) 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **id**: string 
    - **client_id**: string 
    - **incident_id**: string 
    - **incident_name**: string 
    - **status**: integer 
    - **score**: number 
    - **comment**: string 
    - **remarks**: string 
    - **tags**: array<string> 
    - **ttp**:       - **target**: string 
      - **technique**: string 
      - **course**: string  
    - **release**: integer 
    - **actors**: array<string> 
    - **multihost**: boolean 
    - **scene**: integer 
    - **associated_hosts**: array<string> 
    - **host_id**: string 
    - **host_name**: string 
    - **operating_system**: string 
    - **username**: string 
    - **external_ip**: string 
    - **connection_ip**: string 
    - **client_version**: string 
    - **isolation**: integer 
    - **host_status**: string 
    - **actor**: string 
    - **actor_type**: string 
    - **t_names**: array<string> 
    - **start_time**: integer 
    - **end_time**: integer 
    - **keep_alive_status**: integer 
    - **platform**: integer 

---

### 事件详情视图

**POST** `/open_api/rm/v1/incident/view`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/incident/view

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **incident_id**: string (required) 
  - **client_id**: string (required) 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

## Proxy Detection

### 更新 detection 处置状态

**POST** `/open_api/rm/v1/detection/deal_status`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/detection/deal_status

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **ids**: array<string> (required) 
  - **deal_status**: integer (required) 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 检测列表（console 代理）

**POST** `/open_api/rm/v1/detection/get_list`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/detection/get_list

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **page**: integer 
  - **limit**: integer 
  - **from**: string 
  - **threat_level**: string 
  - **ta_id**: string 
  - **t_id**: string 
  - **hash**: string 
  - **p_name**: string 
  - **detect_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
  - **hostname**: string 
  - **client_id**: string 
  - **username**: string 
  - **deal_status**: string 
  - **incident_id**: string 
  - **deal_status_array**: array<integer> 
  - **start_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
  - **root_name**: string 
  - **main_p_name**: string 
  - **malware_name**: string 
  - **detection_source**: string 
  - **view_type**: string 
  - **detection_ids**: array<string> 
  - **rm_connect_ip**: string 
  - **org_connect_ip**: string 
  - **client_ip**: string 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **results**: array<object> 
    - **total**: integer 

---

### 检测详情视图

**POST** `/open_api/rm/v1/detection/view`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/detection/view

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **detection_id**: string (required) 
  - **client_id**: string (required) 
  - **view_type**: string 
  - **process_uuid**: string 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **client_id**: string 
    - **org_name**: string 
    - **incident_name**: string 
    - **score**: integer 
    - **last_update_time**: integer 
    - **start_time**: integer 
    - **view_type**: string 
    - **timeline**: object 
    - **file_relations**: array<object> 
    - **process_relations**: array<object> 
    - **ioa_timeline**: array<object> 
    - **ioc_list**: object 
    - **host_info**: object 
    - **processes**: array<object> 

---

## Proxy Virus Scan

### 新建病毒扫描计划

**POST** `/open_api/rm/v1/virus_scan/add`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/virus_scan/add

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **scan_type**: integer (required) 
  - **plan_name**: string (required) 
  - **plan_type**: integer (required) 
  - **scope**: integer 
  - **contents**: object 
  - **client_id**: string 
  - **execute_start_time**: integer 
  - **execute_cycle**: integer 
  - **cycle_setting**: boolean 
  - **group_ids**: array<integer> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 取消病毒扫描计划

**POST** `/open_api/rm/v1/virus_scan/cancel`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/virus_scan/cancel

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **rid**: string (required) 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 病毒扫描计划列表

**POST** `/open_api/rm/v1/virus_scan/list`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/virus_scan/list

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **page**: integer 
  - **limit**: integer 
  - **plan_name**: string 
  - **scope**: integer 
  - **plan_type**: integer 
  - **cycle_setting**: integer 
  - **scan_type**: integer 
  - **update_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
  - **status**: string 
  - **operation_user**: string 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **total**: integer 
    - **results**: array<object> 

---

### 病毒扫描执行记录

**POST** `/open_api/rm/v1/virus_scan/scan_record`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/virus_scan/scan_record

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **page**: integer 
  - **limit**: integer 
  - **rid**: string 
  - **task_id**: string 
  - **execution_batch**: string 
  - **host_name**: string 
  - **client_id**: string 
  - **scan_type**: string 
  - **status**: string 
  - **start_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
  - **end_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **total**: integer 
    - **results**: array<object> 

---

### 编辑病毒扫描计划

**POST** `/open_api/rm/v1/virus_scan/update`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/virus_scan/update

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **rid**: string (required) 
  - **scan_type**: integer 
  - **plan_name**: string 
  - **plan_type**: integer 
  - **scope**: integer 
  - **contents**: object 
  - **client_id**: string 
  - **execute_start_time**: integer 
  - **execute_cycle**: integer 
  - **cycle_setting**: boolean 
  - **group_ids**: array<integer> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

## Proxy Virus

### 按 Hash 查看主机明细

**POST** `/open_api/rm/v1/virus/hash/host/list`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/virus/hash/host/list

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **page**: integer 
  - **limit**: integer 
  - **sha1**: string 
  - **client_id**: string 
  - **username**: string 
  - **host_name**: string 
  - **importance**: integer 
  - **mac_address**: string 
  - **client_ip**: string 
  - **rmconnectip**: string 
  - **status**: integer 
  - **host_status**: string 
  - **path**: string 
  - **last_checked_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **total**: integer 
    - **results**: array<object> 

---

### 按文件 Hash 统计病毒命中

**POST** `/open_api/rm/v1/virus/hash/list`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/virus/hash/list

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **page**: integer 
  - **limit**: integer 
  - **last_checked_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
  - **name**: string 
  - **sha1**: string 
  - **md5**: string 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **total**: integer 
    - **results**: array<object> 

---

### 按主机统计病毒命中

**POST** `/open_api/rm/v1/virus/host/list`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api/v1/overseas/virus/host/list

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **page**: integer 
  - **limit**: integer 
  - **client_id**: string 
  - **username**: string 
  - **host_name**: string 
  - **importance**: integer 
  - **mac_address**: string 
  - **client_ip**: string 
  - **rmconnectip**: string 
  - **status**: integer 
  - **last_checked_time**:     - **time_range**:       - **start**: integer 
      - **end**: integer  
    - **quick_time**:       - **time_num**: integer 
      - **time_span**: string 
      - **time_type**: string   
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **total**: integer 
    - **results**: array<object> status 条件存在时结果项会额外带 sha1/md5

---

## Proxy Strategy

### 获取单个策略

**GET** `/open_api/rm/v1/strategy/:strategy_type/single`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api2/v1/strategy/:strategy_type/single

> Downstream: ../ratp-console/services/console

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 创建策略

**POST** `/open_api/rm/v1/strategy/create`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api2/v1/strategy/create

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **name**: string (required) 策略名称
  - **type**: string (required) 策略类型
  - **content**: string 策略内容
  - **range_type**: integer (required) 范围类型: 1-全局, 2-分组
  - **group_ids**: array<integer> 
  - **config_content**: string 
  - **status**: integer 状态: 1-启用, 0-禁用
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 删除策略

**POST** `/open_api/rm/v1/strategy/delete`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api2/v1/strategy/delete

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **strategy_id**: string (required) 
  - **type**: string (required) 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 策略详情

**POST** `/open_api/rm/v1/strategy/detail`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api2/v1/strategy/detail

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **type**: string (required) 
  - **strategy_id**: string (required) 
  - **status**: integer 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 获取默认策略

**POST** `/open_api/rm/v1/strategy/get_default`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api2/v1/strategy/get_default

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **type**: string (required) 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 策略列表

**POST** `/open_api/rm/v1/strategy/list`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api2/v1/strategy/list

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **type**: string 策略类型
  - **name**: string 
  - **status**: integer 
  - **range_type**: integer 
  - **group_ids**: array<integer> 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 排序策略

**POST** `/open_api/rm/v1/strategy/sort`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api2/v1/strategy/sort

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **type**: string (required) 
  - **sort_ids**: array<string> (required) 排序后的策略ID列表
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 获取策略状态

**GET** `/open_api/rm/v1/strategy/state`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api2/v1/strategy/state

> Downstream: ../ratp-console/services/console

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 更新策略状态

**POST** `/open_api/rm/v1/strategy/status`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api2/v1/strategy/status

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **strategy_id**: string (required) 
  - **type**: string (required) 
  - **status**: integer (required) 状态: 1-启用, 0-禁用
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 更新策略

**POST** `/open_api/rm/v1/strategy/update`

open_api 代理接口；按 proxy_routes.go 转发到 ratp-console/services/console，下游 request/response 以当前代码为准。

> Proxy Route: /api2/v1/strategy/update

> Downstream: ../ratp-console/services/console

**Request Body**:

Content-Type: `application/json`

```json
  - **strategy_id**: string (required) 策略ID
  - **name**: string 
  - **type**: string 
  - **content**: string 
  - **range_type**: integer 
  - **group_ids**: array<integer> 
  - **config_content**: string 
  - **status**: integer 
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object