# Tool: edr_plan_edit

用途：
- 编辑计划，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 rid（计划ID）明确无歧义。

输入说明：
- rid：计划ID，必填，用于指定要编辑的计划
- scan_type：扫描/操作类型，可选，1-快速扫描 2-全盘扫描 3-自定义路径扫描 4-漏洞修复 5-安装软件 6-卸载软件 7-更新软件 8-发送文件
- plan_type：执行方式，可选，1-立即执行 2-定时执行 3-周期执行
- scope：范围，可选，1-特定主机 2-主机组 3-全网主机
- type：业务类型，可选，kill_plan/leak_repair/distribute_software/distribute_file
- plan_name：计划名称
- contents：内容对象 (scan_path/software/file等)
- execute_start_time：执行开始时间
- execute_cycle：执行周期
- repeat_cycle：重复周期
- execution_time：执行时间 hh:mm
- group_ids：主机组
- device_client_ids：主机id数组

输出建议：
- 成功后汇报计划ID和更新的内容。
- 如果只更新了部分字段，要说明保留了哪些原有值。
