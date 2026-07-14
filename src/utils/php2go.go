package utils

func ContainsString(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func ContainsInt(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ContainsInt32(slice []int32, item int32) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ArrayColumnStruct[T any, U any](slice []T, getter func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = getter(item)
	}
	return result
}

// ArrayColumnGeneric 泛型实现，支持任意类型
func ArrayColumnGeneric[T any, K comparable](maps []map[K]T, columnKey K) []T {
	column := make([]T, len(maps))
	for i, m := range maps {
		column[i] = m[columnKey]
	}
	return column
}
