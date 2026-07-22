package test

import (
	"context"
	"gin-demo/src/core/conf"
	"gin-demo/src/core/db"
	"os"
	"path/filepath"
	"testing"
)

// findProjectRootByGoMod 从当前工作目录向上查找 go.mod，定位项目根。
// 测试中用来验证 db 包内 findProjectRoot 的行为是否正确。
func findProjectRootByGoMod(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd FAIL: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found in test")
		}
		dir = parent
	}
}

// cleanMockDBIfExists 在测试前后清理 mock SQLite 文件，保证用例独立。
func cleanMockDBIfExists(t *testing.T) {
	t.Helper()
	root := findProjectRootByGoMod(t)
	p := filepath.Join(root, "data", "mock.db.sqlite")
	if _, err := os.Stat(p); err == nil {
		if err := os.Remove(p); err != nil {
			t.Fatalf("清理 mock.db.sqlite 失败: %v", err)
		}
	}
}

// TestDBInitSQLite 验证当 DBCfg 中 reader/writer 均为空时，NewDB 会启用 SQLite mock。
func TestDBInitSQLite(t *testing.T) {
	cleanMockDBIfExists(t)
	defer cleanMockDBIfExists(t)

	// DSN 为空 → 触发 SQLite mock
	cfg := &conf.DBCfg{Reader: "", Writer: ""}
	d, err := db.NewDB(cfg)
	if err != nil {
		t.Fatalf("db.NewDB(SQLite mock) FAIL: %v", err)
	}
	defer d.Close()

	// 验证 mock SQLite 文件被自动创建
	root := findProjectRootByGoMod(t)
	mockPath := filepath.Join(root, "data", "mock.db.sqlite")
	if _, err := os.Stat(mockPath); os.IsNotExist(err) {
		t.Fatalf("SQLite mock 未被创建: %s", mockPath)
	}
}

// mockModel 用于测试 SQLite 读写的一个极简 GORM 模型。
type mockModel struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:64"`
}

// TestDBSQLiteReadWrite 验证启用 SQLite mock 后，写库写一条数据、读库能读回来。
// 同时验证 reader 与 writer 在 mock 模式下指向同一个 SQLite 文件。
func TestDBSQLiteReadWrite(t *testing.T) {
	cleanMockDBIfExists(t)
	defer cleanMockDBIfExists(t)

	cfg := &conf.DBCfg{Reader: "", Writer: ""}
	d, err := db.NewDB(cfg)
	if err != nil {
		t.Fatalf("db.NewDB(SQLite mock) FAIL: %v", err)
	}
	defer d.Close()

	ctx := context.Background()

	// AutoMigrate 建表，测试写连接是否可用
	if err := d.Writer(ctx).AutoMigrate(&mockModel{}); err != nil {
		t.Fatalf("Writer AutoMigrate FAIL: %v", err)
	}

	// 通过写连接插入一条记录
	want := mockModel{Name: "mock-user"}
	if err := d.Writer(ctx).Create(&want).Error; err != nil {
		t.Fatalf("Writer Create FAIL: %v", err)
	}
	if want.ID == 0 {
		t.Fatal("Writer Create 未生成主键 ID")
	}

	// 通过读连接查询，验证 reader / writer 指向同一个 SQLite 文件
	var got mockModel
	if err := d.Reader(ctx).First(&got, want.ID).Error; err != nil {
		t.Fatalf("Reader First FAIL: %v", err)
	}
	if got.Name != want.Name {
		t.Fatalf("Reader 读到的数据不一致: want=%s got=%s", want.Name, got.Name)
	}
}

// TestDBTransaction 验证在 SQLite mock 模式下事务的原子性：
// 事务内返回错误时，已写入的数据应当被回滚。
func TestDBTransaction(t *testing.T) {
	cleanMockDBIfExists(t)
	defer cleanMockDBIfExists(t)

	cfg := &conf.DBCfg{Reader: "", Writer: ""}
	d, err := db.NewDB(cfg)
	if err != nil {
		t.Fatalf("db.NewDB(SQLite mock) FAIL: %v", err)
	}
	defer d.Close()

	ctx := context.Background()

	if err := d.Writer(ctx).AutoMigrate(&mockModel{}); err != nil {
		t.Fatalf("Writer AutoMigrate FAIL: %v", err)
	}

	wantErr := context.Canceled
	// 事务内插入一条记录，然后返回错误，期望记录被回滚
	err = d.Transaction(ctx, func(txCtx context.Context) error {
		if err := d.Writer(txCtx).Create(&mockModel{Name: "tx-rollback"}).Error; err != nil {
			t.Fatalf("Transaction Writer Create FAIL: %v", err)
		}
		return wantErr
	})
	if err == nil {
		t.Fatal("Transaction 未返回预期错误")
	}

	// 验证事务回滚后记录数量为 0
	var count int64
	if err := d.Reader(ctx).Model(&mockModel{}).Count(&count).Error; err != nil {
		t.Fatalf("Reader Count FAIL: %v", err)
	}
	if count != 0 {
		t.Fatalf("事务未回滚，记录数: %d", count)
	}
}

// TestContextTxDB 验证 SetContextTxDB / GetContextTxDB 能够正确地在 context 中存取事务句柄。
func TestContextTxDB(t *testing.T) {
	cleanMockDBIfExists(t)
	defer cleanMockDBIfExists(t)

	cfg := &conf.DBCfg{Reader: "", Writer: ""}
	d, err := db.NewDB(cfg)
	if err != nil {
		t.Fatalf("db.NewDB(SQLite mock) FAIL: %v", err)
	}
	defer d.Close()

	ctx := context.Background()

	if err := d.Writer(ctx).AutoMigrate(&mockModel{}); err != nil {
		t.Fatalf("Writer AutoMigrate FAIL: %v", err)
	}

	var capturedTx interface{}
	_ = d.Transaction(ctx, func(txCtx context.Context) error {
		capturedTx = db.GetContextTxDB(txCtx)
		if capturedTx == nil {
			t.Fatal("GetContextTxDB 在事务内返回 nil，预期为非空 *gorm.DB")
		}

		// 事务内插入记录，不返回错误以便提交；后续验证能查到
		if err := d.Writer(txCtx).Create(&mockModel{Name: "tx-commit"}).Error; err != nil {
			t.Fatalf("事务内 Create FAIL: %v", err)
		}
		return nil
	})

	var count int64
	if err := d.Reader(ctx).Model(&mockModel{}).Where("name = ?", "tx-commit").Count(&count).Error; err != nil {
		t.Fatalf("Reader Count FAIL: %v", err)
	}
	if count != 1 {
		t.Fatalf("事务提交后记录数应为 1，实际为 %d", count)
	}
}
