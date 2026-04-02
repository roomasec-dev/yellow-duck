# Tool: edr_ioc_update

用途：
- 更新威胁情报指标（IOC），属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 IOC ID 明确无歧义。

输入说明：
- ioc_id：IOC ID，必填，用于指定要更新的 IOC。
- ioc_hash：IOC Hash，必填。
- ioc_action：新的动作类型，可选。
- ioc_description：新的描述，可选。
- ioc_expiration_date：新的过期时间，可选。

输出建议：
- 成功后汇报 IOC ID 和更新的内容。
- 如果只更新了部分字段，要说明保留了哪些原有值。
