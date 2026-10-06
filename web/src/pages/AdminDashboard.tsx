import { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { adminWsUrl, api, asArray, Dashboard } from '../api'

type FeedItem = {
  id: string
  type: string
  text: string
  tone: 'ok' | 'warn' | 'bad' | 'info'
  at: string
}

function relTime(iso: string) {
  const diff = Math.max(0, Date.now() - new Date(iso).getTime())
  const s = Math.floor(diff / 1000)
  if (s < 5) return 'just now'
  if (s < 60) return `${s}s ago`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ${m % 60}m ago`
  return new Date(iso).toLocaleString()
}

function pct(n: number, d: number) {
  if (!d) return 0
  return Math.round((n / d) * 100)
}

function statusTone(status: string): FeedItem['tone'] {
  if (status === 'Accepted') return 'ok'
  if (status === 'Partial') return 'warn'
  if (status === 'pending') return 'info'
  return 'bad'
}

function describeEvent(type: string, p: any, at: string): FeedItem | null {
  const name = p?.name ?? p?.student_name ?? 'Student'
  const problem = p?.problem_title ? `“${p.problem_title}”` : ''
  switch (type) {
    case 'student.login':
      return { id: '', type, at, tone: 'info', text: `${name} logged in (${p?.secret ?? ''})` }
    case 'student.start':
      return { id: '', type, at, tone: 'info', text: `${name} started the contest` }
    case 'submission.created':
      return {
        id: '',
        type,
        at,
        tone: 'info',
        text: `${name} ${p?.kind === 'run' ? 'ran code on' : 'submitted'} ${problem}`,
      }
    case 'submission.finished': {
      const score = typeof p?.score === 'number' ? ` · ${p.score}/${p.max_score}` : ''
      return {
        id: '',
        type,
        at,
        tone: p?.kind === 'run' ? 'info' : statusTone(p?.status),
        text: `${name} — ${p?.kind === 'run' ? 'Run' : 'Submit'} ${problem}: ${p?.status ?? ''}${p?.kind === 'submit' ? score : ''}`,
      }
    }
    case 'config.updated':
      return {
        id: '',
        type,
        at,
        tone: 'warn',
        text: `Contest config updated: “${p?.title}”, ${p?.duration_minutes} min, ${p?.is_active ? 'ACTIVE' : 'inactive'}`,
      }
    case 'secrets.generated':
      return { id: '', type, at, tone: 'info', text: `Generated ${p?.count} secret IDs` }
    case 'secrets.deleted_all':
      return {
        id: '',
        type,
        at,
        tone: 'bad',
        text: `Deleted all secrets (${p?.secrets_deleted}) and students (${p?.students_deleted})`,
      }
    case 'contest.reset':
      return {
        id: '',
        type,
        at,
        tone: 'bad',
        text: `Contest reset — students ${p?.students_deleted}, secrets ${p?.secrets_deleted}, submissions ${p?.submissions_deleted}. Questions kept.`,
      }
    case 'problem.created':
      return { id: '', type, at, tone: 'info', text: `Problem added: “${p?.title}”` }
    case 'problem.updated':
      return { id: '', type, at, tone: 'info', text: `Problem updated: “${p?.title}”` }
    case 'problem.deleted':
      return { id: '', type, at, tone: 'warn', text: `Problem #${p?.id} deleted` }
    case 'problem.test_cases_updated':
      return { id: '', type, at, tone: 'info', text: `Test cases updated for problem #${p?.problem_id} (${p?.count})` }
    case 'leaderboard.updated':
      return null
    default:
      return { id: '', type, at, tone: 'info', text: type }
  }
}

export default function AdminDashboard() {
  const [data, setData] = useState<Dashboard | null>(null)
  const [feed, setFeed] = useState<FeedItem[]>([])
  const [subs, setSubs] = useState<any[]>([])
  const [ws, setWs] = useState<'connecting' | 'live' | 'offline'>('connecting')
  const [error, setError] = useState('')
  const [, setTick] = useState(0)
  const refreshTimer = useRef<number | null>(null)

  const refresh = useCallback(async () => {
    try {
      const [d, s] = await Promise.all([api.adminDashboard(), api.adminSubmissions()])
      setData(d)
      setSubs(asArray(s).slice(0, 12))
      setError('')
    } catch (e: any) {
      setError(e.message)
    }
  }, [])

  const scheduleRefresh = useCallback(() => {
    if (refreshTimer.current) return
    refreshTimer.current = window.setTimeout(() => {
      refreshTimer.current = null
      refresh()
    }, 600)
  }, [refresh])

  useEffect(() => {
    refresh()
    api
      .adminActivity()
      .then((evs) => {
        const items: FeedItem[] = []
        for (const ev of asArray<any>(evs)) {
          const item = describeEvent(ev.event_type, ev.payload, ev.created_at)
          if (item) items.push({ ...item, id: `a-${ev.id}` })
        }
        setFeed(items.slice(0, 60))
      })
      .catch(() => {})

    let sock: WebSocket | null = null
    let closed = false
    let retry: number | null = null
    const connect = () => {
      sock = new WebSocket(adminWsUrl())
      sock.onopen = () => setWs('live')
      sock.onclose = () => {
        setWs('offline')
        if (!closed) retry = window.setTimeout(connect, 3000)
      }
      sock.onerror = () => setWs('offline')
      sock.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data)
          const payload = msg.payload?.payload ?? msg.payload
          const at = msg.payload?.event?.created_at ?? new Date().toISOString()
          const item = describeEvent(msg.type, payload, at)
          if (item) {
            setFeed((prev) => [{ ...item, id: `w-${Date.now()}-${Math.random()}` }, ...prev].slice(0, 60))
          }
          scheduleRefresh()
        } catch {
          /* ignore malformed */
        }
      }
    }
    connect()

    const poll = window.setInterval(refresh, 30000)
    const tick = window.setInterval(() => setTick((t) => t + 1), 10000)
    return () => {
      closed = true
      if (retry) clearTimeout(retry)
      if (refreshTimer.current) clearTimeout(refreshTimer.current)
      clearInterval(poll)
      clearInterval(tick)
      sock?.close()
    }
  }, [refresh, scheduleRefresh])

  if (!data && !error) return <p style={{ color: 'var(--muted)' }}>Loading dashboard…</p>

  const p = data?.participants
  const s = data?.submissions
  const sc = data?.scoring
  const c = data?.contest

  return (
    <div className="dash">
      <div className="dash-head">
        <div>
          <h1 style={{ marginBottom: '.2rem' }}>Byte Battle 2026</h1>
          <div className="dash-sub">
            <span className={`pill ${c?.is_active ? 'pill-ok' : 'pill-muted'}`}>
              <span className="dot" /> {c?.is_active ? 'Contest active' : 'Contest inactive'}
            </span>
            <span className="pill pill-muted">{c?.duration_minutes} min per student</span>
            <span className={`pill ${ws === 'live' ? 'pill-ok' : ws === 'offline' ? 'pill-bad' : 'pill-muted'}`}>
              <span className="dot" /> {ws === 'live' ? 'Live' : ws === 'offline' ? 'Reconnecting…' : 'Connecting…'}
            </span>
          </div>
        </div>
        <div className="row-actions" style={{ margin: 0 }}>
          <button className="btn secondary" type="button" onClick={refresh}>
            Refresh
          </button>
          <Link className="btn" to="/admin/leaderboard">
            Leaderboard
          </Link>
        </div>
      </div>

      {error && <div className="error">{error}</div>}

      {data && p && s && sc && (
        <>
          <div className="kpis">
            <div className="kpi">
              <div className="kpi-l">Participants</div>
              <div className="kpi-n">
                {p.registered}
                <span className="kpi-sub"> / {p.secrets_total} IDs</span>
              </div>
              <div className="kpi-foot">{p.secrets_total - p.secrets_claimed} IDs unused</div>
            </div>
            <div className="kpi kpi-accent">
              <div className="kpi-l">In progress</div>
              <div className="kpi-n">{p.in_progress}</div>
              <div className="kpi-foot">{p.active_last_5min} active in last 5 min</div>
            </div>
            <div className="kpi">
              <div className="kpi-l">Finished</div>
              <div className="kpi-n">{p.finished}</div>
              <div className="kpi-foot">{p.not_started} logged in, not started</div>
            </div>
            <div className="kpi">
              <div className="kpi-l">Submissions</div>
              <div className="kpi-n">{s.total}</div>
              <div className="kpi-foot">
                {s.last_15min} in last 15 min · {s.runs} runs
              </div>
            </div>
            <div className={`kpi ${s.acceptance_rate >= 50 ? 'kpi-good' : s.total ? 'kpi-warn' : ''}`}>
              <div className="kpi-l">Acceptance rate</div>
              <div className="kpi-n">{s.total ? `${Math.round(s.acceptance_rate)}%` : '—'}</div>
              <div className="kpi-foot">
                {s.accepted} AC · {s.partial} partial · {s.failed} failed
                {s.errored ? ` · ${s.errored} judge errors` : ''}
              </div>
            </div>
            <div className="kpi">
              <div className="kpi-l">Avg score</div>
              <div className="kpi-n">
                {p.registered ? sc.avg_total.toFixed(1) : '—'}
                <span className="kpi-sub"> / {sc.max_possible}</span>
              </div>
              <div className="kpi-foot">
                median {sc.median_total.toFixed(0)} · top {sc.top_total} · {sc.full_solvers} full score
              </div>
            </div>
          </div>

          <div className="dash-grid">
            <div className="panel">
              <h2>Participation funnel</h2>
              <Funnel
                rows={[
                  ['Secret IDs generated', p.secrets_total, p.secrets_total],
                  ['Logged in', p.registered, p.secrets_total],
                  ['Started contest', p.in_progress + p.finished, p.secrets_total],
                  ['Currently solving', p.in_progress, p.secrets_total],
                  ['Time finished', p.finished, p.secrets_total],
                ]}
              />
            </div>

            <div className="panel">
              <h2>Submission outcomes</h2>
              {s.total === 0 ? (
                <p className="muted">No final submissions yet.</p>
              ) : (
                <>
                  <div className="stackbar">
                    <span className="seg ok" style={{ width: `${pct(s.accepted, s.total)}%` }} title="Accepted" />
                    <span className="seg warn" style={{ width: `${pct(s.partial, s.total)}%` }} title="Partial" />
                    <span className="seg bad" style={{ width: `${pct(s.failed, s.total)}%` }} title="Failed" />
                    <span className="seg err" style={{ width: `${pct(s.errored, s.total)}%` }} title="Judge error" />
                  </div>
                  <div className="legend">
                    <span>
                      <i className="sw ok" /> Accepted {s.accepted}
                    </span>
                    <span>
                      <i className="sw warn" /> Partial {s.partial}
                    </span>
                    <span>
                      <i className="sw bad" /> Failed {s.failed}
                    </span>
                    {s.errored > 0 && (
                      <span>
                        <i className="sw err" /> Judge error {s.errored}
                      </span>
                    )}
                  </div>
                </>
              )}
              <div className="mini-stats">
                <div>
                  <div className="mini-n">{s.runs}</div>
                  <div className="mini-l">Total runs</div>
                </div>
                <div>
                  <div className="mini-n">{s.runs_last_15min}</div>
                  <div className="mini-l">Runs / 15 min</div>
                </div>
                <div>
                  <div className="mini-n">{s.last_15min}</div>
                  <div className="mini-l">Submits / 15 min</div>
                </div>
                <div>
                  <div className="mini-n">{sc.zero_scorers}</div>
                  <div className="mini-l">Zero score</div>
                </div>
              </div>
            </div>
          </div>

          <div className="panel">
            <div className="panel-head">
              <h2>Problem breakdown</h2>
              <Link to="/admin/questions" className="muted-link">
                Manage questions →
              </Link>
            </div>
            {data.problems.length === 0 ? (
              <p className="muted">No problems configured.</p>
            ) : (
              <table className="table">
                <thead>
                  <tr>
                    <th>#</th>
                    <th>Problem</th>
                    <th>Difficulty</th>
                    <th className="num">Attempted</th>
                    <th className="num">Solved</th>
                    <th style={{ width: '26%' }}>Solve rate</th>
                    <th className="num">Avg best</th>
                    <th className="num">Submits</th>
                    <th className="num">Runs</th>
                  </tr>
                </thead>
                <tbody>
                  {data.problems.map((pr) => {
                    const rate = pct(pr.solved, pr.attempted)
                    return (
                      <tr key={pr.id}>
                        <td className="mono muted">{pr.order_index}</td>
                        <td>
                          <strong>{pr.title}</strong>
                          <div className="muted" style={{ fontSize: '.75rem' }}>max {pr.max_score} pts</div>
                        </td>
                        <td>
                          <span className={`badge ${pr.difficulty}`}>{pr.difficulty}</span>
                        </td>
                        <td className="num mono">{pr.attempted}</td>
                        <td className="num mono">{pr.solved}</td>
                        <td>
                          <div className="bar">
                            <span
                              className={`bar-fill ${rate >= 60 ? 'ok' : rate >= 25 ? 'warn' : 'bad'}`}
                              style={{ width: `${rate}%` }}
                            />
                          </div>
                          <div className="muted" style={{ fontSize: '.72rem', marginTop: 2 }}>
                            {pr.attempted ? `${rate}% of attempters` : 'no attempts'}
                          </div>
                        </td>
                        <td className="num mono">{pr.attempted ? `${Math.round(pr.avg_best_pct)}%` : '—'}</td>
                        <td className="num mono">{pr.submits}</td>
                        <td className="num mono">{pr.runs}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            )}
          </div>

          <div className="dash-grid">
            <div className="panel">
              <div className="panel-head">
                <h2>Top performers</h2>
                <Link to="/admin/leaderboard" className="muted-link">
                  Full leaderboard →
                </Link>
              </div>
              {data.top.length === 0 ? (
                <p className="muted">No students yet.</p>
              ) : (
                <ol className="toplist">
                  {data.top.map((t, i) => (
                    <li key={t.student_id}>
                      <span className={`rank r${i + 1}`}>{i + 1}</span>
                      <span className="who">
                        <strong>{t.name}</strong>
                        <span className="mono muted"> {t.secret_code}</span>
                      </span>
                      <span className="score mono">
                        {t.total_score}
                        <span className="muted"> / {sc.max_possible}</span>
                      </span>
                    </li>
                  ))}
                </ol>
              )}
            </div>

            <div className="panel">
              <h2>Live activity</h2>
              <div className="feed">
                {feed.length === 0 && <p className="muted">Waiting for events…</p>}
                {feed.map((f) => (
                  <div className={`feed-row tone-${f.tone}`} key={f.id}>
                    <span className="feed-dot" />
                    <span className="feed-text">{f.text}</span>
                    <span className="feed-time mono">{relTime(f.at)}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="panel">
            <h2>Recent submissions</h2>
            {subs.length === 0 ? (
              <p className="muted">No submissions yet.</p>
            ) : (
              <table className="table">
                <thead>
                  <tr>
                    <th>When</th>
                    <th>Student</th>
                    <th>Problem</th>
                    <th>Type</th>
                    <th>Verdict</th>
                    <th className="num">Score</th>
                  </tr>
                </thead>
                <tbody>
                  {subs.map((row) => (
                    <tr key={row.id}>
                      <td className="mono muted">{relTime(row.created_at)}</td>
                      <td>{row.student_name}</td>
                      <td>{row.problem_title}</td>
                      <td>
                        <span className={`pill ${row.kind === 'submit' ? 'pill-accent' : 'pill-muted'}`}>
                          {row.kind}
                        </span>
                      </td>
                      <td>
                        <span className={`verdict v-${statusTone(row.status)}`}>{row.status}</span>
                      </td>
                      <td className="num mono">
                        {row.kind === 'submit' ? `${row.score}/${row.max_score}` : '—'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </>
      )}
    </div>
  )
}

function Funnel({ rows }: { rows: [string, number, number][] }) {
  return (
    <div className="funnel">
      {rows.map(([label, n, d]) => (
        <div className="funnel-row" key={label}>
          <div className="funnel-l">{label}</div>
          <div className="bar">
            <span className="bar-fill accent" style={{ width: `${pct(n, d)}%` }} />
          </div>
          <div className="funnel-n mono">
            {n}
            <span className="muted"> ({pct(n, d)}%)</span>
          </div>
        </div>
      ))}
    </div>
  )
}
