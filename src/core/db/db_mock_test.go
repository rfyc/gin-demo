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

// mockModel 用于测试 SQLite 读写与事务的极简 GORM 模型。
type mockModel struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:64"`
}

// cleanMockDB 删除 mock SQLite 文件，保证用例间数据隔离。
// 必须等 DB 连接关闭后才能删除，因此调用方需保证清理时序。
func cleanMockDB(t *testing.T) {

	t.Helper()
	path, err := db.GetMockPath()
	require.NoError(t, err, "db.GetMockPath FAIL")
	if _, err := os.Stat(path); err == nil {
		require.NoError(t, os.Remove(path), "删除 mock SQLite 文件失败")
	}
}

// newMockDB 创建基于 SQLite 的 mock DB，并注册统一的清理逻辑：
// 测试结束时先关闭连接，再删除 mock 文件，避免文件锁残留。
func newMockDB(t *testing.T) *db.DB {

	t.Helper()
	cleanMockDB(t)
	d, err := db.NewMockDB()
	require.NoError(t, err, "db.NewMockDB FAIL")
	t.Cleanup(func() {
		d.Close()
		cleanMockDB(t)
	})
	return d
}

// migrateModel 在 mock DB 上建表，供读写/事务用例复用。
func migrateModel(t *testing.T, d *db.DB) {

	t.Helper()
	ctx := context.Background()
	require.NoError(t, d.Writer(ctx).AutoMigrate(&mockModel{}), "AutoMigrate FAIL")
}

// TestNewDBEmptyDSN 覆盖异常场景：NewDB 在 Reader/Writer DSN 任一为空时必须返回错误，
// 不能静默创建缺失连接的 DB 实例。
func TestNewDBEmptyDSN(t *testing.T) {

	tests := []struct {
		name string
		cfg  *conf.DBCfg
	}{
		{"reader 为空", &conf.DBCfg{Reader: "", Writer: "mysql://writer"}},
		{"writer 为空", &conf.DBCfg{Reader: "mysql://reader", Writer: ""}},
		{"reader 与 writer 均为空", &conf.DBCfg{Reader: "", Writer: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			d, err := db.NewDB(tt.cfg)
			require.Error(t, err)
			if d != nil {
				d.Close()
			}
		})
	}
}

// TestNewMockDB 覆盖基本场景：NewMockDB 应成功创建，且 Writer/Reader 均可用。
func TestNewMockDB(t *testing.T) {

	// case: NewMockDB 成功创建, Writer/Reader 均可用
	d := newMockDB(t)
	assert.NotNil(t, d.Writer(context.Background()))
	assert.NotNil(t, d.Reader(context.Background()))
}

// TestMockDBFileCreated 验证 NewMockDB 会自动创建 SQLite mock 文件，
// 且文件路径与 GetMockPath 返回的路径一致。
func TestMockDBFileCreated(t *testing.T) {

	// case: NewMockDB 自动创建 SQLite 文件, 路径与 GetMockPath 一致
	cleanMockDB(t)
	d, err := db.NewMockDB()
	require.NoError(t, err, "db.NewMockDB FAIL")
	defer func() {
		d.Close()
		cleanMockDB(t)
	}()

	path, err := db.GetMockPath()
	require.NoError(t, err, "db.GetMockPath FAIL")
	assert.FileExists(t, path)
}

// TestDBMockReadWrite 验证 mock 模式下数据一致性：通过 Writer 写入的数据，
// Reader 必须能读回，即 reader 与 writer 指向同一个 SQLite 文件。
func TestDBMockReadWrite(t *testing.T) {

	// case: Writer 写入的数据, Reader 必须能读回(读写指向同一 SQLite)
	d := newMockDB(t)
	migrateModel(t, d)
	ctx := context.Background()

	want := mockModel{Name: "mock-user"}
	require.NoError(t, d.Writer(ctx).Create(&want).Error, "Writer Create FAIL")
	assert.NotZero(t, want.ID)

	var got mockModel
	require.NoError(t, d.Reader(ctx).First(&got, want.ID).Error, "Reader First FAIL")
	assert.Equal(t, want.Name, got.Name)
}

// TestDBTransactionRollback 覆盖异常场景：事务内返回错误时，已写入的数据必须被回滚。
func TestDBTransactionRollback(t *testing.T) {

	// case: 事务内返回错误, 已写入的数据必须被回滚
	d := newMockDB(t)
	migrateModel(t, d)
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

// TestDBTransactionCommit 覆盖基本场景：事务内不返回错误时，数据应被提交并可被读回。
func TestDBTransactionCommit(t *testing.T) {

	// case: 事务内无错误, 数据应被提交并可读回
	d := newMockDB(t)
	migrateModel(t, d)
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

// TestWriterReaderWithoutTx 验证无事务时 Writer 与 Reader 返回各自独立的连接。
func TestWriterReaderWithoutTx(t *testing.T) {

	// case: 无事务时 Writer 与 Reader 应为独立的连接
	d := newMockDB(t)
	ctx := context.Background()
	assert.NotSame(t, d.Writer(ctx), d.Reader(ctx))
}

// TestContextTxPropagation 验证事务内 GetContextTx 返回非空事务句柄，
// 且 Writer/Reader 均复用该句柄，保证事务内读写落在同一事务上。
func TestContextTxPropagation(t *testing.T) {

	// case: 事务内 GetContextTx 非空, 且 Writer/Reader 复用同一事务句柄
	d := newMockDB(t)
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

// TestSetContextTxAndGetContextTx 验证 SetContextTx/GetContextTx 往返后句柄一致。
func TestSetContextTxAndGetContextTx(t *testing.T) {

	// case: SetContextTx/GetContextTx 往返后句柄一致
	d := newMockDB(t)
	dummy := d.Writer(context.Background())

	ctx := db.SetContextTx(context.Background(), dummy)
	assert.Same(t, dummy, db.GetContextTx(ctx))
}

// TestGetContextTxEmpty 覆盖异常场景：未注入事务的 ctx 应返回 nil。
func TestGetContextTxEmpty(t *testing.T) {

	// case: 未注入事务的 ctx, GetContextTx 应返回 nil
	assert.Nil(t, db.GetContextTx(context.Background()))
}

// TestDBCloseIdempotent 覆盖异常场景：Close 重复调用（含底层连接已关闭）不应 panic。
func TestDBCloseIdempotent(t *testing.T) {

	// case: Close 重复调用(含底层已关闭)不应 panic
	d := newMockDB(t)
	d.Close()
	d.Close()
}
