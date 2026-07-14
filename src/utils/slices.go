package utils

import "github.com/spf13/cast"

func UniqueIntSlices(s []int) []int {
	if len(s) < 2 {
		return s
	}

	mark := make(map[int]bool, 0)
	res := make([]int, 0)
	for _, v := range s {
		if _, ok := mark[v]; !ok {
			res = append(res, v)
		}
		mark[v] = true
	}

	return res
}

func StringInSlice(s string, slices []string) bool {
	if slices == nil || len(slices) == 0 {
		return false
	}
	for _, v := range slices {
		if v == s {
			return true
		}
	}
	return false
}

func IntSliceToInt64Slice(s []int) []int64 {
	res := make([]int64, 0)
	for _, v := range s {
		res = append(res, int64(v))
	}
	return res
}

func StringSliceToUniqueIntSlices(s []string) []int {
	if len(s) == 0 {
		return []int{}
	}

	mark := make(map[string]struct{})
	res := make([]int, 0)
	for _, v := range s {
		if _, ok := mark[v]; !ok {
			res = append(res, cast.ToInt(v))
		}
		mark[v] = struct{}{}
	}
	return res
}

func IntSlicesToIntMap(s []int) map[int]struct{} {
	res := make(map[int]struct{})
	for _, v := range s {
		res[v] = struct{}{}
	}
	return res
}

func UniqueSlice[T comparable](slice []T) []T {
	keys := make(map[T]bool)
	list := []T{}

	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func RemoveNilItem(s []string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == "" {
			s = append(s[:i], s[i+1:]...)
			i--
		}
	}
	return s
}

func InSlice[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
