import { useCallback, useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { adminWsUrl, api, asArray, LeaderboardEntry } from '../api'
import logo from '../../asset/logo.png'

type Problem = { id: number; title: string; order_index: number }

function useClock() {
  const [now, setNow] = useState(new Date())
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(t)
  }, [])
  return now
}

export default function LeaderboardPresent() {
  const [params] = useSearchParams()
  const limit = Math.max(3, Math.min(100, Number(params.get('limit')) || 20))
  const pageSeconds = Math.max(5, Number(params.get('page')) || 12)

  const [rows, setRows] = useState<LeaderboardEntry[]>([])
  const [problems, setProblems] = useState<Problem[]>([])
  const [maxScore, setMaxScore] = useState(0)
  const [live, setLive] = useState<'connecting' | 'live' | 'offline'>('connecting')
  const [changed, setChanged] = useState<Record<number, 'up' | 'down' | 'score'>>({})
  const [page, setPage] = useState(0)
  const [isFs, setIsFs] = useState(false)
  const prevRef = useRef<Map<number, { rank: number; total: number }>>(new Map())
  const now = useClock()

  const applyRows = useCallback((next: LeaderboardEntry[]) => {
    const prev = prevRef.current
    const marks: Record<number, 'up' | 'down' | 'score'> = {}
    next.forEach((r, i) => {
      const p = prev.get(r.student_id)
      if (!p) return
      if (i < p.rank) marks[r.student_id] = 'up'
      else if (i > p.rank) marks[r.student_id] = 'down'
      else if (r.total_score !== p.total) marks[r.student_id] = 'score'
    })
    const m = new Map<number, { rank: number; total: number }>()
    next.forEach((r, i) => m.set(r.student_id, { rank: i, total: r.total_score }))
    prevRef.current = m
    setRows(next)
    if (Object.keys(marks).length) {
      setChanged(marks)
      window.setTimeout(() => setChanged({}), 2500)
    }
  }, [])

  const refresh = useCallback(async () => {
    const [lb, ps, dash] = await Promise.all([api.adminLeaderboard(), api.adminProblems(), api.adminDashboard()])
    applyRows(asArray(lb))
    setProblems(
      asArray<any>(ps)
        .map((p) => ({ id: p.id, title: p.title, order_index: p.order_index }))
        .sort((a, b) => a.order_index - b.order_index || a.id - b.id),
    )
    setMaxScore(dash.scoring.max_possible)
  }, [applyRows])

  useEffect(() => {
    refresh().catch(console.error)
    let sock: WebSocket | null = null
    let closed = false
    let retry: number | null = null
    const connect = () => {
      sock = new WebSocket(adminWsUrl())
      sock.onopen = () => setLive('live')
      sock.onerror = () => setLive('offline')
      sock.onclose = () => {
        setLive('offline')
        if (!closed) retry = window.setTimeout(connect, 3000)
      }
      sock.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data)
          if (msg.type === 'leaderboard.updated' && Array.isArray(msg.payload)) {
            applyRows(msg.payload)
          } else if (
            msg.type === 'submission.finished' ||
            msg.type === 'student.login' ||
            msg.type === 'config.updated' ||
            String(msg.type).startsWith('problem.')
          ) {
            refresh().catch(() => {})
          }
        } catch {
          /* ignore */
        }
      }
    }
    connect()
    const poll = window.setInterval(() => refresh().catch(() => {}), 30000)
    return () => {
      closed = true
      if (retry) clearTimeout(retry)
      clearInterval(poll)
      sock?.close()
    }
  }, [refresh, applyRows])

  // `limit` rows per page; rotate pages when there are more participants than fit.
  const pageSize = limit
  const totalPages = Math.max(1, Math.ceil(rows.length / pageSize))
  useEffect(() => {
    if (totalPages <= 1) {
      setPage(0)
      return
    }
    const t = setInterval(() => setPage((p) => (p + 1) % totalPages), pageSeconds * 1000)
    return () => clearInterval(t)
  }, [totalPages, pageSeconds])
  const safePage = Math.min(page, totalPages - 1)
  const visible = rows.slice(safePage * pageSize, (safePage + 1) * pageSize)

  const [problemMax, setProblemMax] = useState<Map<number, number>>(new Map())
  useEffect(() => {
    api
      .adminDashboard()
      .then((d) => setProblemMax(new Map(d.problems.map((p) => [p.id, p.max_score]))))
      .catch(() => {})
  }, [problems.length])

  const cellClass = (score: number, id: number) => {
    const p = problemMax.get(id) ?? 0
    if (p > 0 && score >= p) return 'pc full'
    if (score > 0) return 'pc part'
    return 'pc none'
  }

  useEffect(() => {
    const onFs = () => setIsFs(Boolean(document.fullscreenElement))
    document.addEventListener('fullscreenchange', onFs)
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'f' || e.key === 'F') toggleFs()
      if (e.key === 'Escape' && document.fullscreenElement) document.exitFullscreen().catch(() => {})
    }
    window.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('fullscreenchange', onFs)
      window.removeEventListener('keydown', onKey)
    }
  }, [])

  function toggleFs() {
    if (document.fullscreenElement) document.exitFullscreen().catch(() => {})
    else document.documentElement.requestFullscreen().catch(() => {})
  }

  const compact = problems.length > 8

  return (
    <div className={`present ${compact ? 'compact' : ''}`}>
      <header className="present-head">
        <div className="present-title">
          <img className="present-logo" src={logo} alt="Byte Battle" />
          <div className="present-sub">
            <span className={`pill ${live === 'live' ? 'pill-ok' : live === 'offline' ? 'pill-bad' : 'pill-muted'}`}>
              <span className="dot" /> {live === 'live' ? 'LIVE' : live === 'offline' ? 'RECONNECTING' : 'CONNECTING'}
            </span>
            <span className="pill pill-muted">{rows.length} participants</span>
            {totalPages > 1 && (
              <span className="pill pill-muted">
                page {safePage + 1}/{totalPages}
              </span>
            )}
          </div>
        </div>
        <h1 className="present-heading">Leaderboard</h1>
        <div className="present-aside">
          <div className="present-clock mono">
            {now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
          </div>
          <button className="present-fs" onClick={toggleFs} title="Toggle fullscreen (F)">
            {isFs ? 'Exit fullscreen' : 'Fullscreen'}
          </button>
        </div>
      </header>

      <main className="present-body">
        {visible.length === 0 ? (
          <div className="present-empty">Waiting for participants…</div>
        ) : (
          <table className="ptable">
            <thead>
              <tr>
                <th className="c-rank">#</th>
                <th className="c-name">Participant</th>
                <th className="c-total">Score</th>
                {problems.map((p, i) => (
                  <th key={p.id} className="c-prob" title={p.title}>
                    {compact ? `P${i + 1}` : p.title}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {visible.map((r, idx) => {
                const rank = safePage * pageSize + idx + 1
                const mark = changed[r.student_id]
                return (
                  <tr key={r.student_id} className={`prow ${rank <= 3 ? `top top${rank}` : ''} ${mark ? `mark-${mark}` : ''}`}>
                    <td className="c-rank mono">
                      <span className={`prank r${rank}`}>{rank}</span>
                    </td>
                    <td className="c-name">
                      <span className="pname">{r.name}</span>
                      {mark === 'up' && <span className="arrow up">▲</span>}
                      {mark === 'down' && <span className="arrow down">▼</span>}
                    </td>
                    <td className="c-total mono">
                      <span className="ptotal">{r.total_score}</span>
                      {maxScore > 0 && <span className="pmax">/{maxScore}</span>}
                      <div className="pbar">
                        <span style={{ width: `${maxScore ? Math.min(100, (r.total_score / maxScore) * 100) : 0}%` }} />
                      </div>
                    </td>
                    {problems.map((p) => {
                      const sc = r.problem_scores?.[String(p.id)] ?? 0
                      return (
                        <td key={p.id} className="c-prob">
                          <span className={cellClass(sc, p.id)}>{sc || '·'}</span>
                        </td>
                      )
                    })}
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </main>

      <footer className="present-foot">
        <span>Press <kbd>F</kbd> for fullscreen · <kbd>Esc</kbd> to exit</span>
        <span className="muted">
          {limit} per page{totalPages > 1 ? ` · rotating every ${pageSeconds}s` : ''} · tune with{' '}
          <span className="mono">?limit=30&amp;page=15</span>
        </span>
      </footer>
    </div>
  )
}
