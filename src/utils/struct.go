package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// 通用的结构体转JSON字段类型映射函数
func StructToJSONSchema(s interface{}) (map[string]string, error) {
	result := make(map[string]string)
	t := reflect.TypeOf(s)

	// 如果是指针，获取其元素类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 确保是结构体类型
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("输入必须是结构体或结构体指针")
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")

		// 如果json标签为空或者有omitempty，只取主标签
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// 处理omitempty情况
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}

		// 映射数据类型
		var dataType string
		switch field.Type.Kind() {
		case reflect.String:
			dataType = "text"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			dataType = "int"
		case reflect.Float32, reflect.Float64:
			dataType = "float"
		case reflect.Bool:
			dataType = "bool"
		default:
			dataType = field.Type.String()
		}

		result[jsonTag] = dataType
	}

	return result, nil
}

// GetFieldJsonNames 获取结构体字段名
func GetFieldJsonNames(s interface{}) string {

	t := reflect.TypeOf(s)

	var fields []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			fields = append(fields, jsonTag)
		} else {
			fields = append(fields, field.Name)
		}
	}

	return strings.Join(fields, ",")
}

// 将结构体转换为 map[string]*string
func StructToMapStringPtr(obj interface{}) (map[string]*string, error) {
	result := make(map[string]*string)

	val := reflect.ValueOf(obj)

	// 如果是指针，获取指向的值
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// 确保是结构体类型
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("输入必须是一个结构体")
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// 获取字段名，可以使用tag自定义
		fieldName := field.Name

		// 检查json tag
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			// 处理 omitempty
			if idx := strings.Index(jsonTag, ","); idx != -1 {
				fieldName = jsonTag[:idx]
			} else {
				fieldName = jsonTag
			}
		}

		// 如果字段不可导出，跳过
		if !field.IsExported() {
			continue
		}

		// 转换值为字符串指针
		strPtr := ConvertToStrPtr(fieldValue)
		if strPtr != nil {
			result[fieldName] = strPtr
		}
	}

	return result, nil
}

// 辅助函数：将任意值转换为字符串指针
func ConvertToStrPtr(v reflect.Value) *string {
	if !v.IsValid() {
		return nil
	}

	// 处理指针
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	var strValue string

	switch v.Kind() {
	case reflect.String:
		strValue = v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		strValue = strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		strValue = strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		strValue = strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Bool:
		strValue = strconv.FormatBool(v.Bool())
	case reflect.Struct:
		// 特殊处理 time.Time
		if t, ok := v.Interface().(time.Time); ok {
			strValue = t.Format(time.RFC3339)
		} else {
			// 其他结构体可以递归处理或跳过
			return nil
		}
	default:
		// 其他类型跳过或根据需求处理
		return nil
	}

	return &strValue
}

// 将字符串拆分为引用数组
func SplitQuoteToArray(str string) []*string {
	arr := strings.Split(str, ",")
	result := make([]*string, 0, len(arr))
	for _, item := range arr {
		result = append(result, &item)
	}
	return result
}

// 将结构体转换为 map[string]interface{}
func StructToMap(obj interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	val := reflect.ValueOf(obj)

	// 如果是指针，获取指向的值
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// 确保是结构体类型
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("输入必须是一个结构体")
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// 获取字段名，可以使用tag自定义
		fieldName := field.Name

		// 检查json tag
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			// 处理 omitempty
			if idx := strings.Index(jsonTag, ","); idx != -1 {
				fieldName = jsonTag[:idx]
			} else {
				fieldName = jsonTag
			}
		}
		// 如果字段不可导出，跳过
		if !field.IsExported() {
			continue
		}

		// 转换值为接口类型
		result[fieldName] = fieldValue.Interface()
	}

	return result, nil
}
