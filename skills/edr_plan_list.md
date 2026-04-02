# Tool: edr_plan_list

用途：
- 查询计划列表，适合回答"有哪些计划""计划的状态"。

输入建议：
- 支持分页查询，page 和 page_size 一起规划。
- 如果用户提到第几页、下一页、每页多少条，要把 page 和 page_size 一起规划出来。
- 如果用户只说"下一页"，优先根据最近一次计划列表的页码继续翻页。
- type 参数用于区分业务类型：kill_plan(查杀计划), leak_repair(漏洞修复), distribute_software(分发软件), distribute_file(发送文件)

输入说明：
- scan_type：扫描类型，1-快速扫描 2-全盘扫描 3-自定义路径扫描 4-漏洞修复 5-安装软件 6-卸载软件 7-更新软件 8-发送文件
- plan_type：执行方式，1-立即执行 2-定时执行 3-周期执行
- type：业务类型，必填，kill_plan/leak_repair/distribute_software/distribute_file
- search_content：搜索内容
- page：页码，必填
- limit：每页数量，必填

输出建议：
- 优先展示计划名称、计划类型、执行方式、范围、创建时间、状态。
- contents 字段是 JSON 字符串（如 "null" 或 "{\"scan_path\":[]}"}），如需查看详情需要解析。
- 按最新创建时间优先理解结果。
- 回复里要保留当前页、总页数、是否还有下一页，方便继续翻页。
