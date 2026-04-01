# Tool: edr_strategy_create

用途：
- 创建策略，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认策略名称、类型等关键参数明确无歧义。

输入说明：
- plan_name：策略名称，必填。
- instruction_name：策略类型，必填，支持 `virus_scan_settings`、`asset_registration` 等。
- scope：范围类型，必填，1=全局，2=主机组。

输出建议：
- 成功后汇报策略 ID 和创建结果。
- 同时提醒用户这是高影响动作。
