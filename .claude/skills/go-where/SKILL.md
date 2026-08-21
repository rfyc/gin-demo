---
name: go-where
description: 确定当前项目可用的 go 命令路径(与 go.mod 版本一致),并将结果写入 MEMORY.md 供后续引用。激活时机
   需要运行 go 命令(go build/go test/go vet/go mod 等)前, 或询问"go 命令在哪/用什么 go 版本"时。
---

# Go 命令定位

在运行任何 go 命令之前,先确定本项目对应的 go 可执行文件,避免误用系统默认或其它项目的 go 版本。

## 激活时机

- 需要执行 go 命令(go build / go test / go vet / go mod tidy 等)时
- 用户询问 go 命令位置或 go 版本时

## 步骤

1. **读取 go.mod**,获取当前项目声明的 go 版本。

2. **确定 go 命令**,按以下优先级依次尝试:
   - `GOROOT` 环境变量:若已设置,go 命令为 `$GOROOT/bin/go`
   - `GOPATH` 环境变量:若已设置,go 命令为 `$GOPATH/go/bin/go`(若存在)
   - 本项目 IDE 配置:查看 `.idea/workspace.xml`(GoLand 的 GOROOT 配置)或 `.vscode/settings.json`(VSCode 的 go.goroot 配置)

3. **验证版本一致**:运行 `go version` 确认 go 命令版本与 go.mod 声明的版本一致(主版本即可,如 go 1.18)。若不一致,继续尝试下一个候选路径。

4. **写入 MEMORY.md**:将结果保存到用户 auto-memory 目录(由 Claude 记忆系统管理,路径形如 `~/.claude/projects/<项目路径slug>/memory/`,**不要**写到项目目录 `.claude/` 下)的 `go_command.md`,内容需标明:
   - 项目路径与项目名称(不同项目的 go 命令可能不同)
   - go 命令的绝对路径
   - go 版本与选择依据
   - 该命令仅适用于本项目,不能与其它项目混用
   并在该目录的 `MEMORY.md` 索引中追加一行指向 `go_command.md`(索引行控制在 150 字符内)。

5. **后续引用**:本次会话及后续会话中,执行 go 命令时直接使用 MEMORY.md 记录的路径,无需重复探测。

## 示例

```bash
# 读取 go.mod 声明的版本
cat go.mod

# 候选: GOROOT 环境变量
echo $GOROOT

# 候选: IDE 配置
grep -o 'GOROOT[^/]*' .idea/workspace.xml 2>/dev/null

# 验证版本
/Users/hong/go/go1.18.10/bin/go version
```
