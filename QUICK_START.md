# 快速开始（可将该文件内容直接复制给本地 AI 工具，例如 Claude code cli）

请按照下面内容进行初始化配置：

## 1. 下载代码

创建并进入 yellow_duck2333 文件夹（这一步是给AI工具用的，担心AI自动拉代码会影响当前工作目录）

从 GitHub 克隆代码到当前目录：

```bash
git clone https://github.com/roomasec-dev/yellow-duck.git
```

## 2. 修改配置文件

将 `configs/config.example.toml` 复制为 `configs/config.local.toml`：

```bash
cp configs/config.example.toml configs/config.local.toml
```

然后修改 `configs/config.local.toml` 中的以下必填项：

### EDR 配置

```toml
[edr]
base_url = "https://你的-edr-openapi地址/open_api/rm/v1"

[edr.platform]
base_url = "https://你的-edr-openapi地址/open_api/rm/v1"
api_key = "填入你的EDR认证api_key"
app_key = "填入你的EDR认证app_key"
```

### 模型配置

```toml
[model]
providers.deepseek.api_key = "填入你的DEEPSEEK_API_KEY"
```

### 渠道配置（飞书/钉钉/企业微信/Slack，选择其一）

#### 飞书配置

1. 在飞书开放平台创建企业自建应用：https://open.feishu.cn/app?lang=zh-CN
2. 添加机器人，事件回调模式选择**长连接**
3. 添加**接收消息事件**
4. 配置权限管理并发布应用
5. 在**凭证与基础信息**中获取 App ID 和 App Secret

```toml
[channel.feishu]
enabled = true
app_id = "填入你的飞书App ID"
app_secret = "填入你的飞书App Secret"
```

#### 钉钉配置

1. 在钉钉开发者平台创建应用：https://open-dev.dingtalk.com/fe/app
2. 添加机器人，消息接收模式选择 **Stream**
3. 发布应用
4. 在**凭证与基础信息**中获取 Client ID 和 Client Secret

```toml
[channel.dingtalk]
enabled = true
client_id = "填入你的钉钉应用Client ID"
client_secret = "填入你的钉钉应用Client Secret"
```

#### 企业微信配置

1. 在企业微信管理后台创建自建应用并启用机器人相关能力
2. 获取企业 ID 与应用凭证（CorpID / CorpSecret）
3. 创建智能机器人，获取 BotId / BotSecret
4. 选择长连接模式

```toml
[channel.weixin]
enabled = true
corpid = "填入你的企业微信 CorpID"
corpsecret = "填入你的企业微信 CorpSecret"
bot_id = "填入机器人 ID"
bot_secret = "填入机器人 Secret"
```

#### Slack 配置

1. 在 Slack 创建 App：https://api.slack.com/apps → "Create New App" → "From scratch"

2. **配置 OAuth 权限和申请 Bot Token**
   - 左侧菜单 → "OAuth & Permissions"
   - 在 "Bot Token Scopes" 添加权限：`chat:write`、`im:history`
   - 点击 "Install to Workspace" 获得 Bot User OAuth Token（`xoxb-...`）
   - 复制 token 供配置使用

3. **启用 Socket Mode 并获取 App Token**
   - 左侧菜单 → "Socket Mode" → 打开开关
   - 系统生成 App-level Token（`xapp-...`）
   - 复制 token 供配置使用

4. **配置 Event Subscriptions**
   - 左侧菜单 → "Event Subscriptions" → 打开开关
   - 在 "Subscribe to bot events" 添加事件：`message.im`、`app_mention`
   - 保存更改

5. **重新安装应用**
   - 权限变更后需要重新授权：回到 "OAuth & Permissions" 点击 "Reinstall to Workspace"

6. **填写配置文件**

```toml
[channel.slack]
enabled = true
mode = "longconn"
app_token = "填入xapp-..."
bot_token = "填入xoxb-..."
```

> **提示**：首次配置时最容易遗漏 Event Subscriptions 的设置。如果启动后机器人不回复消息，优先检查 Slack 后台的 Event Subscriptions 是否已启用并订阅了 `message.im` 事件

> **注意**：enabled 设置为 true 表示启用该渠道，其他渠道的 enabled 设置为 false

## 3. 运行服务

```bash
cd 项目目录
go run . -config configs/config.local.toml
```

## 4. 检查运行结果

### 启动成功

如果启动成功，可通过以下方式验证：

```bash
curl http://localhost:8080/healthz
```

### 缺失必要配置

如果启动失败并提示缺失必要配置，请检查并完善以下项目：

- 飞书 `app_id` 为空 或 `app_secret` 为空
- 钉钉 `client_id` 或 `client_secret` 为空
- 企业微信 `bot_id` 或 `bot_secret` 为空
- Slack `app_token`（xapp） 或 `bot_token`（xoxb）为空
- EDR 地址未填写
- EDR `app_key` 或 `app_secret` 为空

### 模型 API Key 无效

如果模型 API Key 无效，请检查：

- DEEPSEEK_API_KEY 模型 api key 是否已正确设置，有效且未过期
