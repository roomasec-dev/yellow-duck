# Tool: edr_release_isolate_files

用途：
- 放行被隔离的文件，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 GUID 明确无歧义。
- 放行后文件将恢复正常访问，请确认该文件安全后再放行。

输入说明：
- isolate_file_guids：隔离文件 GUID 列表，多个用逗号分隔，必填。
- isolate_file_add_exclusion：是否同时添加为例外，可选。
- isolate_file_release_all_hash：是否放行同哈希的所有文件，可选。

输出建议：
- 成功后汇报已放行的文件数量。
- 如果同时添加为例外，要说明。
- 提醒用户这是高影响动作。
