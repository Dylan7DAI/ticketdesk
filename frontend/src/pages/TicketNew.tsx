import { useState, type FormEvent } from 'react'
import { isApiFailure } from '../api/client'
import { errorText } from '../api/errorText'
import { api, type CreateBody } from '../api/endpoints'
import { ErrorBanner } from '../components/Feedback'
import { Field } from '../components/Field'
import { useGo } from '../components/nav'
import { useToast } from '../components/Toast'
import { useCurrentUser } from '../context/CurrentUser'
import { PRIORITY_LABEL, PRIORITY_ORDER, PRIORITY_SLA } from '../labels'
import { LIMITS, validateTicket, type FormErrors } from '../validation'

const EMPTY: CreateBody = {
  title: '', description: '', customer_name: '', customer_contact: '', priority: 'normal',
}

export function TicketNewPage() {
  const { me } = useCurrentUser()
  const go = useGo()
  const toast = useToast()
  const [form, setForm] = useState<CreateBody>(EMPTY)
  const [errors, setErrors] = useState<FormErrors>({})
  const [banner, setBanner] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const set = <K extends keyof CreateBody>(k: K, v: CreateBody[K]) => {
    setForm((f) => ({ ...f, [k]: v }))
    setErrors((e) => ({ ...e, [k]: undefined }))
  }

  async function submit(ev: FormEvent) {
    ev.preventDefault()
    setBanner(null)
    const local = validateTicket(form)
    if (Object.keys(local).length > 0) {
      setErrors(local) // 前端拦截，不发请求
      return
    }
    setSubmitting(true)
    try {
      const t = await api.createTicket(me.id, form)
      toast('success', '工单已创建')
      go(`/tickets/${t.id}`)
    } catch (e) {
      if (!isApiFailure(e)) throw e
      // 后端 400 的 details 显示在对应字段下。
      setErrors(e.error.details ?? {})
      setBanner(errorText(e.error.code, e.error.message))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <h1>新建工单</h1>
      {banner && <ErrorBanner text={banner} />}
      <form className="form" onSubmit={submit} noValidate>
        <Field label="标题" error={errors.title} counter={{ value: form.title, max: LIMITS.title }}>
          <input value={form.title} onChange={(e) => set('title', e.target.value)} />
        </Field>
        <Field label="描述" error={errors.description} counter={{ value: form.description, max: LIMITS.description }}>
          <textarea rows={6} value={form.description} onChange={(e) => set('description', e.target.value)} />
        </Field>
        <Field label="客户名称" error={errors.customer_name} counter={{ value: form.customer_name, max: LIMITS.customer_name }}>
          <input value={form.customer_name} onChange={(e) => set('customer_name', e.target.value)} />
        </Field>
        <Field label="联系方式" error={errors.customer_contact} counter={{ value: form.customer_contact, max: LIMITS.customer_contact }}>
          <input value={form.customer_contact} onChange={(e) => set('customer_contact', e.target.value)} />
        </Field>
        <Field label="优先级（决定 SLA 时长）" error={errors.priority}>
          <select value={form.priority} onChange={(e) => set('priority', e.target.value as CreateBody['priority'])}>
            {PRIORITY_ORDER.map((p) => (
              <option key={p} value={p}>{PRIORITY_LABEL[p]}（{PRIORITY_SLA[p]}）</option>
            ))}
          </select>
        </Field>
        <div className="actions">
          <button type="submit" className="primary" disabled={submitting}>
            {submitting ? '提交中…' : '创建工单'}
          </button>
        </div>
      </form>
    </>
  )
}
