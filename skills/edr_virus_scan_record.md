# Tool: edr_virus_scan_record

用途：
- 查询病毒扫描执行记录，适合回答"扫描记录""扫描历史""某次扫描的结果"。

输入建议：
- 支持分页查询，page 和 page_size 一起规划。
- 如果用户提到第几页、下一页、每页多少条，要把 page 和 page_size 一起规划出来。
- 可以通过 hostname、client_id、task_id、rid 等条件筛选。

输入说明：
- page：页码，必填
- limit：每页数量，必填
- rid：计划ID，可选
- task_id：任务ID，可选
- execution_batch：执行批次，可选
- host_name：主机名，可选
- client_id：主机ID，可选
- scan_type：扫描类型，可选，1-快速扫描 2-全盘扫描 3-自定义路径扫描
- status：状态，可选
- start_time：开始时间过滤，可选
- end_time：结束时间过滤，可选

输出建议：
- 优先展示主机名、扫描类型、状态、发现威胁数、清理威胁数。
- 如果结果为空，明确说"当前没有扫描记录"。
- 按最新创建时间优先理解结果。
- 回复里要保留当前页、总页数、是否还有下一页，方便继续翻页。
