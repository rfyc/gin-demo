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
// 当 conf.DBCfg 中的 DSN 为空时，读写连接会回退到同一个 SQLite 文件作为 mock 数据库。
type DB struct {
	writer *gorm.DB
	reader *gorm.DB
}

// NewDB 根据配置创建读写分离的数据库连接。
//   - cfg.Reader / cfg.Writer 提供 MySQL DSN，则走 MySQL 连接；
//   - 对应 DSN 为空，则自动使用 data/mock.db.sqlite 作为 mock 数据库。
//
// 参数:
//   - cfg: 数据库配置（MySQL DSN 或留空启用 SQLite mock）
//
// 返回:
//   - *DB: 初始化完成的读写分离 DB 实例
//   - error: 初始化失败时返回带上下文的错误信息
func NewDB(cfg *conf.DBCfg) (*DB, error) {
	var err error
	var db = &DB{}
	// 初始化读连接：MySQL DSN 为空时自动走 SQLite mock
	if db.reader, err = dbInit(cfg.Reader); err != nil {
		return nil, err
	}
	// 初始化写连接：MySQL DSN 为空时自动走 SQLite mock
	if db.writer, err = dbInit(cfg.Writer); err != nil {
		return nil, err
	}
	return db, nil
}

// getMockDBPath 返回 mock SQLite 数据库文件的绝对路径，并确保 data 目录已创建。
// SQLite 文件固定放在 <项目根>/data/mock.db.sqlite；如果 data 目录不存在则自动创建（权限 0755）。
//
// 返回:
//   - string: mock SQLite 文件的绝对路径，例如 /path/to/project/data/mock.db.sqlite
//   - error: 无法找到项目根或无法创建 data 目录时返回错误
func getMockDBPath() (string, error) {
	var (
		root string
		dir  string
		err  error
	)
	// 从当前工作目录向上递归查找 go.mod，确定项目根
	if root, err = findProjectRoot(); err != nil {
		return "", err
	}
	// 拼接 data 目录并确保其存在（多级目录自动创建）
	dir = filepath.Join(root, "data")
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建 data 目录失败: %w", err)
	}
	// data/mock.db.sqlite 是 mock 的固定文件名
	return filepath.Join(dir, "mock.db.sqlite"), nil
}

// findProjectRoot 从当前工作目录向上递归查找 go.mod，将所在目录视为项目根。
// 该函数用于定位 data/、config/ 等项目级目录的位置，避免因启动目录不同导致路径错乱。
//
// 返回:
//   - string: 包含 go.mod 的目录绝对路径
//   - error: 已到达文件系统根仍未找到 go.mod 时返回 "go.mod not found"
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		// go.mod 存在则认为此目录为项目根
		if _, err = os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		// 向上一级回溯
		parent := filepath.Dir(dir)
		// dir 与 parent 相同说明已到文件系统根，查找失败
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
	// 关闭写连接对应的底层 sql.DB，忽略错误（GORM 内部已处理）
	if sqlDB, err := db.writer.DB(); err == nil {
		sqlDB.Close()
	}
	// 关闭读连接对应的底层 sql.DB
	if sqlDB, err := db.reader.DB(); err == nil {
		sqlDB.Close()
	}
}

// Transaction 开启写库事务，并通过 context 传递事务句柄给后续的 Reader/Writer 调用。
// 注意：事务内的所有读写操作都会走同一个写连接，保证一致性。
//
// 参数:
//   - ctx: 调用方上下文
//   - fn: 事务闭包，闭包内部应使用传入的 ctx 执行 DB 操作
//
// 返回:
//   - error: 闭包返回的错误或事务本身的错误，会被包装为 "DB Transaction FAIL: %w"
func (db *DB) Transaction(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	// 在 writer 上开启事务，将同一 tx 注入 ctx，fn 内读写操作均走该 tx
	if err = db.Writer(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(SetContextTxDB(ctx, tx))
	}); err != nil {
		return fmt.Errorf("DB Transaction FAIL: %w", err)
	}
	return nil
}

type dbTxKey struct{}

// SetContextTxDB 将事务句柄注入 context。
// 后续调用 Reader/Writer 时若检测到 ctx 中存在事务，则优先返回事务 DB，保证读写一致性。
//
// 参数:
//   - ctx: 原始上下文
//   - tx: 当前开启的 GORM 事务句柄
//
// 返回:
//   - context.Context: 包含事务句柄的新上下文
func SetContextTxDB(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, dbTxKey{}, tx)
}

// GetContextTxDB 从 context 中取出当前事务句柄。
// 若 ctx 中没有事务，返回 nil，此时 Reader/Writer 会返回原始读写连接。
//
// 参数:
//   - ctx: 调用方上下文
//
// 返回:
//   - *gorm.DB: 事务句柄或 nil
func GetContextTxDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(dbTxKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

// dbInit 根据 dsn 初始化数据库连接。
// dsn 为空字符串时自动回退为 SQLite mock（data/mock.db.sqlite）；否则使用 MySQL。
// 该函数是 NewDB 初始化读写连接的通用入口。
//
// 参数:
//   - dsn: MySQL DSN 字符串；传 "" 则启用 SQLite mock
//
// 返回:
//   - *gorm.DB: 已配置连接池的 GORM DB 实例
//   - error: 初始化失败时返回带上下文的错误
func dbInit(dsn string) (db *gorm.DB, err error) {
	if dsn == "" {
		// DSN 为空 → 使用本地 SQLite 作为 mock 数据库
		return dbInitSQLite()
	}
	// DSN 非空 → 连接 MySQL
	return dbInitMySQL(dsn)
}

// dbInitMySQL 使用 GORM 打开 MySQL 连接并设置连接池参数。
// 适用于生产/测试环境中 reader 或 writer DSN 不为空的场景。
//
// 参数:
//   - dsn: 标准 MySQL DSN，格式示例：user:pass@tcp(host:port)/db?charset=utf8mb4&parseTime=true
//
// 返回:
//   - *gorm.DB: 已配置的 MySQL GORM 连接
//   - error: 连接失败或获取底层 sql.DB 失败时返回错误
func dbInitMySQL(dsn string) (db *gorm.DB, err error) {
	// 打开 MySQL 驱动，并将 GORM 日志设为 Silent 以避免生产环境大量 SQL 输出
	if db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}); err != nil {
		return nil, fmt.Errorf("连接 MySQL 主库失败: %w", err)
	}

	var sqlDB *sql.DB
	// 获取底层 sql.DB 以便配置连接池
	if sqlDB, err = db.DB(); err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	// 连接池配置 —— 生产环境合理值：50 活跃连接、10 空闲，生命周期与空闲时间防僵死连接
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return db, nil
}

// dbInitSQLite 打开或创建 data/mock.db.sqlite 文件并配置 SQLite 连接池。
// SQLite 为单文件数据库，连接池限制为 1 连接（避免并发写冲突导致 "database is locked"）。
// 本函数仅在 MySQL DSN 为空（本地 mock 模式）时被调用。
//
// 返回:
//   - *gorm.DB: 已配置的 SQLite GORM 连接
//   - error: 无法定位项目根、无法创建 data 目录或打开 SQLite 文件失败时返回错误
func dbInitSQLite() (db *gorm.DB, err error) {
	var path string
	// 先获取 mock SQLite 文件的绝对路径，并确保 data 目录存在
	if path, err = getMockDBPath(); err != nil {
		return nil, err
	}
	// 打开 SQLite；如果文件不存在，sqlite3 驱动会自动创建
	if db, err = gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	}); err != nil {
		return nil, fmt.Errorf("打开 SQLite mock 失败 [path=%s]: %w", path, err)
	}

	var sqlDB *sql.DB
	if sqlDB, err = db.DB(); err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	// SQLite 连接池配置 —— 单库文件，限制为 1 个活跃/空闲连接，避免并发写锁库
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	// 生命周期与空闲时间设为 0，即不强制回收连接（本地 mock，简单稳定优先）
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetConnMaxIdleTime(0)

	return db, nil
}
