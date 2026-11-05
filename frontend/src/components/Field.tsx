import type { ReactNode } from 'react'
import { charCount } from '../validation'

interface Props {
  label: string
  error?: string
  children: ReactNode
  // 传入 value 与 max 时显示字数计数，达到 90% 时变色提示。
  counter?: { value: string; max: number }
}

export function Field({ label, error, children, counter }: Props) {
  const n = counter ? charCount(counter.value) : 0
  const near = counter ? n >= counter.max * 0.9 : false
  return (
    <div className={`field${error ? ' invalid' : ''}`}>
      <span className="label">{label}</span>
      {children}
      <div className={`hint${near ? ' near' : ''}`}>
        <span className="field-error" role={error ? 'alert' : undefined}>{error}</span>
        {counter && <span>{n} / {counter.max}</span>}
      </div>
    </div>
  )
}
