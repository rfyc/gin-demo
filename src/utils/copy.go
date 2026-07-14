package utils

import (
	"encoding/json"
)

// DeepCopy 使用json.Marshal和json.Unmarshal实现深拷贝
func DeepCopy(src, dst interface{}) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, dst)
}
