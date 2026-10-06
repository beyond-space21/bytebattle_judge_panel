import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Editor from '@monaco-editor/react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import {
  api,
  asArray,
  formatTime,
  ProblemDetail,
  ProblemSummary,
  setStudentToken,
} from '../api'
import logo from '../../asset/logo.png'

export default function StudentWorkspace() {
  const nav = useNavigate()
  const [teamName, setTeamName] = useState('')
  const [remaining, setRemaining] = useState(0)
  const [expired, setExpired] = useState(false)
  const [problems, setProblems] = useState<ProblemSummary[]>([])
  const [selected, setSelected] = useState<number | null>(null)
  const [problem, setProblem] = useState<ProblemDetail | null>(null)
  const [code, setCode] = useState('')
  const [customInput, setCustomInput] = useState('')
  const [useCustom, setUseCustom] = useState(false)
  const [result, setResult] = useState<any>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const codeCache = useMemo(() => new Map<number, string>(), [])

  const refreshMe = useCallback(async () => {
    const me = await api.studentMe()
    setTeamName(me.student.name)
    setRemaining(me.remaining_seconds)
    setExpired(me.expired)
    if (!me.student.started_at) {
      nav('/lobby', { replace: true })
      return
    }
  }, [nav])

  const loadProblems = useCallback(async () => {
    const list = asArray(await api.studentProblems())
    setProblems(list)
    if (selected == null && list.length) setSelected(list[0].id)
  }, [selected])

  useEffect(() => {
    refreshMe().catch((e) => setError(e.message))
    loadProblems().catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    const t = setInterval(() => {
      setRemaining((r) => {
        if (r <= 1) {
          setExpired(true)
          return 0
        }
        return r - 1
      })
    }, 1000)
    return () => clearInterval(t)
  }, [])

  useEffect(() => {
    if (selected == null) return
    setError('')
    setResult(null)
    api
      .studentProblem(selected)
      .then((p) => {
        setProblem(p)
        const cached = codeCache.get(p.id)
        setCode(cached ?? p.starter_code)
        if (p.samples?.[0]) setCustomInput(p.samples[0].input)
      })
      .catch((e) => setError(e.message))
  }, [selected, codeCache])

  function onCodeChange(v: string | undefined) {
    const next = v ?? ''
    setCode(next)
    if (selected != null) codeCache.set(selected, next)
  }

  async function run() {
    if (selected == null || expired) return
    setBusy(true)
    setError('')
    try {
      const res = await api.studentRun(selected, code, useCustom ? customInput : undefined)
      setResult(res.result)
      await loadProblems()
    } catch (e: any) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }

  async function submit() {
    if (selected == null || expired) return
    if (!confirm('Submit counts toward your 3-attempt limit. Continue?')) return
    setBusy(true)
    setError('')
    try {
      const res = await api.studentSubmit(selected, code)
      setResult(res.result)
      await loadProblems()
      await refreshMe()
    } catch (e: any) {
      setError(e.message)
    } finally {
      setBusy(false)
    }
  }

  const timerClass = remaining <= 60 ? 'danger' : remaining <= 300 ? 'warn' : ''
  const submitsLeft = problem?.progress?.submits_left ?? problems.find((p) => p.id === selected)?.submits_left ?? 3
  const score = problems.reduce((sum, p) => sum + (p.best_score ?? 0), 0)
  const maxScore = problems.reduce((sum, p) => sum + (p.max_score ?? 0), 0)

  return (
    <div className="workspace">
      <div className="topbar">
        <div className="brand">
          <img className="brand-logo" src={logo} alt="Byte Battle" />
          <span className="team-name">{teamName}</span>
          <span className="team-score">
            {score}
            <span className="team-score-max"> / {maxScore}</span>
          </span>
        </div>
        <div style={{ display: 'flex', gap: '.75rem', alignItems: 'center' }}>
          <span className={`timer ${timerClass}`}>{expired ? 'TIME UP' : formatTime(remaining)}</span>
          <button
            className="btn ghost"
            onClick={() => {
              setStudentToken(null)
              nav('/')
            }}
          >
            Log out
          </button>
        </div>
      </div>
      <div className="workspace-body">
        <aside className="problem-list">
          <h3>Problems</h3>
          {problems.map((p) => (
            <button
              key={p.id}
              className={`problem-item ${selected === p.id ? 'active' : ''}`}
              onClick={() => setSelected(p.id)}
            >
              <div className="title">{p.title}</div>
              <div className="meta-row">
                <span className={`badge ${p.difficulty}`}>{p.difficulty}</span>
                <span className="pts" title="Your best score / total points for this question">
                  {p.best_score ?? 0}/{p.max_score ?? 0} pts
                </span>
                <span>{p.submits_left} left</span>
              </div>
            </button>
          ))}
        </aside>

        <section className="statement">
          {problem ? (
            <>
              <h2 style={{ marginTop: 0 }}>{problem.title}</h2>
              <div className="meta-row" style={{ marginBottom: '1rem', color: 'var(--muted)', fontSize: '.85rem' }}>
                <span className={`badge ${problem.difficulty}`}>{problem.difficulty}</span>
                <span className="pts">
                  {problem.progress?.best_score ?? 0}/{problem.progress?.max_score ?? 0} pts
                </span>
                <span>
                  {problem.time_limit_ms}ms · {(problem.memory_limit_kb / 1024).toFixed(0)}MB
                </span>
              </div>
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{problem.statement_md}</ReactMarkdown>
              {problem.samples?.length > 0 && (
                <>
                  <h3>Examples</h3>
                  {problem.samples.map((s, i) => (
                    <div key={s.id} style={{ marginBottom: '1rem' }}>
                      <strong>Example {i + 1}</strong>
                      <div className="sample-box" style={{ marginTop: '.35rem' }}>
                        <div>Input:</div>
                        <pre style={{ margin: '.25rem 0' }}>{s.input}</pre>
                        <div>Output:</div>
                        <pre style={{ margin: '.25rem 0 0' }}>{s.expected_output}</pre>
                      </div>
                    </div>
                  ))}
                </>
              )}
            </>
          ) : (
            <p style={{ color: 'var(--muted)' }}>Select a problem</p>
          )}
        </section>

        <section className="editor-col">
          <div className="editor-toolbar">
            <button className="btn secondary" disabled={busy || expired || !problem} onClick={run}>
              Run
            </button>
            <button className="btn success" disabled={busy || expired || !problem || submitsLeft <= 0} onClick={submit}>
              Submit ({submitsLeft} left)
            </button>
            <label style={{ display: 'flex', alignItems: 'center', gap: '.35rem', color: 'var(--muted)', fontSize: '.85rem' }} title="Runs your code and the official solution on the same stdin, then compares outputs.">
              <input type="checkbox" checked={useCustom} onChange={(e) => setUseCustom(e.target.checked)} />
              Custom input
            </label>
          </div>
          <div className="editor-wrap">
            <Editor
              height="100%"
              defaultLanguage="python"
              theme="vs-dark"
              value={code}
              onChange={onCodeChange}
              options={{ fontSize: 14, minimap: { enabled: false }, automaticLayout: true }}
            />
          </div>
          {useCustom && (
            <textarea
              value={customInput}
              onChange={(e) => setCustomInput(e.target.value)}
              rows={4}
              style={{
                width: '100%',
                border: 'none',
                borderTop: '1px solid var(--border)',
                background: '#0b1016',
                padding: '.75rem',
                fontFamily: 'var(--mono)',
                resize: 'vertical',
              }}
              placeholder="Custom stdin…"
            />
          )}
          <div className="results">
            {error && <div className="error">{error}</div>}
            {!result && !error && <div style={{ color: 'var(--muted)' }}>Run sample tests or submit for scoring.</div>}
            {result && (
              <>
                <h4>
                  {result.status || 'Result'}
                  {typeof result.score === 'number' && result.cases && (
                    <span>
                      {' '}
                      — {result.score}/{result.max_score}
                    </span>
                  )}
                  {result.stdout != null && !result.cases && (
                    <span style={{ color: 'var(--muted)', fontWeight: 400 }}> — vs official solution</span>
                  )}
                </h4>
                {result.cases?.map((c: any, i: number) => (
                  <div className="case-row" key={i}>
                    <span className={c.passed ? 'pass' : 'fail'}>{c.passed ? 'PASS' : 'FAIL'}</span>
                    <span>
                      Case {c.index ?? i + 1}: {c.status}
                      {typeof c.points === 'number' ? ` (+${c.points})` : ''}
                      {c.stdout != null && (
                        <>
                          <br />
                          stdout: {c.stdout || '(empty)'}
                        </>
                      )}
                      {c.stderr ? (
                        <>
                          <br />
                          stderr: {c.stderr}
                        </>
                      ) : null}
                      {c.expected != null && (
                        <>
                          <br />
                          expected: {c.expected}
                        </>
                      )}
                    </span>
                  </div>
                ))}
                {result.stdout != null && !result.cases && (
                  <div className="case-row">
                    <span className={result.passed ? 'pass' : 'fail'}>{result.passed ? 'PASS' : 'FAIL'}</span>
                    <pre style={{ whiteSpace: 'pre-wrap', margin: 0 }}>
                      your output: {result.stdout || '(empty)'}
                      {result.expected != null ? `\nexpected: ${result.expected || '(empty)'}` : ''}
                      {result.stderr ? `\nstderr: ${result.stderr}` : ''}
                    </pre>
                  </div>
                )}
              </>
            )}
          </div>
        </section>
      </div>
    </div>
  )
}
