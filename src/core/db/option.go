package db

import "gorm.io/gorm"

type Option func(db *gorm.DB) *gorm.DB

func Append(db *gorm.DB, dbOpts ...Option) *gorm.DB {
	for _, fn := range dbOpts {
		db = fn(db)
	}
	return db
}

func Where(query interface{}, args ...interface{}) Option {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(query, args...)
	}
}

func Or(query interface{}, args ...interface{}) Option {
	return func(db *gorm.DB) *gorm.DB {
		return db.Or(query, args...)
	}
}

func Table(name string, args ...interface{}) Option {
	return func(db *gorm.DB) *gorm.DB {
		return db.Table(name, args...)
	}
}

func OrderBy(orderBy interface{}) Option {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(orderBy)
	}
}

func Limit(limit int) Option {
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit)
	}
}
func Page(page, pageSize uint32) Option {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		return db.Offset(int((page - 1) * pageSize)).Limit(int(pageSize))
	}
}

func Empty() Option {
	return func(db *gorm.DB) *gorm.DB {
		return db
	}
}
