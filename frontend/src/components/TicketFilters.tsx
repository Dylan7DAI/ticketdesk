import { useEffect, useState } from 'react'
import type { Priority, Status, User } from '../api/types'
import { PRIORITY_LABEL, PRIORITY_ORDER, STATUS_LABEL, STATUS_ORDER } from '../labels'

export interface FilterValues {
  status: Status[]
  priority: Priority[]
  assignee: string // '' 全部 | 'none' 未分配 | 用户 ID
  q: string
}

interface Props {
  value: FilterValues
  agents: User[]
  onChange: (patch: Partial<FilterValues>) => void
  onClear: () => void
}

function toggle<T>(list: T[], v: T): T[] {
  return list.includes(v) ? list.filter((x) => x !== v) : [...list, v]
}

// 搜索框本地持有输入，300ms 无输入后才通知父组件，避免每个字符一次请求。
function SearchBox({ value, onCommit }: { value: string; onCommit: (q: string) => void }) {
  const [text, setText] = useState(value)
  useEffect(() => setText(value), [value])
  useEffect(() => {
    if (text === value) return
    const timer = setTimeout(() => onCommit(text), 300)
    return () => clearTimeout(timer)
  }, [text, value, onCommit])
  return (
    <label>
      搜索（标题 / 客户）
      <input type="search" value={text} onChange={(e) => setText(e.target.value)} placeholder="输入关键词" />
    </label>
  )
}

export function TicketFilters({ value, agents, onChange, onClear }: Props) {
  return (
    <div className="filters">
      <div>
        <div className="muted" style={{ fontSize: 12 }}>状态</div>
        <div className="checks">
          {STATUS_ORDER.map((s) => (
            <label key={s}>
              <input type="checkbox" checked={value.status.includes(s)}
                onChange={() => onChange({ status: toggle(value.status, s) })} />
              {STATUS_LABEL[s]}
            </label>
          ))}
        </div>
      </div>
      <div>
        <div className="muted" style={{ fontSize: 12 }}>优先级</div>
        <div className="checks">
          {PRIORITY_ORDER.map((p) => (
            <label key={p}>
              <input type="checkbox" checked={value.priority.includes(p)}
                onChange={() => onChange({ priority: toggle(value.priority, p) })} />
              {PRIORITY_LABEL[p]}
            </label>
          ))}
        </div>
      </div>
      <label>
        处理人
        <select value={value.assignee} onChange={(e) => onChange({ assignee: e.target.value })}>
          <option value="">全部</option>
          <option value="none">未分配</option>
          {agents.map((u) => <option key={u.id} value={u.id}>{u.name}</option>)}
        </select>
      </label>
      <SearchBox value={value.q} onCommit={(q) => onChange({ q })} />
      <button type="button" onClick={onClear}>清除筛选</button>
    </div>
  )
}
