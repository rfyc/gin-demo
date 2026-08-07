# 修改或添加代码的规则（高优先级）

## 1. 代码规范

* 函数要短小精悍，避免过长或过复杂的逻辑
* 函数要明确入参出参, 每个函数都要有注释和测试用例
* 函数格式严格为：
  * 第一个参数必须是 context.Context 类型，用于传递取消信号、上下文信息等
  * 提前声明变量
  * 处理错误 要包含上下文 如: 产生错误的函数名、参数值等
  * good example:

```golang
func XXX(ctx context.Context,input****) (output***, error) {
	var err error
	var list ****
	// 注释*****
	if list,err = query(a1,a2);err!= nil{
	    return nil, fmt.Errorf("query fail: %w - a1: %v - a2: %v",err, a1, a2)
	}
	return nil,nil
}
```

* bad example:
  * 尽量不要使用 `:=`, 因为看起来结构乱

```golang
  func XXX(ctx context.Context,input****) (output***, error) {
      // 注释*****
      list,err := query(a1,a2)
	  if err!= nil{
          return nil, fmt.Errorf("query fail: %w - a1: %v - a2: %v",err, a1, a2)
      }
      return nil,nil
  }
```

## 2. 注释要求

**所有修改/新增的代码块、变量、函数、方法，必须添加详细注释**：

* 函数必须写 GoDoc 风格注释，说明用途、参数、返回值、错误场景
* 非公开函数至少说明用途和关键逻辑
* 复杂逻辑块需要行内注释说明意图

## 3. 测试要求

### 测试类型

### 测试通用规则

**所有修改/新增的代码，必须以「函数」为基本单元编写单元测试**：

* 未完成测试的代码，不得视为「完成」
* 测试文件位置
  * 在被测代码所在目录下创建测试文件
  * 测试文件路径：`test/***_test.go`（与被测代码对应命名）
  * 测试case要覆盖基本的场景和异常场景
  * 测试函数格式严格为：

```golang
func TestXxx(t *testing.T) {
    // 被测函数或代码逻辑块, 批量测试case
}
```