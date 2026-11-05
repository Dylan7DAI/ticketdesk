// Package migrations 内嵌 SQL 迁移文件并提供 Up/Down，
// 评审者不必另装 migrate CLI；T3.1 的测试辅助函数也复用这里。
package migrations

import (
	"embed"
	"errors"
	"io/fs"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // 注册 pgx5:// 驱动
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// FS 内嵌本目录全部文件。iofs 会跳过名字不符合
// {版本}_{名称}.{up|down}.sql 的文件（如本文件），所以用 * 即可。
//
//go:embed *
var FS embed.FS

// ErrNoMigrations 表示目录中还没有任何迁移文件（T1.1 之前的状态）。
var ErrNoMigrations = errors.New("migrations: 没有找到迁移文件")

// Up 执行全部未应用的迁移；已是最新时返回 nil。
func Up(dsn string) error {
	return run(dsn, func(m *migrate.Migrate) error { return m.Up() })
}

// Down 回滚到版本 0；已经是空库时返回 nil。
func Down(dsn string) error {
	return run(dsn, func(m *migrate.Migrate) error { return m.Down() })
}

// Reset 回滚到版本 0 再重新迁移，用于把演示数据恢复到初始状态。
func Reset(dsn string) error {
	return run(dsn, func(m *migrate.Migrate) error {
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		return m.Up()
	})
}

func run(dsn string, op func(*migrate.Migrate) error) error {
	src, err := iofs.New(FS, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, toPgx5(dsn))
	if err != nil {
		return err
	}
	defer m.Close()

	err = op(m)
	switch {
	case err == nil, errors.Is(err, migrate.ErrNoChange):
		return nil
	case errors.Is(err, fs.ErrNotExist):
		return ErrNoMigrations
	default:
		return err
	}
}

// golang-migrate 的 pgx v5 驱动注册的 scheme 是 pgx5，
// 而 .env 里用的是通用的 postgres://，这里做一次转换。
func toPgx5(dsn string) string {
	for _, p := range []string{"postgres://", "postgresql://"} {
		if strings.HasPrefix(dsn, p) {
			return "pgx5://" + strings.TrimPrefix(dsn, p)
		}
	}
	return dsn
}
