package utils

import "github.com/spf13/cast"

// 返回2个int的最小值
func MinInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func IntSliceToStringSlice(intSlice []int) []string {
	strSlice := make([]string, len(intSlice))
	for i, v := range intSlice {
		strSlice[i] = cast.ToString(v)
	}

	return strSlice
}
