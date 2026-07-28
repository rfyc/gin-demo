package db

import (
	"context"
	"database/sql"
	"fmt"
	"gin-demo/src/core/conf"
	"gin-demo/src/utils"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type DB struct {
	writer *gorm.DB
	reader *gorm.DB
}

func NewDB(cfg *conf.DBCfg) (*DB, error) {
	if cfg.Reader == "" {
		return nil, fmt.Errorf("MySQL Reader DSN 为空")
	}
	if cfg.Writer == "" {
		return nil, fmt.Errorf("MySQL Writer DSN 为空")
	}
	var err error
	var db = &DB{}
	if db.reader, err = dbInitMySQL(cfg.Reader); err != nil {
		return nil, err
	}
	if db.writer, err = dbInitMySQL(cfg.Writer); err != nil {
		return nil, err
	}
	return db, nil
}

func NewMockDB() (*DB, error) {
	var err error
	var db = &DB{}
	if db.reader, err = dbInitSQLite(); err != nil {
		return nil, err
	}
	if db.writer, err = dbInitSQLite(); err != nil {
		return nil, err
	}
	return db, nil
}

func getMockDBPath() (string, error) {
	root, err := utils.FindProjectRoot()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "data/mock")
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建 data/mock 目录失败: %w", err)
	}
	return filepath.Join(dir, "db.sqlite"), nil
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

func dbInitMySQL(dsn string) (db *gorm.DB, err error) {
	if db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}); err != nil {
		return nil, fmt.Errorf("连接 MySQL 主库失败: %w", err)
	}

	var sqlDB *sql.DB
	if sqlDB, err = db.DB(); err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return db, nil
}

func dbInitSQLite() (db *gorm.DB, err error) {
	path, err := getMockDBPath()
	if err != nil {
		return nil, err
	}
	if db, err = gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}); err != nil {
		return nil, fmt.Errorf("打开 SQLite mock 失败 [path=%s]: %w", path, err)
	}

	var sqlDB *sql.DB
	if sqlDB, err = db.DB(); err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetConnMaxIdleTime(0)

	return db, nil
}
