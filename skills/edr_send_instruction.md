# Tool: edr_send_instruction

用途：
- 向指定主机下发指令，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 client_id 和 instruction_name 明确无歧义。

输入说明：
- client_id：目标主机的客户端 ID，必填。
- instruction_name：指令名称，必填。支持以下指令：
  - list_ps：进程列表
  - quarantine_file：隔离文件
  - recover_file：恢复文件
  - quarantine_network：隔离主机
  - recover_network：恢复主机
  - kill_ps：结束进程
  - process_analyze：查看进程分析
  - image_analyze：查看模块分析
  - process_dump：下载进程 dump
  - batch_quarantine_file：批量隔离文件
  - batch_kill_ps：批量结束进程
  - get_suspicious_file：获取可疑文件
  - full_malware_scan：全盘扫描
  - quick_malware_scan：快速扫描
  - custom_malware_scan：自定义扫描

输出建议：
- 成功后汇报任务 ID 和下发结果。
- 同时提醒用户这是高影响动作。
- 如果指令需要时间执行完成，建议用户稍后使用 edr_task_result 查询结果。
