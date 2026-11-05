import { Route, Routes } from 'react-router-dom'
import { UserLink } from './components/nav'
import { UserSwitcher } from './components/UserSwitcher'
import { ToastProvider } from './components/Toast'
import { CurrentUserProvider } from './context/CurrentUser'
import { TicketDetailPage } from './pages/TicketDetail'
import { TicketListPage } from './pages/TicketList'
import { TicketNewPage } from './pages/TicketNew'
import './App.css'

export default function App() {
  return (
    <ToastProvider>
      <CurrentUserProvider>
        <header className="topbar">
          <UserLink to="/" className="brand">客服工单</UserLink>
          <nav>
            <UserLink to="/">工单队列</UserLink>
            <UserLink to="/tickets/new">新建工单</UserLink>
          </nav>
          <UserSwitcher />
        </header>
        <main className="content">
          <Routes>
            <Route path="/" element={<TicketListPage />} />
            <Route path="/tickets/new" element={<TicketNewPage />} />
            <Route path="/tickets/:id" element={<TicketDetailPage />} />
            <Route path="*" element={<p>页面不存在</p>} />
          </Routes>
        </main>
      </CurrentUserProvider>
    </ToastProvider>
  )
}
