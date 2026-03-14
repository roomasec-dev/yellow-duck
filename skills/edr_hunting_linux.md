# Capability: edr_hunting_linux

适用：
- `os_type=linux`
- 当前真实样本里已经看到 `systemd -> logrotate` 这类 `exec` 进程链。

优先字段：
- 主体：`process`、`process_name`、`process_id`、`command_line`
- 新进程：`new_process_name`、`newprocess`、`newcommandline`、`newprocessuuid`
- 父链：`parent_process*`、`grand1_*`、`grand2_*`
- 哈希：`md5`、`sha1`、`processmd5`、`processsha1`
- 文件/目录：`file`、`directory`
- 风险等级：`processlevel`、`newprocesslevel`、`newimagelevel`
- 签名：如果有 `signing_info` / `signed` 再用，没有就别强行说

等级字段怎么用：
- `processlevel` / `newprocesslevel` / `newimagelevel` 使用同一套语义：`10-20` 可信，`30` 未知，`40` 未入库，`50-70` 恶意或高风险。
- Linux 上高等级对象如果再叠加 shell 拉起、下载执行、计划任务/服务修改，应优先扩父子进程链和命令行。
- 不要把 detection 的 `threat_level`、事件分数误当成这里的进程等级。

怎么猎：
- 优先看 `exec`：谁从 `systemd`、shell、cron、服务进程派生出来。
- 看是否出现异常工具链：shell -> downloader -> script -> persistence。
- 看命令行里是否有下载、反弹、提权、计划任务、systemd service 修改、定时任务修改。
- 看父进程是否异常：例如业务进程突然拉起 shell、解释器、网络工具。
- 看同一 `client_id` 下是否重复出现相同路径、相同哈希、相同父子进程模式。

高价值 IOC：
- 可执行路径、脚本路径、命令行
- 父子进程链
- 哈希
- 异常服务/计划任务相关路径

contain 用法：
- 可用于模糊试探路径或命令行片段，例如 `/tmp/`、`curl `、`wget `、`bash -c`。
- 如果是明确进程名（如 `bash`、`sh`、`python3`、`curl`），优先用 `is`，不要先用 `contain`。
- 但命中后一定要再核对真实字段值，不要直接当成确认命中。
