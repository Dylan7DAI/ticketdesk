import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { ApiFailure, isApiFailure } from '../api/client'
import { errorText } from '../api/errorText'
import { api } from '../api/endpoints'
import type { TicketDetail } from '../api/types'
import { ActionPanel, type Command } from '../components/ActionPanel'
import { OverdueBadge, PriorityBadge, StatusBadge } from '../components/Badges'
import { CommentSection } from '../components/CommentSection'
import { EventTimeline } from '../components/EventTimeline'
import { ErrorBanner, Loading } from '../components/Feedback'
import { UserLink } from '../components/nav'
import { useToast } from '../components/Toast'
import { useCurrentUser } from '../context/CurrentUser'
import { formatTime } from '../labels'

function remaining(t: TicketDetail): string {
  const ms = new Date(t.sla_due_at).getTime() - Date.now()
  const hours = Math.floor(Math.abs(ms) / 3_600_000)
  const mins = Math.floor((Math.abs(ms) % 3_600_000) / 60_000)
  const text = hours > 0 ? `${hours} 小时 ${mins} 分钟` : `${mins} 分钟`
  return ms < 0 ? `已逾期 ${text}` : `剩余 ${text}`
}

// switch 按 kind 逐个窄化 Command，新增命令时漏处理会在编译期报错。
function send(me: number, id: number, c: Command): Promise<TicketDetail> {
  switch (c.kind) {
    case 'claim':
      return api.claim(me, id)
    case 'assign':
      return api.assign(me, id, c.assigneeId)
    case 'reassign':
      return api.reassign(me, id, c.assigneeId)
    case 'resolve':
    case 'reopen':
    case 'close':
      return api.transition(me, id, c.kind, c.note)
  }
}

const SUCCESS: Record<Command['kind'], string> = {
  claim: '已领取工单', assign: '已分配', reassign: '已改派',
  resolve: '已标记为解决', reopen: '已重开', close: '工单已关闭',
}

export function TicketDetailPage() {
  const { me, users } = useCurrentUser()
  const toast = useToast()
  const id = Number(useParams().id)
  const agents = useMemo(() => users.filter((u) => u.role === 'agent'), [users])

  const [ticket, setTicket] = useState<TicketDetail | null>(null)
  const [fatal, setFatal] = useState<ApiFailure | null>(null) // 403 / 404：整页提示
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    try {
      setTicket(await api.getTicket(me.id, id))
      setFatal(null)
      setError(null)
    } catch (e) {
      if (!isApiFailure(e)) throw e
      if (e.status === 403 || e.status === 404) setFatal(e)
      else setError(errorText(e.error.code, e.error.message))
    }
  }, [me.id, id])

  // 切换用户或工单时重新请求；先清空，避免短暂显示上一个用户可见的内容。
  useEffect(() => {
    setTicket(null)
    setFatal(null)
    void load()
  }, [load])

  // 409/403 表示页面已过期：提示错误文案并重新拉取最新详情。
  const onFailure = useCallback(async (e: unknown) => {
    if (!isApiFailure(e)) throw e
    toast('error', errorText(e.error.code, e.error.message))
    if (e.status === 409 || e.status === 403) await load()
  }, [load, toast])

  async function run(c: Command) {
    setBusy(true)
    try {
      setTicket(await send(me.id, id, c)) // 用响应替换页面状态，不做乐观更新
      toast('success', SUCCESS[c.kind])
    } catch (e) {
      await onFailure(e)
    } finally {
      setBusy(false)
    }
  }

  async function comment(body: string): Promise<boolean> {
    setBusy(true)
    try {
      await api.comment(me.id, id, body)
      toast('success', '评论已发表')
      await load() // 评论接口只返回评论，历史和评论列表需要重新拉取
      return true
    } catch (e) {
      await onFailure(e)
      return false
    } finally {
      setBusy(false)
    }
  }

  if (!Number.isInteger(id) || id <= 0) return <FullPage text="工单 ID 无效" />
  if (fatal) {
    return <FullPage text={fatal.status === 404 ? '工单不存在' : errorText(fatal.error.code, fatal.error.message)} />
  }
  if (error) return <ErrorBanner text={error} onRetry={() => void load()} />
  if (!ticket) return <Loading />

  return (
    <>
      <p><UserLink to="/">← 返回列表</UserLink></p>
      <h1>#{ticket.id} {ticket.title}</h1>
      <section className="panel">
        <dl className="detail-grid">
          <dt>状态</dt><dd><StatusBadge status={ticket.status} /> <PriorityBadge priority={ticket.priority} /> <OverdueBadge overdue={ticket.is_overdue} /></dd>
          <dt>处理人</dt><dd>{ticket.assignee ? ticket.assignee.name : <span className="muted">未分配</span>}</dd>
          <dt>客户</dt><dd>{ticket.customer_name}（{ticket.customer_contact}）</dd>
          <dt>创建人</dt><dd>{ticket.created_by.name}</dd>
          <dt>创建时间</dt><dd>{formatTime(ticket.created_at)}</dd>
          <dt>更新时间</dt><dd>{formatTime(ticket.updated_at)}</dd>
          <dt>SLA 截止</dt><dd>{formatTime(ticket.sla_due_at)}（{ticket.status === 'resolved' || ticket.status === 'closed' ? '已结束' : remaining(ticket)}）</dd>
          <dt>描述</dt><dd style={{ whiteSpace: 'pre-wrap' }}>{ticket.description}</dd>
        </dl>
      </section>
      <section className="panel">
        <h2>操作</h2>
        <ActionPanel ticket={ticket} agents={agents} busy={busy} onRun={run} />
      </section>
      <CommentSection comments={ticket.comments} status={ticket.status} submitting={busy}
        canComment={ticket.allowed_actions.includes('comment')} onSubmit={comment} />
      <section className="panel">
        <h2>历史</h2>
        <EventTimeline events={ticket.events} />
      </section>
    </>
  )
}

function FullPage({ text }: { text: string }) {
  return (
    <div className="panel" style={{ textAlign: 'center' }}>
      <h1>{text}</h1>
      <UserLink to="/">返回列表</UserLink>
    </div>
  )
}
