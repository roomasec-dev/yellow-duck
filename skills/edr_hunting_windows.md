# Capability: edr_hunting_windows

适用：
- `os_type=windows`
- 重点看进程创建、文件访问、模块、签名、完整性、RPC、线程、磁盘访问。

如果用户在找“写自启动”/“持久化”：
- 默认理解成在 hunting 注册表 Run/RunOnce、计划任务、服务、启动目录，不要先泛化成普通 CreateProcess 排查。
- 第一步优先试与持久化更接近的命令行/路径关键词，而不是直接全局扫 `CreateProcess`。

优先字段：
- 主体：`process`、`process_name`、`process_id`、`command_line`
- 新进程：`new_process_name`、`newprocess`、`newcommandline`、`newprocessuuid`
- 父链：`parent_process*`、`grand1_*`、`grand2_*`
- 哈希：`md5`、`sha1`、`processmd5`、`processsha1`
- 签名：`signing_info`、`new_signing_info`、`signed`、`signerexex`、`signerserialex`
- PE 信息：`pe.*`、`process_pe.*`
- 权限/风险：`processintegrity`、`processsigned`、`processisnetworked`、`processlevel`、`newprocesslevel`、`newimagelevel`、`rpcprocesslevel`
- 文件访问：`file`、`directory`、`fltrid`、`fltrname`
- 线程/RPC：`thread.*`、`rpc*`、`callstack`

等级字段怎么用：
- `processlevel` / `newprocesslevel` / `newimagelevel` / `rpcprocesslevel`：都是同一套等级语义。
- `10-20` 先按可信对象看，`30` 视为未知，`40` 视为未入库，`50-70` 视为恶意或高风险优先排查对象。
- Windows 上如果高等级对象再叠加异常父进程、异常命令行、无效签名、敏感目录访问，优先级应继续上调。
- 不要把 detection 的 `threat_level`、事件分数误当成这里的进程等级。

怎么猎：
- 先按 `operation` 分桶，再按 `process_name` / `new_process_name` 看异常组合。
- 对 `CreateProcess`，优先看“父进程 -> 子进程 -> 命令行”。
- 对磁盘/文件访问，优先看 `file`、`directory`、`fltrname` 是否涉及物理盘、系统目录、启动项、敏感文件。
- 对可疑二进制，优先拉哈希、签名、PE 信息，看是不是系统签名、微软签名、公司名伪装。
- 对注入/线程/RPC 类线索，优先结合 `thread.*`、`rpc*`、`callstack` 和父进程链。

Windows 持久化 hunting 优先顺序：
- 先看注册表自启动：`Run`、`RunOnce`、`CurrentVersion\\Run`、`CurrentVersion\\RunOnce`。
- 再看计划任务：`schtasks`、`TaskScheduler`、任务 XML、计划任务目录。
- 再看服务：`sc create`、`services.exe`、服务安装/修改、服务相关二进制路径。
- 再看启动目录：`Startup`、开始菜单启动项、用户启动文件夹。
- 如果平台里有更贴近写入语义的 operation（如注册表写入、服务创建、文件写入），优先用这些；如果没有，再退回命令行和路径关键词。

Windows 持久化关键词建议：
- 注册表：`reg add`、`RunOnce`、`CurrentVersion\\Run`、`CurrentVersion\\RunOnce`
- 计划任务：`schtasks`、`TaskScheduler`、`\\Tasks\\`
- 服务：`sc create`、`CreateService`、`services.exe`
- 启动目录：`Startup`

Windows 持久化默认过滤思路：
- 优先 `os_type=windows`
- 再加时间范围
- 然后优先在 `command_line` / `newcommandline` / `file` / `process` 做 `contain` 试探关键词
- 只有当用户明确给了某个进程名或服务名时，再优先用 `is`

不要这样做：
- 不要一上来就只查 `operation is CreateProcess` 然后把系统正常进程创建当成“写自启动”结果。
- 不要因为看到 `svchost.exe`、`services.exe` 就直接下结论，必须结合命令行、文件路径、父子链和落点。

高价值 IOC：
- 可执行路径、DLL/模块路径、MD5/SHA1
- 父子进程链
- 命令行参数
- 签名信息、公司名、产品名
- 文件访问目标、过滤器名 `fltrname`

contain 用法：
- 可拿来试探 `process`、`command_line`、`file` 里的子串。
- 如果是明确进程名（如 `powershell.exe`、`cmd.exe`、`rundll32.exe`），优先用 `is`，不要先用 `contain`。
- 但命中后必须回看真实返回值，不要因为 contain 成功就默认相关。
