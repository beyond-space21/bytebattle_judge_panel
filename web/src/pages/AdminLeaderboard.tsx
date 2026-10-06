import { useEffect, useState } from 'react'
import { adminWsUrl, api, asArray, LeaderboardEntry } from '../api'

type QuestionCol = { id: string; number: number; title: string }

export default function AdminLeaderboard() {
  const [rows, setRows] = useState<LeaderboardEntry[]>([])
  const [questions, setQuestions] = useState<QuestionCol[]>([])
  const [live, setLive] = useState('')

  async function refresh() {
    const [lb, problems] = await Promise.all([api.adminLeaderboard(), api.adminProblems()])
    setRows(asArray(lb))
    setQuestions(
      asArray<any>(problems)
        .map((p) => ({ id: String(p.id), number: p.order_index, title: p.title }))
        .sort((a, b) => a.number - b.number || Number(a.id) - Number(b.id)),
    )
  }

  useEffect(() => {
    refresh().catch(console.error)
    const ws = new WebSocket(adminWsUrl())
    ws.onopen = () => setLive('live')
    ws.onclose = () => setLive('offline')
    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data)
        if (msg.type === 'leaderboard.updated' && Array.isArray(msg.payload)) {
          setRows(msg.payload)
        } else if (msg.type === 'submission.finished' && msg.payload?.kind === 'submit') {
          refresh().catch(() => {})
        }
      } catch {
        /* ignore */
      }
    }
    return () => ws.close()
  }, [])

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '1rem', flexWrap: 'wrap' }}>
        <h1 style={{ margin: 0 }}>Leaderboard</h1>
        <div className="row-actions" style={{ margin: 0, alignItems: 'center' }}>
          <span className={`pill ${live === 'live' ? 'pill-ok' : live === 'offline' ? 'pill-bad' : 'pill-muted'}`}>
            <span className="dot" /> {live || 'connecting'}
          </span>
          <a className="btn" href="/admin/present" target="_blank" rel="noopener">
            Present (fullscreen)
          </a>
        </div>
      </div>
      <p className="muted" style={{ marginTop: '.5rem', fontSize: '.85rem' }}>
        Present mode opens a clean, large-type view for streaming or projection. Secret codes are hidden there. Press{' '}
        <kbd>F</kbd> inside it to go fullscreen.
      </p>
      <div className="panel">
        <table className="table">
          <thead>
            <tr>
              <th>#</th>
              <th>Name</th>
              <th>Secret</th>
              <th>Total</th>
              {questions.map((q) => (
                <th key={q.id} title={q.title}>
                  {q.number}
                </th>
              ))}
              <th>Last improve</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={r.student_id}>
                <td>{i + 1}</td>
                <td>{r.name}</td>
                <td className="mono">{r.secret_code}</td>
                <td className="mono">
                  <strong>{r.total_score}</strong>
                </td>
                {questions.map((q) => (
                  <td key={q.id} className="mono" title={q.title}>
                    {r.problem_scores?.[q.id] ?? 0}
                  </td>
                ))}
                <td>{r.last_improve_at ? new Date(r.last_improve_at).toLocaleString() : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
