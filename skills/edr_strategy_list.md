# Tool: edr_strategy_list

用途：
- 获取策略列表，适合回答"有哪些策略""策略的详细信息"。

输入说明：
- type：策略类型，必填，支持 `virus_scan_settings`、`asset_registration` 等。
- page：页码，默认 1。
- limit：每页数量，默认 20。
- name：策略名称，可选，用于过滤。
- status：状态，可选，1=启用，0=禁用。
- range_type：范围类型，可选，1=全局，2=主机组。

输出建议：
- 优先展示策略 ID、名称、类型、状态、范围类型、创建时间。
- 如果结果为空，说明当前没有匹配的策略。
