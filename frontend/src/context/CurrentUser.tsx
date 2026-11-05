import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api/endpoints'
import { isApiFailure } from '../api/client'
import type { User } from '../api/types'

interface Ctx {
  users: User[]
  me: User
  setMe: (id: number) => void
}

const CurrentUserContext = createContext<Ctx | null>(null)

// 当前用户保存在 URL 参数 ?as=<id>（默认 1），便于开两个窗口模拟两名客服并发操作。
// 用户列表加载完成前不渲染子树，所以子组件拿到的 me 一定有效。
export function CurrentUserProvider({ children }: { children: ReactNode }) {
  const [params, setParams] = useSearchParams()
  const [users, setUsers] = useState<User[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  const asId = Number(params.get('as')) || 1

  useEffect(() => {
    // 用户列表本身要带身份头；任意有效 ID 即可，这里用当前选择。
    api.users(asId)
      .then(setUsers)
      .catch((e) => setError(isApiFailure(e) ? e.error.message : '加载用户失败'))
    // 只在首次加载时取一次。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const value = useMemo<Ctx | null>(() => {
    if (!users || users.length === 0) return null
    const me = users.find((u) => u.id === asId) ?? users[0]
    return {
      users,
      me,
      setMe: (id) => {
        const next = new URLSearchParams(params)
        next.set('as', String(id))
        setParams(next)
      },
    }
  }, [users, asId, params, setParams])

  if (error) return <div className="page-error" role="alert">{error}。请确认后端已启动（npm run dev）。</div>
  if (!value) return <div className="page-loading">加载中…</div>
  return <CurrentUserContext.Provider value={value}>{children}</CurrentUserContext.Provider>
}

export function useCurrentUser(): Ctx {
  const v = useContext(CurrentUserContext)
  if (!v) throw new Error('useCurrentUser 必须在 CurrentUserProvider 内使用')
  return v
}
