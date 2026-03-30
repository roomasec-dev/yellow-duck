# Tool: edr_ioc_add

用途：
- 添加威胁情报指标（IOC），属于高危写操作。

风险要求：
- 必须等待用户明确回复"确认"后再执行。
- 执行前要确认 IOC 的哈希值、动作类型等关键参数明确无歧义。

输入说明：
- ioc_action：动作类型，必填，如"block"、"watch"等。
- ioc_hash：哈希值，必填，支持 MD5/SHA1/SHA256。
- ioc_description：描述，可选，说明为什么要添加这个 IOC。
- ioc_expiration_date：过期时间，可选，格式如"2024-12-31"。
- ioc_file_name：文件名，可选。
- ioc_host_type：主机类型，可选。

输出建议：
- 成功后汇报 IOC 的哈希值和添加结果。
- 同时提醒用户这是高影响动作。
