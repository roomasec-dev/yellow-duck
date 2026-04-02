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

---

## Proxy Plan

### 新建计划

**POST** `/open_api/rm/v1/plan/add`

sase-console-api 新建计划接口

> Proxy Route: /api/v1/plan/add

> Downstream: ../sase-console-api

**Request Body**:

Content-Type: `application/json`

```json
  - **rid**: string 计划id (编辑时必填)
  - **scan_type**: integer (required) 1:快速扫描 2:全盘扫描 3:自定义路径扫描 4:漏洞修复 5:安装软件 6:卸载软件 7:更新软件 8:发送文件
  - **plan_name**: string 计划名称
  - **plan_type**: integer (required) 1:立即执行 2:定时执行 3:周期执行
  - **scope**: integer (required) 1:特定主机 2:主机组 3:全网主机
  - **contents**: object 内容 (scan_path/software/file等)
  - **execute_start_time**: integer 执行开始时间
  - **execute_cycle**: integer 执行周期 1:每天 2:每周 3:每月
  - **repeat_cycle**: array<integer> 重复周期 周0-6 月1-31
  - **execution_time**: string 执行时间 hh:mm
  - **group_ids**: array<integer> 主机组
  - **type**: string (required) 业务类型: kill_plan/leak_repair/distribute_software/distribute_file
  - **device_client_ids**: array<string> 主机id数组
  - **expired_setting**: integer 过期设置 0:永不过期 1:指定过期时间
  - **expired_time**: integer 过期时间
  - **search_content**: array<string> 搜索内容
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 取消计划

**PUT** `/open_api/rm/v1/plan/cancel/:rid`

sase-console-api 取消计划接口

> Proxy Route: /api/v1/plan/cancel/:rid

> Downstream: ../sase-console-api

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 编辑计划

**POST** `/open_api/rm/v1/plan/edit`

sase-console-api 编辑计划接口

> Proxy Route: /api/v1/plan/edit

> Downstream: ../sase-console-api

**Request Body**:

Content-Type: `application/json`

```json
  - **rid**: string 计划id (编辑时必填)
  - **scan_type**: integer (required) 1:快速扫描 2:全盘扫描 3:自定义路径扫描 4:漏洞修复 5:安装软件 6:卸载软件 7:更新软件 8:发送文件
  - **plan_name**: string 计划名称
  - **plan_type**: integer (required) 1:立即执行 2:定时执行 3:周期执行
  - **scope**: integer (required) 1:特定主机 2:主机组 3:全网主机
  - **contents**: object 内容 (scan_path/software/file等)
  - **execute_start_time**: integer 执行开始时间
  - **execute_cycle**: integer 执行周期 1:每天 2:每周 3:每月
  - **repeat_cycle**: array<integer> 重复周期 周0-6 月1-31
  - **execution_time**: string 执行时间 hh:mm
  - **group_ids**: array<integer> 主机组
  - **type**: string (required) 业务类型: kill_plan/leak_repair/distribute_software/distribute_file
  - **device_client_ids**: array<string> 主机id数组
  - **expired_setting**: integer 过期设置 0:永不过期 1:指定过期时间
  - **expired_time**: integer 过期时间
  - **search_content**: array<string> 搜索内容
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data: object

---

### 计划列表

**POST** `/open_api/rm/v1/plan/list`

sase-console-api 病毒扫描计划列表接口

> Proxy Route: /api/v1/plan/list

> Downstream: ../sase-console-api

**Request Body**:

Content-Type: `application/json`

```json
  - **scan_type**: integer 1:快速扫描 2:全盘扫描 3:自定义路径扫描 4:漏洞修复 5:安装软件 6:卸载软件 7:更新软件 8:发送文件
  - **plan_type**: integer 1:立即执行 2:计划执行
  - **type**: string (required) 业务类型: kill_plan查杀计划, leak_repair漏洞修复, distribute_software分发软件, distribute_file发送文件
  - **search_content**: string 
  - **page**: integer (required) 页码
  - **limit**: integer (required) 每页数量
```

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **total**: integer 
    - **items**: array<object> 

---

### 计划任务记录

**GET** `/open_api/rm/v1/plan/task/:rid`

sase-console-api 计划任务记录接口

> Proxy Route: /api/v1/plan/task/:rid

> Downstream: ../sase-console-api

**Responses**:

- `200`: HTTP 200 + error/message/data
  - `application/json`:
    - error: integer
    - message: string
    - data:     - **total**: integer 
    - **items**: array<object> 



# 自动响应的策略

## POST 列表查询

POST /open_api/rm/v1/instruction_policy/list

> Body 请求参数

```json
{
  "policy_type": 1,
  "name": "sss",
  "operation_user": "水水水水",
  "scopes": "1",
  "action": 1,
  "status": 1
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 是 |none|
|» policy_type|body|integer| 否 |1 内置策略 2 自定义策略|
|» name|body|string| 否 |none|
|» operation_user|body|string| 否 |none|
|» scopes|body|string| 否 |// 1 隔离网络 // 2 智能响应|
|» action|body|integer| 否 |// 1 隔离网络 // 2 智能响应|
|» status|body|integer| 否 |// 1 启用 2 禁用|
|» create_time|body|object| 否 |none|
|»» time_range|body|object| 是 |none|
|»»» start|body|integer| 是 |none|
|»»» end|body|integer| 是 |none|
|» update_time|body|object| 否 |none|
|»» time_range|body|object| 是 |none|
|»»» start|body|integer| 是 |none|
|»»» end|body|integer| 是 |none|

> 返回示例

> 200 Response

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "result": [
      {
        "rid": "1962463468078501888",
        "policy_type": 2,
        "name": "testmf",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2,
          1
        ],
        "client_id": "d0bf954479f84a1fa3e176ee079021d7",
        "scope": 2,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "DESKTOP-I5L1OH7",
        "group_ids": [],
        "operation_user": "171****5221",
        "create_time": 1756722726,
        "update_time": 1769482142,
        "status": 1,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1998702940830830592",
        "policy_type": 2,
        "name": "adfasdfasdf",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "98574e36f33ebbadd153fcf3a4177c2a",
        "scope": 2,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "kylin-pc",
        "group_ids": [],
        "operation_user": "123456789011111",
        "create_time": 1765362889,
        "update_time": 1765871279,
        "status": 1,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1996146956669292544",
        "policy_type": 2,
        "name": "士大夫但是",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "123456789011111",
        "create_time": 1764753495,
        "update_time": 1765952748,
        "status": 1,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1960987025683255296",
        "policy_type": 2,
        "name": "test",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "4b52dd63e9fb4c94820a8c5f34f1f0c9",
        "scope": 2,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "DESKTOP-I5L1OH7",
        "group_ids": [],
        "operation_user": "176****2470",
        "create_time": 1756370714,
        "update_time": 1767956912,
        "status": 1,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1960988795289473024",
        "policy_type": 2,
        "name": "fdsf",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 3,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "",
        "group_ids": [
          292
        ],
        "operation_user": "176****2470",
        "create_time": 1756371136,
        "update_time": 1767956913,
        "status": 1,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "2019715677614510080",
        "policy_type": 2,
        "name": "手动阀手动阀撒",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "123456789011111",
        "create_time": 1770372716,
        "update_time": 1770372734,
        "status": 1,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1284141750721056768",
        "policy_type": 1,
        "name": "CobaltStrike远控响应策略",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2,
          1
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "全网主机",
        "group_ids": [],
        "operation_user": "",
        "create_time": 1761891231,
        "update_time": 1764301552,
        "status": 1,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "2032325973063503872",
        "policy_type": 2,
        "name": "拦截龙虾进程",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2,
          1
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "155****3036",
        "create_time": 1773379245,
        "update_time": 1773379362,
        "status": 1,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "2039626466089504768",
        "policy_type": 2,
        "name": "侧是",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "177****6010",
        "create_time": 1775119818,
        "update_time": 1775119818,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "2036728724501565440",
        "policy_type": 2,
        "name": "test auto",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "135****1247",
        "create_time": 1774428943,
        "update_time": 1774428943,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "2019973335835742208",
        "policy_type": 2,
        "name": "test-m",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "135****4246",
        "create_time": 1770434147,
        "update_time": 1770434406,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "2019974316434657280",
        "policy_type": 2,
        "name": "test-m1",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "135****4246",
        "create_time": 1770434381,
        "update_time": 1770434404,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1237061059751841792",
        "policy_type": 1,
        "name": "银狐远控响应策略",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2,
          1
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "全网主机",
        "group_ids": [],
        "operation_user": "",
        "create_time": 1750666320,
        "update_time": 1764301557,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1237061185803259904",
        "policy_type": 1,
        "name": "Ghost远控响应策略",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2,
          1
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "全网主机",
        "group_ids": [],
        "operation_user": "",
        "create_time": 1750666350,
        "update_time": 1764301555,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1284148151589670912",
        "policy_type": 1,
        "name": "勒索攻击响应策略",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2,
          1
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "全网主机",
        "group_ids": [],
        "operation_user": "",
        "create_time": 1761892757,
        "update_time": 1764301554,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1284143021666471936",
        "policy_type": 1,
        "name": "钓鱼攻击响应策略",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2,
          1
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "全网主机",
        "group_ids": [],
        "operation_user": "",
        "create_time": 1761891534,
        "update_time": 1764301554,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1962415055412662272",
        "policy_type": 2,
        "name": "698363",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "1p8_g357joa2oc",
        "create_time": 1756711183,
        "update_time": 1756713655,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1962416311812231168",
        "policy_type": 2,
        "name": "tttt",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "1p8_g357joa2oc",
        "create_time": 1756711483,
        "update_time": 1756711483,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1962414762859958272",
        "policy_type": 2,
        "name": "123123",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "1p8_g357joa2oc",
        "create_time": 1756711114,
        "update_time": 1756711114,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1962413808420917248",
        "policy_type": 2,
        "name": "test77",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 1,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "All",
        "group_ids": [],
        "operation_user": "1p8_g357joa2oc",
        "create_time": 1756710886,
        "update_time": 1756710886,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      },
      {
        "rid": "1962412804409397248",
        "policy_type": 2,
        "name": "fdsf_test9",
        "condition_list": {
          "sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "metas": {}
            }
          ],
          "version": "1.0",
          "metas": {}
        },
        "action": [
          2
        ],
        "client_id": "",
        "scope": 3,
        "tq_group": {
          "groups": null,
          "show_data": ""
        },
        "scope_content": "",
        "group_ids": [
          296,
          292
        ],
        "operation_user": "1p8_g357joa2oc",
        "create_time": 1756710647,
        "update_time": 1756710742,
        "status": 2,
        "task_num": 0,
        "task_start_time": 1775059200,
        "task_end_time": 1775121820
      }
    ]
  }
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» error|integer|true|none||none|
|» message|string|true|none||none|
|» data|object|true|none||none|
|»» result|[object]|true|none||none|
|»»» rid|string|true|none||none|
|»»» policy_type|integer|true|none||none|
|»»» name|string|true|none||none|
|»»» condition_list|object|true|none||none|
|»»»» sets|[object]|true|none||none|
|»»»»» sub_sets_logical|string|true|none||none|
|»»»»» sub_sets|[object]|true|none||none|
|»»»»»» sub_sets_logical|string|true|none||none|
|»»»»»» sub_sets|[object]|true|none||none|
|»»»»»»» sub_sets_logical|string|true|none||none|
|»»»»»»» sub_sets|null|true|none||none|
|»»»»»»» access_rules|[object]|true|none||none|
|»»»»»»»» key|string|true|none||none|
|»»»»»»»» value|string|true|none||none|
|»»»»»»»» compare_method|string|true|none||none|
|»»»»»» access_rules|null|true|none||none|
|»»»»» metas|object|true|none||none|
|»»»» version|string|true|none||none|
|»»»» metas|object|true|none||none|
|»»» action|[integer]|true|none||none|
|»»» client_id|string|true|none||none|
|»»» scope|integer|true|none||none|
|»»» tq_group|object|true|none||none|
|»»»» groups|null|true|none||none|
|»»»» show_data|string|true|none||none|
|»»» scope_content|string|true|none||none|
|»»» group_ids|[integer]|true|none||none|
|»»» operation_user|string|true|none||none|
|»»» create_time|integer|true|none||none|
|»»» update_time|integer|true|none||none|
|»»» status|integer|true|none||none|
|»»» task_num|integer|true|none||none|
|»»» task_start_time|integer|true|none||none|
|»»» task_end_time|integer|true|none||none|

## POST 更新

POST /open_api/rm/v1/instruction_policy/update

> Body 请求参数

```json
{
  "name": "testmf",
  "condition_list": {
    "sets": [
      {
        "sub_sets_logical": "AND",
        "sub_sets": [
          {
            "sub_sets_logical": "AND",
            "sub_sets": [
              {
                "sub_sets_logical": "",
                "sub_sets": null,
                "access_rules": [
                  {
                    "key": "score",
                    "value": "1",
                    "compare_method": "greater_or_equal"
                  }
                ]
              }
            ],
            "access_rules": null
          }
        ],
        "metas": {}
      }
    ],
    "version": "1.0",
    "metas": {}
  },
  "action": [
    2,
    1
  ],
  "scope": 2,
  "client_id": "46148b9c343b41a2be0c54b630a4fd71",
  "group_ids": [],
  "rid": "1962463468078501888",
  "policy_type": 2,
  "tq_group": {
    "groups": null,
    "show_data": ""
  },
  "scope_content": "DESKTOP-I5L1OH7",
  "operation_user": "171****5221",
  "create_time": 1756722726,
  "update_time": 1769482142,
  "status": 1,
  "task_num": 0,
  "task_start_time": 1775059200,
  "task_end_time": 1775121820,
  "index": 1
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 否 |none|
|» name|body|string| 是 |none|
|» condition_list|body|object| 是 |none|
|»» sets|body|[object]| 是 |none|
|»»» sub_sets_logical|body|string| 否 |none|
|»»» sub_sets|body|[object]| 否 |none|
|»»»» sub_sets_logical|body|string| 否 |none|
|»»»» sub_sets|body|[object]| 否 |none|
|»»»»» sub_sets_logical|body|string| 否 |none|
|»»»»» sub_sets|body|null| 否 |none|
|»»»»» access_rules|body|[object]| 否 |none|
|»»»»»» key|body|string| 否 |none|
|»»»»»» value|body|string| 否 |none|
|»»»»»» compare_method|body|string| 否 |none|
|»»»» access_rules|body|null| 否 |none|
|»»» metas|body|object| 否 |none|
|»» version|body|string| 是 |none|
|»» metas|body|object| 是 |none|
|» action|body|[integer]| 是 |none|
|» scope|body|integer| 是 |none|
|» client_id|body|string| 是 |none|
|» group_ids|body|[string]| 是 |none|
|» rid|body|string| 是 |none|
|» policy_type|body|integer| 是 |none|
|» tq_group|body|object| 是 |none|
|»» groups|body|null| 是 |none|
|»» show_data|body|string| 是 |none|
|» scope_content|body|string| 是 |none|
|» operation_user|body|string| 是 |none|
|» create_time|body|integer| 是 |none|
|» update_time|body|integer| 是 |none|
|» status|body|integer| 是 |none|
|» task_num|body|integer| 是 |none|
|» task_start_time|body|integer| 是 |none|
|» task_end_time|body|integer| 是 |none|
|» index|body|integer| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## POST 更新状态

POST /open_api/rm/v1/instruction_policy/save_status

> Body 请求参数

```json
{
  "rid": "1962463468078501888",
  "rids": [
    "1998702940830830592",
    "1996146956669292544",
    "1960987025683255296",
    "1960988795289473024",
    "2019715677614510080",
    "1284141750721056768",
    "2032325973063503872"
  ]
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 是 |none|
|» rid|body|string| 是 |none|
|» rids|body|[string]| 是 |none|

> 返回示例

> 200 Response

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "name": "testmf"
  }
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» error|integer|true|none||none|
|» message|string|true|none||none|
|» data|object|true|none||none|
|»» name|string|true|none||none|

## POST 删除

POST /open_api/rm/v1/instruction_policy/delete

> Body 请求参数

```json
{
  "rid": ""
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 是 |none|
|» rid|body|string| 是 |none|

> 返回示例

> 200 Response

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "rid": "1998702940830830592",
    "hit_continue": "0",
    "is_deleted": 0,
    "policy_type": 2,
    "sub_type": "process",
    "have_resp": true,
    "name": "adfasdfasdf",
    "name_en": "adfasdfasdf",
    "condition_list": {
      "sets": [
        {
          "sub_sets_logical": "AND",
          "sub_sets": [
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "",
                  "sub_sets": null,
                  "access_rules": [
                    "[Object]"
                  ]
                }
              ],
              "access_rules": null
            }
          ],
          "metas": {}
        }
      ],
      "version": "1.0",
      "metas": {}
    },
    "operator_list": {
      "sets": [
        {
          "sub_sets_logical": "AND",
          "sub_sets": [
            {
              "sub_sets_logical": "",
              "sub_sets": null,
              "access_rules": [
                {
                  "key": "org_name",
                  "value": "rmalpha",
                  "compare_method": "equal"
                },
                {
                  "key": "client_id",
                  "value": "98574e36f33ebbadd153fcf3a4177c2a",
                  "compare_method": "equal"
                }
              ]
            },
            {
              "sub_sets_logical": "AND",
              "sub_sets": [
                {
                  "sub_sets_logical": "AND",
                  "sub_sets": [
                    "[Object]"
                  ],
                  "access_rules": null
                }
              ],
              "access_rules": null
            }
          ],
          "metas": {}
        }
      ],
      "version": "1.0",
      "metas": {}
    },
    "action": [
      2
    ],
    "func_list": [
      "DoQuarantineOperation:intelligent_response"
    ],
    "client_id": "98574e36f33ebbadd153fcf3a4177c2a",
    "group_ids": [],
    "scope": 2,
    "tq_group": {
      "groups": null,
      "show_data": ""
    },
    "scope_content": "kylin-pc",
    "operation_user": "123456789011111",
    "operation_uid": "57",
    "create_time": 1765362889,
    "update_time": 1765871279
  }
}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

状态码 **200**

|名称|类型|必选|约束|中文名|说明|
|---|---|---|---|---|---|
|» error|integer|true|none||none|
|» message|string|true|none||none|
|» data|object|true|none||none|
|»» rid|string|true|none||none|
|»» hit_continue|string|true|none||none|
|»» is_deleted|integer|true|none||none|
|»» policy_type|integer|true|none||none|
|»» sub_type|string|true|none||none|
|»» have_resp|boolean|true|none||none|
|»» name|string|true|none||none|
|»» name_en|string|true|none||none|
|»» condition_list|object|true|none||none|
|»»» sets|[object]|true|none||none|
|»»»» sub_sets_logical|string|false|none||none|
|»»»» sub_sets|[object]|false|none||none|
|»»»»» sub_sets_logical|string|false|none||none|
|»»»»» sub_sets|[object]|false|none||none|
|»»»»»» sub_sets_logical|string|false|none||none|
|»»»»»» sub_sets|null|false|none||none|
|»»»»»» access_rules|[object]|false|none||none|
|»»»»»»» key|string|false|none||none|
|»»»»»»» value|string|false|none||none|
|»»»»»»» compare_method|string|false|none||none|
|»»»»» access_rules|null|false|none||none|
|»»»» metas|object|false|none||none|
|»»» version|string|true|none||none|
|»»» metas|object|true|none||none|
|»» operator_list|object|true|none||none|
|»»» sets|[object]|true|none||none|
|»»»» sub_sets_logical|string|false|none||none|
|»»»» sub_sets|[object]|false|none||none|
|»»»»» sub_sets_logical|string|true|none||none|
|»»»»» sub_sets|[object]|true|none||none|
|»»»»»» sub_sets_logical|string|false|none||none|
|»»»»»» sub_sets|[object]|false|none||none|
|»»»»»»» sub_sets_logical|string|false|none||none|
|»»»»»»» sub_sets|null|false|none||none|
|»»»»»»» access_rules|[object]|false|none||none|
|»»»»»»»» key|string|false|none||none|
|»»»»»»»» value|string|false|none||none|
|»»»»»»»» compare_method|string|false|none||none|
|»»»»»» access_rules|null|false|none||none|
|»»»»» access_rules|[object]¦null|true|none||none|
|»»»»»» key|string|true|none||none|
|»»»»»» value|string|true|none||none|
|»»»»»» compare_method|string|true|none||none|
|»»»» metas|object|false|none||none|
|»»» version|string|true|none||none|
|»»» metas|object|true|none||none|
|»» action|[integer]|true|none||none|
|»» func_list|[string]|true|none||none|
|»» client_id|string|true|none||none|
|»» group_ids|[string]|true|none||none|
|»» scope|integer|true|none||none|
|»» tq_group|object|true|none||none|
|»»» groups|null|true|none||none|
|»»» show_data|string|true|none||none|
|»» scope_content|string|true|none||none|
|»» operation_user|string|true|none||none|
|»» operation_uid|string|true|none||none|
|»» create_time|integer|true|none||none|
|»» update_time|integer|true|none||none|

## POST 操作顺序

POST /open_api/rm/v1/instruction_policy/save_sort

> Body 请求参数

```json
{
  "rids": [
    "1996146956669292544",
    "1960987025683255296",
    "1960988795289473024",
    "2019715677614510080",
    "2032325973063503872",
    "1284141750721056768"
  ]
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 是 |none|
|» rids|body|[string]| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|

### 返回数据结构

## POST 新增策略

POST /open_api/rm/v1/instruction_policy/add_policy

> Body 请求参数

```json
{
  "name": "侧是",
  "condition_list": {
    "sets": [
      {
        "sub_sets_logical": "AND",
        "sub_sets": [
          {
            "sub_sets_logical": "AND",
            "sub_sets": [
              {
                "access_rules": [
                  {
                    "key": "incident_name",
                    "value": "本地机器学习检测到\"Ransom.Win32.Wannacrypt.C\"木马事件",
                    "compare_method": "regex"
                  },
                  {
                    "key": "score",
                    "value": "10",
                    "compare_method": "greater_or_equal"
                  }
                ]
              }
            ]
          }
        ]
      }
    ]
  },
  "action": [
    2
  ],
  "scope": 1,
  "client_id": "",
  "group_ids": []
}
```

### 请求参数

|名称|位置|类型|必选|说明|
|---|---|---|---|---|
|body|body|object| 是 |none|
|» name|body|string| 否 |none|
|» condition_list|body|object| 否 |none|
|»» sets|body|[object]| 是 |none|
|»»» sub_sets_logical|body|string| 否 |none|
|»»» sub_sets|body|[object]| 否 |none|
|»»»» sub_sets_logical|body|string| 否 |none|
|»»»» sub_sets|body|[object]| 否 |none|
|»»»»» access_rules|body|[object]| 否 |none|
|»»»»»» key|body|string| 是 |none|
|»»»»»» value|body|string| 是 |none|
|»»»»»» compare_method|body|string| 是 |none|
|» action|body|[integer]| 是 |none|
|» scope|body|integer| 是 |none|
|» client_id|body|string| 是 |none|
|» group_ids|body|[string]| 是 |none|

> 返回示例

> 200 Response

```json
{}
```

### 返回结果

|状态码|状态码含义|说明|数据模型|
|---|---|---|---|
|200|[OK](https://tools.ietf.org/html/rfc7231#section-6.3.1)|none|Inline|