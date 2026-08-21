package db

import "gorm.io/gorm"

// Option 定义对 *gorm.DB 的变换函数，用于链式组装查询条件。
// 每次调用返回新的查询链，不修改原查询。
type Option func(db *gorm.DB) *gorm.DB

// Opts 是查询选项链的起点（identity 变换），提供 Opts.Where(...).Where(...).Page(...)
// 的链式引用方式：链上每个方法都会生效并逐级叠加。
var Opts Option = func(db *gorm.DB) *gorm.DB {

	return db
}

// Append 依次将 dbOpts 应用到 db，返回处理后的查询链。
// 参数 db 为待查询的 *gorm.DB，dbOpts 为按序应用的 Option 列表。
func (Option) Append(db *gorm.DB, dbOpts ...Option) *gorm.DB {

	for _, fn := range dbOpts {
		db = fn(db)
	}
	return db
}

// Where 在已有 Option 链上追加 where 条件，返回新的 Option。
// 参数 query 为条件表达式，args 为占位符参数值。
func (o Option) Where(query interface{}, args ...interface{}) Option {

	return func(db *gorm.DB) *gorm.DB {

		return o(db).Where(query, args...)
	}
}

// Or 在已有 Option 链上追加 or 条件，通常与 Where 组合使用，返回新的 Option。
// 参数 query 为条件表达式，args 为占位符参数值。
func (o Option) Or(query interface{}, args ...interface{}) Option {

	return func(db *gorm.DB) *gorm.DB {

		return o(db).Or(query, args...)
	}
}

// Table 在已有 Option 链上指定查询表名，返回新的 Option。
// 参数 name 为目标表名，args 为可选的表级参数。
func (o Option) Table(name string, args ...interface{}) Option {

	return func(db *gorm.DB) *gorm.DB {

		return o(db).Table(name, args...)
	}
}

// OrderBy 在已有 Option 链上追加排序，返回新的 Option。
// 参数 orderBy 为排序表达式，如 "id DESC"。
func (o Option) OrderBy(orderBy interface{}) Option {

	return func(db *gorm.DB) *gorm.DB {

		return o(db).Order(orderBy)
	}
}

// Limit 在已有 Option 链上追加返回行数限制，返回新的 Option。
// 参数 limit 为最大返回行数，0 表示不限制。
func (o Option) Limit(limit int) Option {

	return func(db *gorm.DB) *gorm.DB {

		return o(db).Limit(limit)
	}
}

// Page 在已有 Option 链上追加分页：page 从 1 开始，pageSize 为每页行数，
// 任一参数为 0 时取默认值（page=1，pageSize=20），返回新的 Option。
func (o Option) Page(page, pageSize uint32) Option {

	return func(db *gorm.DB) *gorm.DB {

		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		return o(db).Offset(int((page - 1) * pageSize)).Limit(int(pageSize))
	}
}

// Empty 在已有 Option 链上不做任何变换，返回新的 Option，用于占位。
func (o Option) Empty() Option {

	return func(db *gorm.DB) *gorm.DB {

		return o(db)
	}
}
