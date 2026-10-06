import { FormEvent, useEffect, useState } from 'react'
import { api, ContestConfig } from '../api'

export default function AdminConfig() {
  const [config, setConfig] = useState<ContestConfig | null>(null)
  const [count, setCount] = useState(10)
  const [secrets, setSecrets] = useState<any[]>([])
  const [generated, setGenerated] = useState<any[]>([])
  const [error, setError] = useState('')
  const [msg, setMsg] = useState('')
  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState('')
  const [deleting, setDeleting] = useState(false)
  const [showResetModal, setShowResetModal] = useState(false)
  const [resetConfirm, setResetConfirm] = useState('')
  const [resetting, setResetting] = useState(false)

  async function refresh() {
    const [c, s] = await Promise.all([api.adminConfig(), api.adminSecrets()])
    setConfig(c)
    setSecrets(Array.isArray(s) ? s : [])
  }

  useEffect(() => {
    refresh().catch((e) => setError(e.message))
  }, [])

  async function save(e: FormEvent) {
    e.preventDefault()
    if (!config) return
    setError('')
    try {
      const updated = await api.adminUpdateConfig({
        title: config.title,
        duration_minutes: config.duration_minutes,
        is_active: config.is_active,
      })
      setConfig(updated)
      setMsg('Configuration saved')
    } catch (err: any) {
      setError(err.message)
    }
  }

  async function generate() {
    setError('')
    try {
      const secs = await api.adminGenerateSecrets(count)
      setGenerated(secs)
      await refresh()
      setMsg(`Generated ${secs.length} secrets`)
    } catch (err: any) {
      setError(err.message)
    }
  }

  function downloadSecrets(list: any[]) {
    const text = list.map((s) => s.code).join('\n')
    const blob = new Blob([text], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'secrets.txt'
    a.click()
    URL.revokeObjectURL(url)
  }

  function openDeleteModal() {
    if (!secrets.length) {
      setError('No secret IDs to delete')
      return
    }
    setDeleteConfirm('')
    setShowDeleteModal(true)
  }

  async function confirmDeleteAll() {
    if (deleteConfirm !== 'DELETE') {
      setError('Type DELETE exactly to confirm')
      return
    }
    setDeleting(true)
    setError('')
    try {
      const res = await api.adminDeleteAllSecrets()
      setGenerated([])
      setShowDeleteModal(false)
      setDeleteConfirm('')
      await refresh()
      setMsg(`Deleted ${res.secrets_deleted} secret(s) and ${res.students_deleted} student(s)`)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setDeleting(false)
    }
  }

  async function confirmReset() {
    if (resetConfirm !== 'RESET') {
      setError('Type RESET exactly to confirm')
      return
    }
    setResetting(true)
    setError('')
    try {
      const res = await api.adminResetContest()
      setGenerated([])
      setShowResetModal(false)
      setResetConfirm('')
      await refresh()
      setMsg(
        `Reset complete. Removed ${res.students_deleted} student(s), ${res.secrets_deleted} secret(s), and ${res.submissions_deleted} submission(s). Questions were kept.`,
      )
    } catch (err: any) {
      setError(err.message)
    } finally {
      setResetting(false)
    }
  }

  const claimed = secrets.filter((s) => s.claimed_at).length

  if (!config) return <p>Loading…</p>

  return (
    <div>
      <h1>Configuration</h1>
      {error && <div className="error">{error}</div>}
      {msg && <p style={{ color: 'var(--accent-2)' }}>{msg}</p>}

      <form className="panel" onSubmit={save}>
        <h2>Contest settings</h2>
        <div className="field">
          <label>Title</label>
          <input value={config.title} onChange={(e) => setConfig({ ...config, title: e.target.value })} />
        </div>
        <div className="field">
          <label>Duration (minutes)</label>
          <input
            type="number"
            min={1}
            value={config.duration_minutes}
            onChange={(e) => setConfig({ ...config, duration_minutes: Number(e.target.value) })}
          />
        </div>
        <label style={{ display: 'flex', gap: '.5rem', alignItems: 'center', marginBottom: '1rem' }}>
          <input
            type="checkbox"
            checked={config.is_active}
            onChange={(e) => setConfig({ ...config, is_active: e.target.checked })}
          />
          Contest active (students can log in)
        </label>
        <button className="btn" type="submit">
          Save configuration
        </button>
      </form>

      <div className="panel">
        <h2>Reset contest</h2>
        <p style={{ color: 'var(--muted)', marginTop: 0 }}>
          Clears secret IDs, student logins, timers, submissions, the leaderboard, and the activity feed.
          Questions, test cases, and official solutions stay.
        </p>
        <button className="btn danger" type="button" onClick={() => { setResetConfirm(''); setShowResetModal(true) }}>
          Reset all
        </button>
      </div>

      <div className="panel">
        <h2>Secret IDs</h2>
        <div className="row-actions">
          <div className="field" style={{ width: 140 }}>
            <label>Count</label>
            <input type="number" min={1} max={500} value={count} onChange={(e) => setCount(Number(e.target.value))} />
          </div>
          <button className="btn" type="button" onClick={generate}>
            Generate
          </button>
          <button className="btn secondary" type="button" onClick={() => downloadSecrets(secrets)} disabled={!secrets.length}>
            Download all
          </button>
          <button className="btn danger" type="button" onClick={openDeleteModal}>
            Delete all IDs
          </button>
        </div>
        {generated.length > 0 && (
          <p className="mono" style={{ whiteSpace: 'pre-wrap' }}>
            Newly generated:{'\n'}
            {generated.map((s) => s.code).join('\n')}
          </p>
        )}
        <table className="table">
          <thead>
            <tr>
              <th>Code</th>
              <th>Claimed</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            {secrets.map((s) => (
              <tr key={s.id}>
                <td className="mono">{s.code}</td>
                <td>{s.claimed_at ? new Date(s.claimed_at).toLocaleString() : '—'}</td>
                <td>{new Date(s.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showResetModal && (
        <div className="modal-backdrop" onClick={() => !resetting && setShowResetModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
            <h3>Reset the contest?</h3>
            <p>
              This permanently removes every secret ID, student, timer, and submission, and clears the activity feed.
              Questions and their test cases are kept. This cannot be undone.
            </p>
            <div className="field">
              <label>Type RESET to confirm</label>
              <input
                value={resetConfirm}
                onChange={(e) => setResetConfirm(e.target.value)}
                autoFocus
                placeholder="RESET"
                disabled={resetting}
              />
            </div>
            <div className="modal-actions">
              <button className="btn secondary" type="button" onClick={() => setShowResetModal(false)} disabled={resetting}>
                Cancel
              </button>
              <button
                className="btn danger"
                type="button"
                onClick={confirmReset}
                disabled={resetting || resetConfirm !== 'RESET'}
              >
                {resetting ? 'Resetting…' : 'Reset all'}
              </button>
            </div>
          </div>
        </div>
      )}

      {showDeleteModal && (
        <div className="modal-backdrop" onClick={() => !deleting && setShowDeleteModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
            <h3>Delete all secret IDs?</h3>
            <p>
              This will permanently delete <strong>{secrets.length}</strong> secret ID(s)
              {claimed > 0 && (
                <>
                  {' '}
                  and <strong>{claimed}</strong> claimed student account(s) with their submissions
                </>
              )}
              . This cannot be undone.
            </p>
            <div className="field">
              <label>Type DELETE to confirm</label>
              <input
                value={deleteConfirm}
                onChange={(e) => setDeleteConfirm(e.target.value)}
                autoFocus
                placeholder="DELETE"
                disabled={deleting}
              />
            </div>
            <div className="modal-actions">
              <button className="btn secondary" type="button" onClick={() => setShowDeleteModal(false)} disabled={deleting}>
                Cancel
              </button>
              <button
                className="btn danger"
                type="button"
                onClick={confirmDeleteAll}
                disabled={deleting || deleteConfirm !== 'DELETE'}
              >
                {deleting ? 'Deleting…' : 'Delete all'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
