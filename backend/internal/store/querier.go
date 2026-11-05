package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"ticketdesk/internal/domain"
)

// Querier 同时由连接池和事务实现：读操作用 Store.Q()，写操作在 InTx 内使用。
type Querier interface {
	GetUser(ctx context.Context, id int64) (*domain.User, error) // 不存在返回 nil, nil
	ListUsers(ctx context.Context) ([]domain.User, error)
	// LockTicket 即 SELECT ... FOR UPDATE；不存在返回 nil, nil。
	LockTicket(ctx context.Context, id int64) (*domain.Ticket, error)
	GetTicket(ctx context.Context, id int64) (*domain.Ticket, error) // 不加锁
	// InsertTicket 原样写入 t 的全部字段（含三个时间），回填 ID。
	InsertTicket(ctx context.Context, t *domain.Ticket) error
	// UpdateTicketState 同时设 updated_at = now()。
	UpdateTicketState(ctx context.Context, id int64, s domain.Status, assignee *int64) error
	InsertComment(ctx context.Context, c *Comment) error // 回填 ID、CreatedAt
	InsertEvent(ctx context.Context, e *Event) error     // 回填 ID、CreatedAt
	ListTickets(ctx context.Context, f TicketFilter) (items []TicketRow, total int, err error)
	ListComments(ctx context.Context, ticketID int64) ([]Comment, error)
	ListEvents(ctx context.Context, ticketID int64) ([]Event, error)
}

// dbtx 是 *pgxpool.Pool 和 pgx.Tx 共有的方法集。
type dbtx interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type queries struct{ db dbtx }

var _ Querier = (*queries)(nil)
