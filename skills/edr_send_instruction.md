# Tool: edr_send_instruction

用途：
- 向指定主机下发指令，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 client_id 和 instruction_name 明确无歧义。

输入说明：
- client_id：目标主机的客户端 ID，必填。
- instruction_name：指令名称，必填，如"list_ps"、"get_info"等。

输出建议：
- 成功后汇报任务 ID 和下发结果。
- 同时提醒用户这是高影响动作。
- 如果指令需要时间执行完成，建议用户稍后使用 edr_task_result 查询结果。
