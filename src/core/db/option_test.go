package db_test

import (
	"fmt"
	"gin-demo/src/core/db"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newInMemoryDB 创建内存 SQLite 的 gorm DB，并建好 mock_models 表，供 Option 测试使用。
func newInMemoryDB(t *testing.T) *gorm.DB {

	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "gorm.Open(sqlite) FAIL")
	require.NoError(t, gdb.AutoMigrate(&mockModel{}), "AutoMigrate FAIL")
	return gdb
}

// seedModels 向 gdb 插入 n 条有序数据（Name 依次为 n0..n(n-1)）。
func seedModels(t *testing.T, gdb *gorm.DB, n int) {

	t.Helper()
	seeds := make([]mockModel, 0, n)
	for i := 0; i < n; i++ {
		seeds = append(seeds, mockModel{Name: fmt.Sprintf("n%d", i)})
	}
	require.NoError(t, gdb.Create(&seeds).Error, "Create seeds FAIL")
}

// TestOptAppendWhere 验证 Where 方法生成的 Option 能正确过滤数据。
func TestOptAppendWhere(t *testing.T) {

	// case: Where 生成的 Option 能正确过滤数据
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 3)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.Where("name = ?", "n1"))
	require.NoError(t, gdb.Find(&list).Error)
	assert.Len(t, list, 1)
	assert.Equal(t, "n1", list[0].Name)
}

// TestOptAppendOr 验证 Or 方法生成的 Option 与 Where 组合时能取并集。
func TestOptAppendOr(t *testing.T) {

	// case: Or 与 Where 组合时取并集
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 3)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.Where("name = ?", "n0"), db.Opts.Or("name = ?", "n2"))
	require.NoError(t, gdb.Find(&list).Error)
	assert.Len(t, list, 2)
}

// TestOptAppendLimit 验证 Limit 方法生成的 Option 能限制返回行数。
func TestOptAppendLimit(t *testing.T) {

	// case: Limit 生成的 Option 能限制返回行数
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 5)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.Limit(2))
	require.NoError(t, gdb.Find(&list).Error)
	assert.Len(t, list, 2)
}

// TestOptAppendOrderBy 验证 OrderBy 方法生成的 Option 能按指定排序返回。
func TestOptAppendOrderBy(t *testing.T) {

	// case: OrderBy 生成的 Option 能按指定排序返回
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 3)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.OrderBy("id DESC"))
	require.NoError(t, gdb.Find(&list).Error)
	assert.Len(t, list, 3)
	assert.Equal(t, "n2", list[0].Name)
}

// TestOptAppendPage 验证 Page 方法生成的分页 Option 能正确取指定页数据。
func TestOptAppendPage(t *testing.T) {

	// case: Page 生成的分页 Option 能正确取指定页数据
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 5)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.OrderBy("id"), db.Opts.Page(2, 2))
	require.NoError(t, gdb.Find(&list).Error)
	// 第 2 页、每页 2 条 → 第 3、4 条（n2, n3）
	assert.Len(t, list, 2)
	assert.Equal(t, "n2", list[0].Name)
	assert.Equal(t, "n3", list[1].Name)
}

// TestOptPageDefaultSize 验证 Page 在 page/pageSize 均为 0 时取默认每页 20 条。
func TestOptPageDefaultSize(t *testing.T) {

	// case: Page 的 page/pageSize 均为 0 时取默认每页 20 条
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 25)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.OrderBy("id"), db.Opts.Page(0, 0))
	require.NoError(t, gdb.Find(&list).Error)
	assert.Len(t, list, 20)
}

// TestOptAppendTable 验证 Table 方法生成的 Option 能指定查询表名。
func TestOptAppendTable(t *testing.T) {

	// case: Table 生成的 Option 能指定查询表名
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 3)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.Table("mock_models"))
	require.NoError(t, gdb.Find(&list).Error)
	assert.Len(t, list, 3)
}

// TestOptAppendEmpty 验证 Empty 方法生成的 Option 不改变查询结果。
func TestOptAppendEmpty(t *testing.T) {

	// case: Empty 生成的 Option 不改变查询结果
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 3)

	var list []mockModel
	gdb = db.Opts.Append(gdb, db.Opts.Empty())
	require.NoError(t, gdb.Find(&list).Error)
	assert.Len(t, list, 3)
}

// TestOptChainTwoWhere 验证链式调用时两个 Where 均生效（逐级叠加），
// 而非只有最后一个生效。
func TestOptChainTwoWhere(t *testing.T) {

	// case: 链式调用时两个 Where 均生效(逐级叠加)
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 4)

	var list []mockModel
	gdb = db.Opts.Append(gdb,
		db.Opts.Where("name != ?", "n0").Where("name != ?", "n1").Page(1, 2),
	)
	require.NoError(t, gdb.Find(&list).Error)
	// 排除 n0、n1 后取第 1 页 2 条 → n2, n3
	assert.Len(t, list, 2)
	assert.Equal(t, "n2", list[0].Name)
	assert.Equal(t, "n3", list[1].Name)
}

// TestOptAppendChain 验证多个 Option 按序组合后的查询结果正确。
func TestOptAppendChain(t *testing.T) {

	// case: 多个 Option 按序组合后的查询结果正确
	gdb := newInMemoryDB(t)
	seedModels(t, gdb, 6)

	var list []mockModel
	gdb = db.Opts.Append(gdb,
		db.Opts.Where("name != ?", "n0"),
		db.Opts.OrderBy("id DESC"),
		db.Opts.Limit(2),
	)
	require.NoError(t, gdb.Find(&list).Error)
	// 排除 n0 后倒序取前 2 → n5, n4
	assert.Len(t, list, 2)
	assert.Equal(t, "n5", list[0].Name)
	assert.Equal(t, "n4", list[1].Name)
}
