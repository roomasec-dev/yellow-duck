# Capability: edr_hunting_macos

适用：
- `os_type=macos`
- 当前真实样本里已经看到 `Feishu -> bash -> grep` 这类 `exec` 链。

优先字段：
- 主体：`process`、`process_name`、`command_line`
- 新进程：`new_process_name`、`newprocess`、`newcommandline`
- 父链：`parent_process*`、`grand1_*`
- 哈希：`md5`、`sha1`、`processmd5`、`processsha1`
- 风险等级：`processlevel`、`newprocesslevel`、`newimagelevel`
- 签名：`signing_info`、`new_signing_info`
- 用户：`process_user_name`、`new_user_name`

等级字段怎么用：
- `processlevel` / `newprocesslevel` / `newimagelevel` 也是同一套语义：`10-20` 可信，`30` 未知，`40` 未入库，`50-70` 恶意或高风险。
- macOS 上如果高等级对象又是 GUI App 拉起 shell、脚本、网络工具，优先看签名、父链、命令行和持久化路径。
- 不要把 detection 的 `threat_level`、事件分数误当成这里的进程等级。

怎么猎：
- 优先看 `exec`，尤其是 GUI 应用、浏览器、IM、Office、launchd 派生 shell/脚本/工具链。
- 看是否出现异常链路：App -> shell -> curl/wget/bash/python/osascript。
- 看签名是否可信：开发者 ID、Apple 签名、签名主体是否和进程身份匹配。
- 看是否出现 LaunchAgent / LaunchDaemon / 偏好设置 / 持久化目录相关路径。
- 看是否有用户空间应用突然拉起终端、网络探测、系统命令。

高价值 IOC：
- App 路径、二进制路径、命令行
- 开发者签名信息
- 父子进程链
- 哈希
- 用户名与宿主范围

contain 用法：
- 可试探 `process`、`command_line`、`newcommandline` 里的路径或命令片段。
- 如果是明确进程名（如 `bash`、`zsh`、`osascript`、`curl`），优先用 `is`，不要先用 `contain`。
- 但当前 contain 返回并不稳定，命中后必须人工核对是否真的包含目标字符串。
