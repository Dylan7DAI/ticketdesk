package store

import (
	"context"
	"encoding/json"
)

func (q *queries) InsertComment(ctx context.Context, c *Comment) error {
	return q.db.QueryRow(ctx,
		`INSERT INTO ticket_comments (ticket_id, author_id, body) VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		c.TicketID, c.AuthorID, c.Body,
	).Scan(&c.ID, &c.CreatedAt)
}

func (q *queries) InsertEvent(ctx context.Context, e *Event) error {
	payload := e.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	return q.db.QueryRow(ctx,
		`INSERT INTO ticket_events (ticket_id, actor_id, type, payload) VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		e.TicketID, e.ActorID, e.Type, []byte(payload),
	).Scan(&e.ID, &e.CreatedAt)
}

// ListComments 按 created_at, id 升序，id 保证同一时刻的记录顺序稳定。
func (q *queries) ListComments(ctx context.Context, ticketID int64) ([]Comment, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id, ticket_id, author_id, body, created_at FROM ticket_comments
		 WHERE ticket_id = $1 ORDER BY created_at, id`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (q *queries) ListEvents(ctx context.Context, ticketID int64) ([]Event, error) {
	rows, err := q.db.Query(ctx,
		`SELECT id, ticket_id, actor_id, type, payload, created_at FROM ticket_events
		 WHERE ticket_id = $1 ORDER BY created_at, id`, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		var payload []byte
		if err := rows.Scan(&e.ID, &e.TicketID, &e.ActorID, &e.Type, &payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Payload = payload
		out = append(out, e)
	}
	return out, rows.Err()
}
