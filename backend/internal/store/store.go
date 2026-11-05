// Package store 集中所有 SQL、事务和行锁。这里不做业务判断（不写 if status == ...），
// 业务规则在 domain，流程编排在 service。
package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store 持有连接池。
type Store struct{ pool *pgxpool.Pool }

// Open 建立连接池并 Ping 一次，连不上时立即报错。
func Open(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

// Q 返回池上的非事务 Querier，读操作直接使用。
func (s *Store) Q() Querier { return &queries{db: s.pool} }

// InTx 在一个事务中执行 fn：fn 返回错误或 panic 时回滚，否则提交。
// 回滚用 WithoutCancel，请求被取消时也要把回滚发出去，而不是留给连接超时。
func (s *Store) InTx(ctx context.Context, fn func(q Querier) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(p)
		}
	}()
	if err := fn(&queries{db: tx}); err != nil {
		_ = tx.Rollback(context.WithoutCancel(ctx))
		return err
	}
	return tx.Commit(ctx)
}
