import { useState } from 'react'
import type { Action, TicketDetail, User } from '../api/types'
import { ACTION_LABEL } from '../labels'
import { LIMITS, checkText } from '../validation'

export type Command =
  | { kind: 'claim' }
  | { kind: 'assign' | 'reassign'; assigneeId: number }
  | { kind: 'resolve' | 'reopen' | 'close'; note: string }

interface Props {
  ticket: TicketDetail
  agents: User[]
  busy: boolean
  onRun: (c: Command) => void
}

// 只渲染 allowed_actions 中的按钮，不在前端推导权限（comment 由评论区处理）。
export function ActionPanel({ ticket, agents, busy, onRun }: Props) {
  const allowed = new Set<Action>(ticket.allowed_actions)
  const [target, setTarget] = useState('')
  const [note, setNote] = useState('')
  const [noteError, setNoteError] = useState<string>()

  // 改派时排除当前处理人。
  const candidates = agents.filter((a) => a.id !== ticket.assignee?.id)
  const hasTransition = ['resolve', 'reopen', 'close'].some((a) => allowed.has(a as Action))
  const hasAssign = allowed.has('assign') || allowed.has('reassign')
  const targetId = Number(target)

  function transition(kind: 'resolve' | 'reopen' | 'close') {
    const msg = checkText('note', note, 0)
    if (msg) return setNoteError(msg)
    // close 需要二次确认。
    if (kind === 'close' && !window.confirm('关闭后工单不可再修改，确定关闭吗？')) return
    onRun({ kind, note })
    setNote('')
  }

  if (!allowed.has('claim') && !hasAssign && !hasTransition) {
    return <p className="muted">当前没有可执行的操作</p>
  }
  return (
    <div className="actions-wrap">
      <div className="actions">
        {allowed.has('claim') && (
          <button type="button" className="primary" disabled={busy} onClick={() => onRun({ kind: 'claim' })}>
            {ACTION_LABEL.claim}
          </button>
        )}
        {hasAssign && (
          <>
            <select value={target} onChange={(e) => setTarget(e.target.value)} aria-label="选择客服">
              <option value="">选择客服…</option>
              {candidates.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
            </select>
            {(['assign', 'reassign'] as const).filter((k) => allowed.has(k)).map((k) => (
              <button key={k} type="button" disabled={busy || !targetId}
                onClick={() => onRun({ kind: k, assigneeId: targetId })}>
                {ACTION_LABEL[k]}
              </button>
            ))}
          </>
        )}
      </div>
      {hasTransition && (
        <div className="actions" style={{ marginTop: 8 }}>
          <input value={note} placeholder={`备注（可选，最多 ${LIMITS.note} 字）`} style={{ width: 260 }}
            onChange={(e) => { setNote(e.target.value); setNoteError(undefined) }} />
          {(['resolve', 'reopen', 'close'] as const).filter((k) => allowed.has(k)).map((k) => (
            <button key={k} type="button" className={k === 'close' ? 'danger' : undefined}
              disabled={busy} onClick={() => transition(k)}>
              {ACTION_LABEL[k]}
            </button>
          ))}
          {noteError && <span className="field-error" role="alert">{noteError}</span>}
        </div>
      )}
    </div>
  )
}
