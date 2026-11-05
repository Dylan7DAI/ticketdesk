package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"ticketdesk/internal/domain"
)

// ticketCols 与 scanTicket 的字段顺序必须一致。枚举转成 text，避免 pgx 对自定义类型的解码问题。
const ticketCols = `id, title, description, customer_name, customer_contact,
	priority::text, status::text, assignee_id, created_by, sla_due_at, created_at, updated_at`

func scanTicket(row pgx.Row, extra ...any) (*domain.Ticket, error) {
	var t domain.Ticket
	var prio, status string
	dest := append([]any{
		&t.ID, &t.Title, &t.Description, &t.CustomerName, &t.CustomerContact,
		&prio, &status, &t.AssigneeID, &t.CreatedBy, &t.SLADueAt, &t.CreatedAt, &t.UpdatedAt,
	}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	t.Priority, t.Status = domain.Priority(prio), domain.Status(status)
	return &t, nil
}

func (q *queries) getTicket(ctx context.Context, id int64, suffix string) (*domain.Ticket, error) {
	t, err := scanTicket(q.db.QueryRow(ctx,
		`SELECT `+ticketCols+` FROM tickets WHERE id = $1`+suffix, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (q *queries) GetTicket(ctx context.Context, id int64) (*domain.Ticket, error) {
	return q.getTicket(ctx, id, "")
}

// LockTicket 锁住工单行直到事务结束；并发的写操作在这里排队（WF§7）。
func (q *queries) LockTicket(ctx context.Context, id int64) (*domain.Ticket, error) {
	return q.getTicket(ctx, id, " FOR UPDATE")
}

func (q *queries) InsertTicket(ctx context.Context, t *domain.Ticket) error {
	return q.db.QueryRow(ctx,
		`INSERT INTO tickets (title, description, customer_name, customer_contact,
			priority, status, assignee_id, created_by, sla_due_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`,
		t.Title, t.Description, t.CustomerName, t.CustomerContact,
		string(t.Priority), string(t.Status), t.AssigneeID, t.CreatedBy,
		t.SLADueAt, t.CreatedAt, t.UpdatedAt,
	).Scan(&t.ID)
}

func (q *queries) UpdateTicketState(ctx context.Context, id int64, s domain.Status, assignee *int64) error {
	tag, err := q.db.Exec(ctx,
		`UPDATE tickets SET status = $2, assignee_id = $3, updated_at = now() WHERE id = $1`,
		id, string(s), assignee)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("store: UpdateTicketState 未更新到恰好一行")
	}
	return nil
}
