import { useCallback, useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { isApiFailure } from '../api/client'
import { errorText } from '../api/errorText'
import { api, type ListParams } from '../api/endpoints'
import type { Priority, Status, TicketPage } from '../api/types'
import { OverdueBadge, PriorityBadge, StatusBadge } from '../components/Badges'
import { EmptyState, ErrorBanner, Loading, Pagination } from '../components/Feedback'
import { UserLink } from '../components/nav'
import { TicketFilters, type FilterValues } from '../components/TicketFilters'
import { useCurrentUser } from '../context/CurrentUser'
import { formatTime, PRIORITY_ORDER, STATUS_ORDER } from '../labels'

const PAGE_SIZE = 20

// 筛选状态完全来自 URL 查询参数，刷新和返回后保持。
function readFilters(p: URLSearchParams): FilterValues & { page: number } {
  const list = <T extends string>(key: string, allowed: T[]) =>
    (p.get(key)?.split(',') ?? []).filter((x): x is T => allowed.includes(x as T))
  return {
    status: list<Status>('status', STATUS_ORDER),
    priority: list<Priority>('priority', PRIORITY_ORDER),
    assignee: p.get('assignee_id') ?? '',
    q: p.get('q') ?? '',
    page: Math.max(1, Number(p.get('page')) || 1),
  }
}

export function TicketListPage() {
  const { me, users } = useCurrentUser()
  const [params, setParams] = useSearchParams()
  const f = useMemo(() => readFilters(params), [params])
  const agents = useMemo(() => users.filter((u) => u.role === 'agent'), [users])

  const [data, setData] = useState<TicketPage | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [reloadKey, setReloadKey] = useState(0)

  // 改变筛选条件时页码回到 1；其余参数（如 as）原样保留。
  const update = useCallback((patch: Partial<FilterValues>, page = 1) => {
    const next = new URLSearchParams(params)
    const merged = { ...readFilters(params), ...patch }
    const set = (k: string, v: string) => (v ? next.set(k, v) : next.delete(k))
    set('status', merged.status.join(','))
    set('priority', merged.priority.join(','))
    set('assignee_id', merged.assignee)
    set('q', merged.q)
    set('page', page > 1 ? String(page) : '')
    setParams(next)
  }, [params, setParams])

  const clear = () => update({ status: [], priority: [], assignee: '', q: '' })

  useEffect(() => {
    let stale = false
    setLoading(true)
    setError(null)
    const query: ListParams = {
      status: f.status, priority: f.priority, assignee_id: f.assignee || undefined,
      q: f.q || undefined, page: f.page, page_size: PAGE_SIZE,
    }
    api.listTickets(me.id, query)
      .then((d) => { if (!stale) setData(d) })
      .catch((e) => { if (!stale) setError(isApiFailure(e) ? errorText(e.error.code, e.error.message) : '加载失败') })
      .finally(() => { if (!stale) setLoading(false) })
    return () => { stale = true }
  }, [me.id, f, reloadKey])

  const filtered = f.status.length > 0 || f.priority.length > 0 || f.assignee !== '' || f.q !== ''

  return (
    <>
      <h1>工单队列</h1>
      {me.role === 'agent' && <div className="notice">仅显示待领取和分配给你的工单</div>}
      <TicketFilters value={f} agents={agents} onChange={(p) => update(p)} onClear={clear} />
      {error && <ErrorBanner text={error} onRetry={() => setReloadKey((k) => k + 1)} />}
      {loading && !data && <Loading />}
      {data && data.items.length === 0 && !loading && (
        <EmptyState text="没有符合条件的工单"
          action={filtered ? <button type="button" onClick={clear}>清除筛选</button> : undefined} />
      )}
      {data && data.items.length > 0 && (
        <>
          <table aria-busy={loading}>
            <thead>
              <tr>
                <th>标题</th><th>客户</th><th>优先级</th><th>状态</th>
                <th>处理人</th><th>SLA 截止时间</th><th>逾期</th><th>创建时间</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map((t) => (
                <tr key={t.id} className={t.is_overdue ? 'row-overdue' : undefined}>
                  <td><UserLink to={`/tickets/${t.id}`}>{t.title}</UserLink></td>
                  <td>{t.customer_name}</td>
                  <td><PriorityBadge priority={t.priority} /></td>
                  <td><StatusBadge status={t.status} /></td>
                  <td>{t.assignee ? t.assignee.name : <span className="muted">未分配</span>}</td>
                  <td>{formatTime(t.sla_due_at)}</td>
                  <td><OverdueBadge overdue={t.is_overdue} /></td>
                  <td>{formatTime(t.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <Pagination page={data.page} pageSize={data.page_size} total={data.total}
            onChange={(p) => update({}, p)} />
        </>
      )}
    </>
  )
}
