# Tool: edr_delete_isolate_files

用途：
- 删除隔离文件记录，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 GUID 明确无歧义。
- 此操作只是删除隔离文件记录，不是放行文件，文件本身仍处于隔离状态。
- 只能删除 recover_status = 1 的隔离文件。

输入说明：
- isolate_file_guids：隔离文件 GUID 列表，多个用逗号分隔，必填。

输出建议：
- 成功后汇报已删除的记录数量和 GUID。
- 提醒用户这只是删除记录，文件本身仍隔离，如需放行需使用 edr_release_isolate_files。
