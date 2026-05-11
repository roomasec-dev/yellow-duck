# 小黄鸭机器人

一个以 Go 实现的 AI 控制面脚手架，目标是做成类似 OpenClaw 的主 Chat 机器人：

- 从 IM 接收消息，当前先接飞书应用机器人
- 用一个主 Chat 维护上下文和工具路由
- 通过 EDR Open API 控制终端动作
- 使用 TOML 配置模型、会话、压缩、EDR 和渠道
- 使用无 CGO 的 SQLite 落地会话、摘要和消息去重

## 当前交互方式

当前既支持显式命令（39个），也支持自然语言工具规划：

**EDR 主机管理**
- `/edr hosts [hostname]` 查询主机
- `/edr host-isolate <client_id>` 隔离主机
- `/edr host-release <client_id>` 恢复主机
- `/edr host-black <client_id_csv> <reason>` 加入主机黑名单
- `/edr host-remove <client_id_csv>` 从管控中移除主机
- `/edr host-offline-get` 查询主机离线配置
- `/edr host-offline-set <status> <time>` 保存主机离线配置（status: 1-开启 2-关闭，time: 超时天数）

**EDR 事件与检出**
- `/edr incidents [client_id] [page] [page_size]` 查询平台事件
- `/edr detections [incident_id] [page] [page_size]` 查询平台行为检出
- `/edr incident-view <incident_id> <client_id>` 查询事件详情
- `/edr detection-view <detection_id> <client_id> [view_type] [process_uuid]` 查询检出详情
- `/edr detection-update <detection_id|ids_csv> <status>` 更新检出处置状态（状态：1-待处置 2-处置中 3-已处理 4-误报）
- `/edr incident-update <ids_csv> <status> [scene] [allow] [comment]` 批量处置事件（状态：1-未研判 2-研判中 3-真攻击 4-误报）
- `/edr incident-summary <incident_id>` 查询相关事件

**EDR 计划管理**
- `/edr plans [page] [page_size]` 查询计划列表
- `/edr plan-add <plan_name> <scan_type> <plan_type> <scope> [type]` 创建计划
  - `scan_type` 可选值：`1`-快速扫描 `2`-全盘扫描 `3`-自定义路径扫描 `4`-漏洞修复 `5`-安装软件 `6`-卸载软件 `7`-更新软件 `8`-发送文件
  - `plan_type` 可选值：`1`-立即执行 `2`-定时执行 `3`-周期执行
  - `scope` 可选值：`1`-特定主机 `2`-主机组 `3`-全网主机
  - `type` 可选值：`kill_plan` / `leak_repair` / `distribute_software` / `distribute_file`（可选，不填默认 `kill_plan`）
- `/edr plan-edit <rid> <plan_name> <scan_type> <plan_type> <scope> <type>` 编辑计划
- `/edr plan-cancel <rid>` 取消计划

**EDR 病毒管理**
- `/edr virus-scan-record [hostname] [client_id] [page] [page_size]` 病毒扫描记录
- `/edr virus-by-host [hostname] [client_id] [page] [page_size]` 按主机查病毒
- `/edr virus-by-hash [hash] [page] [page_size]` 按哈希查病毒
- `/edr virus-hash-hosts <sha1> [hostname] [page] [page_size]` 按哈希查关联主机

**EDR 隔离文件管理**
- `/edr isolate-files [client_id] [hostname] [page] [page_size]` 隔离文件列表
- `/edr isolate-files-release <isolate_file_guids_csv> [isolate_file_add_exclusion] [isolate_file_release_all_hash]` 放行隔离文件
  - `isolate_file_guids_csv` 为必填，多个 GUID 用英文逗号分隔
  - `isolate_file_add_exclusion` 可选，`true/false`，默认 `false`
  - `isolate_file_release_all_hash` 可选，`true/false`，默认 `false`
- `/edr isolate-files-delete <isolate_file_guids_csv>` 删除隔离文件记录
  - `isolate_file_guids_csv` 为必填，多个 GUID 用英文逗号分隔

**EDR IOC管理**
- `/edr iocs [hash] [page] [page_size]` IOC列表
- `/edr ioc-add <ioc_action> <ioc_hash> [ioc_description] [ioc_expiration_date] [ioc_file_name] [ioc_host_type]` 添加IOC
  - `ioc_action` 为必填，动作类型（如 `block` / `watch`）
  - `ioc_hash` 为必填，支持 MD5/SHA1/SHA256
  - `ioc_description`、`ioc_expiration_date`（如 `2026-12-31`）、`ioc_file_name`、`ioc_host_type` 为可选
- `/edr ioc-update <ioc_id> <ioc_hash> [ioc_action] [ioc_description] [ioc_expiration_date]` 更新IOC
  - `ioc_id`、`ioc_hash` 为必填
  - `ioc_action`、`ioc_description`、`ioc_expiration_date` 为可选
- `/edr ioc-delete <ioc_id>` 删除IOC
  - `ioc_id` 为必填

**EDR IOA管理**
- `/edr ioas [page] [page_size]` IOA列表
- `/edr ioa-networks [page] [page_size]` IOA网络排除列表

**EDR 策略管理**
- `/edr strategies [type] [page] [page_size]` 策略列表
- `/edr strategy-single <strategy_type>` 策略详情
- `/edr strategy-state` 策略状态
- `/edr strategy-update <strategy_type> <strategy_id> [key=value ...]` 更新策略
  - `strategy_type` 为必填，`virus_scan_settings`（病毒查杀设置）或 `asset_registration`（资产登记）
  - `strategy_id` 为必填，策略 ID
  - 可选参数支持 key=value 格式，包括：`scan_file_scope`、`startup_scan_mode`、`archive_size_limit_enabled`、`archive_size_limit`、`realtime_mem_cache_tech_enabled`、`dynamic_cpu_monitor_enabled`、`dynamic_cpu_high_percent`、`stop_realtime_on_cpu_high_enabled`、`stop_realtime_cpu_high_percent`、`owl_on_realtime_enabled`、`realtime_scan_archive_enabled`、`runtime_max_file_size_mb`、`custom_max_file_size_mb`

**EDR 响应**
- `/edr send-instruction <instruction_name> <client_id> [path]` 发送指令到主机（人工响应）
- `/edr tasks [page] [page_size]` 查询指令任务列表（人工响应）
- `/edr task-result <task_id>` 查询任务结果（人工响应）
- `/edr instruction-policies [page] [page_size]` 自动响应策略

**EDR 威胁狩猎**
- `/edr logs [client_id] [page] [page_size]` 日志调查
- `/edr log-alarms [page] [page_size]` 狩猎告警

高危操作（隔离、恢复、放行隔离文件、删除隔离文件记录、IOC管理写操作、加入黑名单、移除主机、发送指令、计划管理、更新检出处置状态、批量处置事件）需要回复「确认」后才能执行。

现在也支持知识库工具：

- 自然语言搜索知识库，底层会递归遍历 `knowledge_base.path` 下的 `.md` 文件
- 自然语言新增、编辑、追加、删除知识库条目
- `knowledge_base.path` 可以直接指向外部目录，不要求知识库文件放在当前仓库里

普通消息会进入主 Chat，并使用真实模型层实现回复。当前默认配置已经切到 DeepSeek 的 thinking 模型 `deepseek-reasoner`，并保留 `deepseek-chat` 和 `stub` 作为 fallback。

如果主模型思考时间较长，系统会额外用 `progress.model` 把内部步骤改写成一条更友好的进度消息，再通过渠道发给用户。现在默认是用 `deepseek-chat` 做这层“进度播报”。

同时，系统会先用 `routing.model` 对自然语言做一次 EDR 意图路由。对于只读查询（主机、事件、检出、日志），会直接自动调用 EDR，再把真实结果交给主模型组织答案；高危写操作默认仍要求显式命令确认。

另外系统会把 `prompts/AGENTS.md` 和 `skills/*.md` 注入提示词，并维护当前会话的长期记忆。长期记忆会自动限制数量，超过上限时裁剪较旧条目。

对于超大的 incident/detection 详情，系统不会直接把整块 JSON 塞给模型，而是先保存成 artifact，再按问题做定向搜索和分段读取，避免把上下文窗口一次性打爆。

## 飞书接入

- 现在默认使用 `longconn` 长连接模式，不需要公网回调地址
- `channel.feishu.mode` 支持 `longconn`、`webhook`、`both`
- 如果用长连接，只需要 `app_id` 和 `app_secret`
- `verification_token` / `encrypt_key` 主要用于 webhook 模式

## 钉钉接入
- 现在默认使用 `longconn` 长连接模式，不需要公网回调地址
- `channel.dingtalk.mode` 支持 `longconn`、`webhook`、`both`
- 如果用长连接，只需要 `client_id` 和 `client_secret`
- `verification_token` / `encrypt_key` 主要用于 webhook 模式

## 安装与运行

### 预编译文件

项目根目录下的 `dist/` 会放这四个可执行文件：

- `dist/rm-ai-agent-darwin-amd64`
- `dist/rm-ai-agent-darwin-arm64`
- `dist/rm-ai-agent-linux-amd64`
- `dist/rm-ai-agent-windows-amd64.exe`

说明：

- Linux 和 Windows 目前默认是 `amd64` 架构
- macOS 同时提供 `amd64` 和 `arm64` 两个版本：
  - Apple Silicon（M1/M2/M3）建议优先使用 `dist/rm-ai-agent-darwin-arm64`
  - Intel Mac 使用 `dist/rm-ai-agent-darwin-amd64`

### 通用安装步骤

1. 准备目录和配置文件

- 复制 `configs/config.example.toml`，保存成 `configs/config.local.toml`
- 可执行文件和 `configs/`、`skills/`、`prompts/` 建议放在同一个项目目录结构下运行
- 运行时会自动生成 `data/rm-ai-agent.sqlite` 和 `data/rm-ai-agent.log`

2. 准备模型 API Key

- 当前默认模型是 DeepSeek，需要先去模型提供方申请 API Key
- 推荐把模型 Key 配成环境变量 `DEEPSEEK_API_KEY`
- `configs/config.local.toml` 里默认已经通过 `api_key_env = "DEEPSEEK_API_KEY"` 读取，不建议把 key 明文写进配置

3. 去 EDR 后台申请凭证

- 先确认你们接的是哪一套接口：
  - `edr.base_url`：本地/控制面 Open API
  - `edr.platform`：平台 API，主要用于事件、检出、日志、详情等能力
- 如果你们的 EDR Open API 用 Bearer Token，就向 EDR 管理后台或管理员申请 `API Token`，填到 `edr.auth_token`
- 如果你们的 EDR Open API 用自定义请求头 APIKey，就向后台申请 `API Key`，然后填到 `[edr.headers]`，例如 `X-API-Key = "你的 key"`
- 如果你们的平台 API 需要 `app_key/app_secret`，就在 EDR 后台申请这对凭证；很多团队也会把 `app_secret` 叫 `SK`
- 推荐把平台凭证走环境变量：
  - `EDR_PLATFORM_APP_KEY`
  - `EDR_PLATFORM_APP_SECRET`
- 如果要启用平台 API，把 `configs/config.local.toml` 里的 `edr.platform.enabled` 改成 `true`

4. 去飞书开放平台申请 Bot 并开权限

- 进入飞书开放平台，创建企业自建应用
- 给应用开通机器人能力（Bot）
- 在应用凭证页拿到：
  - `app_id`
  - `app_secret`
- 在权限管理里至少开通 IM 相关权限，重点是：
  - 接收用户消息
  - 发送/回复消息
  - 在群聊或会话中使用机器人
- 如果你走 `webhook` 或 `both` 模式，还要在事件订阅里配置回调地址，并拿到：
  - `verification_token`
  - `encrypt_key`
- 权限改完后，记得发布应用到企业内可用范围，并把机器人加到目标群聊或单聊场景里，否则它收不到消息

5. 按实际环境填写配置

- 下面是一份更接近真实部署的 `configs/config.local.toml` 示例：

```toml
[channel.feishu]
enabled = true
mode = "longconn"
app_id = "cli_xxx"
app_secret = "xxx"
verification_token = ""
encrypt_key = ""

[models.providers.deepseek]
type = "deepseek"
base_url = "https://api.deepseek.com"
api_key_env = "DEEPSEEK_API_KEY"
model = "deepseek-reasoner"

[edr]
base_url = "https://你的-edr-openapi/open_api/rm/v1"
auth_token = ""

[edr.platform]
enabled = true
base_url = "https://你的-edr-openapi/open_api/rm/v1"
app_key_env = "EDR_PLATFORM_APP_KEY"
app_secret_env = "EDR_PLATFORM_APP_SECRET"

[edr.headers]
# 如果你们 EDR 要求 API Key，就在这里加，例如：
# X-API-Key = "your-api-key"
```

6. 设置环境变量

- Windows PowerShell:

```powershell
$env:DEEPSEEK_API_KEY="你的模型 API Key"
$env:EDR_PLATFORM_APP_KEY="你的 EDR AppKey"
$env:EDR_PLATFORM_APP_SECRET="你的 EDR AppSecret / SK"
```

- Linux/macOS:

```bash
export DEEPSEEK_API_KEY="你的模型 API Key"
export EDR_PLATFORM_APP_KEY="你的 EDR AppKey"
export EDR_PLATFORM_APP_SECRET="你的 EDR AppSecret / SK"
```

7. 启动服务

Windows:

```powershell
.\dist\rm-ai-agent-windows-amd64.exe -config configs/config.local.toml
```

Linux:

```bash
chmod +x ./dist/rm-ai-agent-linux-amd64
./dist/rm-ai-agent-linux-amd64 -config configs/config.local.toml
```

macOS (Apple Silicon / arm64):

```bash
chmod +x ./dist/rm-ai-agent-darwin-arm64
./dist/rm-ai-agent-darwin-arm64 -config configs/config.local.toml
```

macOS (Intel / amd64):

```bash
chmod +x ./dist/rm-ai-agent-darwin-amd64
./dist/rm-ai-agent-darwin-amd64 -config configs/config.local.toml
```

8. 验证是否接通

- 访问 `GET /healthz`，返回 `ok` 说明服务已启动
- 看本地日志 `data/rm-ai-agent.log`，确认没有模型、飞书、EDR 鉴权报错
- 在飞书里给机器人发一条普通消息，例如“现在几点”或“查一下最近事件”
- 如果机器人没回复，优先检查：
  - 飞书应用是否已发布到企业内
  - 机器人是否已被拉进群
  - `app_id/app_secret` 是否正确
  - EDR 凭证是否正确
  - 模型 API Key 是否正确

### 环境变量建议

- 模型 API Key 建议走环境变量，不要直接写到仓库配置
- EDR 平台 `app_key/app_secret` 也建议走环境变量

示例：

Windows PowerShell:

```powershell
$env:DEEPSEEK_API_KEY="你的 key"
$env:EDR_PLATFORM_APP_KEY="你的平台 AppKey"
$env:EDR_PLATFORM_APP_SECRET="你的平台 AppSecret"
.\dist\rm-ai-agent-windows-amd64.exe -config configs/config.local.toml
```

Linux/macOS:

```bash
export DEEPSEEK_API_KEY="你的 key"
export EDR_PLATFORM_APP_KEY="你的平台 AppKey"
export EDR_PLATFORM_APP_SECRET="你的平台 AppSecret"
./dist/rm-ai-agent-linux-amd64 -config configs/config.local.toml
```

### 健康检查

- 程序启动后默认监听 `:8080`
- 可用 `GET /healthz` 检查服务是否正常
- 如果只跑飞书长连接，也会同时起本地 HTTP 服务用于健康检查

## 模型配置

- 当前实现支持 `openai_compatible` / `deepseek` / `stub` 三种 provider 类型
- DeepSeek 走 OpenAI 兼容接口，默认地址是 `https://api.deepseek.com`
- thinking 模型用 `deepseek-reasoner`，普通对话 fallback 用 `deepseek-chat`
- 进度播报模型单独配置在 `progress.model`，推荐保持为 `deepseek/deepseek-chat`
- 工具规划默认也走 `routing.model`
- API Key 建议通过环境变量注入，不要直接写进仓库配置

PowerShell 示例：

```powershell
$env:DEEPSEEK_API_KEY="你的 key"
go run . -config configs/config.example.toml
```

测试环境如果你直接把密钥写入本地配置，可以用：

```powershell
go run . -config configs/config.local.toml
```

## EDR 配置

- 本地/控制面 Open API 继续走 `edr.base_url`
- 平台 API 额外走 `edr.platform`，会先用 `app_key + app_secret + timestamp` 计算 MD5 签名，再调用 `/get_open_api_token` 换 token
- 平台 token 获取成功后，客户端会自动缓存一段时间，并在请求失败时尝试刷新
- 建议把平台凭证放到环境变量 `EDR_PLATFORM_APP_KEY`、`EDR_PLATFORM_APP_SECRET`
- 如果你们的 EDR 控制面不是 Bearer Token，而是 API Key 请求头，就在 `[edr.headers]` 里补自定义 header
- `edr.auth_token`、`[edr.headers]`、`edr.platform.app_key/app_secret` 具体用哪种，取决于你们后台开放的是哪套鉴权方式

PowerShell 示例：

```powershell
$env:EDR_PLATFORM_APP_KEY="你的平台 AppKey"
$env:EDR_PLATFORM_APP_SECRET="你的平台 AppSecret"
go run . -config configs/config.example.toml
```

## EDR API 文档
https://qax-console.zboundary.com/main/docs?id=19c548c5-1f18-4439-9a5b-fa3ad5f96c44

## 知识库配置

- 知识库目录由 `knowledge_base.path` 控制，可以是相对路径，也可以是外部绝对路径
- 系统只会处理这个目录下递归找到的 `.md` 文件
- 搜索命中后会返回文件标题、相对路径和片段，方便继续让 AI 做“改这篇”“删这篇”这类自然语言操作

示例：

```toml
[knowledge_base]
enabled = true
path = "D:/sec-kb"
search_limit = 5
snippet_length = 240
```

## 目录结构

```text
main.go
configs/
internal/app
internal/channel/feishu
internal/artifact
internal/compression
internal/config
internal/edr
internal/memory
internal/model
internal/planner
internal/prompt
internal/protocol
internal/session
internal/store
skills/
prompts/
```

## 设计决策

- `Gateway 控制面优先`：先把 webhook、会话、存储、EDR 工具边界搭起来
- `单主 Chat 优先`：先让一个主 Chat 控 EDR，再迭代子 agent
- `上下文治理分层`：最近对话 + 持久摘要，而不是只截断历史
- `渠道与会话解耦`：飞书只做 adapter，主会话不依赖飞书字段
- `无 CGO SQLite`：便于部署，减少目标机依赖
