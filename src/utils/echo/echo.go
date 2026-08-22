// Package echo 提供一组调试辅助函数，用于在开发阶段以 JSON、YAML 及原始结构等
// 多种格式打印变量内容，方便快速查看和排查数据。
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

// Json 以 JSON 格式化缩进的方式打印任意类型的值到标准输出。
// 参数:
//   - res: 待打印的值，可为任意类型（结构体、切片、map 等）
//
// 说明:
//   - 使用 json.MarshalIndent 序列化，缩进为 4 个空格
//   - 输出格式为 "类型名: JSON内容"，便于区分不同类型的数据
//   - res 为 nil 时打印 "nil"，不进行序列化
//   - 序列化失败时打印错误信息，便于排查
func Json(res any) {

	var jbytes []byte
	var err error

	// res 为 nil 时 reflect.TypeOf 返回 nil, 取类型名会触发 nil 指针解引用, 需提前返回
	if res == nil {
		fmt.Println("nil")
		return
	}

	jbytes, err = json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Printf("%s: <marshal error> %v\n", reflect.TypeOf(res).String(), err)
		return
	}
	fmt.Printf("%s: %v\n", reflect.TypeOf(res).String(), string(jbytes))
}

// Yml 以 YAML 格式打印任意类型的值到标准输出，并过滤掉内容中含 "xxx_" 的行（通常用于隐藏脱敏字段）。
// 参数:
//   - res: 待打印的值，可为任意类型
//
// 说明:
//   - 序列化成功时按行过滤掉包含 "xxx_" 的行后逐行打印（行首缩进 4 个空格）
//   - res 为 nil 时打印 "nil"，不进行序列化
//   - 序列化失败时降级为直接打印原始值（%v 格式）
func Yml(res interface{}) {

	var ymls []byte
	var err error

	// res 为 nil 时 reflect.TypeOf 返回 nil, 取类型名会触发 nil 指针解引用, 需提前返回
	if res == nil {
		fmt.Println("nil")
		return
	}

	if ymls, err = yaml.Marshal(res); err == nil {
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

// Dump 使用 spew 库以深度展开的方式打印任意类型的值到标准输出。
// 参数:
//   - res: 待打印的值，可为任意类型
//
// 说明:
//   - 与 Json/Yml 不同，Dump 会展开指针、通道、接口内部等底层信息，输出更详细
//   - 适用于调试复杂的嵌套结构（如循环引用、未导出字段等）
func Dump(res interface{}) {

	//config := spew.ConfigState{
	//	DisableCapacities: true, // 禁用容量信息
	//	DisableMethods:    true, // 禁用自定义方法调用
	//}
	// 使用自定义配置打印
	spew.Dump(res)
}

// Context 打印 context.Context 的运行时信息，包括类型名、可导出的字段值，
// 以及内部携带的 key-value 链数据，用于调试 context 的传递和内容。
// 参数:
//   - ctx: 待调试的 context，若为 nil 则打印提示并直接返回
//
// 说明:
//   - 使用反射遍历 context 的导出字段，对嵌套结构体/指针字段打印其内容
//   - 内部 key-value 数据通过 contextValues 递归遍历，结果依赖 Go 标准库的
//     具体实现（如 *context.valueCtx），不同 Go 版本可能不一致，仅供调试使用
func Context(ctx context.Context) {

	if ctx == nil {
		fmt.Println("Context is nil")
		return
	}

	var (
		v reflect.Value // ctx 的反射值
		t reflect.Type  // ctx 的反射类型
	)

	// 1. 获取 context 的反射值, 指针类型时解引用
	{
		v = reflect.ValueOf(ctx)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
	}

	// 2. 获取 context 的类型并打印基础信息
	{
		t = v.Type()
		fmt.Printf("Context type: %s\n", t.String())
		fmt.Println("Context values:")
	}

	// 3. 非结构体类型没有字段(如 context.Background 的底层类型是 int), 跳过字段遍历
	if v.Kind() != reflect.Struct {
		fmt.Println("  (no fields)")
		return
	}

	// 使用反射遍历context结构体的所有字段
	for i := 0; i < v.NumField(); i++ {
		var (
			field     = v.Field(i) // field 当前字段值
			fieldType = t.Field(i) // fieldType 当前字段类型信息
		)

		// 跳过未导出的字段
		if !fieldType.IsExported() {
			continue
		}

		// 处理不同类型的字段
		switch field.Kind() {
		case reflect.Ptr:
			// 指针字段仅当非 nil 时打印其指向的值
			if field.IsValid() && !field.IsNil() {
				fmt.Printf("  %s: %+v\n", fieldType.Name, field.Interface())
			}
		case reflect.Struct:
			// 结构体字段直接打印（结构体不能调用 IsNil）
			if field.IsValid() {
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

// contextValues 递归遍历 context 的内部字段，打印标准库 context 实现
// （如 *context.valueCtx）中携带的 key/value 数据，用于辅助 Context 的调试。
// 参数:
//   - ctx: 待遍历的 context，nil 或非结构体类型（如 emptyCtx）时直接返回
//
// 说明:
//   - 依赖 context 内部未导出字段名（key/value）及匿名嵌入的 context.Context
//     字段，与 Go 版本实现强相关，仅用于调试目的
func contextValues(ctx context.Context) {

	var v reflect.Value
	var t reflect.Type

	// 递归场景（遍历父 context）可能传入 nil, 需提前返回
	if ctx == nil {
		return
	}

	v = reflect.ValueOf(ctx)

	// 检查是否是*context.valueCtx类型
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// 非结构体类型没有字段（如 context.Background 的底层类型是 int）, 直接返回
	if v.Kind() != reflect.Struct {
		return
	}

	t = v.Type()

	// 查找并打印key和value字段
	for i := 0; i < v.NumField(); i++ {
		var (
			field     = v.Field(i)     // field 当前字段值
			fieldType = t.Field(i)     // fieldType 当前字段类型信息
			fieldName = fieldType.Name // fieldName 字段名(未导出时为原始名)
		)

		// 对于未导出字段，使用小写名称
		if !fieldType.IsExported() {
			// 获取原始名称（可能需要使用unsafe包，这里简化处理）
			fieldName = t.Field(i).Name
		}

		// 检查是否是key或value字段
		if (fieldName == "key" || fieldName == "value") && field.IsValid() {
			// 未导出字段无法用 Interface() 读取值(会 panic), 仅打印其类型名
			if !field.CanInterface() {
				fmt.Printf("  %s: <unexported %s>\n", fieldName, field.Type().String())
				continue
			}
			fmt.Printf("  %s: %v\n", fieldName, field.Interface())
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
