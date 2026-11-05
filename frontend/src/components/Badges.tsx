import type { Priority, Status } from '../api/types'
import { PRIORITY_LABEL, STATUS_LABEL } from '../labels'

export const StatusBadge = ({ status }: { status: Status }) => (
  <span className={`badge status-${status}`}>{STATUS_LABEL[status]}</span>
)

export const PriorityBadge = ({ priority }: { priority: Priority }) => (
  <span className={`badge priority-${priority}`}>{PRIORITY_LABEL[priority]}</span>
)

// 逾期用颜色加文字，不只依赖颜色。
export const OverdueBadge = ({ overdue }: { overdue: boolean }) =>
  overdue ? <span className="badge overdue">已逾期</span> : null
