import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { setAdminToken } from '../api'

export default function AdminLayout() {
  const nav = useNavigate()
  return (
    <div className="admin-shell">
      <nav className="admin-nav">
        <div className="logo">Judge Admin</div>
        <NavLink to="/admin" end>
          Dashboard
        </NavLink>
        <NavLink to="/admin/questions">Questions</NavLink>
        <NavLink to="/admin/config">Configuration</NavLink>
        <NavLink to="/admin/leaderboard">Leaderboard</NavLink>
        <button
          className="btn ghost"
          style={{ marginTop: 'auto', justifyContent: 'flex-start' }}
          onClick={() => {
            setAdminToken(null)
            nav('/admin/login')
          }}
        >
          Log out
        </button>
      </nav>
      <main className="admin-main">
        <Outlet />
      </main>
    </div>
  )
}
