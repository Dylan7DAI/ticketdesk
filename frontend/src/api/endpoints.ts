import { request } from './client'
import type { Comment, Priority, Status, TicketDetail, TicketPage, User } from './types'

// 每个接口一个函数，第一个参数都是当前用户 ID（X-User-ID）。

export interface ListParams {
  status?: Status[]
  priority?: Priority[]
  assignee_id?: string // 用户 ID 或 'none'
  q?: string
  page?: number
  page_size?: number
}

export const api = {
  users: (me: number) => request<User[]>('/api/users', { userId: me }),

  listTickets: (me: number, p: ListParams) =>
    request<TicketPage>('/api/tickets', {
      userId: me,
      query: {
        status: p.status?.join(','),
        priority: p.priority?.join(','),
        assignee_id: p.assignee_id,
        q: p.q,
        page: p.page,
        page_size: p.page_size,
      },
    }),

  getTicket: (me: number, id: number) => request<TicketDetail>(`/api/tickets/${id}`, { userId: me }),

  createTicket: (me: number, body: CreateBody) =>
    request<TicketDetail>('/api/tickets', { userId: me, method: 'POST', body }),

  claim: (me: number, id: number) =>
    request<TicketDetail>(`/api/tickets/${id}/claim`, { userId: me, method: 'POST' }),

  assign: (me: number, id: number, assignee_id: number) =>
    request<TicketDetail>(`/api/tickets/${id}/assign`, { userId: me, method: 'POST', body: { assignee_id } }),

  reassign: (me: number, id: number, assignee_id: number) =>
    request<TicketDetail>(`/api/tickets/${id}/reassign`, { userId: me, method: 'POST', body: { assignee_id } }),

  transition: (me: number, id: number, action: 'resolve' | 'reopen' | 'close', note: string) =>
    request<TicketDetail>(`/api/tickets/${id}/${action}`, { userId: me, method: 'POST', body: { note } }),

  comment: (me: number, id: number, body: string) =>
    request<Comment>(`/api/tickets/${id}/comments`, { userId: me, method: 'POST', body: { body } }),
}

export interface CreateBody {
  title: string
  description: string
  customer_name: string
  customer_contact: string
  priority: Priority
}
