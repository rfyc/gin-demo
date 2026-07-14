# Agent 编排

## 可用 Agent

位于 `~/.claude/agents/`：

| Agent | 用途 | 使用时机 |
|-------|------|----------|
| planner | 实现规划 | 复杂功能、重构 |
| architect | 系统设计 | 架构决策 |
| tdd-guide | 测试驱动开发 | 新功能、Bug 修复 |
| code-reviewer | 代码审查 | 编写代码后 |
| security-reviewer | 安全分析 | 提交前 |
| build-error-resolver | 修复构建错误 | 构建失败时 |
| e2e-runner | E2E 测试 | 关键用户流程 |
| refactor-cleaner | 清理死代码 | 代码维护 |
| doc-updater | 文档 | 更新文档 |
| rust-reviewer | Rust 代码审查 | Rust 项目 |

## 立即使用 Agent

无需用户提示，直接使用：
1. 复杂功能请求 —— 使用 **planner** agent
2. 刚刚编写/修改代码 —— 使用 **code-reviewer** agent
3. Bug 修复或新功能 —— 使用 **tdd-guide** agent
4. 架构决策 —— 使用 **architect** agent

## 并行任务执行

对独立操作**始终**使用并行 Task 执行：

```markdown
# 正确：并行执行
同时启动 3 个 agent：
1. Agent 1：分析 auth 模块的安全性
2. Agent 2：审查缓存系统的性能
3. Agent 3：检查工具函数的类型

# 错误：不必要的串行执行
先 agent 1，再 agent 2，最后 agent 3
```

## 多视角分析

对于复杂问题，使用分角色子 agent：
- 事实审查员
- 资深工程师
- 安全专家
- 一致性审查员
- 冗余检查员
