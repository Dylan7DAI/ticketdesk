package service

import (
	"context"

	"ticketdesk/internal/domain"
	"ticketdesk/internal/store"
)

// Authenticate 按用户 ID 取出用户；不存在返回 UNAUTHENTICATED。角色只取自数据库（WF§2）。
func (s *Service) Authenticate(ctx context.Context, userID int64) (*domain.User, error) {
	u, err := s.db.Q().GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, domain.ErrUnauthenticated()
	}
	return u, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.db.Q().ListUsers(ctx)
}

// userNames 一次性取出全部用户的名称，供列表和历史渲染使用，避免循环里逐条查询。
func userNames(ctx context.Context, q store.Querier) (names, error) {
	users, err := q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	n := make(names, len(users))
	for _, u := range users {
		n[u.ID] = u.Name
	}
	return n, nil
}

// ListTickets 返回分页列表。actor 是 agent 时强制设置 VisibleTo，覆盖调用方传入的任何值（WF§11）。
func (s *Service) ListTickets(ctx context.Context, actor domain.User, f store.TicketFilter) (*TicketPage, error) {
	if actor.Role == domain.RoleAgent {
		id := actor.ID
		f.VisibleTo = &id
	} else {
		f.VisibleTo = nil
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	rows, total, err := s.db.Q().ListTickets(ctx, f)
	if err != nil {
		return nil, err
	}
	now := s.now()
	items := make([]TicketSummary, len(rows))
	for i, r := range rows {
		var a *UserRef
		if r.AssigneeID != nil && r.AssigneeName != nil {
			a = &UserRef{ID: *r.AssigneeID, Name: *r.AssigneeName}
		}
		items[i] = summarize(r.Ticket, a, now)
	}
	return &TicketPage{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}

// GetTicketDetail 返回详情。不存在 → 404；agent 不可见 → 403 NOT_ASSIGNEE。
// 用 Authorize(view) 判定可见性，与写操作共用同一套规则。
func (s *Service) GetTicketDetail(ctx context.Context, actor domain.User, id int64) (*TicketDetail, error) {
	q := s.db.Q()
	t, err := q.GetTicket(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, domain.ErrTicketNotFound()
	}
	if err := domain.Authorize(actor, *t, domain.ActionView); err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, q, actor, t)
}
