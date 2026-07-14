package utils

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/spf13/cast"
)

type CompareType interface {
	~string | ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64
}

type NormalType interface {
	~string | ~bool | ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64
}

func ArrayToString[T NormalType](array []T) []string {
	var result []string
	for _, item := range array {
		result = append(result, cast.ToString(item))
	}
	return result
}

func ArraySortToString[T CompareType](array []T) (result []string) {
	var tmp []T
	for _, item := range array {
		tmp = append(tmp, item)
	}
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i] < tmp[j]
	})
	for _, val := range tmp {
		result = append(result, cast.ToString(val))
	}
	return result
}

// 判断数组中是否存在某个元素
func ArrayExist[T NormalType](array []T, value T) bool {
	for _, item := range array {
		if value == item {
			return true
		}
	}
	return false
}

// 过滤掉数组中的指定元素
func ArrayFilter[T NormalType](array []T, val ...T) (newArray []T) {
	for _, v := range array {
		if !ArrayExist(val, v) {
			newArray = append(newArray, v)
		}
	}
	return
}

// 过滤数组中的重复元素
func ArrayUnique[T NormalType](array []T) (newArray []T) {
	var values = map[T]bool{}
	for _, val := range array {
		values[val] = true
	}
	for v, _ := range values {
		newArray = append(newArray, v)
	}
	return
}

func ArrayRandomElement[T any](array []T) (element T, err error) {
	if len(array) == 0 {
		return element, fmt.Errorf("the array is empty")
	}
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(array))
	return array[index], nil
}

func DeduplicateStrings(slice []string) []string {
	seen := make(map[string]struct{}) // 使用空结构体节省内存
	result := make([]string, 0)

	for _, item := range slice {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{} // 标记为已存在
			result = append(result, item)
		}
	}
	return result
}

// RemoveDuplicates 去重
func RemoveDuplicates(nums []int) []int {
	seen := make(map[int]bool)
	result := []int{}

	for _, num := range nums {
		if !seen[num] {
			seen[num] = true
			result = append(result, num)
		}
	}
	return result
}
