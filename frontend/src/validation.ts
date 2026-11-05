// 与 WF§6 字段表同规则：去首尾空白后按 Unicode 字符数计算（不是 UTF-16 长度）。
// 只是体验优化，后端会重新校验。
import type { CreateBody } from './api/endpoints'
import type { Priority } from './api/types'

export const LIMITS = {
  title: 200,
  description: 5000,
  customer_name: 100,
  customer_contact: 200,
  body: 2000,
  note: 500,
} as const

const LABEL: Record<keyof typeof LIMITS, string> = {
  title: '标题', description: '描述', customer_name: '客户名称',
  customer_contact: '联系方式', body: '评论内容', note: '备注',
}

export const charCount = (s: string): number => [...s.trim()].length

export function checkText(field: keyof typeof LIMITS, v: string, min = 1): string | undefined {
  const n = charCount(v)
  if (n < min) return `${LABEL[field]}不能为空`
  if (n > LIMITS[field]) return `${LABEL[field]}不能超过 ${LIMITS[field]} 个字符`
  return undefined
}

export type FormErrors = Partial<Record<keyof CreateBody, string>>

export function validateTicket(f: CreateBody & { priority: Priority }): FormErrors {
  const e: FormErrors = {}
  for (const k of ['title', 'description', 'customer_name', 'customer_contact'] as const) {
    const msg = checkText(k, f[k])
    if (msg) e[k] = msg
  }
  return e
}
