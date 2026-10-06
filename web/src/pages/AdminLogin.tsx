import { FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, setAdminToken } from '../api'

export default function AdminLogin() {
  const nav = useNavigate()
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const res = await api.adminLogin(username, password)
      setAdminToken(res.token)
      nav('/admin')
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="auth-page">
      <form className="auth-card" onSubmit={onSubmit}>
        <h1>Admin</h1>
        <p className="sub">Manage the competition, questions, and live activity.</p>
        {error && <div className="error">{error}</div>}
        <div className="field">
          <label>Username</label>
          <input value={username} onChange={(e) => setUsername(e.target.value)} required />
        </div>
        <div className="field">
          <label>Password</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        </div>
        <button className="btn" type="submit" disabled={loading} style={{ width: '100%' }}>
          {loading ? 'Signing in…' : 'Sign in'}
        </button>
        <p className="sub" style={{ marginTop: '1.25rem', marginBottom: 0, textAlign: 'center' }}>
          <Link to="/">Student login</Link>
        </p>
      </form>
    </div>
  )
}
