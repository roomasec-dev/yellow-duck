# Tool: edr_strategy_sort

用途：
- 对策略进行排序，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认策略 ID 列表和顺序明确无歧义。

输入说明：
- sort_ids：排序后的策略 ID 列表，必填，顺序即为最终排列顺序。
- type：策略类型，必填，支持 `virus_scan_settings`、`asset_registration` 等。

输出建议：
- 成功后汇报排序更新的策略数量。
