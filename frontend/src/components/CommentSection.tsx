import { useState, type FormEvent } from 'react'
import type { Comment, Status } from '../api/types'
import { formatTime } from '../labels'
import { LIMITS, checkText } from '../validation'
import { Field } from './Field'

interface Props {
  comments: Comment[]
  canComment: boolean
  status: Status
  submitting: boolean
  onSubmit: (body: string) => Promise<boolean>
}

// 评论框是否显示完全由 allowed_actions 里有没有 comment 决定；隐藏时说明原因。
export function CommentSection({ comments, canComment, status, submitting, onSubmit }: Props) {
  const [body, setBody] = useState('')
  const [error, setError] = useState<string>()

  async function submit(ev: FormEvent) {
    ev.preventDefault()
    const msg = checkText('body', body)
    if (msg) return setError(msg)
    if (await onSubmit(body)) setBody('')
  }

  return (
    <section className="panel">
      <h2>评论（{comments.length}）</h2>
      {comments.length === 0 && <p className="muted">暂无评论</p>}
      {comments.map((c) => (
        <div key={c.id} className="comment">
          <strong>{c.author.name}</strong>{' '}
          <span className="muted">{formatTime(c.created_at)}</span>
          <div>{c.body}</div>
        </div>
      ))}
      {canComment ? (
        <form className="form" onSubmit={submit} noValidate style={{ marginTop: 12 }}>
          <Field label="添加评论" error={error} counter={{ value: body, max: LIMITS.body }}>
            <textarea rows={3} value={body} onChange={(e) => { setBody(e.target.value); setError(undefined) }} />
          </Field>
          <div className="actions">
            <button type="submit" className="primary" disabled={submitting}>发表评论</button>
          </div>
        </form>
      ) : (
        <p className="muted" style={{ marginTop: 12 }}>
          {status === 'closed' ? '工单已关闭，不能再评论' : '仅处理人或主管可评论'}
        </p>
      )}
    </section>
  )
}
