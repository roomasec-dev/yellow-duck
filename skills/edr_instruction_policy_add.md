# Tool: edr_instruction_policy_add

用途：
- 创建自动响应策略，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 name、scope、policy_type 明确无歧义。

输入说明：
- name：策略名称，必填
- policy_type：策略类型，必填，1-内置策略 2-自定义策略
- scope：范围，必填，1-特定主机 2-主机组 3-全网
- client_id：特定主机ID（scope=1时使用）
- group_ids：主机组ID列表（scope=2时使用）
- scope_content：范围内容
- action：动作列表，如 [1]隔离网络 [2]智能响应
- condition_list：条件列表，复杂结构由系统根据用户需求构造

输出建议：
- 成功后汇报策略名称和策略ID。
- 同时提醒用户这是高影响动作。
