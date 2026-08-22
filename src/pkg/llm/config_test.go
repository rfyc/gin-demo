// config_test.go 是 ModelConf.Clone 的单元测试:
// 覆盖指针字段深拷贝、map 拷贝、零值场景。
package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestClone 覆盖正常场景: 各字段完整拷贝且指针/map 不共享底层。
func TestClone(t *testing.T) {

	var (
		temperature         = float32(0.3)
		maxCompletionTokens = 4096
		reasoningEffort     = "high"
	)
	var cfg = ModelConf{
		Name:            "doubao-seed-2.0-lite",
		Temperature:     &temperature,
		MaxTokens:       &maxCompletionTokens,
		ReasoningEffort: &reasoningEffort,
		Extra:           map[string]any{"response_format": map[string]string{"type": "json_object"}},
	}

	var cloned = cfg.Clone()

	// case: 标量与指针字段值相等
	assert.Equal(t, cfg.Name, cloned.Name)
	assert.Equal(t, cfg.Temperature, cloned.Temperature)
	assert.Equal(t, cfg.MaxTokens, cloned.MaxTokens)
	assert.Equal(t, cfg.ReasoningEffort, cloned.ReasoningEffort)
	assert.Equal(t, cfg.Extra, cloned.Extra)

	// case: 指针字段不共享底层(修改拷贝不影响原配置)
	*cloned.Temperature = 0.9
	*cloned.MaxTokens = 1
	*cloned.ReasoningEffort = "low"
	assert.Equal(t, float32(0.3), *cfg.Temperature)
	assert.Equal(t, 4096, *cfg.MaxTokens)
	assert.Equal(t, "high", *cfg.ReasoningEffort)

	// case: map 不共享底层(修改拷贝不影响原配置)
	cloned.Extra["response_format"] = "changed"
	assert.Equal(t, map[string]string{"type": "json_object"}, cfg.Extra["response_format"])
}

// TestCloneZero 覆盖边界场景: 零值配置拷贝后仍为零值, 不产生 nil 解引用。
func TestCloneZero(t *testing.T) {

	var cfg ModelConf

	var cloned = cfg.Clone()

	// case: 零值拷贝后指针字段仍为 nil, map 仍为 nil
	assert.Equal(t, ModelConf{}, cloned)
	assert.Nil(t, cloned.Temperature)
	assert.Nil(t, cloned.MaxTokens)
	assert.Nil(t, cloned.ReasoningEffort)
	assert.Nil(t, cloned.Extra)
}
