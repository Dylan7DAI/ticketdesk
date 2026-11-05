import type { ReactNode } from 'react'

export const Loading = ({ text = '加载中…' }: { text?: string }) => (
  <div className="loading" role="status">{text}</div>
)

export const EmptyState = ({ text, action }: { text: string; action?: ReactNode }) => (
  <div className="empty">
    <p>{text}</p>
    {action}
  </div>
)

export const ErrorBanner = ({ text, onRetry }: { text: string; onRetry?: () => void }) => (
  <div className="error-banner" role="alert">
    <span>{text}</span>
    {onRetry && <button type="button" onClick={onRetry}>重试</button>}
  </div>
)

interface PaginationProps {
  page: number
  pageSize: number
  total: number
  onChange: (page: number) => void
}

export function Pagination({ page, pageSize, total, onChange }: PaginationProps) {
  const pages = Math.max(1, Math.ceil(total / pageSize))
  return (
    <nav className="pagination" aria-label="分页">
      <span>共 {total} 条，第 {page} / {pages} 页</span>
      <button type="button" disabled={page <= 1} onClick={() => onChange(page - 1)}>上一页</button>
      <button type="button" disabled={page >= pages} onClick={() => onChange(page + 1)}>下一页</button>
    </nav>
  )
}
