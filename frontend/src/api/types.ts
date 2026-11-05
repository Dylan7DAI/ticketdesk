export type Role = 'agent' | 'supervisor'
export type Status = 'open' | 'in_progress' | 'resolved' | 'closed'
export type Priority = 'urgent' | 'high' | 'normal' | 'low'
export type Action = 'claim' | 'assign' | 'reassign' | 'resolve' | 'reopen' | 'close' | 'comment'

export interface UserRef { id: number; name: string }
export interface User extends UserRef { role: Role }

export interface TicketSummary {
  id: number; title: string; customer_name: string
  priority: Priority; status: Status; assignee: UserRef | null
  sla_due_at: string; is_overdue: boolean; created_at: string   // RFC3339
}
export interface TicketPage { items: TicketSummary[]; total: number; page: number; page_size: number }

export interface Comment { id: number; author: UserRef; body: string; created_at: string }
export interface TicketEvent {
  id: number; actor: UserRef; created_at: string
  type: 'created' | 'claimed' | 'assigned' | 'reassigned' | 'status_changed' | 'commented'
  payload: Record<string, unknown>
  summary: string            // 后端渲染好的中文说明，见 WF§17.3
}
export interface TicketDetail extends TicketSummary {
  description: string; customer_contact: string; created_by: UserRef; updated_at: string
  comments: Comment[]; events: TicketEvent[]; allowed_actions: Action[]
}

export interface ApiError { code: string; message: string; details?: Record<string, string> }
