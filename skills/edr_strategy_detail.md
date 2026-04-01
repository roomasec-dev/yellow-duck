# Tool: edr_strategy_detail

用途：
- 获取策略详情，适合回答"某个策略的完整配置是什么"。

输入说明：
- strategy_id：策略 ID，必填。
- type：策略类型，必填，支持 `virus_scan_settings`、`asset_registration` 等。

输出建议：
- 优先展示策略名称、类型、状态、内容、配置信息、排除对象、范围类型等完整信息。
- 如果结果为空，说明当前没有该策略的详情。
