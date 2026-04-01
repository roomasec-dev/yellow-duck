# Tool: edr_ioa_add

用途：
- 添加 IOA（入侵检测规则），属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 IOA 的名称、严重程度等关键参数明确无歧义。

输入说明：
- kb_query：严重程度，必填，如"high"、"medium"、"low"。
- operation：命令行，可选。
- reason：描述，可选。
- ioc_file_name：文件名，可选。
- ioc_host_type：主机类型，可选。

输出建议：
- 成功后汇报添加的 IOA 名称和添加结果。
- 同时提醒用户这是高影响动作。
