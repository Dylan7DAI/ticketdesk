// Package service 按 WF§5 的固定模板编排写操作：开事务 → 锁行 → 调 domain 判定 → 写库 → 写事件。
// 业务规则只写在 domain 中，这里不重复判断。
package service

import (
	"context"
	"time"

	"ticketdesk/internal/store"
)

// DB 是 service 依赖的存储接口。依赖接口而非 *store.Store，
// 以便测试注入"写事件必定失败"的 Querier 包装（I7）。
type DB interface {
	InTx(ctx context.Context, fn func(q store.Querier) error) error
	Q() store.Querier
}

type Service struct {
	db  DB
	now func() time.Time
}

// New 创建 Service。now 为 nil 时使用系统时钟；测试可注入固定时钟。
func New(db DB, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{db: db, now: now}
}

var _ DB = (*store.Store)(nil)
