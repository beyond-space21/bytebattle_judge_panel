import { FormEvent, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, setStudentToken } from '../api'

export default function StudentLogin() {
  const nav = useNavigate()
  const [secret, setSecret] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res = await api.studentLogin(secret.trim(), name.trim())
      setStudentToken(res.token)
      if (res.student.started_at) nav('/contest')
      else nav('/lobby')
    } catch (err: any) {
      setError(err.message || 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="auth-page">
      <form className="auth-card" onSubmit={onSubmit}>
        <h1>Byte Battle</h1>
        <p className="sub">Enter your 6-character secret ID and name to join.</p>
        {error && <div className="error">{error}</div>}
        <div className="field">
          <label>Secret ID</label>
          <input
            value={secret}
            onChange={(e) => setSecret(e.target.value.toUpperCase())}
            maxLength={6}
            placeholder="ABC123"
            autoComplete="off"
            required
          />
        </div>
        <div className="field">
          <label>Your name</label>
          <input value={name} onChange={(e) => setName(e.target.value)} placeholder="Jane Doe" required />
        </div>
        <button className="btn" type="submit" disabled={loading} style={{ width: '100%' }}>
          {loading ? 'Signing in…' : 'Enter contest'}
        </button>
        <p className="sub" style={{ marginTop: '1.25rem', marginBottom: 0, textAlign: 'center' }}>
          <Link to="/admin/login">Admin login</Link>
        </p>
      </form>
    </div>
  )
}
