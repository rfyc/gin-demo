// Package db 提供统一的数据库访问层：
//
//   - 生产环境：基于 MySQL 读写分离，writer 连接池负责写与事务，
//     reader 连接池负责读，可在只读实例上水平扩展；
//   - 本地开发：降级为 SQLite 单文件 mock（见 NewMockDB），
//     便于离线调试与单元测试；
//   - 事务传播：通过 context 注入事务句柄（SetContextTx / GetContextTx），
//     保证事务内所有读写自动复用同一连接，无需显式传递 *gorm.DB。
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

// DB 封装读写分离的数据库连接：
//   - writer 负责所有写操作（INSERT/UPDATE/DELETE）与事务；
//   - reader 负责读操作，可指向独立的只读数据库实例。
type DB struct {
	writer *gorm.DB
	reader *gorm.DB
}

// NewDB 基于配置创建生产级 DB 实例。
//
// 参数 cfg 中 Reader/Writer 为 MySQL DSN，任一为空都会返回错误，
// 避免静默使用错误连接导致数据写入不存在的库。
// 返回的 DB 已建立读写两个连接池，调用方退出前必须调用 Close() 释放。
func NewDB(cfg *conf.DBCfg) (*DB, error) {
	if cfg.Reader == "" {
		return nil, fmt.Errorf("MySQL Reader DSN 为空")
	}
	if cfg.Writer == "" {
		return nil, fmt.Errorf("MySQL Writer DSN 为空")
	}
	var err error
	var db = &DB{}
	if db.reader, err = InitMySQL(cfg.Reader); err != nil {
		return nil, err
	}
	if db.writer, err = InitMySQL(cfg.Writer); err != nil {
		return nil, err
	}
	return db, nil
}

// NewMockDB 创建基于 SQLite 单文件的本地 mock DB，用于单元测试与本地开发。
// reader 与 writer 指向同一个 SQLite 文件（见 GetMockPath），
// 使读写路径均可用且数据一致。失败时返回包含上下文信息的错误。
func NewMockDB() (*DB, error) {
	var err error
	var db = &DB{}
	if db.reader, err = InitSQLite(); err != nil {
		return nil, err
	}
	if db.writer, err = InitSQLite(); err != nil {
		return nil, err
	}
	return db, nil
}

// GetMockPath 返回 SQLite mock 数据库文件的绝对路径。
// 路径位于项目根目录（go.mod 所在目录）下的 data/mock/db.sqlite，
// 目标目录不存在时自动创建。依赖当前工作目录向上查找项目根（utils.FindProjectRoot）。
func GetMockPath() (string, error) {
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

// Writer 返回写库连接。
// 若 ctx 中已注入事务句柄（见 Transaction），返回该事务句柄，
// 保证事务内的写操作落在同一事务上；否则返回写连接池的 *gorm.DB。
func (db *DB) Writer(ctx context.Context) *gorm.DB {
	if tx := GetContextTx(ctx); tx != nil {
		return tx
	}
	return db.writer.WithContext(ctx)
}

// Reader 返回读库连接。
// 若 ctx 中已注入事务句柄，返回该事务句柄（事务内读写一致）；
// 否则返回读连接池的 *gorm.DB。
func (db *DB) Reader(ctx context.Context) *gorm.DB {
	if tx := GetContextTx(ctx); tx != nil {
		return tx
	}
	return db.reader.WithContext(ctx)
}

// Close 释放读写两个连接池的底层数据库连接。
// 幂等安全：重复调用或底层连接已关闭时不会 panic。
func (db *DB) Close() {
	if sqlDB, err := db.writer.DB(); err == nil {
		sqlDB.Close()
	}
	if sqlDB, err := db.reader.DB(); err == nil {
		sqlDB.Close()
	}
}

// Transaction 在写库上执行一个事务。
//
// fn 通过返回 error 控制提交（nil）或回滚（非 nil）；
// 事务句柄通过 SetContextTx 注入 ctx，事务内应使用 db.Writer(ctx)/db.Reader(ctx)
// 读写，以保证落在同一事务上。
// 若 ctx 已处于外层事务中，gorm 以 savepoint 形式嵌套，不会新建连接。
func (db *DB) Transaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if err = db.Writer(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(SetContextTx(ctx, tx))
	}); err != nil {
		return fmt.Errorf("DB Transaction FAIL: %w", err)
	}
	return nil
}

// dbTxKey 是 context 中事务句柄的私有 key 类型。
// 使用私有类型而非 string，避免与其他包存入 ctx 的值冲突。
type dbTxKey struct{}

// SetContextTx 将事务句柄 tx 注入 ctx，供事务内的 Writer/Reader 调用复用。
func SetContextTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, dbTxKey{}, tx)
}

// GetContextTx 从 ctx 取出注入的事务句柄；ctx 中未注入事务时返回 nil。
func GetContextTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(dbTxKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

// InitMySQL 根据 DSN 建立 MySQL 连接池，并应用连接池参数（最大连接数、空闲连接数、连接生命周期）。
// 连接失败时返回包含 DSN 的错误上下文。SQL 日志级别为 Silent，避免刷屏。
func InitMySQL(dsn string) (db *gorm.DB, err error) {
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

// InitSQLite 打开 SQLite mock 数据库（文件不存在时自动创建）。
// 连接池限制为单连接，避免 SQLite 单写锁导致的文件锁冲突。
func InitSQLite() (db *gorm.DB, err error) {
	path, err := GetMockPath()
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
