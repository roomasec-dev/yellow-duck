# Tool: edr_strategy_get_default

用途：
- 获取默认策略，适合回答"某个类型的默认策略是什么"。

输入说明：
- strategy_id：策略 ID，必填。
- type：策略类型，必填，支持 `virus_scan_settings`、`asset_registration` 等。

输出建议：
- 优先展示默认策略的名称、类型、状态、内容、配置信息。
- 如果结果为空，说明当前没有该类型的默认策略。
