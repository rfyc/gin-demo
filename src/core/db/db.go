package db

import (
	"context"
	"database/sql"
	"fmt"
	"gin-demo/src/core/conf"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB 封装了读写分离的 GORM 连接。
// writer 用于写操作（INSERT / UPDATE / DELETE），reader 用于读操作（SELECT）。
type DB struct {
	writer *gorm.DB
	reader *gorm.DB
}

// NewDB 根据配置创建读写分离的 MySQL 数据库连接。
// 要求 cfg.Reader 和 cfg.Writer 必须包含有效的 MySQL DSN，否则返回错误。
// 若需要本地 mock，请使用 NewMockDB。
//
// 参数:
//   - cfg: MySQL 数据库配置
//
// 返回:
//   - *DB: 初始化完成的读写分离 DB 实例
//   - error: DSN 为空或初始化失败时返回带上下文的错误信息
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

// NewMockDB 创建基于 SQLite 文件的本地 mock 数据库，读写均走同一个 SQLite 文件。
// 本函数与真实 MySQL 连接逻辑完全独立，仅用于本地开发环境且无可用 MySQL 时的场景。
// SQLite 文件固定放在 <项目根>/data/mock.db.sqlite。
//
// 返回:
//   - *DB: 初始化完成的 mock DB 实例
//   - error: 初始化失败时返回错误
func NewMockDB() (*DB, error) {
	var err error
	var db = &DB{}
	// 读写均使用同一个 SQLite mock 连接
	if db.reader, err = dbInitSQLite(); err != nil {
		return nil, err
	}
	if db.writer, err = dbInitSQLite(); err != nil {
		return nil, err
	}
	return db, nil
}

// getMockDBPath 返回 mock SQLite 数据库文件的绝对路径，并确保 data 目录已创建。
func getMockDBPath() (string, error) {
	var (
		root string
		dir  string
		err  error
	)
	if root, err = findProjectRoot(); err != nil {
		return "", err
	}
	dir = filepath.Join(root, "data")
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建 data 目录失败: %w", err)
	}
	return filepath.Join(dir, "mock.db.sqlite"), nil
}

// findProjectRoot 从当前工作目录向上递归查找 go.mod，将所在目录视为项目根。
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err = os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
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

// Transaction 开启写库事务，并通过 context 传递事务句柄给后续的 Reader/Writer 调用。
func (db *DB) Transaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if err = db.Writer(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(SetContextTxDB(ctx, tx))
	}); err != nil {
		return fmt.Errorf("DB Transaction FAIL: %w", err)
	}
	return nil
}

type dbTxKey struct{}

// SetContextTxDB 将事务句柄注入 context。
func SetContextTxDB(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, dbTxKey{}, tx)
}

// GetContextTxDB 从 context 中取出当前事务句柄。
func GetContextTxDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(dbTxKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

// dbInitMySQL 使用 GORM 打开 MySQL 连接并设置连接池参数。
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

// dbInitSQLite 打开或创建 data/mock.db.sqlite 文件并配置 SQLite 连接池。
// 仅用于本地 mock 场景。
func dbInitSQLite() (db *gorm.DB, err error) {
	var path string
	if path, err = getMockDBPath(); err != nil {
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
