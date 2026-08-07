package array

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/spf13/cast"
)

// CompareType 是一个类型约束，包含所有支持比较运算符（如 <, >）的基础类型。
type CompareType interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64
}

// NormalType 是一个类型约束，包含常见的标量类型（字符串、布尔值、整数、浮点数等）。
type NormalType interface {
	~string | ~bool | ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64
}

// ToString 将数组中的每个元素转换为字符串。
// 参数:
//   - arr: 包含各种标量类型的切片。
// 返回:
//   - []string: 转换后的字符串切片。
func ToString[T NormalType](arr []T) []string {
	var result []string
	for _, item := range arr {
		result = append(result, cast.ToString(item))
	}
	return result
}

// SortToString 对数组进行排序，并返回排序后的字符串切片。
// 该函数不会修改原数组，而是操作其副本。
// 参数:
//   - arr: 支持比较运算的元素切片。
// 返回:
//   - []string: 排序后的字符串切片。
func SortToString[T CompareType](arr []T) (result []string) {
	// 创建副本以保证原数组不可变性
	tmp := make([]T, len(arr))
	copy(tmp, arr)
	// 使用切片排序
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i] < tmp[j]
	})
	for _, val := range tmp {
		result = append(result, cast.ToString(val))
	}
	return result
}

// Random 从数组中随机抽取一个元素。
// 参数:
//   - arr: 需要从中抽取元素的切片。
// 返回:
//   - element: 抽取的随机元素。
//   - err: 如果数组为空，则返回 error。
func Random[T any](arr []T) (element T, err error) {
	if len(arr) == 0 {
		return element, fmt.Errorf("Random fail: the array is empty")
	}
	// 使用当前纳秒时间作为随机数种子
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	index := r.Intn(len(arr))
	return arr[index], nil
}

// Exclude 根据提供的值过滤掉数组中的指定元素。
// 返回一个新数组，其中包含原数组中不在 `val` 列表中的元素。
// 参数:
//   - arr: 源数组。
//   - val: 需要被过滤掉的目标值列表。
// 返回:
//   - []T: 过滤后的新数组。
func Exclude[T comparable](arr []T, val ...T) (newArray []T) {
	for _, v := range arr {
		// 如果当前元素不在过滤值列表中，则保留
		if !contains(val, v) {
			newArray = append(newArray, v)
		}
	}
	return
}

// contains 判断切片中是否包含指定元素。
// 参数:
//   - slice: 待查找的切片。
//   - item: 目标元素。
// 返回:
//   - bool: 包含返回 true，否则返回 false。
func contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
