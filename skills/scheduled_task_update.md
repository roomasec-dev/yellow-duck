# Tool: scheduled_task_update

用途：
- 修改已有定时任务的频率、内容、状态。

适用场景：
- “把刚才那个改成一小时一次”
- “把任务暂停一下”
- “恢复那个定时任务”
- “把任务内容改成盯 incident”

输入建议：
- 优先带 `task_id`。
- 修改频率时填 `task_interval_minutes`。
- 暂停/恢复时可填 `task_status=paused|active`。
- 改任务内容时填 `task_prompt`。

规则：
- 用户没明确任务 id 但上下文里只有一个明显目标任务时，可以沿用那个任务。
