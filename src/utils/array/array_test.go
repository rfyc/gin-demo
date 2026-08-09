package array

import (
	"reflect"
	"strings"
	"testing"
)

func TestToString(t *testing.T) {
	// int 类型
	toStringCheck(t, []int{1, 2, 3}, []string{"1", "2", "3"})
	// 空数组返回 nil 切片
	toStringCheck(t, []int{}, nil)
	// string 类型保持原样
	toStringCheck(t, []string{"a", "b"}, []string{"a", "b"})
	// bool 类型
	toStringCheck(t, []bool{true, false}, []string{"true", "false"})
	// float 类型
	toStringCheck(t, []float64{1.5, 2.0}, []string{"1.5", "2"})
}

// toStringCheck 校验 ToString 对指定类型数组的转换结果。
func toStringCheck[T NormalType](t *testing.T, arr []T, want []string) {
	t.Helper()
	got := ToString(arr)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ToString(%v) = %v, want %v", arr, got, want)
	}
}

func TestSortToString(t *testing.T) {
	// 乱序 int 排序
	sortToStringCheck(t, []int{3, 1, 2}, []string{"1", "2", "3"})
	// 乱序 string 排序
	sortToStringCheck(t, []string{"c", "a", "b"}, []string{"a", "b", "c"})
	// 单元素
	sortToStringCheck(t, []int{5}, []string{"5"})
	// 空数组返回 nil 切片
	sortToStringCheck(t, []int{}, nil)
}

// sortToStringCheck 校验 ToSortString 对指定类型数组的排序转换结果。
func sortToStringCheck[T CompareType](t *testing.T, arr []T, want []string) {
	t.Helper()
	got := ToSortString(arr)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SortToString(%v) = %v, want %v", arr, got, want)
	}
}

func TestSortToStringNotMutateOriginal(t *testing.T) {
	// 验证排序不会修改原数组（不可变性）
	src := []int{3, 1, 2}
	_ = ToSortString(src)
	if !reflect.DeepEqual(src, []int{3, 1, 2}) {
		t.Errorf("SortToString 修改了原数组: %v", src)
	}
}

func TestRandom(t *testing.T) {
	randomCheck(t, []int{1, 2, 3})
	randomCheck(t, []string{"a", "b", "c"})
	// 单元素
	randomCheck(t, []int{7})
}

// randomCheck 校验 Rand 抽取的元素属于原数组。
func randomCheck[T comparable](t *testing.T, arr []T) {
	t.Helper()
	got, err := Rand(arr)
	if err != nil {
		t.Fatalf("Random(%v) 意外报错: %v", arr, err)
	}
	if !Exist(arr, got) {
		t.Errorf("Random(%v) = %v, 不在原数组中", arr, got)
	}
}

func TestRandomEmpty(t *testing.T) {
	// 异常场景：空数组必须返回错误
	_, err := Rand([]int{})
	if err == nil {
		t.Fatal("Random([]int{}) 未返回错误")
	}
	if !strings.Contains(err.Error(), "the array is empty") {
		t.Errorf("错误信息不符合预期: %v", err)
	}
}

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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Exclude(%v, %v) = %v, want %v", tt.arr, tt.val, got, tt.want)
			}
		})
	}
}
