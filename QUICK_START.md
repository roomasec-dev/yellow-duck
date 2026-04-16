# 快速开始（可将该文件内容直接复制给本地 AI 工具，例如 Claude code cli）

请按照下面内容进行初始化配置：

## 1. 下载代码

创建并进入 yellow_duck2333 文件夹

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

### 渠道配置（飞书或钉钉，选择其一）

#### 飞书配置

1. 在飞书开放平台创建企业自建应用：https://open.feishu.cn/app?lang=zh-CN
2. 添加机器人，事件回调模式选择**长连接**
3. 添加**接收消息事件**
4. 配置权限管理并发布应用
5. 在**凭证与基础信息**中获取 App ID 和 App Secret

```toml
[channel.feishu]
app_id = "填入你的飞书App ID"
app_secret = "填入你的飞书App Secret"
enabled = true
```

#### 钉钉配置

1. 在钉钉开发者平台创建应用：https://open-dev.dingtalk.com/fe/app
2. 添加机器人，消息接收模式选择 **Stream**
3. 发布应用
4. 在**凭证与基础信息**中获取 Client ID 和 Client Secret

```toml
[channel.dingtalk]
client_id = "填入你的钉钉应用Client ID"
app_secret = "填入你的钉钉应用Client Secret"
enabled = true
```

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

- 飞书 `app_id` 为空
- 飞书 `app_secret` 为空
- EDR 地址未填写
- EDR `app_key` 或 `app_secret` 为空

### 模型 API Key 无效

如果模型 API Key 无效，请检查：

- DEEPSEEK_API_KEY 模型 api key 是否已正确设置，有效且未过期
