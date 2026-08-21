package array

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestToString 覆盖正常与边界场景: int/string/bool/float 各类数组与空数组的字符串转换。
func TestToString(t *testing.T) {

	// case: int 数组转换
	toStringCheck(t, []int{1, 2, 3}, []string{"1", "2", "3"})
	// case: 空数组返回 nil 切片
	toStringCheck(t, []int{}, nil)
	// case: string 数组保持原样
	toStringCheck(t, []string{"a", "b"}, []string{"a", "b"})
	// case: bool 数组转换
	toStringCheck(t, []bool{true, false}, []string{"true", "false"})
	// case: float 数组转换
	toStringCheck(t, []float64{1.5, 2.0}, []string{"1.5", "2"})
}

// toStringCheck 校验 ToString 对指定类型数组的转换结果。
func toStringCheck[T NormalType](t *testing.T, arr []T, want []string) {

	t.Helper()
	got := ToString(arr)
	assert.Equal(t, want, got)
}

// TestSortToString 覆盖正常与边界场景: 乱序数组排序转换、单元素与空数组。
func TestSortToString(t *testing.T) {

	// case: 乱序 int 排序转换
	sortToStringCheck(t, []int{3, 1, 2}, []string{"1", "2", "3"})
	// case: 乱序 string 排序转换
	sortToStringCheck(t, []string{"c", "a", "b"}, []string{"a", "b", "c"})
	// case: 单元素数组
	sortToStringCheck(t, []int{5}, []string{"5"})
	// case: 空数组返回 nil 切片
	sortToStringCheck(t, []int{}, nil)
}

// sortToStringCheck 校验 ToSortString 对指定类型数组的排序转换结果。
func sortToStringCheck[T CompareType](t *testing.T, arr []T, want []string) {

	t.Helper()
	got := ToSortString(arr)
	assert.Equal(t, want, got)
}

// TestSortToStringNotMutateOriginal 覆盖边界场景: 排序转换不修改原数组(不可变性)。
func TestSortToStringNotMutateOriginal(t *testing.T) {

	// case: 排序不修改原数组(不可变性)
	src := []int{3, 1, 2}
	_ = ToSortString(src)
	assert.Equal(t, []int{3, 1, 2}, src)
}

// TestRandom 覆盖正常场景: 各类型数组与单元素数组抽取的元素均属于原数组。
func TestRandom(t *testing.T) {

	// case: 各类型数组与单元素数组抽取的元素均属于原数组
	randomCheck(t, []int{1, 2, 3})
	randomCheck(t, []string{"a", "b", "c"})
	// case: 单元素数组
	randomCheck(t, []int{7})
}

// randomCheck 校验 Rand 抽取的元素属于原数组。
func randomCheck[T comparable](t *testing.T, arr []T) {

	t.Helper()
	got, err := Rand(arr)
	assert.NoError(t, err)
	assert.True(t, Exist(arr, got), "Random(%v) = %v 不在原数组中", arr, got)
}

// TestRandomEmpty 覆盖异常场景: 空数组抽取必须返回错误。
func TestRandomEmpty(t *testing.T) {

	// case: 空数组抽取返回错误
	_, err := Rand([]int{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "the array is empty")
}

// TestExclude 表驱动覆盖正常与边界场景: 过滤部分/多个元素、无过滤值、全过滤与空数组。
func TestExclude(t *testing.T) {

	tests := []struct {
		name string
		arr  []int
		val  []int
		want []int
	}{
		{name: "过滤部分元素", arr: []int{1, 2, 3, 2}, val: []int{2}, want: []int{1, 3}},
		{name: "过滤多个值", arr: []int{1, 2, 3, 4}, val: []int{1, 3}, want: []int{2, 4}},
		{name: "无过滤值时返回全量", arr: []int{1, 2, 3}, val: nil, want: []int{1, 2, 3}},
		{name: "全部被过滤返回空", arr: []int{1, 1}, val: []int{1}, want: nil},
		{name: "空数组", arr: []int{}, val: []int{1}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := Filter(tt.arr, tt.val...)
			assert.Equal(t, tt.want, got)
		})
	}
}
