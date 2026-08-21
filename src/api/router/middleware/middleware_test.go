// middleware_test.go 是 Trace / CORS / Recover / Logger 中间件的单元测试:
// 通过 gin 引擎与 httptest 验证各中间件的注入、拦截与恢复行为。
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gin-demo/schema"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestTraceWithHeader 覆盖正常场景: 请求头携带 trace_id 时原样透传。
func TestTraceWithHeader(t *testing.T) {

	// case: 请求头携带 trace_id, 应原样透传到 gin 上下文与响应头
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Trace())
	r.GET("/", func(c *gin.Context) {

		assert.Equal(t, "fixed-id", c.GetString(schema.CTXTraceKey))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(schema.CTXTraceKey, "fixed-id")
	r.ServeHTTP(w, req)

	assert.Equal(t, "fixed-id", w.Header().Get(schema.CTXTraceKey))
}

// TestTraceGenerate 覆盖边界场景: 请求头缺失时自动生成非空 trace_id。
func TestTraceGenerate(t *testing.T) {

	// case: 请求头缺失 trace_id, 应自动生成非空 UUID
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Trace())
	var got string
	r.GET("/", func(c *gin.Context) {

		got = c.GetString(schema.CTXTraceKey)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.NotEmpty(t, got)
	assert.NotEmpty(t, w.Header().Get(schema.CTXTraceKey))
}

// TestCORSPreflight 覆盖异常场景: OPTIONS 预检请求以 204 终止, 不进入业务 handler。
func TestCORSPreflight(t *testing.T) {

	// case: OPTIONS 预检请求, 应以 204 终止且不进入业务 handler
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	var nextCalled bool
	r.POST("/", func(c *gin.Context) {

		nextCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/", nil))

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.False(t, nextCalled)
}

// TestCORSGet 覆盖正常场景: 非预检请求放行, 并设置跨域响应头。
func TestCORSGet(t *testing.T) {

	// case: 非预检请求, 应放行并设置跨域响应头
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	var nextCalled bool
	r.GET("/", func(c *gin.Context) {

		nextCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.True(t, nextCalled)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// TestRecoverPanic 覆盖异常场景: 业务 handler panic 时被捕获, 返回统一错误响应而非进程崩溃。
func TestRecoverPanic(t *testing.T) {

	// case: 业务 handler panic, 应被捕获并返回统一错误响应而非进程崩溃
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recover())
	r.GET("/panic", func(c *gin.Context) {

		panic("boom")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "服务器内部错误")
}

// TestLogger 覆盖正常场景: 请求走完 Trace+Logger 链路后正常返回, 不 panic。
func TestLogger(t *testing.T) {

	// case: 请求走完 Trace+Logger 链路, 应正常返回不 panic
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Trace())
	r.Use(Logger())
	r.GET("/", func(c *gin.Context) {

		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/?q=1", nil))

	assert.Equal(t, http.StatusOK, w.Code)
}
