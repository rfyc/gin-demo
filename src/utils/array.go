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
	tmp := make([]T, len(array))
	copy(tmp, array)
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i] < tmp[j]
	})
	for _, val := range tmp {
		result = append(result, cast.ToString(val))
	}
	return result
}

func ArrayRandomElement[T any](array []T) (element T, err error) {
	if len(array) == 0 {
		return element, fmt.Errorf("the array is empty")
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	index := r.Intn(len(array))
	return array[index], nil
}

func ArrayFilter[T comparable](array []T, val ...T) (newArray []T) {
	for _, v := range array {
		if !InSlice(val, v) {
			newArray = append(newArray, v)
		}
	}
	return
}
