// db_test.go 是 db 包的 MySQL 集成测试，与 db_mock_test.go（SQLite mock 测试）互补。
//
// 连接信息：
//   - 优先读取环境变量 TEST_MYSQL_DSN（完整 DSN，库需自行存在）；
//   - 未设置时使用本地默认凭据 root/root@127.0.0.1:3306，
//     并自动创建测试库 gin_demo_test（CREATE DATABASE IF NOT EXISTS）。
//
// 本地 MySQL 不可用（端口不通或凭据错误）时测试会 t.Skip，
// 不影响无数据库环境的 CI 运行。
package db_test

import (
	"context"
	"fmt"
	"gin-demo/src/core/conf"
	"gin-demo/src/core/db"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 本地 MySQL 默认连接参数。
// 若需覆盖，请设置环境变量 TEST_MYSQL_DSN（优先级最高）。
const (
	mysqlHost = "127.0.0.1:3306"
	mysqlUser = "root"
	mysqlPass = "root"
	mysqlDB   = "gin_demo_test"
)

// mysqlDSN 返回 MySQL 测试 DSN：
// 优先取环境变量 TEST_MYSQL_DSN，否则拼装本地默认凭据 + 测试库。
func mysqlDSN() string {

	if dsn := os.Getenv("TEST_MYSQL_DSN"); dsn != "" {
		return dsn
	}
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		mysqlUser, mysqlPass, mysqlHost, mysqlDB)
}

// ensureMySQLTestDB 确保测试数据库存在：
// 仅当未设置 TEST_MYSQL_DSN 时执行，连接无库 DSN 并自动创建测试库（幂等）。
func ensureMySQLTestDB(t *testing.T) error {

	t.Helper()
	if os.Getenv("TEST_MYSQL_DSN") != "" {
		// 用户显式提供 DSN，假定库已存在，跳过自动建库
		return nil
	}
	adminDSN := fmt.Sprintf("%s:%s@tcp(%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		mysqlUser, mysqlPass, mysqlHost)
	admin, err := db.InitMySQL(adminDSN)
	if err != nil {
		return fmt.Errorf("连接本地 MySQL 失败: %w", err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		return fmt.Errorf("获取 admin sql.DB 失败: %w", err)
	}
	defer adminSQL.Close()

	if err := admin.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4", mysqlDB)).Error; err != nil {
		return fmt.Errorf("创建测试数据库失败 [db=%s]: %w", mysqlDB, err)
	}
	return nil
}

// newMySQLDB 创建 MySQL 测试 DB：
// 本地 MySQL 不可用时跳过测试，可用的失败则直接 Fatal。
// 测试结束时统一 Close，释放连接池。
func newMySQLDB(t *testing.T) *db.DB {

	t.Helper()
	if err := ensureMySQLTestDB(t); err != nil {
		t.Skipf("跳过 MySQL 集成测试: %v", err)
	}
	d, err := db.NewDB(&conf.DBCfg{Reader: mysqlDSN(), Writer: mysqlDSN()})
	require.NoError(t, err, "db.NewDB(MySQL) FAIL")
	t.Cleanup(func() {

		d.Close()
	})
	return d
}

// migrateMySQLModel 在 MySQL 测试库上重建 mockModel 表：
// 先删旧表再建，保证重复运行用例时数据干净。
func migrateMySQLModel(t *testing.T, d *db.DB) {

	t.Helper()
	ctx := context.Background()
	require.NoError(t, d.Writer(ctx).Migrator().DropTable(&mockModel{}), "DropTable FAIL")
	require.NoError(t, d.Writer(ctx).AutoMigrate(&mockModel{}), "AutoMigrate FAIL")
}

// TestMySQLNewDB 覆盖基本场景：有效 DSN 下 NewDB 应成功创建，Writer/Reader 均可用。
func TestMySQLNewDB(t *testing.T) {

	// case: 有效 DSN 下 NewDB 成功, Writer/Reader 均可用
	d := newMySQLDB(t)
	assert.NotNil(t, d.Writer(context.Background()))
	assert.NotNil(t, d.Reader(context.Background()))
}

// TestMySQLReadWrite 覆盖基本场景：通过 Writer 写入的数据，Reader 必须能读回。
func TestMySQLReadWrite(t *testing.T) {

	// case: 通过 Writer 写入的数据, Reader 必须能读回
	d := newMySQLDB(t)
	migrateMySQLModel(t, d)
	ctx := context.Background()

	want := mockModel{Name: "mysql-user"}
	require.NoError(t, d.Writer(ctx).Create(&want).Error, "Writer Create FAIL")
	assert.NotZero(t, want.ID)

	var got mockModel
	require.NoError(t, d.Reader(ctx).First(&got, want.ID).Error, "Reader First FAIL")
	assert.Equal(t, want.Name, got.Name)
}

// TestMySQLTransactionRollback 覆盖异常场景：事务内返回错误时，已写入的数据必须被回滚。
func TestMySQLTransactionRollback(t *testing.T) {

	// case: 事务内返回错误, 已写入的数据必须被回滚
	d := newMySQLDB(t)
	migrateMySQLModel(t, d)
	ctx := context.Background()

	err := d.Transaction(ctx, func(txCtx context.Context) error {

		if err := d.Writer(txCtx).Create(&mockModel{Name: "tx-rollback"}).Error; err != nil {
			return fmt.Errorf("事务内 Create FAIL: %w", err)
		}
		return context.Canceled
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, d.Reader(ctx).Model(&mockModel{}).Count(&count).Error, "Reader Count FAIL")
	assert.Zero(t, count, "事务未回滚")
}

// TestMySQLTransactionCommit 覆盖基本场景：事务内不返回错误时，数据应被提交并可被读回。
func TestMySQLTransactionCommit(t *testing.T) {

	// case: 事务内无错误, 数据应被提交并可读回
	d := newMySQLDB(t)
	migrateMySQLModel(t, d)
	ctx := context.Background()

	require.NoError(t, d.Transaction(ctx, func(txCtx context.Context) error {

		if err := d.Writer(txCtx).Create(&mockModel{Name: "tx-commit"}).Error; err != nil {
			return fmt.Errorf("事务内 Create FAIL: %w", err)
		}
		return nil
	}), "Transaction FAIL")

	var count int64
	require.NoError(t, d.Reader(ctx).Model(&mockModel{}).Where("name = ?", "tx-commit").Count(&count).Error, "Reader Count FAIL")
	assert.Equal(t, int64(1), count)
}

// TestMySQLContextTxPropagation 验证事务内 GetContextTx 返回非空事务句柄，
// 且 Writer/Reader 均复用该句柄，保证事务内读写落在同一事务上。
func TestMySQLContextTxPropagation(t *testing.T) {

	// case: 事务内 GetContextTx 非空, 且 Writer/Reader 复用同一事务句柄
	d := newMySQLDB(t)
	ctx := context.Background()

	err := d.Transaction(ctx, func(txCtx context.Context) error {

		var inTx *gorm.DB = db.GetContextTx(txCtx)
		require.NotNil(t, inTx, "GetContextTx 在事务内返回 nil")
		assert.Same(t, inTx, d.Writer(txCtx), "事务内 Writer 未复用事务句柄")
		assert.Same(t, inTx, d.Reader(txCtx), "事务内 Reader 未复用事务句柄")
		return nil
	})
	require.NoError(t, err)
}

// TestMySQLWriterReaderWithoutTx 验证无事务时 Writer 与 Reader 返回各自独立的连接池。
func TestMySQLWriterReaderWithoutTx(t *testing.T) {

	// case: 无事务时 Writer 与 Reader 应为独立的连接池
	d := newMySQLDB(t)
	ctx := context.Background()
	assert.NotSame(t, d.Writer(ctx), d.Reader(ctx))
}

// TestMySQLPoolParams 覆盖连接池配置：InitMySQL 设置的最大连接数应为 50。
func TestMySQLPoolParams(t *testing.T) {

	// case: InitMySQL 设置的最大连接数应为 50
	d := newMySQLDB(t)
	sqlDB, err := d.Writer(context.Background()).DB()
	require.NoError(t, err, "获取 sql.DB FAIL")
	assert.Equal(t, 50, sqlDB.Stats().MaxOpenConnections)
}

// TestMySQLInitFail 覆盖异常场景：InitMySQL 连接不可达地址时返回错误。
func TestMySQLInitFail(t *testing.T) {

	// case: InitMySQL 连接不可达地址, 应返回错误
	bad, err := db.InitMySQL("user:pass@tcp(127.0.0.1:1)/gin_demo_test?parseTime=True")
	require.Error(t, err)
	if bad != nil {
		if badSQL, dbErr := bad.DB(); dbErr == nil {
			badSQL.Close()
		}
	}
}
