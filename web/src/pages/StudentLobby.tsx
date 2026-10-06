import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, ContestConfig, formatTime, setStudentToken, Student } from '../api'

export default function StudentLobby() {
  const nav = useNavigate()
  const [student, setStudent] = useState<Student | null>(null)
  const [config, setConfig] = useState<ContestConfig | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    api.studentMe()
      .then((me) => {
        setStudent(me.student)
        setConfig(me.config)
        if (me.student.started_at) nav('/contest', { replace: true })
      })
      .catch((e) => setError(e.message))
  }, [nav])

  async function start() {
    setLoading(true)
    setError('')
    try {
      await api.studentStart()
      nav('/contest')
    } catch (e: any) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  function logout() {
    setStudentToken(null)
    nav('/')
  }

  return (
    <div className="lobby">
      <div className="lobby-card">
        <h1>{config?.title || 'Byte Battle'}</h1>
        <p className="meta">
          Welcome{student ? `, ${student.name}` : ''}. You have{' '}
          <strong className="mono">{formatTime((config?.duration_minutes || 0) * 60)}</strong> once you start.
          The timer begins only when you click Start.
        </p>
        {error && <div className="error">{error}</div>}
        <div className="row-actions" style={{ justifyContent: 'center' }}>
          <button className="btn success" onClick={start} disabled={loading || !config?.is_active}>
            {loading ? 'Starting…' : 'Start Contest'}
          </button>
          <button className="btn secondary" onClick={logout}>
            Log out
          </button>
        </div>
        {!config?.is_active && (
          <p className="meta" style={{ marginTop: '1rem' }}>
            Contest is not active yet. Wait for the admin to activate it.
          </p>
        )}
      </div>
    </div>
  )
}
