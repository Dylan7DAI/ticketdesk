// Package testutil 提供集成测试用的测试库辅助函数。只被 _test.go 文件引用。
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"ticketdesk/internal/store"
	"ticketdesk/migrations"
)

// moduleRoot 从当前目录逐级向上找含 go.mod 的目录。go test 的工作目录是各包目录，
// 不能用固定的相对路径。
func moduleRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("testutil: 向上找不到 go.mod")
		}
		dir = parent
	}
}

// testDSN 读取 TEST_DATABASE_URL：系统环境变量优先，其次是仓库根目录的 .env。
// 未设置时 Fatal 而不是跳过，避免集成测试没有真正运行却显示通过。
func testDSN(t testing.TB) string {
	t.Helper()
	_ = godotenv.Load(filepath.Join(moduleRoot(t), "..", ".env"))
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL 未设置：先运行 npm run db:up 和 npm run setup")
	}
	return dsn
}

// 固定的演示用户，ID 与 WF§1 一致。
const seedUsers = `
INSERT INTO users (id, name, role) VALUES
  (1, '主管 王敏', 'supervisor'), (2, '客服 李雷', 'agent'),
  (3, '客服 韩梅', 'agent'),      (4, '客服 张伟', 'agent');
SELECT setval(pg_get_serial_sequence('users', 'id'), 4);`

// NewSeededStore 对测试库执行 migrations.Reset（回滚再迁移），得到 0002_seed 的真实演示数据。
// 下一次 NewTestStore 会清表并重置用户，所以不会影响其他测试。
func NewSeededStore(t testing.TB) *store.Store {
	t.Helper()
	dsn := testDSN(t)
	if err := migrations.Reset(dsn); err != nil {
		t.Fatalf("重置测试库失败: %v", err)
	}
	s, err := store.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("连接测试库失败: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

// NewTestStore 迁移测试库、清空全部表并插入 4 个固定用户，返回连到测试库的 Store。
// 各包共用一个测试库，所以集成测试必须串行（-p 1）。
func NewTestStore(t testing.TB) *store.Store {
	t.Helper()
	dsn := testDSN(t)
	if err := migrations.Up(dsn); err != nil {
		t.Fatalf("迁移测试库失败: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `TRUNCATE ticket_events, ticket_comments, tickets, users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("清表失败: %v", err)
	}
	if _, err := pool.Exec(ctx, seedUsers); err != nil {
		t.Fatalf("插入测试用户失败: %v", err)
	}

	s, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("连接测试库失败: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}
