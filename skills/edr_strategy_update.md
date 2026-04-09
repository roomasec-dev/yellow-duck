# Tool: edr_strategy_update

用途：
- 更新查杀设置/病毒扫描策略，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认策略 ID 明确无歧义。

输入说明：
- strategy_id：**必填，当用户明确说明想要更改查杀设置时，请用户指定 id，或者从上文的策略查询结果（pending_high_risk 或 tool_result）中提取对应的 id 字段，禁止编造。**
- type：策略类型，必填，支持以下值：
  - `virus_scan_settings`：病毒查杀设置
  - `asset_registration`：资产登记
- scan_file_scope：查杀范围，可选值：`all`（全部）、`recommended`（推荐）。
- startup_scan_mode：启动查杀模式，可选值：`known_dangerous`（已知威胁）、`all_unknown`（全部未知）。
- archive_size_limit_enabled：压缩包大小限制开关，可选值：`true`（自定义大小）、`false`（不限大小）。
- archive_size_limit：压缩包大小限制值，单位 MB，仅当 archive_size_limit_enabled=true 时生效，默认 40。
- realtime_mem_cache_tech_enabled：实时预防启动开关，可选值：`true`、`false`。
- dynamic_cpu_monitor_enabled：扫描启动CPU自动避让开关，可选值：`true`、`false`。
- dynamic_cpu_high_percent：CPU 利用率阈值，范围 1-100，仅当 dynamic_cpu_monitor_enabled=true 时生效。
- stop_realtime_on_cpu_high_enabled：实时防护CPU避让开关，可选值：`true`、`false`。
- stop_realtime_cpu_high_percent：实时防护CPU阈值，范围 1-100，仅当 stop_realtime_on_cpu_high_enabled=true 时生效。
- owl_on_realtime_enabled：实时预防启动猫头鹰，可选值：`true`、`false`。
- realtime_scan_archive_enabled：实时防护扫描压缩包，可选值：`true`、`false`。
- runtime_max_file_size_mb：实时防护文件大小限制，单位 MB，默认 80。
- custom_max_file_size_mb：扫描文件大小限制，单位 MB，默认 80。

**重要**：当用户要修改上述参数为新值时，需要从上下文中：
1. 提取 strategy_id（上文中的 id 字段）
2. 根据用户描述识别出用户要修改的参数名（如 scan_file_scope，startup_scan_mode， archive_size_limit 等等）
3. 填入用户指定的新值

输出建议：
- 成功后汇报策略 ID 和更新的内容。
- 同时提醒用户这是高影响动作。

