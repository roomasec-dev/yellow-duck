# Capability: edr_hunting

用途：
- 用行为日志做威胁狩猎，围绕进程链、父子进程、文件、模块、哈希、签名、线程、RPC、磁盘访问等线索扩分析。
- 适合回答“最近有没有某类行为”“某个 IOC 在哪些主机出现过”“这条事件还能往哪扩”。

意图理解：
- 如果用户说“写自启动”“持久化”“开机启动”“自启动项”“Run/RunOnce”“计划任务”“服务”“LaunchAgent/Daemon”“cron/systemd”，默认理解为在做持久化狩猎，不要先退化成普通进程创建排查。
- 持久化狩猎的第一目标是：找注册表自启动、计划任务、服务、启动目录、LaunchAgent/Daemon、cron/systemd 这类落点或相关写入行为，而不是泛泛看所有 CreateProcess / exec。
- 用户没给明确主机时，也先做一轮聚焦持久化线索的样本查询，不要先反问用户要不要继续。

先怎么做：
- 第一步先定范围：优先按 `client_id`、`os_type`、`operation` 缩小目标。
- 如果用户给了时间范围，再把范围限制到 `start_time` / `end_time`。
- 如果用户在查持久化 / 自启动，优先先定 `os_type` 和时间，再把关键词落到 `command_line`、`newcommandline`、`file`、`process` 等字段上。
- 第二步找主体：先看 `process`、`process_name`、`command_line`。
- 第三步拉进程链：补 `parent_process*`、`grand1_*`、`grand2_*`、`process_rootprocess*`。
- 第四步抽 IOC：文件路径、哈希、命令行、签名、模块、线程/RPC 线索。
- 第五步再扩：同 `client_id`、同 `md5/sha1`、同 `processuuid/newprocessuuid`、同父进程链。

过滤建议：
- 已验证 `operator=is` 可用，适合精确匹配。
- `operator=contain` 接口会接受，但当前真实返回表现不稳定，容易出现和查询词不完全相关的结果。
- 所以 `contain` 只能当“模糊试探”来用，命中后必须再人工核对字段值，不能直接当成强结论。
- 真正下结论时，优先回到 `is`、明确 IOC、明确 `client_id` 或明确进程链。
- 如果用户已经给了明确进程名、操作名、系统类型，例如 `bash`、`powershell.exe`、`exec`、`CreateProcess`，优先用 `is`，不要先用 `contain`。
- `contain` 更适合拿来试探路径片段、命令行片段、模块名片段、目录片段。
- 如果用户提到时间，优先先按时间切片，再做其他过滤，避免在全局海量日志里直接 hunting。
- 如果用户在找“写自启动”，不要优先用 `operation is CreateProcess` 这种过宽条件；应优先试和持久化更接近的条件：注册表/计划任务/服务/启动目录相关关键词，或更贴近写入语义的 operation。
- 如果平台日志里没有直接给出注册表键名字段，也要优先从 `command_line`、`newcommandline`、`file` 里找 `reg add`、`RunOnce`、`CurrentVersion\\Run`、`schtasks`、`TaskScheduler`、`sc create`、`services.exe`、`Startup`、`LaunchAgents`、`LaunchDaemons`、`cron`、`systemd` 等落点线索。

已验证的基础字段族：
- 资产与定位：`client_id`、`client_ip`、`os_type`、`hosttype`、`account`、`domain`、`sid`
- 事件元信息：`alert_name`、`operation`、`datasource`、`detection_source`、`timestamp`、`time`
- 当前进程：`process`、`process_name`、`process_id`、`command_line`、`processuuid`、`processmd5`、`processsha1`
- 父子进程：`parent_process*`、`grand1_*`、`grand2_*`、`process_rootprocess*`
- 新建进程：`new_process_name`、`newprocess`、`newcommandline`、`newprocessuuid`
- 文件/模块/哈希：`file`、`directory`、`module`、`md5`、`sha1`
- 签名：`signing_info`、`new_signing_info`、`signed`
- 线程/RPC：`thread.*`、`rpc*`、`callstack`

常用字段说明：
- `process`：当前事件主体进程的完整路径。
- `process_name`：当前事件主体进程名，适合先做快速分桶。
- `command_line`：当前主体进程命令行，适合看参数、落地路径、网络目标、脚本内容。
- `parent_process` / `parent_process_name`：父进程完整路径 / 进程名。
- `grand1_*`、`grand2_*`：更上层祖先进程，用来还原完整进程链。
- `newprocess` / `new_process_name`：当前主体派生出来的新进程路径 / 进程名。
- `newcommandline`：新进程命令行，适合看二次拉起、下载执行、脚本执行。
- `processuuid`、`newprocessuuid`、`parent_processuuid`：进程实例级唯一标识，适合做链路串联。
- `md5` / `sha1`：当前事件对象哈希；`processmd5` / `processsha1` 更偏主体进程自身哈希。
- `signing_info` / `new_signing_info`：签名详情，适合判断是否为系统签名、厂商签名、开发者签名。
- `datasource`：事件来源类型，例如进程创建、文件访问、线程、RPC 等，适合区分事件语义。
- `operation`：行为动作名，适合做第一层筛选，例如 `exec`、`CreateProcess`。

进程等级字段说明：
- `processlevel`：主体进程等级。
- `newprocesslevel`：新建/子进程等级。
- `newimagelevel`：新镜像/新落地对象等级。
- `rpcprocesslevel`：RPC 相关进程等级。
- 这些等级字段可按同一套语义理解：`10-20` 通常表示可信，`30` 表示未知，`40` 表示未入库，`50-70` 表示恶意程序或高风险对象。
- hunting 时看到 `50-70`，应优先把该进程、其父子链、哈希、签名、落地路径列为高优先级 IOC。
- 如果是 `30` 或 `40`，不要直接下恶意结论，应结合签名、哈希、父子进程链、命令行再判断。

不要混淆的等级：
- `threat_level`、事件 score、处置状态等字段，不等于进程等级。
- 只有 `processlevel` / `newprocesslevel` / `newimagelevel` / `rpcprocesslevel` 才使用 10-70 这套可信/未知/未入库/恶意语义。

输出建议：
- 先给一句 hunting 结论。
- 再给简单进程链：谁启动了谁，谁派生了谁。
- 再给 IOC：路径、哈希、命令行、签名、主机范围。
- 最后说下一步该扩什么。

平台分流：
- Windows 狩猎重点见 `edr_hunting_windows.md`
- Linux 狩猎重点见 `edr_hunting_linux.md`
- macOS 狩猎重点见 `edr_hunting_macos.md`

## 预设狩猎条件

当用户说"日志调查"、"hunting"、"狩猎"时，系统会询问用户选择预设条件：

1. 检测未知进程启动 - processlevel 在 30-70 之间的未知进程
2. 检测可疑进程创建计划任务 - schtasks.exe create
3. 检索通过 wmic 创建进程 - wmic.exe create process
4. 检索可疑 PowerShell 命令 - 包含 FromBase64String
5. 检索 cmd 命令输入 - fltrid: 8000
6. 检索 lsass 内存访问 - lsass.exe 访问
7. 检索未知进程查询计算机名称 - QueryValueKey
8. 检索进程白利用 - 低等级进程加载高风险 DLL
9. 不使用条件查询 - 直接查询指定时间范围内的日志（默认最近15分钟）
10. 自定义条件查询 - 用户输入 KQL 语句
