package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"ticketdesk/internal/domain"
)

func (q *queries) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	var role string
	err := q.db.QueryRow(ctx,
		`SELECT id, name, role::text FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Name, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.Role = domain.Role(role)
	return &u, nil
}

func (q *queries) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := q.db.Query(ctx, `SELECT id, name, role::text FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.User{}
	for rows.Next() {
		var u domain.User
		var role string
		if err := rows.Scan(&u.ID, &u.Name, &role); err != nil {
			return nil, err
		}
		u.Role = domain.Role(role)
		out = append(out, u)
	}
	return out, rows.Err()
}
