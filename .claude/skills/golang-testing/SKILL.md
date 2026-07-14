---
name: golang-testing
description: Go 测试模式，包括表驱动测试、子测试、基准测试、模糊测试和测试覆盖率。遵循 TDD 方法论和惯用 Go 实践。
origin: ECC
---

# Go 测试模式

遵循 TDD 方法论编写可靠、可维护测试的全面 Go 测试模式。

## 激活时机

- 编写新 Go 函数或方法时
- 为已有代码补充测试覆盖时
- 为性能关键代码创建基准测试时
- 为输入验证实现模糊测试时
- 在 Go 项目中遵循 TDD 工作流时

## Go 的 TDD 工作流

### RED-GREEN-REFACTOR 循环

```
RED     → 先写一个失败的测试
GREEN   → 编写最少量代码使测试通过
REFACTOR → 在保持测试通过的同时改进代码
REPEAT  → 继续下一个需求
```

### Go 中的 TDD 逐步指南

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
    got := Add(2, 3)
    want := 5
    if got != want {
        t.Errorf("Add(2, 3) = %d; 期望 %d", got, want)
    }
}

// 第 3 步：运行测试 —— 验证失败
// $ go test
// --- FAIL: TestAdd (0.00s)
// panic: 尚未实现

// 第 4 步：实现最少量代码（GREEN）
func Add(a, b int) int {
    return a + b
}

// 第 5 步：运行测试 —— 验证通过
// $ go test
// PASS

// 第 6 步：如需重构，验证测试仍然通过
```

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

## 最佳实践

**应当：**
- 先写测试（TDD）
- 使用表驱动测试实现全面覆盖
- 测试行为，而非实现
- 在辅助函数中使用 `t.Helper()`
- 对独立测试使用 `t.Parallel()`
- 用 `t.Cleanup()` 清理资源
- 使用有意义的测试名称描述场景

**不应当：**
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

**记住**：测试就是文档。它们展示了代码该如何使用。清晰地编写测试，并保持其及时更新。
