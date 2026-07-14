# Hooks 系统

## Hook 类型

- **PreToolUse**：工具执行前（验证、参数修改）
- **PostToolUse**：工具执行后（自动格式化、检查）
- **Stop**：会话结束时（最终验证）

## 自动接受权限

谨慎使用：
- 对可信、定义明确的计划启用
- 探索性工作时禁用
- 永远不使用 dangerously-skip-permissions 标志
- 改为在 `~/.claude.json` 中配置 `allowedTools`

## TodoWrite 最佳实践

使用 TodoWrite 工具来：
- 跟踪多步骤任务的进度
- 验证对指令的理解
- 支持实时调整方向
- 展示细粒度的实现步骤

待办列表能揭示：
- 乱序的步骤
- 遗漏的事项
- 多余不必要的事项
- 粒度不合适
- 误解的需求
