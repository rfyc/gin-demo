package db_test

import (
	"fmt"
	"gin-demo/src/core/db"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newInMemoryDB 创建内存 SQLite 的 gorm DB，并建好 mock_models 表，供 Option 测试使用。
func newInMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(sqlite) FAIL: %v", err)
	}
	if err := gdb.AutoMigrate(&mockModel{}); err != nil {
		t.Fatalf("AutoMigrate FAIL: %v", err)
	}
	return gdb
}

// seedModels 向 gdb 插入 n 条有序数据（Name 依次为 n0..n(n-1)）。
func seedModels(t *testing.T, gdb *gorm.DB, n int) {
	t.Helper()
	seeds := make([]mockModel, 0, n)
	for i := 0; i < n; i++ {
		seeds = append(seeds, mockModel{Name: fmt.Sprintf("n%d", i)})
	}
	if err := gdb.Create(&seeds).Error; err != nil {
		t.Fatalf("Create seeds FAIL: %v", err)
	}
}

// TestOptAppendWhere 验证 Where 方法生成的 Option 能正确过滤数据。
func TestOptAppendWhere(t *testing.T) {
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 3)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.Where("name = ?", "n1"))
	if err := gdb.Find(&list).Error; err != nil {
		t.Fatalf("Find FAIL: %v", err)
	}
	if len(list) != 1 || list[0].Name != "n1" {
		t.Fatalf("Where 过滤结果不符: %+v", list)
	}
}

// TestOptAppendOr 验证 Or 方法生成的 Option 与 Where 组合时能取并集。
func TestOptAppendOr(t *testing.T) {
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 3)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.Where("name = ?", "n0"), db.Opts.Or("name = ?", "n2"))
	if err := gdb.Find(&list).Error; err != nil {
		t.Fatalf("Find FAIL: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Or 组合结果数量不符: want=2 got=%d (%+v)", len(list), list)
	}
}

// TestOptAppendLimit 验证 Limit 方法生成的 Option 能限制返回行数。
func TestOptAppendLimit(t *testing.T) {
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 5)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.Limit(2))
	if err := gdb.Find(&list).Error; err != nil {
		t.Fatalf("Find FAIL: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Limit 结果数量不符: want=2 got=%d", len(list))
	}
}

// TestOptAppendOrderBy 验证 OrderBy 方法生成的 Option 能按指定排序返回。
func TestOptAppendOrderBy(t *testing.T) {
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 3)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.OrderBy("id DESC"))
	if err := gdb.Find(&list).Error; err != nil {
		t.Fatalf("Find FAIL: %v", err)
	}
	if len(list) != 3 || list[0].Name != "n2" {
		t.Fatalf("OrderBy 结果不符: %+v", list)
	}
}

// TestOptAppendPage 验证 Page 方法生成的分页 Option 能正确取指定页数据。
func TestOptAppendPage(t *testing.T) {
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 5)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.OrderBy("id"), db.Opts.Page(2, 2))
	if err := gdb.Find(&list).Error; err != nil {
		t.Fatalf("Find FAIL: %v", err)
	}
	// 第 2 页、每页 2 条 → 第 3、4 条（n2, n3）
	if len(list) != 2 || list[0].Name != "n2" || list[1].Name != "n3" {
		t.Fatalf("Page 结果不符: %+v", list)
	}
}

// TestOptPageDefaultSize 验证 Page 在 page/pageSize 均为 0 时取默认每页 20 条。
func TestOptPageDefaultSize(t *testing.T) {
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 25)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.OrderBy("id"), db.Opts.Page(0, 0))
	if err := gdb.Find(&list).Error; err != nil {
		t.Fatalf("Find FAIL: %v", err)
	}
	if len(list) != 20 {
		t.Fatalf("Page 默认每页应为 20，实际为 %d", len(list))
	}
}

// TestOptAppendTable 验证 Table 方法生成的 Option 能指定查询表名。
func TestOptAppendTable(t *testing.T) {
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 3)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.Table("mock_models"))
	if err := gdb.Find(&list).Error; err != nil {
		t.Fatalf("Find FAIL: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("Table 查询结果数量不符: want=3 got=%d", len(list))
	}
}

// TestOptAppendEmpty 验证 Empty 方法生成的 Option 不改变查询结果。
func TestOptAppendEmpty(t *testing.T) {
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 3)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.Empty())
	if err := gdb.Find(&list).Error; err != nil {
		t.Fatalf("Find FAIL: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("Empty 不应改变查询，数量不符: want=3 got=%d", len(list))
	}
}

// TestOptChainTwoWhere 验证链式调用时两个 Where 均生效（逐级叠加），
// 而非只有最后一个生效。
func TestOptChainTwoWhere(t *testing.T) {
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 4)

	var list []mockModel
	gdb = db.Opts.Append(gdb,
		db.Opts.Where("name != ?", "n0").Where("name != ?", "n1").Page(1, 2),
	)
	if err := gdb.Find(&list).Error; err != nil {
		t.Fatalf("Find FAIL: %v", err)
	}
	// 排除 n0、n1 后取第 1 页 2 条 → n2, n3
	if len(list) != 2 || list[0].Name != "n2" || list[1].Name != "n3" {
		t.Fatalf("链式双 Where 结果不符: %+v", list)
	}
}

// TestOptAppendChain 验证多个 Option 按序组合后的查询结果正确。
func TestOptAppendChain(t *testing.T) {
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 6)

	var list []mockModel
	gdb = db.Opts.Append(gdb,
		db.Opts.Where("name != ?", "n0"),
		db.Opts.OrderBy("id DESC"),
		db.Opts.Limit(2),
	)
	if err := gdb.Find(&list).Error; err != nil {
		t.Fatalf("Find FAIL: %v", err)
	}
	// 排除 n0 后倒序取前 2 → n5, n4
	if len(list) != 2 || list[0].Name != "n5" || list[1].Name != "n4" {
		t.Fatalf("链式组合结果不符: %+v", list)
	}
}
