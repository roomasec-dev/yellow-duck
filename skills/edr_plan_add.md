# Tool: edr_plan_add

用途：
- 创建计划，属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 plan_name、scan_type、plan_type、scope、type 明确无歧义。

输入说明：
- scan_type：扫描/操作类型，必填，1-快速扫描 2-全盘扫描 3-自定义路径扫描 4-漏洞修复 5-安装软件 6-卸载软件 7-更新软件 8-发送文件
- plan_type：执行方式，必填，1-立即执行 2-定时执行 3-周期执行
- scope：范围，必填，1-特定主机 2-主机组 3-全网主机
- type：业务类型，必填，kill_plan/leak_repair/distribute_software/distribute_file
- plan_name：计划名称
- client_id：当 scope=1(特定主机) 时必填，指定单台主机
- contents：内容对象 (scan_path/software/file等)，格式为 JSON 字符串
- execute_start_time：执行开始时间
- execute_cycle：执行周期 1:每天 2:每周 3:每月
- repeat_cycle：重复周期 周0-6 月1-31
- execution_time：执行时间 hh:mm
- group_ids：主机组
- device_client_ids：主机id数组
- expired_setting：过期设置 0:永不过期 1:指定过期时间
- expired_time：过期时间

输出建议：
- 成功后汇报计划名称和计划ID。
- 同时提醒用户这是高影响动作。
