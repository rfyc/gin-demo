# Git 工作流

## 提交信息格式
```
<类型>: <描述>

<可选正文>
```

类型：feat、fix、refactor、docs、test、chore、perf、ci

注意：归因已通过 ~/.claude/settings.json 全局禁用。

## Pull Request 工作流

创建 PR 时：
1. 分析完整提交历史（不只是最新提交）
2. 使用 `git diff [base-branch]...HEAD` 查看所有变更
3. 撰写全面的 PR 摘要
4. 包含带 TODO 的测试计划
5. 新分支使用 `-u` 标志推送

> 完整开发流程（规划、TDD、代码审查）在 git 操作之前，
> 参见 [development-workflow.md](./development-workflow.md)。
