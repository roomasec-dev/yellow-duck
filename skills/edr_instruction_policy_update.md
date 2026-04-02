# Tool: edr_instruction_policy_update

用途：
- 编辑自动响应策略，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 rid（策略ID）明确无歧义。

输入说明：
- rid：策略ID，必填，用于指定要编辑的策略
- name：策略名称，可选
- policy_type：策略类型，可选，1-内置策略 2-自定义策略
- scope：范围，可选，1-特定主机 2-主机组 3-全网
- client_id：特定主机ID
- group_ids：主机组ID列表
- scope_content：范围内容
- action：动作列表
- condition_list：条件列表

输出建议：
- 成功后汇报策略ID和更新的内容。
- 如果只更新了部分字段，要说明保留了哪些原有值。
