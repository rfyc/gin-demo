package db

import (
	"context"
	"database/sql"
	"fmt"
	"ginext/src/core/conf"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB 实例。
type DB struct {
	writer *gorm.DB
	reader *gorm.DB
}

func NewDB(cfg *conf.DBCfg) (*DB, error) {
	var err error
	var db = &DB{}
	if db.reader, err = dbInit(cfg.Reader); err != nil {
		return nil, err
	}
	if db.writer, err = dbInit(cfg.Writer); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) Writer(ctx context.Context) *gorm.DB {
	if tx := GetContextTxDB(ctx); tx != nil {
		return tx
	}
	return db.writer.WithContext(ctx)
}

func (db *DB) Reader(ctx context.Context) *gorm.DB {
	if tx := GetContextTxDB(ctx); tx != nil {
		return tx
	}
	return db.reader.WithContext(ctx)
}

func (db *DB) Close() {
	if sqlDB, err := db.writer.DB(); err == nil {
		sqlDB.Close()
	}
	if sqlDB, err := db.reader.DB(); err == nil {
		sqlDB.Close()
	}
}

func (db *DB) Transaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if err = db.Writer(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(SetContextTxDB(ctx, tx))
	}); err != nil {
		return fmt.Errorf("DB Transaction FAIL: %w", err)
	}
	return nil
}

type dbTxKey struct{}

func SetContextTxDB(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, dbTxKey{}, tx)
}

func GetContextTxDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(dbTxKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

// dbInit 初始化 MySQL 连接池。
func dbInit(dsn string) (db *gorm.DB, err error) {
	if db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}); err != nil {
		return nil, fmt.Errorf("连接 MySQL 主库失败: %w", err)
	}

	var sqlDB *sql.DB
	if sqlDB, err = db.DB(); err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	// 连接池配置
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return db, nil
}
