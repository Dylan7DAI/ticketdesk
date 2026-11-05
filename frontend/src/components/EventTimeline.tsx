import type { TicketEvent } from '../api/types'
import { formatTime } from '../labels'

const TYPE_LABEL: Record<TicketEvent['type'], string> = {
  created: '创建', claimed: '领取', assigned: '分配',
  reassigned: '改派', status_changed: '状态变更', commented: '评论',
}

// 后端已按时间升序返回，summary 是渲染好的中文说明（WF§17.3），这里直接显示。
export function EventTimeline({ events }: { events: TicketEvent[] }) {
  return (
    <ol className="timeline">
      {events.map((e) => (
        <li key={e.id}>
          <strong>{e.actor.name}</strong> · {TYPE_LABEL[e.type] ?? e.type} ·{' '}
          <span className="muted">{formatTime(e.created_at)}</span>
          <div>{e.summary}</div>
        </li>
      ))}
    </ol>
  )
}
