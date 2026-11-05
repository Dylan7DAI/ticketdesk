import { useCurrentUser } from '../context/CurrentUser'

const ROLE_LABEL = { agent: '客服', supervisor: '主管' } as const

// 切换后 me 变化，依赖 me.id 的页面会重新请求数据（请求头 X-User-ID 随之变化）。
export function UserSwitcher() {
  const { users, me, setMe } = useCurrentUser()
  return (
    <label className="user-switcher">
      当前用户：
      <select value={me.id} onChange={(e) => setMe(Number(e.target.value))}>
        {users.map((u) => (
          <option key={u.id} value={u.id}>{u.name}（{ROLE_LABEL[u.role]}）</option>
        ))}
      </select>
    </label>
  )
}
