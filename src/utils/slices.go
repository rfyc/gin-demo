package utils

import "github.com/spf13/cast"

func IntSliceToInt64Slice(s []int) []int64 {
	res := make([]int64, len(s))
	for i, v := range s {
		res[i] = int64(v)
	}
	return res
}

func StringSliceToUniqueIntSlices(s []string) []int {
	if len(s) == 0 {
		return []int{}
	}
	mark := make(map[string]struct{})
	res := make([]int, 0, len(s))
	for _, v := range s {
		if _, ok := mark[v]; !ok {
			res = append(res, cast.ToInt(v))
		}
		mark[v] = struct{}{}
	}
	return res
}

func IntSlicesToIntMap(s []int) map[int]struct{} {
	res := make(map[int]struct{}, len(s))
	for _, v := range s {
		res[v] = struct{}{}
	}
	return res
}

func UniqueSlice[T comparable](slice []T) []T {
	keys := make(map[T]struct{}, len(slice))
	list := make([]T, 0, len(slice))
	for _, entry := range slice {
		if _, ok := keys[entry]; !ok {
			keys[entry] = struct{}{}
			list = append(list, entry)
		}
	}
	return list
}

func RemoveNilItem(s []string) []string {
	res := make([]string, 0, len(s))
	for _, v := range s {
		if v != "" {
			res = append(res, v)
		}
	}
	return res
}

func InSlice[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
