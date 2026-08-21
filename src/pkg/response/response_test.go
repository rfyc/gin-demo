// response_test.go 是 Success / Fail / AbortWithError 的单元测试:
// 覆盖正常、边界与异常场景, 验证统一响应体的 HTTP 状态码、业务码与数据字段。
package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// newTestContext 构造带响应记录器的 gin 测试上下文。
func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// TestSuccess 覆盖成功响应的基本场景: HTTP 200, code=0, 数据原样回写。
func TestSuccess(t *testing.T) {

	// case: 非 nil 数据, 应原样回写且 code=0
	gin.SetMode(gin.TestMode)
	c, w := newTestContext()

	Success(c, map[string]interface{}{"k": "v"})

	assert.Equal(t, http.StatusOK, w.Code)
	var body Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Code)
	assert.Equal(t, "success", body.Message)
	assert.Equal(t, map[string]interface{}{"k": "v"}, body.Data)
}

// TestSuccessNilData 覆盖边界场景: data 为 nil 时写出空对象, 不报错。
func TestSuccessNilData(t *testing.T) {

	// case: data 为 nil, 应写出空对象且不报错
	gin.SetMode(gin.TestMode)
	c, w := newTestContext()

	Success(c, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var body Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 0, body.Code)
	assert.Equal(t, map[string]interface{}{}, body.Data)
}

// TestFailDefaultCode 覆盖失败响应默认场景: 未传 code 时业务码为 -1。
func TestFailDefaultCode(t *testing.T) {

	// case: 未传 code, 业务码应为默认 -1
	gin.SetMode(gin.TestMode)
	c, w := newTestContext()

	Fail(c, errors.New("boom"))

	assert.Equal(t, http.StatusOK, w.Code)
	var body Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, -1, body.Code)
	assert.Equal(t, "boom", body.Message)
}

// TestFailWithCode 覆盖失败响应指定业务码的场景。
func TestFailWithCode(t *testing.T) {

	// case: 指定业务码 -2, 应原样回写
	gin.SetMode(gin.TestMode)
	c, w := newTestContext()

	Fail(c, errors.New("boom"), -2)

	var body Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, -2, body.Code)
}

// TestFailNilErr 覆盖异常场景: err 为 nil 时 message 为空字符串。
func TestFailNilErr(t *testing.T) {

	// case: err 为 nil, message 应为空字符串
	gin.SetMode(gin.TestMode)
	c, w := newTestContext()

	Fail(c, nil)

	var body Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "", body.Message)
}

// TestAbortWithError 覆盖异常场景: 写出失败响应并终止后续处理(IsAborted 为 true)。
func TestAbortWithError(t *testing.T) {

	// case: 写出失败响应, 应终止后续处理(IsAborted 为 true)
	gin.SetMode(gin.TestMode)
	c, w := newTestContext()

	AbortWithError(c, errors.New("boom"), 500)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusOK, w.Code)
	var body Response
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 500, body.Code)
	assert.Equal(t, "boom", body.Message)
}
