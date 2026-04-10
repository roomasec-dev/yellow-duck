# Open API 接口文档

> **文档版本**：v1.0
> **更新日期**：2026-04-09
> **Base URL**：`https://qax-openapi.zboundary.com`

---

## 目录

- [Open API 接口文档](#open-api-接口文档)
  - [目录](#目录)
  - [1. API使用介绍](#1-api使用介绍)
    - [1.1 步骤介绍](#11-步骤介绍)
    - [1.2 API校验信息获取](#12-api校验信息获取)
  - [2. 平台提供的API](#2-平台提供的api)
    - [2.1 获取Token](#21-获取token)
  - [2.2 策略管理](#22-策略管理)
    - [策略列表查询](#策略列表查询)
    - [新增策略](#新增策略)
    - [更新策略](#更新策略)
    - [更新策略状态](#更新策略状态)
    - [删除策略](#删除策略)
    - [策略排序](#策略排序)
  - [2.3 IOA 配置](#23-ioa-配置)
    - [IOA 白名单列表](#ioa-白名单列表)
    - [创建 IOA 白名单](#创建-ioa-白名单)
    - [更新 IOA 白名单](#更新-ioa-白名单)
    - [删除 IOA 白名单](#删除-ioa-白名单)
    - [IOA 加白活动审计](#ioa-加白活动审计)
  - [2.4 查杀设置](#24-查杀设置)
    - [获取查杀设置](#获取查杀设置)
    - [更新查杀设置](#更新查杀设置)
  - [2.5 终端管理](#25-终端管理)
    - [终端列表查询](#终端列表查询)
    - [终端加入黑名单](#终端加入黑名单)
    - [移除主机](#移除主机)
    - [离线终端管理查询](#离线终端管理查询)
    - [离线终端管理更新](#离线终端管理更新)
  - [2.6 查杀计划](#26-查杀计划)
    - [2.6 查杀计划列表](#26-查杀计划列表)
    - [新建查杀计划](#新建查杀计划)
    - [编辑查杀计划](#编辑查杀计划)
    - [取消查杀计划](#取消查杀计划)
    - [查看查杀任务](#查看查杀任务)
  - [2.7 病毒统计](#27-病毒统计)
    - [按主机统计](#按主机统计)
    - [按 Hash 统计](#按-hash-统计)
    - [Hash 关联主机列表](#hash-关联主机列表)
  - [2.8 任务下发](#28-任务下发)
    - [下发任务（单终端）](#下发任务单终端)
    - [批量下发任务](#批量下发任务)
  - [2.9 事件管理](#29-事件管理)
    - [事件列表](#事件列表)
    - [事件详情](#事件详情)
    - [事件状态批量更新](#事件状态批量更新)
    - [关联风险检出列表](#关联风险检出列表)
    - [进程树](#进程树)
  - [2.10 人工响应](#210-人工响应)
    - [人工响应任务列表](#人工响应任务列表)
  - [3. 通用响应说明](#3-通用响应说明)
    - [响应结构](#响应结构)
    - [错误码说明](#错误码说明)
    - [分页参数](#分页参数)
    - [时间格式](#时间格式)
    - [认证方式](#认证方式)
    - [常见问题](#常见问题)

---

## 1. API使用介绍

### 1.1 步骤介绍

奇安信SASE提供一系列开放API，支持企业开发者通过开放API将奇安信SASE的数据与内部其他系统打通，赋能企业安全体系建设。

开发者使用平台开放API时，可按以下步骤进行：

1. 进入奇安信SASE控制台
2. 从控制台中获取API校验信息：Application Key、Application SK
3. 通过Application Key和Application SK获取Token
4. 通过Token调用一系列业务接口

### 1.2 API校验信息获取

进入奇安信SASE控制台后，点击左下角的**配置中心/API凭证**，进入API凭证页面，选择**获取API凭证**按钮，获取Application Key和Application SK，用于在后面的步骤中获取Token。

---

## 2. 平台提供的API

### 2.1 获取Token

获取 API 访问令牌。

**请求 URL**

```
POST /open_api/rm/v1/get_open_api_token
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Content-Type | Header | String | 是 | application/json |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| sign | Body | String | 是 | 签名，规则：`md5(appkey + appsecret + time)` |
| time | Body | Integer | 是 | 当前时间戳（毫秒） |
| app_key | Body | String | 是 | 控制台获取的 Application Key |

**请求示例**

```json
{
  "sign": "d41d8cd98f00b204e9800998ecf8427e",
  "time": 1712659200000,
  "app_key": "your_app_key_here"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

**响应参数说明**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| error | Integer | 是 | 错误码，0 表示成功 |
| message | String | 是 | 错误信息 |
| data.token | String | 是 | 访问令牌 |

**使用说明**

获取Token后，在调用其他业务接口时，需要在请求头中携带Token：

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌，直接传递 token 值 |

---

## 2.2 策略管理

### 策略列表查询

查询策略列表。

**请求 URL**

```
POST /open_api/rm/v1/instruction_policy/list
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌，直接传递 token 值 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| policy_type | Body | Integer | 否 | 策略类型：1-内置策略，2-自定义策略 |
| name | Body | String | 否 | 策略名称 |
| operation_user | Body | String | 否 | 操作人 |
| scopes | Body | String | 否 | 范围：1-隔离网络，2-智能响应 |
| action | Body | Integer | 否 | 操作：1-隔离网络，2-智能响应 |
| status | Body | Integer | 否 | 状态：1-启用，2-禁用 |
| create_time | Body | Object | 否 | 创建时间范围 |
| create_time.time_range.start | Body | Integer | 是 | 开始时间戳 |
| create_time.time_range.end | Body | Integer | 是 | 结束时间戳 |
| update_time | Body | Object | 否 | 更新时间范围 |

**请求示例**

```json
{
  "policy_type": 1,
  "name": "测试策略",
  "operation_user": "admin",
  "scopes": "1",
  "action": 1,
  "status": 1,
  "create_time": {
    "time_range": {
      "start": 1712553600,
      "end": 1712640000
    }
  }
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "result": [
      {
        "rid": "1962463468078501888",
        "policy_type": 2,
        "name": "测试策略",
        "action": [2, 1],
        "client_id": "d0bf954479f84a1fa3e176ee079021d7",
        "scope": 2,
        "scope_content": "DESKTOP-I5L1OH7",
        "operation_user": "171****5221",
        "create_time": 1756722726,
        "update_time": 1769482142,
        "status": 1,
        "task_num": 0
      }
    ]
  }
}
```

**响应参数说明**

| 参数名 | 类型 | 说明 |
|--------|------|------|
| error | Integer | 错误码，0 表示成功 |
| message | String | 错误信息 |
| data.result[].rid | String | 策略ID |
| data.result[].policy_type | Integer | 策略类型 |
| data.result[].name | String | 策略名称 |
| data.result[].status | Integer | 状态：1-启用，2-禁用 |

---

### 新增策略

创建新的策略。

**请求 URL**

```
POST /open_api/rm/v1/instruction_policy/add_policy
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| name | Body | String | 是 | 策略名称 |
| condition_list | Body | Object | 是 | 条件列表 |
| condition_list.sets | Body | Array | 是 | 条件集合 |
| condition_list.version | Body | String | 是 | 版本号 |
| action | Body | Array | 否 | 执行动作：[1]-隔离网络，[2]-智能响应 |
| scope | Body | Integer | 是 | 范围：1-全网，2-指定终端，3-指定分组 |
| client_id | Body | String | 否 | 终端ID |
| group_ids | Body | Array | 否 | 分组ID列表 |

**请求示例**

```json
{
  "name": "新策略",
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
                    "value": "木马事件",
                    "compare_method": "regex"
                  }
                ]
              }
            ]
          }
        ]
      }
    ],
    "version": "1.0"
  },
  "action": [2],
  "scope": 1
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

### 更新策略

更新已有策略。

**请求 URL**

```
POST /open_api/rm/v1/instruction_policy/update
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| rid | Body | String | 是 | 策略ID |
| name | Body | String | 是 | 策略名称 |
| condition_list | Body | Object | 是 | 条件列表 |
| action | Body | Array | 是 | 执行动作 |
| scope | Body | Integer | 是 | 范围 |
| client_id | Body | String | 是 | 终端ID |
| group_ids | Body | Array | 是 | 分组ID列表 |
| policy_type | Body | Integer | 是 | 策略类型 |
| status | Body | Integer | 是 | 状态 |

**请求示例**

```json
{
  "name": "更新后的策略",
  "condition_list": {
    "sets": [...],
    "version": "1.0"
  },
  "action": [2, 1],
  "scope": 2,
  "client_id": "46148b9c343b41a2be0c54b630a4fd71",
  "group_ids": [],
  "rid": "1962463468078501888",
  "policy_type": 2,
  "status": 1
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

### 更新策略状态

更新策略的启用/禁用状态。

**请求 URL**

```
POST /open_api/rm/v1/instruction_policy/save_status
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| rid | Body | String | 是 | 策略ID |
| rids | Body | Array | 是 | 策略ID列表（批量操作） |

**请求示例**

```json
{
  "rid": "1962463468078501888",
  "rids": ["1998702940830830592", "1996146956669292544"]
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "name": "testmf"
  }
}
```

---

### 删除策略

删除指定策略。

**请求 URL**

```
POST /open_api/rm/v1/instruction_policy/delete
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| rid | Body | String | 是 | 策略ID |

**请求示例**

```json
{
  "rid": "1998702940830830592"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "rid": "1998702940830830592",
    "hit_continue": "0",
    "is_deleted": 0,
    "policy_type": 2,
    "name": "adfasdfasdf"
  }
}
```

---

### 策略排序

调整策略的执行顺序。

**请求 URL**

```
POST /open_api/rm/v1/instruction_policy/save_sort
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| rids | Body | Array | 是 | 策略ID列表，按排序顺序传入 |

**请求示例**

```json
{
  "rids": [
    "1996146956669292544",
    "1960987025683255296",
    "1960988795289473024"
  ]
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

## 2.3 IOA 配置

### IOA 白名单列表

查询 IOA 加白名单列表。

**请求 URL**

```
POST /open_api_server/rm/v1/configure/ioa/list
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| exclusion_name | Body | String | 否 | 加白名称 |
| t_name | Body | String | 否 | 战术名称 |
| operate_user | Body | String | 否 | 操作人 |
| file_name | Body | String | 否 | 文件名 |
| ioa_name | Body | String | 否 | IOA名称 |
| host_type | Body | String | 否 | 主机类型 |
| command_line | Body | String | 否 | 命令行 |
| ta_name | Body | String | 否 | 战术分类名称 |
| update_time | Body | Array | 是 | 更新时间范围 |
| group_ids | Body | Array | 否 | 分组ID |
| page | Body | Integer | 否 | 页码 |
| limit | Body | Integer | 否 | 每页数量 |

**请求示例**

```json
{
  "exclusion_name": "",
  "ioa_name": "",
  "update_time": ["2026-04-01 00:00:00", "2026-04-30 23:59:59"],
  "group_ids": [],
  "page": 1,
  "limit": 10
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "results": [
      {
        "exclusion_id": "4d4d17bc4c17436b9dca94a7ee013950",
        "ioa_id": "1691726446122242048",
        "ta_name": "凭据访问",
        "t_name": "系统凭据转储",
        "ioa_name": "疑似mimikatz攻击",
        "description": "攻击者可能会尝试转储凭据...",
        "file_name": "C:\\Users\\John\\Desktop\\mimikatz_x64.exe",
        "command_line": "mimikatz_x64.exe",
        "host_type": "ALL",
        "exclusion_name": "测试加白",
        "create_time": 1775137280,
        "update_time": 1775141347,
        "operate_user": "177****6010"
      }
    ],
    "total": 1
  }
}
```

---

### 创建 IOA 白名单

创建新的 IOA 加白规则。

**请求 URL**

```
POST /open_api/rm/v1/configure/ioa/add
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| ioa_name | Body | String | 是 | IOA名称 |
| exclusion_name | Body | String | 是 | 加白名称 |
| description | Body | String | 是 | 描述 |
| file_name | Body | String | 是 | 文件名（支持正则） |
| command_line | Body | String | 是 | 命令行（支持正则） |
| host_type | Body | String | 是 | 主机类型：ALL/WINDOWS/LINUX |
| group_ids | Body | Array | 是 | 分组ID列表 |
| ioa_id | Body | String | 是 | IOA规则ID |
| ta_id | Body | String | 是 | 战术分类ID |
| t_id | Body | String | 是 | 技术ID |

**请求示例**

```json
{
  "ioa_name": "疑似mimikatz攻击",
  "exclusion_name": "测试加白",
  "description": "攻击者可能会尝试转储凭据...",
  "file_name": "C:\\\\Users\\\\John\\\\Desktop\\\\mimikatz_x64\\.exe",
  "command_line": "mimikatz_x64\\.exe",
  "host_type": "ALL",
  "group_ids": [],
  "ioa_id": "1691726446122242048",
  "ta_id": "TA0006",
  "t_id": "T1003"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "exclusion_id": "4d4d17bc4c17436b9dca94a7ee013950",
    "ioa_id": "1691726446122242048",
    "ta_name": "凭据访问",
    "ioa_name": "疑似mimikatz攻击",
    "create_time": 1775137280
  }
}
```

---

### 更新 IOA 白名单

更新已有的 IOA 加白规则。

**请求 URL**

```
POST /open_api/rm/v1/configure/ioa/update
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| ioa_name | Body | String | 是 | IOA名称 |
| exclusion_name | Body | String | 是 | 加白名称 |
| description | Body | String | 是 | 描述 |
| file_name | Body | String | 是 | 文件名 |
| command_line | Body | String | 是 | 命令行 |
| host_type | Body | String | 是 | 主机类型 |
| group_ids | Body | Array | 是 | 分组ID列表 |
| ioa_id | Body | String | 是 | IOA规则ID |
| id | Body | String | 是 | 白名单记录ID |

**请求示例**

```json
{
  "ioa_name": "疑似mimikatz攻击",
  "exclusion_name": "更新后的名称",
  "description": "更新后的描述",
  "file_name": "C:\\\\Users\\\\John\\\\Desktop\\\\mimikatz_x64.exe",
  "command_line": "mimikatz_x64.exe",
  "host_type": "ALL",
  "group_ids": [],
  "ioa_id": "1691726446122242048",
  "id": "4d4d17bc4c17436b9dca94a7ee013950"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success"
}
```

---

### 删除 IOA 白名单

删除指定的 IOA 加白规则。

**请求 URL**

```
POST /open_api/rm/v1/configure/ioa/delete
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| id | Body | String | 是 | 白名单记录ID |

**请求示例**

```json
{
  "id": "4d4d17bc4c17436b9dca94a7ee013950"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

### IOA 加白活动审计

查询 IOA 加白后的活动审计记录。

**请求 URL**

```
POST /open_api/rm/v1/configure/ioa/audit_log
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| event_time | Body | Object | 否 | 事件时间范围 |
| event_time.time_range.start | Body | Integer | 是 | 开始时间戳 |
| event_time.time_range.end | Body | Integer | 是 | 结束时间戳 |
| host_name | Body | String | 否 | 主机名 |
| ioa_name | Body | String | 否 | IOA名称 |
| file_name | Body | String | 否 | 文件名 |
| command_line | Body | String | 否 | 命令行 |
| page | Body | Integer | 否 | 页码 |
| limit | Body | Integer | 否 | 每页数量 |

**请求示例**

```json
{
  "event_time": {
    "time_range": {
      "start": 1712553600,
      "end": 1712640000
    }
  },
  "host_name": "",
  "ioa_name": "",
  "page": 1,
  "limit": 10
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {}
}
```

---

## 2.4 查杀设置

### 获取查杀设置

获取病毒查杀全局策略设置。

**请求 URL**

```
GET /open_api/rm/v1/strategy/virus_scan_settings/single
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "name": "virus_scan_settings 全局策略",
    "type": "virus_scan_settings",
    "content": "virus_scan_settings 全局策略",
    "status": 1,
    "is_default": "normal",
    "strategy_id": "69ce8354aff0f2be4eb260e5",
    "version_id": "69ce8354aff0f2be4eb260e6",
    "range_type": 1,
    "group_ids": [],
    "config_content": "{\"scan_file_scope\":\"recommended\",\"startup_scan_mode\":\"all_unknown\"}",
    "create_time": 1775141716,
    "last_update_time": 0,
    "operator_id": "92",
    "operator_name": "177****6010"
  }
}
```

**config_content 字段说明**

| 字段名 | 类型 | 说明 |
|--------|------|------|
| scan_file_scope | String | 扫描文件范围：recommended-推荐，all-全部 |
| startup_scan_mode | String | 开机扫描模式：all_unknown-全部未知，known_dangerous-已知危险 |
| archive_size_limit_enabled | Boolean | 是否启用压缩包大小限制 |
| archive_size_limit | Integer | 压缩包大小限制（MB） |
| realtime_scan_archive_enabled | Boolean | 是否启用实时扫描压缩包 |

---

### 更新查杀设置

更新病毒查杀策略设置。

**请求 URL**

```
POST /open_api/rm/v1/strategy/update
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| name | Body | String | 是 | 策略名称 |
| type | Body | String | 是 | 策略类型 |
| content | Body | String | 是 | 策略内容描述 |
| status | Body | Integer | 是 | 状态：1-启用，2-禁用 |
| is_default | Body | String | 是 | 是否默认：normal-普通 |
| strategy_id | Body | String | 是 | 策略ID |
| version_id | Body | String | 是 | 版本ID |
| range_type | Body | Integer | 是 | 范围类型：1-全网 |
| group_ids | Body | Array | 是 | 分组ID列表 |
| config_content | Body | String | 是 | 配置内容（JSON字符串） |
| exclude_objects | Body | Object | 是 | 排除对象 |

**请求示例**

```json
{
  "name": "virus_scan_settings 全局策略",
  "type": "virus_scan_settings",
  "content": "virus_scan_settings 全局策略",
  "status": 1,
  "is_default": "normal",
  "strategy_id": "69ce8354aff0f2be4eb260e5",
  "version_id": "69ce8354aff0f2be4eb260e6",
  "range_type": 1,
  "group_ids": [],
  "config_content": "{\"scan_file_scope\":\"recommended\",\"startup_scan_mode\":\"known_dangerous\"}",
  "exclude_objects": {
    "users": [],
    "user_groups": [],
    "host_groups": [],
    "client_ids": [],
    "departs": []
  }
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "name": "virus_scan_settings 全局策略",
    "version_id": "69ce8654aff0f2be4eb260ea",
    "status": 1
  }
}
```

---

## 2.5 终端管理

### 终端列表查询

查询终端列表。

**请求 URL**

```
POST /open_api/rm/v1/hosts/globalization/list
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| page | Body | Integer | 是 | 页码 |
| limit | Body | Integer | 是 | 每页数量 |
| hostname | Body | String | 否 | 主机名 |
| client_ids | Body | Array | 否 | 终端ID列表 |
| client_version | Body | String | 否 | 客户端版本 |
| status | Body | String | 否 | 在线状态：online-在线，offline-离线 |
| importance | Body | Integer | 否 | 重要程度：1-高，2-中，3-低 |
| win_version | Body | String | 否 | Windows版本 |
| system_type | Body | Integer | 否 | 系统类型：1-Windows，2-Linux |
| client_ip | Body | String | 否 | 终端IP |
| mac_address | Body | String | 否 | MAC地址 |
| username | Body | String | 否 | 用户名 |
| last_logon_time | Body | Object | 否 | 最近登录时间范围 |
| last_seen_time | Body | Object | 否 | 最后活跃时间范围 |

**请求示例**

```json
{
  "page": 1,
  "limit": 10,
  "hostname": "",
  "status": "online",
  "importance": 1
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {}
}
```

---

### 终端加入黑名单

将指定终端加入黑名单。

**请求 URL**

```
POST /open_api/rm/v1/hosts/add_blacklist
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| client_ids | Body | Array | 是 | 终端ID列表 |
| reason | Body | String | 是 | 加入黑名单原因 |

**请求示例**

```json
{
  "client_ids": ["0772a7a958c349e6a5266363c2d4d3c6"],
  "reason": "异常行为"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

### 移除主机

从管理中移除指定主机。

**请求 URL**

```
POST /open_api/rm/v1/hosts/remove_host
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| client_ids | Body | Array | 是 | 终端ID列表 |

**请求示例**

```json
{
  "client_ids": ["4fcd2d2073104b66adfc2e8e5869a5d4"]
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

### 离线终端管理查询

查询离线终端管理设置。

**请求 URL**

```
GET /open_api/rm/v1/client_setting/host_offline
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "id": "69ce716c20f4277efc0ee2bc",
    "org_name": "AA0000040900000302",
    "status": 1,
    "type": "host_offline",
    "setting": {
      "offline_days": 180
    },
    "create_time": 1775137132,
    "update_time": 1775137132
  }
}
```

**响应参数说明**

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.status | Integer | 状态：1-开启，2-关闭 |
| data.setting.offline_days | Integer | 离线天数阈值 |

---

### 离线终端管理更新

更新离线终端管理设置。

**请求 URL**

```
POST /open_api/rm/v1/client_setting/host_offline
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| status | Body | Integer | 是 | 状态：1-开启，2-关闭 |
| setting.offline_days | Body | Integer | 是 | 离线天数阈值 |

**请求示例**

```json
{
  "status": 1,
  "setting": {
    "offline_days": 15
  }
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "id": "69ce716c20f4277efc0ee2bc",
    "status": 1,
    "setting": {
      "offline_days": 15
    }
  }
}
```

---

## 2.6 查杀计划

### 2.6 查杀计划列表

查询查杀计划列表。

**请求 URL**

```
POST /open_api/rm/v1/plan/list
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| type | Body | String | 是 | 计划类型：`kill_plan`-查杀计划 |
| page | Body | Integer | 是 | 页码 |
| limit | Body | Integer | 是 | 每页数量 |

**请求示例**

```json
{
  "type": "kill_plan",
  "page": 1,
  "limit": 10
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "total": 17,
    "items": [
      {
        "rid": "2036374019992719360",
        "org_name": "AA0000040900000302",
        "plan_name": "0324",
        "plan_type": 1,
        "execute_cycle": 1,
        "scope": 1,
        "scope_content": "",
        "contents": "{\"scan_path\":[\"C:\\\\Users\\\\John\\\\Desktop\"],\"scan_type\":3}",
        "scan_type": 3,
        "create_time": 1774344375,
        "update_time": 1774540800,
        "operation_user": "admin",
        "status": 4,
        "type": "kill_plan"
      }
    ]
  }
}
```

**响应参数说明**

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.total | Integer | 总数 |
| data.items[].rid | String | 计划ID |
| data.items[].plan_name | String | 计划名称 |
| data.items[].plan_type | Integer | 计划类型 |
| data.items[].execute_cycle | Integer | 执行周期 |
| data.items[].status | Integer | 状态：1-待执行，2-执行中，4-已完成 |
| data.items[].scan_type | Integer | 扫描类型：1-快速扫描，2-全盘扫描，3-自定义扫描 |

---

### 新建查杀计划

创建新的查杀计划。

**请求 URL**

```
POST /open_api/rm/v1/plan/add
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| plan_name | Body | String | 是 | 计划名称 |
| plan_type | Body | Integer | 是 | 计划类型：1-立即执行，2-定时执行，3-周期执行 |
| scope | Body | Integer | 是 | 范围：1-全网，2-指定终端，3-指定分组 |
| scan_type | Body | Integer | 是 | 扫描类型：1-快速扫描，2-全盘扫描，3-自定义扫描 |
| execute_cycle | Body | Integer | 是 | 执行周期 |
| expired_setting | Body | Integer | 是 | 过期设置：0-不过期，1-过期 |
| expired_time | Body | Integer | 否 | 过期时间（时间戳） |
| type | Body | String | 是 | 类型：`kill_plan` |
| contents | Body | Object | 是 | 计划内容 |
| contents.scan_type | Body | Integer | 是 | 扫描类型 |
| contents.scan_path | Body | Array | 否 | 扫描路径 |
| contents.isolation_setting | Body | Array | 否 | 隔离设置 |

**请求示例**

```json
{
  "plan_name": "周常查杀",
  "plan_type": 1,
  "scope": 3,
  "scan_type": 1,
  "execute_cycle": 1,
  "expired_setting": 1,
  "expired_time": 1775318400,
  "type": "kill_plan",
  "contents": {
    "scan_type": 1,
    "scan_path": ["C:\\Users"],
    "isolation_setting": ["auto_isolation"]
  }
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

### 编辑查杀计划

编辑已有的查杀计划。

**请求 URL**

```
POST /open_api/rm/v1/plan/edit
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| plan_name | Body | String | 是 | 计划名称 |
| plan_type | Body | Integer | 是 | 计划类型 |
| scope | Body | Integer | 是 | 范围 |
| scan_type | Body | Integer | 是 | 扫描类型 |
| execute_cycle | Body | Integer | 是 | 执行周期 |
| repeat_cycle | Body | Array | 是 | 重复周期 |
| execution_time | Body | String | 是 | 执行时间 |
| expired_setting | Body | Integer | 是 | 过期设置 |
| device_client_ids | Body | Array | 是 | 终端ID列表 |
| group_ids | Body | Array | 是 | 分组ID列表 |
| type | Body | String | 是 | 类型 |
| contents | Body | Object | 是 | 计划内容 |
| rid | Body | String | 是 | 计划ID |

**请求示例**

```json
{
  "plan_name": "更新后的计划",
  "plan_type": 3,
  "scope": 3,
  "scan_type": 1,
  "execute_cycle": 1,
  "repeat_cycle": [],
  "execution_time": "07:04",
  "expired_setting": 0,
  "device_client_ids": [],
  "group_ids": [],
  "type": "kill_plan",
  "contents": {
    "scan_type": 1,
    "scan_path": [],
    "isolation_setting": ["auto_isolation"]
  },
  "rid": "2039728820453380096"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

### 取消查杀计划

取消指定的查杀计划。

**请求 URL**

```
PUT /open_api/rm/v1/plan/cancel/{id}
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**路径参数**

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | String | 是 | 计划ID |

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

### 查看查杀任务

查看查杀任务的执行记录。

**请求 URL**

```
POST /open_api/rm/v1/virus_scan/scan_record
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| page | Body | Integer | 是 | 页码 |
| limit | Body | Integer | 是 | 每页数量 |
| rid | Body | String | 否 | 计划ID |
| host_name | Body | String | 否 | 主机名 |
| scan_type | Body | String | 否 | 扫描类型 |
| status | Body | String | 否 | 任务状态 |
| task_id | Body | String | 否 | 任务ID |
| start_time | Body | Object | 否 | 开始时间范围 |
| end_time | Body | Object | 否 | 结束时间范围 |

**请求示例**

```json
{
  "page": 1,
  "limit": 10,
  "rid": "2039728820453380096"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {}
}
```

---

## 2.7 病毒统计

### 按主机统计

按主机维度查询病毒统计信息。

**请求 URL**

```
POST /open_api/rm/v1/virus/host/list
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| page | Body | Integer | 是 | 页码 |
| limit | Body | Integer | 是 | 每页数量 |
| status | Body | Integer | 否 | 状态 |
| importance | Body | Integer | 否 | 重要程度 |
| last_checked_time | Body | Object | 否 | 最后查杀时间范围 |
| client_ip | Body | String | 否 | 终端IP |
| host_name | Body | String | 否 | 主机名 |
| client_id | Body | String | 否 | 终端ID |
| mac_address | Body | String | 否 | MAC地址 |

**请求示例**

```json
{
  "page": 1,
  "limit": 10,
  "status": 0,
  "importance": 1,
  "host_name": "WIN10"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "total": 3,
    "results": [
      {
        "host_name": "WINDOWS11",
        "client_id": "f4fa9b70c5f24aac9260cd2ed9bbf195",
        "status": 2,
        "username": "John",
        "importance": 2,
        "client_ip": "192.168.111.75",
        "mac_address": "00:0C:29:32:A8:BF",
        "virus_file_count": 7,
        "virus_memory_count": 0,
        "last_checked_time": 1774349304,
        "host_status": "uninstall"
      }
    ]
  }
}
```

**响应参数说明**

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.total | Integer | 总数 |
| data.results[].host_name | String | 主机名 |
| data.results[].client_id | String | 终端ID |
| data.results[].status | Integer | 状态 |
| data.results[].virus_file_count | Integer | 文件病毒数 |
| data.results[].virus_memory_count | Integer | 内存病毒数 |
| data.results[].last_checked_time | Integer | 最后查杀时间 |

---

### 按 Hash 统计

按文件 Hash 查询病毒统计。

**请求 URL**

```
POST /open_api/rm/v1/virus/hash/list
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| page | Body | Integer | 是 | 页码 |
| limit | Body | Integer | 是 | 每页数量 |
| sha1 | Body | String | 否 | SHA1 值 |
| md5 | Body | String | 否 | MD5 值 |
| name | Body | String | 否 | 病毒名称 |
| end_time | Body | Object | 否 | 结束时间范围 |

**请求示例**

```json
{
  "page": 1,
  "limit": 10,
  "sha1": "",
  "md5": "",
  "name": "WannaCry"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {}
}
```

---

### Hash 关联主机列表

查看指定 Hash 关联的主机列表。

**请求 URL**

```
POST /open_api/rm/v1/virus/hash/host_list
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| page | Body | Integer | 是 | 页码 |
| limit | Body | Integer | 是 | 每页数量 |
| sha1 | Body | String | 是 | SHA1 值 |
| md5 | Body | String | 否 | MD5 值 |

**请求示例**

```json
{
  "page": 1,
  "limit": 10,
  "sha1": "970A81F93F588D24FE2603EC8F760D5F0E52261A"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {}
}
```

---

## 2.8 任务下发

### 下发任务（单终端）

向单个终端下发指令任务。

**请求 URL**

```
POST /open_api/rm/v1/instructions/send_instruction
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| instruction_name | Body | String | 是 | 指令名称：`get_suspicious_file`-获取可疑文件 |
| batch_params | Body | Array | 是 | 批处理参数 |
| batch_params[].id | Body | String | 否 | 文件ID |
| batch_params[].path | Body | String | 否 | 文件路径 |
| batch_params[].sha1 | Body | String | 否 | 文件SHA1 |
| batch_params[].pid | Body | Integer | 否 | 进程ID |
| is_batch | Body | Integer | 是 | 是否批量：1-是 |
| client_id | Body | String | 是 | 终端ID |
| verify_code | Body | String | 否 | 验证码 |

**请求示例**

```json
{
  "instruction_name": "get_suspicious_file",
  "batch_params": [
    {
      "id": "26b25ad1-a6a4-4f8d-aece-399cd43fe23a",
      "path": "C:\\\\cmd.exe",
      "sha1": ""
    }
  ],
  "is_batch": 1,
  "client_id": "4909e4b109454198994a90cf030603b9",
  "verify_code": "454634"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

### 批量下发任务

向多个终端批量下发指令任务。

**请求 URL**

```
POST /open_api/rm/v1/instructions/batch_send_instruction
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| client_ids | Body | Array | 是 | 终端ID列表 |
| instruction_name | Body | String | 否 | 指令名称 |
| batch_params | Body | Array | 否 | 批处理参数 |
| is_batch | Body | Integer | 是 | 是否批量：1 |

**请求示例**

```json
{
  "client_ids": ["4909e4b109454198994a90cf030603b9"],
  "instruction_name": "batch_quarantine_file",
  "batch_params": [
    {
      "id": "1e91a9e1-5ee3-4c0a-9635-bfea59dacb33",
      "path": "C:\\\\d.exe",
      "sha1": ""
    }
  ],
  "is_batch": 1
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "failed": null,
    "success": ["4909e4b109454198994a90cf030603b9"]
  }
}
```

---

## 2.9 事件管理

### 事件列表

查询安全事件列表。

**请求 URL**

```
POST /open_api/rm/v1/incidents/list
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| page | Body | Integer | 是 | 页码 |
| limit | Body | Integer | 是 | 每页数量 |
| client_id | Body | String | 否 | 终端ID |
| hostname | Body | String | 否 | 主机名 |
| order | Body | String | 是 | 排序方式 |
| incident_name | Body | String | 否 | 事件名称 |
| score | Body | String | 否 | 威胁分数 |
| status | Body | String | 否 | 处置状态 |
| start_time | Body | Object | 否 | 开始时间范围 |

**order 参数值说明**

| 值 | 说明 |
|-----|------|
| start_time_desc | 开始时间倒序 |
| end_time_desc | 结束时间倒序 |
| score_asc | 分数升序 |
| score_desc | 分数降序 |

**请求示例**

```json
{
  "page": 1,
  "limit": 10,
  "client_id": "",
  "hostname": "",
  "order": "score_desc",
  "status": "2",
  "start_time": {
    "time_range": {
      "start": 1712553600,
      "end": 1712640000
    }
  }
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "incident_list": [
      {
        "id": "698c69d561e091238897087d",
        "incident_id": "ff55ff8aa2ae484eba0e6560d4e3d88e-20260211191013",
        "incident_name": "qax-PC-20260211191013",
        "score": 0,
        "client_id": "ff55ff8aa2ae484eba0e6560d4e3d88e",
        "start_time": 1770808213,
        "end_time": 1770897210,
        "status": 1,
        "host_name": "qax-PC",
        "host_status": "uninstall",
        "ioa_data": [
          {
            "threat_level": 2,
            "count": 605
          }
        ],
        "operating_system": "UOS 20.1070.11018",
        "external_ip": "192.168.111.1",
        "platform": 4
      }
    ]
  }
}
```

**响应参数说明**

| 参数名 | 类型 | 说明 |
|--------|------|------|
| data.incident_list[].incident_id | String | 事件ID |
| data.incident_list[].incident_name | String | 事件名称 |
| data.incident_list[].score | Integer | 威胁分数 |
| data.incident_list[].status | Integer | 状态：1-未处置，2-处置中，3-已处置，4-误报 |
| data.incident_list[].host_name | String | 主机名 |
| data.incident_list[].ioa_data[].threat_level | Integer | 威胁级别 |

---

### 事件详情

查询事件的详细信息。

**请求 URL**

```
POST /open_api/rm/v1/incident/view
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| incident_id | Body | String | 是 | 事件ID |
| client_id | Body | String | 是 | 终端ID |

**请求示例**

```json
{
  "incident_id": "ff55ff8aa2ae484eba0e6560d4e3d88e-20260211191013",
  "client_id": "ff55ff8aa2ae484eba0e6560d4e3d88e"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "client_id": "ff55ff8aa2ae484eba0e6560d4e3d88e",
    "host_info": {
      "client_id": "ff55ff8aa2ae484eba0e6560d4e3d88e",
      "client_ip": "192.168.111.1",
      "hostname": "qax-PC",
      "mac_address": "00:0C:29:85:1C:D2",
      "org_name": "AA0000030400000278",
      "os_version": "UOS 20.1070.11018",
      "username": "root"
    },
    "incident_name": "qax-PC-20260211191013",
    "ioa_timeline": [
      {
        "event_time_int": 1770808199,
        "ioa_id": "2001576052505186304",
        "name": "进程创建",
        "process_name": "bash",
        "threat_level": "2"
      }
    ]
  }
}
```

---

### 事件状态批量更新

批量更新事件处置状态。

**请求 URL**

```
POST /open_api/rm/v1/incident/batch_deal
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| ids | Body | Array | 是 | 事件ID列表 |
| scene | Body | String | 是 | 场景：`alone`-单独处理 |
| status | Body | Integer | 是 | 状态：1-未处置，2-处置中，3-已处置，4-误报反馈 |
| comment | Body | String | 否 | 备注 |
| allow | Body | Boolean | 是 | 是否同步更新所有风险检出状态 |

**请求示例**

```json
{
  "ids": ["ff55ff8aa2ae484eba0e6560d4e3d88e-20260211191013"],
  "scene": "alone",
  "status": 3,
  "comment": "已确认为正常操作",
  "allow": false
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": null
}
```

---

### 关联风险检出列表

查询事件关联的风险检出列表。

**请求 URL**

```
POST /open_api/rm/v1/detection/get_list
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 否 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| page | Body | Integer | 是 | 页码 |
| limit | Body | Integer | 是 | 每页数量 |
| incident_id | Body | String | 是 | 事件ID |

**请求示例**

```json
{
  "page": 1,
  "limit": 10,
  "incident_id": "f46466b589e34ebfb4de81d321cc324e-20260205172620"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "total": 25,
    "results": [
      {
        "detection_id": "f46466b589e34ebfb4de81d321cc324e-96ead0ce-20260205172620",
        "client_id": "f46466b589e34ebfb4de81d321cc324e",
        "view_type": "process",
        "threat_level": "2",
        "hostname": "dev1-PC",
        "deal_status": 1,
        "start_time": 1770287727,
        "main_ioa": {
          "ioa_id": "2001576052505186304",
          "p_name": "cron",
          "p_command_line": "/usr/sbin/cron -f",
          "threat_level": 2
        }
      }
    ]
  }
}
```

---

### 进程树

获取检测事件的进程树信息。

**请求 URL**

```
POST /open_api/rm/v1/detection/view
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 是 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| detection_id | Body | String | 是 | 风险检出ID |
| client_id | Body | String | 是 | 终端ID |
| view_type | Body | String | 是 | 视图类型：`process`-进程 |
| process_uuid | Body | String | 是 | 进程UUID |

**请求示例**

```json
{
  "detection_id": "8be90dbce4394ed38d19c62008db8ab4-{c51e1a53-324f-11f1-9a48-cc6b1e18ec4e}-20260407153444",
  "client_id": "8be90dbce4394ed38d19c62008db8ab4",
  "view_type": "process",
  "process_uuid": "{c51e1aa5-324f-11f1-9a48-cc6b1e18ec4e}"
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {
    "client_id": "8be90dbce4394ed38d19c62008db8ab4",
    "host_info": {
      "client_ip": "192.168.111.172",
      "hostname": "WIN10-CY02",
      "mac_address": "00:0C:29:AD:1C:5E",
      "org_name": "AA0000030400000278"
    },
    "processes": [
      {
        "process_uuid": "{c51e0e73-324f-11f1-9a48-806e6f6e6963}",
        "process_id": "732",
        "process_name": "winlogon.exe",
        "process_path": "C:\\Windows\\System32\\winlogon.exe",
        "process_md5": "EE86712DDF0C59E6921D548B5548FF9C",
        "command_line": "winlogon.exe",
        "process_create_time": 1775545369,
        "parent_uuid": "{c51e0e6c-324f-11f1-9a48-806e6f6e6963}"
      }
    ]
  }
}
```

---

## 2.10 人工响应

### 人工响应任务列表

查询人工响应任务列表。

**请求 URL**

```
POST /open_api/rm/v1/instructions/tasks
```

**请求头**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| Authorization | Header | String | 否 | 访问令牌 |

**请求参数**

| 参数名 | 位置 | 类型 | 必填 | 说明 |
|--------|------|------|------|------|
| page | Body | Integer | 是 | 页码 |
| limit | Body | Integer | 是 | 每页数量 |
| instruction_type | Body | Integer | 是 | 指令类型 |
| id | Body | String | 否 | 任务ID |
| host_name | Body | String | 否 | 主机名 |
| client_id | Body | String | 否 | 终端ID |
| instruction_name | Body | String | 否 | 指令名称 |
| create_time | Body | Object | 否 | 创建时间范围 |
| status | Body | String | 否 | 任务状态 |
| user | Body | String | 否 | 操作人 |
| content | Body | String | 否 | 任务内容 |
| update_time | Body | Object | 否 | 更新时间范围 |

**请求示例**

```json
{
  "page": 1,
  "limit": 10,
  "instruction_type": 0,
  "status": "0",
  "create_time": {
    "time_range": {
      "start": 1712553600,
      "end": 1712640000
    }
  }
}
```

**响应示例**

```json
{
  "error": 0,
  "message": "success",
  "data": {}
}
```

---

## 3. 通用响应说明

### 响应结构

所有接口均返回以下统一格式：

```json
{
  "error": 0,
  "message": "success",
  "data": {}
}
```

### 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 认证失败 |
| 1003 | 权限不足 |
| 1004 | 资源不存在 |
| 1005 | 操作失败 |
| 5000 | 服务器内部错误 |

### 分页参数

列表类接口通常包含分页参数：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | Integer | 是 | 页码，从1开始 |
| limit | Integer | 是 | 每页数量，最大100 |

返回结果中的分页信息：

```json
{
  "data": {
    "total": 100,
    "results": [...]
  }
}
```

### 时间格式

- 时间戳：Unix 时间戳（秒），如 `1712553600`
- 时间范围：包含 `start` 和 `end` 两个时间戳

### 认证方式

除获取Token接口外，其他所有接口都需要在请求头中携带访问令牌：

```
Authorization: {token}
```

---

### 常见问题

**1. 认证失败 (error: 1002)**
- 检查 token 是否过期
- 检查 token 格式是否正确
- 确认 app_key 和 app_secret 是否有效

**2. 参数错误 (error: 1001)**
- 检查必填参数是否完整
- 检查参数类型是否正确
- 检查时间戳格式是否正确

**3. 权限不足 (error: 1003)**
- 确认账户是否有权限访问该接口
- 确认操作对象是否在授权范围内

---