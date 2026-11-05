import type { ApiError } from './types'

// 统一的请求失败对象：HTTP 错误和网络错误都用它表示，页面只需处理一种形状。
export class ApiFailure extends Error {
  readonly status: number
  readonly error: ApiError
  constructor(status: number, error: ApiError) {
    super(error.message)
    this.status = status
    this.error = error
  }
}

export function isApiFailure(e: unknown): e is ApiFailure {
  return e instanceof ApiFailure
}

export interface RequestOptions {
  userId: number
  method?: 'GET' | 'POST'
  body?: unknown
  query?: Record<string, string | number | undefined>
}

function buildUrl(path: string, query?: RequestOptions['query']): string {
  const qs = new URLSearchParams()
  for (const [k, v] of Object.entries(query ?? {})) {
    if (v !== undefined && v !== '') qs.set(k, String(v))
  }
  const s = qs.toString()
  return s ? `${path}?${s}` : path
}

// 所有 /api 请求都带 X-User-ID（WF§2）。fetch 抛出的网络错误转成 NETWORK_ERROR。
export async function request<T>(path: string, opts: RequestOptions): Promise<T> {
  let res: Response
  try {
    res = await fetch(buildUrl(path, opts.query), {
      method: opts.method ?? 'GET',
      headers: {
        'X-User-ID': String(opts.userId),
        ...(opts.body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      },
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    })
  } catch {
    throw new ApiFailure(0, { code: 'NETWORK_ERROR', message: '无法连接服务器' })
  }
  const data: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    const err = (data as { error?: ApiError } | null)?.error
    throw new ApiFailure(res.status, err ?? { code: 'INTERNAL', message: '服务器返回了无法识别的响应' })
  }
  return data as T
}
