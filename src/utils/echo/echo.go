package echo

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v3"
)

func Json(res any, prefix ...string) {
	jbytes, _ := json.MarshalIndent(res, "", "    ")
	prefixStr := ""
	if len(prefix) > 0 {
		prefixStr = prefix[0]
	}
	fmt.Printf("%s\n%s: %v\n", prefixStr, reflect.TypeOf(res).String(), string(jbytes))
}

func Yml(res interface{}) {
	if ymls, err := yaml.Marshal(res); err == nil {
		fmt.Printf("%s:\n", reflect.TypeOf(res).String())
		for _, line := range strings.Split(string(ymls), "\n") {
			if !strings.Contains(line, "xxx_") {
				fmt.Printf("    %s\n", line)
			}
		}
		return
	}
	fmt.Printf("%s:\n%v\n", reflect.TypeOf(res).String(), res)
	return
}

func Dump(res interface{}) {
	//config := spew.ConfigState{
	//	DisableCapacities: true, // 禁用容量信息
	//	DisableMethods:    true, // 禁用自定义方法调用
	//}
	// 使用自定义配置打印
	spew.Dump(res)
}

func Context(ctx context.Context) {

	if ctx == nil {
		fmt.Println("Context is nil")
		return
	}

	// 获取context的反射值
	v := reflect.ValueOf(ctx)

	// 如果是指针类型，获取其指向的实际值
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// 获取context的类型
	t := v.Type()

	fmt.Printf("Context type: %s\n", t.String())
	fmt.Println("Context values:")

	// 使用反射遍历context结构体的所有字段
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 跳过未导出的字段
		if !fieldType.IsExported() {
			continue
		}

		// 处理不同类型的字段
		switch field.Kind() {
		case reflect.Ptr, reflect.Struct:
			// 递归处理嵌套结构体
			if field.IsValid() && !field.IsNil() {
				fmt.Printf("  %s: %+v\n", fieldType.Name, field.Interface())
			}
		default:
			// 直接打印基本类型
			if field.IsValid() {
				fmt.Printf("  %s: %v\n", fieldType.Name, field.Interface())
			}
		}
	}

	// 对于标准库的context实现，我们需要特殊处理内部字段
	// 注意：这种方式依赖于Go版本和具体实现，可能不稳定
	// 仅用于调试目的
	fmt.Println("\nInternal context values (implementation-specific):")
	contextValues(ctx)
}

func contextValues(ctx context.Context) {
	v := reflect.ValueOf(ctx)

	// 检查是否是*context.valueCtx类型
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()

	// 查找并打印key和value字段
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 对于未导出字段，使用小写名称
		fieldName := fieldType.Name
		if !fieldType.IsExported() {
			// 获取原始名称（可能需要使用unsafe包，这里简化处理）
			fieldName = t.Field(i).Name
		}

		// 检查是否是key或value字段
		if (fieldName == "key" || fieldName == "value") && field.IsValid() {
			fmt.Printf("  %s: %v\n", fieldName, field.String())
		}

		// 检查是否有嵌入的Context字段（用于链表结构）
		if fieldType.Anonymous && fieldType.Type.String() == "context.Context" {
			if field.IsValid() && !field.IsNil() {
				// 递归打印父context的值
				contextValues(field.Interface().(context.Context))
			}
		}
	}
}
