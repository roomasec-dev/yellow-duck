# Open API 接口文档

本文档根据 open_api 服务与 ratp-console 服务的路由映射关系整理，包含详细的 Request 和 Response 结构。

**基础 URL**: `http://{host}:{port}/open_api/rm/v1/`

---

## 1. 主机管理 (Hosts)

### 1.1 获取全球化主机列表
**接口地址**: `POST /open_api/rm/v1/hosts/globalization/list`
**描述**: 过滤查询的时候使用gids 查询不同的主机组的数据。当且仅当是从dashboard的威胁 钓鱼等主机下卡片的地方跳转的，type必须带 其他正常搜索的方案 不带参数type。

**Request Body**:
```json
{
  "business_type": 0,          // 1过滤已卸载的主机 2 过滤正在扫描和已卸载的的主机
  "client_id": "string",       // client_id
  "client_ids": ["string"],    // client_id 列表
  "client_ip": "string",       // external ip
  "first_seen_time": {         // 开始检查时间
    "quick_time": {
      "time_num": 0,
      "time_span": "string",
      "time_type": "string"
    }
  },
  "gids": ["string"],          // 组id列表
  "hostname": "string",        // 主机名
  "importance": 0,             // 重要性 1 重要 2普通
  "ip_address": "string",      // ip_address
  "is_export": 0,              // 是否导出 1 是 0否
  "isolate": false,            // true隔离主机 fasle 未隔离主机
  "last_logon_time": {         // 最后登录时间
    "quick_time": {
      "time_num": 0,
      "time_span": "string",
      "time_type": "string"
    }
  },
  "last_seen_time": {          // 最后检查时间
    "quick_time": {
      "time_num": 0,
      "time_span": "string",
      "time_type": "string"
    }
  },
  "limit": 0,                  // 分页限制
  "mac_address": "string",     // mac_address
  "orgconnectip": "string",    // (Required) 连接ip
  "os_type": 0,                // 1 Workstation, 2Sever
  "page": 0,                   // 页码
  "platform": "string",        // 操作系统来源1:Windows; 3:Mac
  "remarks": "string",         // 备注
  "rmconnectip": "string",     // rmconnectip
  "type": 0,                   // 1 :威胁主机 2:钓鱼主机 3:勒索主机
  "win_version": "string"      // windows 版本
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "total": 0,
    "pages": 0,
    "hosts": [
      {
        "id": "string",                  // (Required) id
        "client_id": "string",           // (Required) client_id
        "hostname": "string",            // (Required) 主机名
        "client_ip": "string",           // (Required) external ip
        "orgconnectip": "string",        // (Required) connectip
        "rmconnectip": "string",         // (Required)
        "mac_address": "string",         // (Required) MAC 地址
        "os_type": 0,                    // (Required) 0 空 ， 1 Workstation  大于1 Sever
        "os_version": "string",          // (Required) 客户端版本
        "win_version": "string",         // (Required) win 版本
        "platform": 0,                   // (Required) 来源 0:NotFound; 1:Windows; 2:Linux; 3:MacOS; 4:iOS; 5:Android; 6:HarmonyOS; 7:UOS; 8:XiaomiHyperOS; 9:OriginOS; 10:BlueOS; 11:ColorOS
        "status": "string",              // (Required) 状态
        "keep_alive_status": 0,          // (Required) 是否为正在主机状态 1是0 否
        "importance": 0,                 // (Required) 重要性 1重要 2普通
        "create_time": 0,                // (Required) 创建时间
        "last_active": 0,                // (Required) 最后活跃时间
        "lastlogontime": 0,              // (Required) 最后登录时间
        "username": "string",            // (Required) 用户名
        "group_name": ["string"],        // (Required) 主机列表(主机组名列表)
        "client_version": "string",      // (Required) 安装戎马客户端版本
        "uninstall_task": "string",      // (Required) 是否存在执行中的卸载任务
        "remarks": "string"              // (Required) 备注
      }
    ]
  }
}
```

---

## 2. 文件隔离管理 (Isolate File)

### 2.1 获取隔离文件列表
**接口地址**: `POST /open_api/rm/v1/isolate_file/get_list`

**Request Body**:
```json
{
  "client_id": "string",       // client_id
  "create_time": {             // 创建时间 (Quick Time / Time Range)
    "quick_time": {
      "time_num": 0,
      "time_span": "string",
      "time_type": "string"
    },
    "time_range": {
      "start": 0,
      "end": 0
    }
  },
  "file_name": "string",       // 文件名
  "hostname": "string",        // 主机名
  "limit": 0,                  // (Required) 每页条数
  "md5": "string",             // md5
  "page": 0,                   // (Required) 当前页
  "path": "string",            // 路径
  "recover_status": "string",  // 隔离状态 1.已隔离，2.已释放 3.已清除 (多个逗号分隔)
  "sha1": "string",            // sha1
  "task_id": "string",         // 任务id
  "username": "string"         // 主机用户名
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "total": 0,
    "results": [
      {
        "client_id": "string",           // client_id
        "create_time": 0,                // 创建时间
        "file_name": "string",           // 文件名
        "guid": "string",                // guid (唯一)
        "hostname": "string",            // 主机名
        "md5": "string",                 // md5
        "org_name": "string",            // orgName
        "recover_status": 0,             // 隔离状态 1.已隔离，2已释放 3..已清除
        "remediation_status": 0,         // 响应状态 0.等待终端获取 1.响应执行中 2.响应成功 3.响应失败 4.已过期 5. 部分响应成功
        "sha1": "string",                // sha1
        "show_action": "string"          // 操作按钮展示：0 不展示 1.释放和删除按钮 2.重新下发按钮
      }
    ]
  }
}
```

### 2.2 删除隔离文件记录
**接口地址**: `POST /open_api/rm/v1/isolate_file/delete`

**Request Body**:
```json
{
  "guids": ["string"]  // guid数组
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": null
}
```

### 2.3 释放隔离文件
**接口地址**: `POST /open_api/rm/v1/isolate_file/release`

**Request Body**:
```json
{
  "guids": ["string"],      // guid数组
  "is_add_exclusion": true, // 是否加白
  "relase_all_hash": false  // 否释放所有hash true：是 false：默认否
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": null
}
```

---

## 3. IOC 配置管理 (Configure - IOC)

### 3.1 获取 IOC 列表
**接口地址**: `POST /open_api/rm/v1/configure/ioc/list`

**Request Body**:
```json
{
  "action": "string",          // 动作 Allow / Detect and prevent
  "date_add": {                // hash添加时间
    "quick_time": {
      "time_num": 0,
      "time_span": "string",
      "time_type": "string"
    },
    "time_range": {
      "start": 0,
      "end": 0
    }
  },
  "group_ids": [0],            // 组IDs
  "hash": "string",            // hash
  "host_type": "string",       // 应用范围 ALL / GROUP
  "last_modified": {           // 最近更新时间
    "quick_time": {
      "time_num": 0,
      "time_span": "string",
      "time_type": "string"
    },
    "time_range": {
      "start": 0,
      "end": 0
    }
  },
  "limit": 0,                  // 页面数量
  "page": 0                    // 页码
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "total": 0,
    "results": [
      {
        "action": "string",              // 行为
        "date_added": "string",          // 创建时间
        "description": "string",         // 描述
        "detection_count": "string",     // 命中的告警的数量
        "exclusion_id": "string",        // 【新增】
        "expiration_date": "string",     // 过期时间
        "file_name": "string",           // 文件名
        "group_ids": [0],                // 组IDs
        "hash": "string",                // hash 值
        "host_type": "string",           // host类型
        "ioc_id": "string",              // 主键id
        "last_modified": "string",       // 最近更新的时间
        "last_seen": "string"            // 最近激活时间
      }
    ]
  }
}
```

### 3.2 添加 IOC
**接口地址**: `POST /open_api/rm/v1/configure/ioc/add`

**Request Body**:
```json
{
  "action": "string",          // 动作【Allow，Detect and prevent】
  "description": "string",     // 描述
  "expiration_date": "string", // 过期时间
  "file_name": "string",       // 文件名
  "group_ids": [0],            // 组IDs
  "hash": "string",            // hash值
  "host_type": "string"        // host生效类型 （目前传：all）主机组传GROUP
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string"
}
```

### 3.3 更新 IOC
**接口地址**: `POST /open_api/rm/v1/configure/ioc/update`

**Request Body**:
```json
{
  "id": "string",              // (Required) 主键id
  "action": "string",          // 动作
  "description": "string",     // 描述
  "expiration_date": "string", // 超时时间
  "group_ids": [0],            // 主机组的ids的列表
  "hash": "string",            // hash值
  "host_type": "string"        // host 类型 （目前为All）GROUP：主机组
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string"
}
```

### 3.4 删除 IOC
**接口地址**: `POST /open_api/rm/v1/configure/ioc/delete`

**Request Body**:
```json
{
  "id": "string"               // (Required) 主键id
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string"
}
```

### 3.5 IOC 详情
**接口地址**: `POST /open_api/rm/v1/configure/ioc/detail`

**Request Body**:
```json
{
  "id": "string"               // (Required) 主键id
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
     // 参见 3.1 中的单个结果对象
  }
}
```

---

## 4. 指令管理 (Instructions)

### 4.1 发送指令 (Send Instruction)
**接口地址**: `POST /open_api/rm/v1/instructions/send_instruction`

**Request Body**:
```json
{
  "client_id": "string",       // 主机ID
  "instruction_name": "string",// 任务类型（见下方说明）
  "params": {                  // 自定义参数集
    "key": "string"
  },
  "batch_params": [            // 批量自定义参数集
    {
      "key": "string"
    }
  ],
  "is_batch": "string",        // 是否批量处理， 1是 0否
  "instruction_type": 0,       // (Required) 指令类型：0人工响应 1自动响应
  "Incident_id": "string",     // 事件下发任务传该参数
  "task_name": "string",       // (Required) 任务名称
  "is_online": "string"        // (Required) 是否在线 1 在线 0离线
}
```

**instruction_name 任务类型说明**:
- `list_ps`: 进程响应
- `quarantine_file`: 隔离文件
- `recover_file`: 恢复文件
- `quarantine_network`: 隔离主机
- `recover_network`: 恢复主机
- `kill_ps`: 结束进程
- `process_analyze`: 查看进程分析
- `image_analyze`: 查看模块分析
- `process_dump`: 下载进程dump
- `batch_quarantine_file`: 批量隔离文件
- `batch_kill_ps`: 批量结束进程
- `get_suspicious_file`: 获取可疑文件

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "task_id": "string",       // 任务ID
    "host_name": "string",     // 主机名称
    "repeat": "string"         // 是否重复 true是 false 否
  }
}
```

### 4.2 获取任务列表 (Get Tasks)
**接口地址**: `POST /open_api/rm/v1/instructions/tasks`

**Request Body**:
```json
{
  "page": 0,                   // 页码
  "limit": 0,                  // 每页条数
  "id": "string",              // 任务ID
  "client_id": "string",       // client_id
  "instruction_name": "string",// 指令名称
  "user": "string",            // 用户
  "content": "string",         // 内容
  "status": "string",          // 状态
  "instruction_type": 0,       // 指令类型
  "policy_name": "string",     // 策略名称
  "create_time": {             // 创建时间
    "time_range": {
      "start": 0,
      "end": 0
    },
    "quick_time": {
      "time_span": "string",   // last
      "time_num": 0,           // 数字
      "time_type": "string"    // hours/days
    }
  },
  "update_time": {             // 更新时间
    "time_range": {
      "start": 0,
      "end": 0
    },
    "quick_time": {
      "time_span": "string",
      "time_num": 0,
      "time_type": "string"
    }
  }
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "results": [
      {
        "id": "string",                    // 任务ID
        "instruction_name": "string",      // 指令名称
        "contents": "string",              // 内容
        "status": 0,                       // 状态
        "org_name": "string",              // 组织名称
        "key": "string",                   // key
        "client_id": "string",             // client_id
        "host_name": "string",             // 主机名
        "response_content": "string",      // 响应内容
        "response_time": 0,                // 响应时间
        "create_time": 0,                  // 创建时间
        "update_time": 0,                  // 更新时间
        "activity_time": 0,                // 活动时间
        "operation_user": "string",        // 操作用户
        "allow_download": true,            // 是否允许下载
        "error_code": 0,                   // 错误码
        "error_message": "string",         // 错误信息
        "instruction_type": 0,             // 指令类型
        "policy_name": "string",           // 策略名称
        "search_content": "string",        // 搜索内容
        "search_content_list": "string",   // 搜索内容列表
        "file_id": "string"                // 文件ID
      }
    ],
    "total": 0                             // 总数
  }
}
```

### 4.3 获取任务结果 (Get Task Result)
**接口地址**: `POST /open_api/rm/v1/instructions/task_result`

**Request Body**:
```json
{
  "task_id": "string"          // (Required) 任务ID
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "collect_time": 0,                 // 收集时间
    "host_name": "string",             // 主机名
    "instruction_name": "string",      // 指令名称
    "message": "string",               // 消息
    "image_detail": [                  // 镜像详情
      {
        "image_path": "string",        // 镜像路径
        "image_level": 0,              // 镜像级别
        "image_sha1": "string",        // 镜像SHA1
        "image_signature": "string",   // 镜像签名
        "is_system": 0                 // 是否系统镜像
      }
    ],
    "process": [                       // 进程列表
      {
        "is_system": 0,                // 是否系统进程
        "level": 0,                    // 级别
        "path": "string",              // 路径
        "pid": 0,                      // 进程ID
        "pname": "string",             // 进程名称
        "sha1": "string",              // SHA1
        "signature": "string"          // 签名
      }
    ],
    "process_detail": [                // 进程详情
      {
        "thread_id": 0,                // 线程ID
        "thread_rip": "string",        // 线程RIP
        "thread_symbol": "string",     // 线程符号
        "thread_feature": 0,           // 线程特征
        "code_feature": 0,             // 代码特征
        "address_feature": 0           // 地址特征
      }
    ]
  }
}
```

---

## 5. 事件管理 (Incident)

### 5.1 事件详情
**接口地址**: `POST /open_api/rm/v1/incident/view`

**Request Body**:
```json
{
  "incident_id": "string",  // IncidentId
  "client_id": "string"     // ClientId
}
```

## 6. 检测管理 (Detection)

### 6.1 检测详情
**接口地址**: `POST /open_api/rm/v1/detection/view`

**Request Body**:
```json
{
  "detection_id": "string",  // DetectionId
  "client_id": "string",     // ClientId
  "view_type": "string",     // ProcessView, HostView
  "process_uuid": "string"   // PUuid, 进程uuid
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "success",
  "data": {
    "client_id": "string",
    "org_name": "string",
    "incident_name": "string",
    "incident_name_en": "string",
    "score": 0,
    "last_update_time": 0,
    "start_time": 0,
    "view_type": "process",
    "timeline": {
      "min_time": 0,
      "max_time": 0
    },
    "file_relations": [],
    "process_relations": [],
    "ioa_timeline": [
      {
        "aggregate_count": 0,
        "command_line": "string",
        "create_time": 0,
        "description": "string",
        "description_en": "string",
        "detection_id": "string",
        "detection_source": "string",
        "event_time": 0,
        "file_name": "string",
        "incident_id": "string",
        "ioa_id": "string",
        "ioc_data": null,
        "key": "string",
        "log_id": "string",
        "name": "string",
        "name_en": "string",
        "name_key": "string",
        "p_id": "string",
        "p_md5": "string",
        "p_name": "string",
        "p_sha1": "string",
        "p_uuid": "string",
        "pp_id": "string",
        "pp_name": "string",
        "pp_uuid": "string",
        "process_name": "string",
        "process_uuid": "string",
        "ta_id": "string",
        "ta_name": "string",
        "ta_name_en": "string",
        "t_id": "string",
        "t_name": "string",
        "t_name_en": "string",
        "threat_level": "string",
        "view_type": "string"
      }
    ],
    "ioc_list": {
      "commands": [],
      "domains": [],
      "ips": [],
      "registries": [],
      "files": [
        {
          "name": "string",
          "path": "string",
          "md5": "string",
          "sha1": "string",
          "threat_type": "string",
          "threat_type_eng": "string"
        }
      ],
      "processes": [
        {
          "name": "string",
          "path": "string",
          "level": "string",
          "md5": "string",
          "sha1": "string"
        }
      ]
    },
    "host_info": {
      "client_id": "string",
      "hostname": "string",
      "client_ip": "string",
      "ip_address": "string",
      "mac_address": "string",
      "platform": 0,
      "os_type": 0,
      "os_version": "string",
      "win_version": "string",
      "client_version": "string",
      "status": "string",
      "importance": 0,
      "isolated": 0,
      "username": "string",
      "last_active": 0,
      "last_active_time": "string",
      "lastlogontime": 0,
      "latest_detection_time": "string",
      "org_name": "string",
      "orgconnectip": "string",
      "group_id": "string",
      "remarks": "string",
      "workgroup": "string",
      "tags": []
    },
    "processes": [
      {
        "process_id": "string",
        "process_uuid": "string",
        "parent_uuid": "string",
        "is_host_root": false,
        "process_name": "string",
        "process_path": "string",
        "process_md5": "string",
        "process_sha1": "string",
        "process_level": "string",
        "event_time": 0,
        "process_create_time": 0,
        "source_from": {},
        "behavior_stats": {
          "disk": 0,
          "registry": 0,
          "dns": 0
        },
        "behaviors": {
          "disk": [
            {
              "create_time": "string",
              "disk_path1": "string",
              "disk_type": 0,
              "file_bytes": "string",
              "md5": "string",
              "sha1": "string",
              "p_uuid": "string",
              "rpc_tag": "string"
            }
          ],
          "registry": [
            {
              "create_time": "string",
              "operation_type": "string",
              "p_uuid": "string",
              "registry_path": "string",
              "registry_type": "string",
              "registry_value": "string",
              "rpc_tag": "string",
              "value_name": "string"
            }
          ],
          "dns": [
            {
              "create_time": "string",
              "dns_type": 0,
              "domain": "string",
              "rpc_tag": "string"
            }
          ]
        },
        "ioas": [
          {
            "aggregate_count": 0,
            "command_line": "string",
            "create_time": 0,
            "description": "string",
            "description_en": "string",
            "detection_id": "string",
            "detection_source": "string",
            "incident_id": "string",
            "ioa_id": "string",
            "ioc_data": null,
            "key": "string",
            "log_id": "string",
            "name": "string",
            "name_en": "string",
            "name_key": "string",
            "event_time": 0,
            "file_name": "string",
            "p_id": "string",
            "p_name": "string",
            "p_md5": "string",
            "p_sha1": "string",
            "p_uuid": "string",
            "pp_id": "string",
            "pp_name": "string",
            "pp_uuid": "string",
            "process_name": "string",
            "process_uuid": "string",
            "rpc_process": "string",
            "threat_level": "string",
            "ta_id": "string",
            "ta_name": "string",
            "ta_name_en": "string",
            "t_id": "string",
            "t_name": "string",
            "t_name_en": "string",
            "view_type": "string"
          }
        ],
        "process_info": {
          "process": "string",
          "processid": "string",
          "processcreatetime": "string",
          "processname": "string",
          "processuuid": "string",
          "processlevel": "string",
          "processmd5": "string",
          "processsha1": "string",
          "processavmeta": "string",
          "processimageavmeta": "string",
          "commandline": "string",
          "account": "string",
          "domain": "string",
          "sid": "string",
          "processintegrity": "string",
          "processsigned": "string",
          "psfilesigntype": "string",
          "pstreesigntype": "string",
          "rootprocess": "string",
          "rootprocessid": "string",
          "rootprocessname": "string",
          "rootprocessuuid": "string",
          "rootprocessmd5": "string",
          "rootprocesssha1": "string",
          "pe.companyname": "string",
          "pe.description": "string",
          "pe.fileversion": "string",
          "pe.internalname": "string",
          "pe.productname": "string",
          "pe.pdbpath": "string",
          "processsignerexex": "string",
          "processsignerserialex": "string",
          "processsignerror": "string",
          "isnetworked": "string",
          "folderimageloadcount": "string",
          "uac": "string",
          "havesecsection": "string",
          "isfilehidden": "string",
          "isfolderhidden": "string",
          "memact": "string",
          "microsoftsignonly": "string",
          "maxprocessdlllevel": "string",
          "processtreelevel": "string"
        },
        "source_from": {
          "c_uuid": "string",
          "c_root_uuid": "string",
          "c_name": "string",
          "c_nt_name": "string",
          "c_md5": "string",
          "c_sha1": "string",
          "cp_level": "string",
          "cp_signer_type": "string",
          "cp_tree_level": "string",
          "cp_tree_signer_type": "string",
          "raw_file_path": "string"
        }
      }
    ]
  }
}
```

说明：`process_info`、`behavior_stats`、`behaviors` 为扩展字段结构，实际返回可能根据 `view_type`、平台和检测来源包含更多键值对。

---

## 7. 病毒扫描计划管理 (Virus Scan)

说明：以下接口由 open_api 透传到 ratp-console，open_api 会自动补充组织、用户、语言等上下文信息，不需要调用方显式传入。

### 7.1 新建扫描计划
**接口地址**: `POST /open_api/rm/v1/virus_scan/add`

**Request Body**:
```json
{
  "scan_type": 1,               // (Required) 1 快速扫描 2 全盘扫描 3 自定义路径扫描
  "plan_name": "string",        // (Required) 计划名称
  "plan_type": 1,               // (Required) 1 立即执行 2 计划执行
  "scope": 1,                   // 1 特定主机 2 主机组 3 全网主机
  "contents": {},               // object，透传下游；scan_type=3 时要求包含 scan_path
  "client_id": "string",        // scope=1 时使用
  "execute_start_time": 0,      // plan_type=2 时使用，Unix 时间戳
  "execute_cycle": 1,           // cycle_setting=true 时：1 每天 2 每周 3 每月
  "cycle_setting": false,       // 是否周期执行
  "group_ids": [1, 2]           // scope=2 时使用
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": null
}
```

**字段约束**:
- `scan_type`、`plan_name`、`plan_type` 为源码 binding 必填字段
- `scope=1` 时必须提供 `client_id`
- `scope=2` 时必须提供 `group_ids`
- `scan_type=3` 时必须在 `contents.scan_path` 中提供至少一个扫描路径
- `cycle_setting=true` 时 `execute_cycle` 仅支持 `1/2/3`
- `plan_type=2` 时通常应提供 `execute_start_time`

### 7.2 编辑扫描计划
**接口地址**: `POST /open_api/rm/v1/virus_scan/update`

**Request Body**:
```json
{
  "rid": "string",              // (Required) 扫描计划ID
  "scan_type": 1,
  "plan_name": "string",
  "plan_type": 2,
  "scope": 2,
  "contents": {},               // object，透传下游；scan_type=3 时要求包含 scan_path
  "client_id": "string",
  "execute_start_time": 0,
  "execute_cycle": 2,
  "cycle_setting": true,
  "group_ids": [1001, 1002]
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": null
}
```

**字段约束**:
- `rid` 为源码 binding 必填字段
- 其余约束与“7.1 新建扫描计划”一致

### 7.3 取消扫描计划
**接口地址**: `POST /open_api/rm/v1/virus_scan/cancel`

**Request Body**:
```json
{
  "rid": "string"               // 扫描计划ID
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": null
}
```

### 7.4 获取扫描计划列表
**接口地址**: `POST /open_api/rm/v1/virus_scan/list`

**Request Body**:
```json
{
  "plan_name": "string",
  "scope": 0,
  "plan_type": 0,
  "cycle_setting": 0,           // 1 周期执行 2 非周期执行
  "scan_type": 0,
  "update_time": {
    "time_range": {
      "start": 0,
      "end": 0
    },
    "quick_time": {
      "time_num": 7,
      "time_span": "last",
      "time_type": "days"
    }
  },
  "status": "0",                // 0 准备中 1 执行中 2 已完成 3 已取消
  "operation_user": "string",
  "page": 1,
  "limit": 10
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "total": 0,
    "results": [
      {
        "rid": "string",
        "org_name": "string",
        "client_id": "string",
        "plan_name": "string",
        "plan_type": 0,
        "execute_start_time": 0,
        "execute_cycle": 0,
        "cycle_setting": false,
        "execute_desc": "string",
        "scope": 0,
        "scope_content": "string",
        "contents": "string",
        "group_ids": [0],
        "scan_type": 0,
        "create_time": 0,
        "update_time": 0,
        "operation_user": "string",
        "operation_uid": "string",
        "status": 0,
        "is_deleted": 0
      }
    ]
  }
}
```

说明：`page/limit` 省略时，服务端默认使用 `page=1`、`limit=10`。

### 7.5 获取扫描执行记录
**接口地址**: `POST /open_api/rm/v1/virus_scan/scan_record`

**Request Body**:
```json
{
  "rid": "string",
  "task_id": "string",
  "execution_batch": "string",
  "host_name": "string",
  "client_id": "string",
  "scan_type": "1,2,3",
  "status": "0,1,2,3",
  "start_time": {
    "time_range": {
      "start": 0,
      "end": 0
    },
    "quick_time": {
      "time_num": 7,
      "time_span": "last",
      "time_type": "days"
    }
  },
  "end_time": {
    "time_range": {
      "start": 0,
      "end": 0
    },
    "quick_time": {
      "time_num": 7,
      "time_span": "last",
      "time_type": "days"
    }
  },
  "page": 1,
  "limit": 10
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "total": 0,
    "results": [
      {
        "id": "string",
        "task_id": "string",
        "execution_batch": "string",
        "rid": "string",
        "org_name": "string",
        "client_id": "string",
        "host_name": "string",
        "scan_type": "string",
        "contents": "string",
        "status": 0,
        "create_time": 0,
        "start_time": 0,
        "end_time": 0,
        "update_time": 0,
        "virus_file_num": 0,
        "memory_virus_num": 0,
        "response_time": 0,
        "plan_name": "string",
        "host_status": "string"
      }
    ]
  }
}
```

说明：
- `page/limit` 省略时，服务端默认使用 `page=1`、`limit=10`
- `start_time` 省略时，服务端会自动补默认时间范围（最近配置限制天数）

---

## 8. 病毒统计与明细 (Virus)

说明：
- `page/limit` 省略时，console 侧默认使用 `page=1`、`limit=10`
- `last_checked_time` 省略时，console 侧会自动补默认查询时间范围（最近配置限制天数）

### 8.1 按主机统计
**接口地址**: `POST /open_api/rm/v1/virus/host/list`

**Request Body**:
```json
{
  "client_id": "string",
  "username": "string",
  "host_name": "string",
  "importance": 0,
  "mac_address": "string",
  "client_ip": "string",
  "rmconnectip": "string",
  "status": 0,                  // 0 未处置 1 部分处置 2 全部处置
  "last_checked_time": {
    "time_range": {
      "start": 0,
      "end": 0
    },
    "quick_time": {
      "time_num": 7,
      "time_span": "last",
      "time_type": "days"
    }
  },
  "page": 1,
  "limit": 10
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "total": 0,
    "results": [
      {
        "host_name": "string",
        "client_id": "string",
        "status": 0,
        "username": "string",
        "importance": 0,
        "client_ip": "string",
        "rm_connect_ip": "string",
        "mac_address": "string",
        "virus_file_count": 0,
        "virus_memory_count": 0,
        "last_checked_time": 0,
        "last_active": 0,
        "host_status": "string",
        "path": "string"
      }
    ]
  }
}
```

说明：当请求体中传入 `status` 筛选时，服务内部会切换到 hash 维度聚合逻辑，`results` 项会从 `VirusHostBaseResponse` 变为 `VirusHostByHashResponse`，额外返回：

```json
{
  "sha1": "string",
  "md5": "string"
}
```

### 8.2 按文件 Hash 统计
**接口地址**: `POST /open_api/rm/v1/virus/hash/list`

**Request Body**:
```json
{
  "last_checked_time": {
    "time_range": {
      "start": 0,
      "end": 0
    },
    "quick_time": {
      "time_num": 7,
      "time_span": "last",
      "time_type": "days"
    }
  },
  "name": "string",
  "sha1": "string",
  "md5": "string",
  "page": 1,
  "limit": 10
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "total": 0,
    "results": [
      {
        "name": "string",
        "sha1": "string",
        "md5": "string",
        "size": 0,
        "host_count": 0,
        "client_ids": ["string"],
        "end_time": 0,
        "id": "string"
      }
    ]
  }
}
```

### 8.3 按文件 Hash 查看主机明细
**接口地址**: `POST /open_api/rm/v1/virus/hash/host/list`

**Request Body**:
```json
{
  "sha1": "string",
  "client_id": "string",
  "username": "string",
  "host_name": "string",
  "importance": 0,
  "mac_address": "string",
  "client_ip": "string",
  "rmconnectip": "string",
  "status": 0,                  // 0 未处置 1 部分处置 2 全部处置
  "host_status": "online,offline",
  "path": "string",
  "last_checked_time": {
    "time_range": {
      "start": 0,
      "end": 0
    },
    "quick_time": {
      "time_num": 7,
      "time_span": "last",
      "time_type": "days"
    }
  },
  "page": 1,
  "limit": 10
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "total": 0,
    "results": [
      {
        "host_name": "string",
        "client_id": "string",
        "status": 0,
        "username": "string",
        "importance": 0,
        "client_ip": "string",
        "rm_connect_ip": "string",
        "mac_address": "string",
        "virus_file_count": 0,
        "virus_memory_count": 0,
        "last_checked_time": 0,
        "last_active": 0,
        "host_status": "string",
        "path": "string",
        "sha1": "string",
        "md5": "string"
      }
    ]
  }
}
```

---

## 9. 查杀设置 (NGAV Settings)

### 9.1 获取查杀配置
**接口地址**: `GET /open_api/rm/v1/settings/get_ngav_conf`

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": {
    "<conf_key>": "<conf_value>"
  }
}
```

说明：该接口返回的是海外设置中心 `rmkernel_config` 反序列化后的 `map[string]interface{}`，即 `data` 为动态键值集合，并非固定字段结构；上面仅展示常见键示例。

### 9.2 切换查杀状态
**接口地址**: `POST /open_api/rm/v1/settings/switch_ngav_status`

**Request Body**:
```json
{
  "switch": "on"               // on / off
}
```

**Response Body**:
```json
{
  "error": 0,
  "message": "string",
  "data": null
}
```

说明：当前实现会联动更新海外设置中的 `enable_quarantine_files` 开关。
