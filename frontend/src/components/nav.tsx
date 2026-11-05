import { Link, useNavigate, useSearchParams, type LinkProps } from 'react-router-dom'

// 页面之间跳转时保留 ?as=<id>，否则切换用户后点链接会回到默认用户。
function withAs(to: string, as: string | null): string {
  if (!as) return to
  return to + (to.includes('?') ? '&' : '?') + `as=${as}`
}

export function UserLink({ to, ...rest }: Omit<LinkProps, 'to'> & { to: string }) {
  const [params] = useSearchParams()
  return <Link {...rest} to={withAs(to, params.get('as'))} />
}

export function useGo() {
  const [params] = useSearchParams()
  const nav = useNavigate()
  return (to: string) => nav(withAs(to, params.get('as')))
}
