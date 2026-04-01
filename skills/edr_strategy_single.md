# Tool: edr_strategy_single

用途：
- 获取单个策略（默认策略），适合回答"某个类型的策略当前配置是什么"。

输入说明：
- strategy_type：策略类型，必填，支持以下值：
  - `virus_scan_settings`：病毒扫描设置
  - `asset_registration`：资产登记

输出建议：
- 优先展示策略名称、类型、状态、内容、配置。
- 如果结果为空，说明当前没有默认策略配置。
