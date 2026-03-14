# Tool: knowledge_base_*

用途：
- 把外部知识库目录里的 `.md` 文件暴露给主 agent 使用。
- 支持搜索、新增、编辑、删除知识库文件。

搜索规则：
- `knowledge_base_search` 会递归遍历 `knowledge_base.path` 下所有 markdown 文件。
- 优先按标题、相对路径、正文内容找最相关的片段。
- 面向用户回复时，优先给出命中文件标题、相对路径和关键片段。

写入规则：
- `knowledge_base_write` 默认用 `kb_mode=upsert`，适合整篇新建或整篇覆盖。
- 如果用户说“追加一段”“补充到末尾”，用 `kb_mode=append`。
- 如果用户明确说“把 A 改成 B”，用 `kb_mode=replace_text`，并填写 `kb_old_text` / `kb_new_text`。
- `kb_title` 可以写文件标题，也可以直接写相对路径，例如 `runbook/windows/rdp.md`。

删除规则：
- `knowledge_base_delete` 用于删除整篇知识库文件。
- 如果用户只说“删掉那篇/那条”，但当前上下文里还不能唯一定位文件，先搜索再删，不要猜。
