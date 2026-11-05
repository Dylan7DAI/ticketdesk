package store

import (
	"context"
	"strconv"
	"strings"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// escapeLike 转义 ILIKE 的通配符，让 q 中的 % _ \ 按字面匹配（WF§17.2）。
// 反斜杠必须最先处理，否则会把后面加上的转义符再转义一次。
var escapeLike = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// where 构造 WHERE 子句。拼进 SQL 的只有固定的列名和 $n 占位符，用户给的值全部走参数。
func (f TicketFilter) where() (string, []any) {
	var conds []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if len(f.Statuses) > 0 {
		vals := make([]string, len(f.Statuses))
		for i, s := range f.Statuses {
			vals[i] = string(s)
		}
		add("t.status::text = ANY(?)", vals)
	}
	if len(f.Priorities) > 0 {
		vals := make([]string, len(f.Priorities))
		for i, p := range f.Priorities {
			vals[i] = string(p)
		}
		add("t.priority::text = ANY(?)", vals)
	}
	if a := f.Assignee; a != nil {
		if a.Unassigned {
			conds = append(conds, "t.assignee_id IS NULL")
		} else {
			add("t.assignee_id = ?", a.UserID)
		}
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		add("(t.title ILIKE ? OR t.customer_name ILIKE ?)", "%"+escapeLike.Replace(q)+"%")
	}
	if f.VisibleTo != nil {
		add("((t.status = 'open' AND t.assignee_id IS NULL) OR t.assignee_id = ?)", *f.VisibleTo)
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func (f TicketFilter) paging() (limit, offset int) {
	page, size := f.Page, f.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return size, (page - 1) * size
}

func (q *queries) ListTickets(ctx context.Context, f TicketFilter) ([]TicketRow, int, error) {
	where, args := f.where()

	var total int
	if err := q.db.QueryRow(ctx, `SELECT count(*) FROM tickets t`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit, offset := f.paging()
	args = append(args, limit, offset)
	n := len(args)
	rows, err := q.db.Query(ctx,
		`SELECT `+listCols+`, u.name
		 FROM tickets t LEFT JOIN users u ON u.id = t.assignee_id`+where+
			` ORDER BY t.created_at DESC, t.id DESC
		 LIMIT $`+strconv.Itoa(n-1)+` OFFSET $`+strconv.Itoa(n), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []TicketRow{}
	for rows.Next() {
		var name *string
		t, err := scanTicket(rows, &name)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, TicketRow{Ticket: *t, AssigneeName: name})
	}
	return items, total, rows.Err()
}

// listCols 是带 t. 前缀的 ticketCols：与 users 联表后 id、created_at 会有歧义。
const listCols = `t.id, t.title, t.description, t.customer_name, t.customer_contact,
	t.priority::text, t.status::text, t.assignee_id, t.created_by, t.sla_due_at, t.created_at, t.updated_at`
