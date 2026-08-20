---
name: go-testing
description: 强制执行 Go TDD 工作流，先写表驱动测试再实现代码，确保覆盖率 80%+。涵盖表驱动、基准、模糊测试等惯用模式。
origin: ECC
---

# Go 测试与 TDD 模式

遵循 TDD 方法论编写可靠、可维护测试的全面 Go 测试模式。强制执行 RED-GREEN-REFACTOR 循环，确保代码质量与测试覆盖率。

## 激活时机

- 实现新 Go 函数或方法时（必须先写测试）
- 为已有代码补充测试覆盖时
- 修复 Bug 时（先写失败测试复现问题）
- 构建核心业务逻辑时
- 为性能关键代码创建基准测试时
- 为输入验证实现模糊测试时
- 学习 Go 中的 TDD 工作流时

## 测试要求

- **所有修改/新增的代码，必须以「函数」为基本单元编写单元测试**：
- 未完成测试的代码，不得视为「完成」
- 测试文件位置:
    - 同包单元测试: 在被测代码所在目录下创建 `*_test.go`（可访问被测包的未导出成员）
    - 跨包/集成验证: 统一放在 `test/` 目录（独立包, 无法访问被测包的未导出成员）
- 测试case要覆盖基本的场景和异常场景
- 测试函数格式严格为:

```golang
func TestXxx(t *testing.T) {
    // 被测函数或代码逻辑块, 批量测试case
}
```

## TDD 强制工作流

### 执行步骤（必须按顺序）

1. **定义类型/接口**：先搭建函数签名，用 `panic("尚未实现")` 占位
2. **编写表驱动测试**：创建全面的测试用例（含边界、错误路径）
3. **运行测试（RED）**：验证测试因正确原因失败
4. **实现代码（GREEN）**：编写最少量代码使测试通过
5. **重构（REFACTOR）**：在保持测试通过的前提下改进代码
6. **检查覆盖率**：确保覆盖率达到目标（通用代码 80%+，公开 API 90%+，核心逻辑 100%）

### RED-GREEN-REFACTOR 循环

```
RED     → 编写失败的表驱动测试
GREEN   → 实现最少量代码使测试通过
REFACTOR → 改进代码，测试保持通过
REPEAT  → 下一个测试用例
```

### TDD 逐步指南（基础示例）

```go
// 第 1 步：定义接口/签名
// calculator.go
package calculator

func Add(a, b int) int {
    panic("尚未实现") // 占位符
}

// 第 2 步：编写失败测试（RED）
// calculator_test.go
package calculator

import "testing"

func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"正数相加", 2, 3, 5},
        {"负数相加", -1, -2, -3},
        {"零值", 0, 0, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.expected {
                t.Errorf("Add(%d, %d) = %d; 期望 %d", tt.a, tt.b, got, tt.expected)
            }
        })
    }
}

// 第 3 步：运行测试 —— 验证 RED 阶段
// $ go test ./...
// --- FAIL: TestAdd/正数相加 (0.00s)
// panic: 尚未实现

// 第 4 步：实现最少量代码（GREEN）
func Add(a, b int) int {
    return a + b
}

// 第 5 步：运行测试 —— 验证 GREEN 阶段
// $ go test ./...
// PASS

// 第 6 步：检查覆盖率
// $ go test -cover ./...
// coverage: 100.0%
```

### TDD 完整示例：邮箱验证器

**目标：实现 ValidateEmail 函数**

#### 第 1 步：定义接口

```go
// validator/email.go
package validator

// ValidateEmail 检查给定字符串是否是有效的邮箱地址。
// 有效返回 nil，否则返回描述问题的错误。
func ValidateEmail(email string) error {
    panic("尚未实现")
}
```

#### 第 2 步：编写表驱动测试（RED）

```go
// validator/email_test.go
package validator

import "testing"

func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        // 有效邮箱
        {"简单邮箱", "user@example.com", false},
        {"带子域名", "user@mail.example.com", false},
        {"带加号标签", "user+tag@example.com", false},
        {"带点号本地部分", "first.last@example.com", false},

        // 无效邮箱
        {"空字符串", "", true},
        {"无 @ 符号", "userexample.com", true},
        {"无域名", "user@", true},
        {"无本地部分", "@example.com", true},
        {"双 @ 符号", "user@@example.com", true},
        {"含空格", "user @example.com", true},
        {"无顶级域名", "user@example", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if tt.wantErr && err == nil {
                t.Errorf("ValidateEmail(%q) = nil；期望错误", tt.email)
            }
            if !tt.wantErr && err != nil {
                t.Errorf("ValidateEmail(%q) = %v；期望 nil", tt.email, err)
            }
        })
    }
}
```

#### 第 3 步：运行测试 —— 验证 RED

```bash
$ go test ./validator/...

--- FAIL: TestValidateEmail (0.00s)
    --- FAIL: TestValidateEmail/简单邮箱 (0.00s)
        panic: 尚未实现

FAIL
```

✓ 测试如预期失败（panic）。

#### 第 4 步：实现最少量代码（GREEN）

```go
// validator/email.go
package validator

import (
    "errors"
    "regexp"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

var (
    ErrEmailEmpty   = errors.New("邮箱不能为空")
    ErrEmailInvalid = errors.New("邮箱格式无效")
)

func ValidateEmail(email string) error {
    if email == "" {
        return ErrEmailEmpty
    }
    if !emailRegex.MatchString(email) {
        return ErrEmailInvalid
    }
    return nil
}
```

#### 第 5 步：运行测试 —— 验证 GREEN

```bash
$ go test ./validator/...

PASS
ok      project/validator    0.003s
```

✓ 所有测试通过！

#### 第 6 步：检查覆盖率

```bash
$ go test -cover ./validator/...

PASS
coverage: 100.0% of statements
ok      project/validator    0.003s
```

✓ 覆盖率：100%，TDD 完成！

### Bug 修复的 TDD 模式

修复 Bug 时必须遵循：

1. **先写失败测试**：用测试复现 Bug，确认测试失败
2. **运行测试**：验证 RED（Bug 存在）
3. **修复 Bug**：修改代码
4. **运行测试**：验证 GREEN（Bug 已修复）
5. **回归测试**：确保原有测试全部通过

## 表驱动测试

Go 测试的标准模式。以最少代码实现全面覆盖。

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"正数", 2, 3, 5},
        {"负数", -1, -2, -3},
        {"零值", 0, 0, 0},
        {"正负混合", -1, 1, 0},
        {"大数", 1000000, 2000000, 3000000},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.expected {
                t.Errorf("Add(%d, %d) = %d; 期望 %d",
                    tt.a, tt.b, got, tt.expected)
            }
        })
    }
}
```

### 含错误用例的表驱动测试

```go
func TestParseConfig(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Config
        wantErr bool
    }{
        {
            name:  "有效配置",
            input: `{"host": "localhost", "port": 8080}`,
            want:  &Config{Host: "localhost", Port: 8080},
        },
        {
            name:    "无效 JSON",
            input:   `{invalid}`,
            wantErr: true,
        },
        {
            name:    "空输入",
            input:   "",
            wantErr: true,
        },
        {
            name:  "最小配置",
            input: `{}`,
            want:  &Config{}, // 零值配置
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseConfig(tt.input)

            if tt.wantErr {
                if err == nil {
                    t.Error("期望错误，但得到 nil")
                }
                return
            }

            if err != nil {
                t.Fatalf("意外错误: %v", err)
            }

            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %+v; 期望 %+v", got, tt.want)
            }
        })
    }
}
```

## 子测试和子基准测试

### 组织相关测试

```go
func TestUser(t *testing.T) {
    // 所有子测试共享的 setup
    db := setupTestDB(t)

    t.Run("Create", func(t *testing.T) {
        user := &User{Name: "Alice"}
        err := db.CreateUser(user)
        if err != nil {
            t.Fatalf("CreateUser 失败: %v", err)
        }
        if user.ID == "" {
            t.Error("期望用户 ID 已设置")
        }
    })

    t.Run("Get", func(t *testing.T) {
        user, err := db.GetUser("alice-id")
        if err != nil {
            t.Fatalf("GetUser 失败: %v", err)
        }
        if user.Name != "Alice" {
            t.Errorf("got name %q; 期望 %q", user.Name, "Alice")
        }
    })

    t.Run("Update", func(t *testing.T) {
        // ...
    })

    t.Run("Delete", func(t *testing.T) {
        // ...
    })
}
```

### 并行子测试

```go
func TestParallel(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"用例1", "input1"},
        {"用例2", "input2"},
        {"用例3", "input3"},
    }

    for _, tt := range tests {
        tt := tt // 捕获范围变量
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // 并行运行子测试
            result := Process(tt.input)
            // 断言...
            _ = result
        })
    }
}
```

## 测试辅助函数

### 辅助函数

```go
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper() // 标记为辅助函数

    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("打开数据库失败: %v", err)
    }

    // 测试结束时清理
    t.Cleanup(func() {
        db.Close()
    })

    // 执行 migration
    if _, err := db.Exec(schema); err != nil {
        t.Fatalf("创建 schema 失败: %v", err)
    }

    return db
}

func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("意外错误: %v", err)
    }
}

func assertEqual[T comparable](t *testing.T, got, want T) {
    t.Helper()
    if got != want {
        t.Errorf("got %v; 期望 %v", got, want)
    }
}
```

### 临时文件和目录

```go
func TestFileProcessing(t *testing.T) {
    // 创建临时目录 —— 自动清理
    tmpDir := t.TempDir()

    // 创建测试文件
    testFile := filepath.Join(tmpDir, "test.txt")
    err := os.WriteFile(testFile, []byte("测试内容"), 0644)
    if err != nil {
        t.Fatalf("创建测试文件失败: %v", err)
    }

    // 运行测试
    result, err := ProcessFile(testFile)
    if err != nil {
        t.Fatalf("ProcessFile 失败: %v", err)
    }

    // 断言...
    _ = result
}
```

## Golden 文件测试

对存储在 `testdata/` 中的预期输出文件进行测试。

```go
var update = flag.Bool("update", false, "更新 golden 文件")

func TestRender(t *testing.T) {
    tests := []struct {
        name  string
        input Template
    }{
        {"简单", Template{Name: "test"}},
        {"复杂", Template{Name: "test", Items: []string{"a", "b"}}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Render(tt.input)

            golden := filepath.Join("testdata", tt.name+".golden")

            if *update {
                // 更新 golden 文件：go test -update
                err := os.WriteFile(golden, got, 0644)
                if err != nil {
                    t.Fatalf("更新 golden 文件失败: %v", err)
                }
            }

            want, err := os.ReadFile(golden)
            if err != nil {
                t.Fatalf("读取 golden 文件失败: %v", err)
            }

            if !bytes.Equal(got, want) {
                t.Errorf("输出不匹配:\ngot:\n%s\nwant:\n%s", got, want)
            }
        })
    }
}
```

## 基于接口的 Mock

### 接口 Mock

```go
// 为依赖定义接口
type UserRepository interface {
    GetUser(id string) (*User, error)
    SaveUser(user *User) error
}

// 生产实现
type PostgresUserRepository struct {
    db *sql.DB
}

func (r *PostgresUserRepository) GetUser(id string) (*User, error) {
    // 真实数据库查询
}

// 测试用的 Mock 实现
type MockUserRepository struct {
    GetUserFunc  func(id string) (*User, error)
    SaveUserFunc func(user *User) error
}

func (m *MockUserRepository) GetUser(id string) (*User, error) {
    return m.GetUserFunc(id)
}

func (m *MockUserRepository) SaveUser(user *User) error {
    return m.SaveUserFunc(user)
}

// 使用 Mock 测试
func TestUserService(t *testing.T) {
    mock := &MockUserRepository{
        GetUserFunc: func(id string) (*User, error) {
            if id == "123" {
                return &User{ID: "123", Name: "Alice"}, nil
            }
            return nil, ErrNotFound
        },
    }

    service := NewUserService(mock)

    user, err := service.GetUserProfile("123")
    if err != nil {
        t.Fatalf("意外错误: %v", err)
    }
    if user.Name != "Alice" {
        t.Errorf("got name %q; 期望 %q", user.Name, "Alice")
    }
}
```

## 基准测试

### 基础基准测试

```go
func BenchmarkProcess(b *testing.B) {
    data := generateTestData(1000)
    b.ResetTimer() // 不计算 setup 时间

    for i := 0; i < b.N; i++ {
        Process(data)
    }
}

// 运行：go test -bench=BenchmarkProcess -benchmem
// 输出：BenchmarkProcess-8   10000   105234 ns/op   4096 B/op   10 allocs/op
```

### 不同规模的基准测试

```go
func BenchmarkSort(b *testing.B) {
    sizes := []int{100, 1000, 10000, 100000}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            data := generateRandomSlice(size)
            b.ResetTimer()

            for i := 0; i < b.N; i++ {
                // 复制以避免对已排序数据再次排序
                tmp := make([]int, len(data))
                copy(tmp, data)
                sort.Ints(tmp)
            }
        })
    }
}
```

### 内存分配基准测试

```go
func BenchmarkStringConcat(b *testing.B) {
    parts := []string{"hello", "world", "foo", "bar", "baz"}

    b.Run("plus", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            var s string
            for _, p := range parts {
                s += p
            }
            _ = s
        }
    })

    b.Run("builder", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            var sb strings.Builder
            for _, p := range parts {
                sb.WriteString(p)
            }
            _ = sb.String()
        }
    })

    b.Run("join", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = strings.Join(parts, "")
        }
    })
}
```

## 模糊测试（Go 1.18+）

### 基础模糊测试

```go
func FuzzParseJSON(f *testing.F) {
    // 添加种子语料库
    f.Add(`{"name": "test"}`)
    f.Add(`{"count": 123}`)
    f.Add(`[]`)
    f.Add(`""`)

    f.Fuzz(func(t *testing.T, input string) {
        var result map[string]interface{}
        err := json.Unmarshal([]byte(input), &result)

        if err != nil {
            // 随机输入导致的无效 JSON 是预期的
            return
        }

        // 如果解析成功，重新编码也应该成功
        _, err = json.Marshal(result)
        if err != nil {
            t.Errorf("Unmarshal 成功后 Marshal 失败: %v", err)
        }
    })
}

// 运行：go test -fuzz=FuzzParseJSON -fuzztime=30s
```

### 多输入的模糊测试

```go
func FuzzCompare(f *testing.F) {
    f.Add("hello", "world")
    f.Add("", "")
    f.Add("abc", "abc")

    f.Fuzz(func(t *testing.T, a, b string) {
        result := Compare(a, b)

        // 属性：Compare(a, a) 应始终等于 0
        if a == b && result != 0 {
            t.Errorf("Compare(%q, %q) = %d; 期望 0", a, b, result)
        }

        // 属性：Compare(a, b) 和 Compare(b, a) 应符号相反
        reverse := Compare(b, a)
        if (result > 0 && reverse >= 0) || (result < 0 && reverse <= 0) {
            if result != 0 || reverse != 0 {
                t.Errorf("Compare(%q, %q) = %d, Compare(%q, %q) = %d; 结果不一致",
                    a, b, result, b, a, reverse)
            }
        }
    })
}
```

## 测试覆盖率

### 运行覆盖率

```bash
# 基础覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 在浏览器中查看覆盖率
go tool cover -html=coverage.out

# 按函数查看覆盖率
go tool cover -func=coverage.out

# 带竞态检测的覆盖率
go test -race -coverprofile=coverage.out ./...
```

### 覆盖率目标

| 代码类型 | 目标 |
|----------|------|
| 核心业务逻辑 | 100% |
| 公开 API | 90%+ |
| 通用代码 | 80%+ |
| 生成代码 | 排除 |

### 从覆盖率中排除生成代码

```go
//go:generate mockgen -source=interface.go -destination=mock_interface.go

// 在覆盖率报告中通过 build tag 排除：
// go test -cover -tags=!generate ./...
```

## HTTP Handler 测试

```go
func TestHealthHandler(t *testing.T) {
    // 创建请求
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    w := httptest.NewRecorder()

    // 调用 handler
    HealthHandler(w, req)

    // 检查响应
    resp := w.Result()
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("got status %d; 期望 %d", resp.StatusCode, http.StatusOK)
    }

    body, _ := io.ReadAll(resp.Body)
    if string(body) != "OK" {
        t.Errorf("got body %q; 期望 %q", body, "OK")
    }
}

func TestAPIHandler(t *testing.T) {
    tests := []struct {
        name       string
        method     string
        path       string
        body       string
        wantStatus int
        wantBody   string
    }{
        {
            name:       "获取用户",
            method:     http.MethodGet,
            path:       "/users/123",
            wantStatus: http.StatusOK,
            wantBody:   `{"id":"123","name":"Alice"}`,
        },
        {
            name:       "未找到",
            method:     http.MethodGet,
            path:       "/users/999",
            wantStatus: http.StatusNotFound,
        },
        {
            name:       "创建用户",
            method:     http.MethodPost,
            path:       "/users",
            body:       `{"name":"Bob"}`,
            wantStatus: http.StatusCreated,
        },
    }

    handler := NewAPIHandler()

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var body io.Reader
            if tt.body != "" {
                body = strings.NewReader(tt.body)
            }

            req := httptest.NewRequest(tt.method, tt.path, body)
            req.Header.Set("Content-Type", "application/json")
            w := httptest.NewRecorder()

            handler.ServeHTTP(w, req)

            if w.Code != tt.wantStatus {
                t.Errorf("got status %d; 期望 %d", w.Code, tt.wantStatus)
            }

            if tt.wantBody != "" && w.Body.String() != tt.wantBody {
                t.Errorf("got body %q; 期望 %q", w.Body.String(), tt.wantBody)
            }
        })
    }
}
```

## 测试命令

```bash
# 运行所有测试
go test ./...

# 详细输出运行测试
go test -v ./...

# 运行特定测试
go test -run TestAdd ./...

# 运行匹配模式的测试
go test -run "TestUser/Create" ./...

# 带竞态检测运行测试
go test -race ./...

# 带覆盖率运行测试
go test -cover -coverprofile=coverage.out ./...

# 只运行短测试
go test -short ./...

# 带超时运行测试
go test -timeout 30s ./...

# 运行基准测试
go test -bench=. -benchmem ./...

# 运行模糊测试
go test -fuzz=FuzzParse -fuzztime=30s ./...

# 统计测试运行次数（用于检测不稳定测试）
go test -count=10 ./...
```

## TDD 最佳实践

**必须遵守：**
- 在任何实现代码之前先写测试（严格 TDD）
- 每次改动后立即运行测试
- RED 阶段验证测试是因正确原因失败（不是编译错误或拼写错误）
- GREEN 阶段只写最少量的代码使测试通过
- 使用表驱动测试全面覆盖正常路径、边界条件、错误路径
- 测试行为，而非实现细节

**应当：**
- 包含边界用例（空值、nil、最大值、最小值）
- 在辅助函数中使用 `t.Helper()`
- 对独立测试使用 `t.Parallel()`
- 用 `t.Cleanup()` 清理资源
- 使用有意义的测试名称描述场景

**不应当：**
- 跳过 RED 阶段直接写实现
- 在测试之前写实现代码
- 直接测试私有函数（通过公开 API 测试）
- 在测试中使用 `time.Sleep()`（使用 channel 或条件变量）
- 忽视不稳定的测试（修复或删除它们）
- mock 所有东西（尽可能优先使用集成测试）
- 跳过错误路径测试

## 与 CI/CD 集成

```yaml
# GitHub Actions 示例
test:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.22'

    - name: 运行测试
      run: go test -race -coverprofile=coverage.out ./...

    - name: 检查覆盖率
      run: |
        go tool cover -func=coverage.out | grep total | awk '{print $3}' | \
        awk -F'%' '{if ($1 < 80) exit 1}'
```

## 与其他技能配合

- 实现完成后使用 `go-review` 进行代码审查
- 编码时参考 `go-coding` 中的惯用 Go 写法
- 测试无法通过编译时先排查构建问题

**记住**：测试就是文档。它们展示了代码该如何使用。清晰地编写测试，并保持其及时更新。严格遵守 TDD 流程：先 RED，再 GREEN，最后 REFACTOR。