import { FormEvent, useEffect, useState } from 'react'
import { api, asArray } from '../api'

type TestCaseDraft = {
  input: string
  expected_output: string
  is_sample: boolean
  points: number
}

const emptyProblem = {
  slug: '',
  title: '',
  statement_md: `## Problem

Describe the problem here.

### Input

### Output

### Constraints
`,
  difficulty: 'medium',
  order_index: 0,
  starter_code: 'import sys\n\ndef main():\n    data = sys.stdin.read().split()\n    # TODO: solve\n    pass\n\nif __name__ == "__main__":\n    main()\n',
  reference_code: '',
  time_limit_ms: 2000,
  memory_limit_kb: 256000,
}

export default function AdminQuestions() {
  const [problems, setProblems] = useState<any[]>([])
  const [editing, setEditing] = useState<any | null>(null)
  const [cases, setCases] = useState<TestCaseDraft[]>([
    { input: '1 2\n', expected_output: '3\n', is_sample: true, points: 0 },
    { input: '3 4\n', expected_output: '7\n', is_sample: false, points: 10 },
  ])
  const [error, setError] = useState('')
  const [msg, setMsg] = useState('')

  async function refresh() {
    setProblems(asArray(await api.adminProblems()))
  }

  useEffect(() => {
    refresh().catch((e) => setError(e.message))
  }, [])

  function startCreate() {
    setEditing({ ...emptyProblem })
    setCases([
      { input: '', expected_output: '', is_sample: true, points: 0 },
      { input: '', expected_output: '', is_sample: false, points: 10 },
    ])
    setMsg('')
    setError('')
  }

  async function startEdit(id: number) {
    setError('')
    const p = await api.adminProblem(id)
    setEditing(p)
    setCases(
      (p.test_cases || []).map((tc: any) => ({
        input: tc.input,
        expected_output: tc.expected_output,
        is_sample: tc.is_sample,
        points: tc.points,
      })),
    )
  }

  async function onSave(e: FormEvent) {
    e.preventDefault()
    if (!editing) return
    setError('')
    try {
      let id = editing.id
      if (id) {
        await api.adminUpdateProblem(id, editing)
      } else {
        const created = await api.adminCreateProblem(editing)
        id = created.id
      }
      await api.adminReplaceTestCases(id, cases)
      setMsg('Saved')
      setEditing(null)
      await refresh()
    } catch (err: any) {
      setError(err.message)
    }
  }

  async function onDelete(id: number) {
    if (!confirm('Delete this problem?')) return
    await api.adminDeleteProblem(id)
    await refresh()
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1>Questions</h1>
        <button className="btn" onClick={startCreate}>
          Add question
        </button>
      </div>
      {error && <div className="error">{error}</div>}
      {msg && <p style={{ color: 'var(--accent-2)' }}>{msg}</p>}

      {!editing && (
        <div className="panel">
          <table className="table">
            <thead>
              <tr>
                <th>Order</th>
                <th>Title</th>
                <th>Slug</th>
                <th>Difficulty</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {problems.map((p) => (
                <tr key={p.id}>
                  <td>{p.order_index}</td>
                  <td>{p.title}</td>
                  <td className="mono">{p.slug}</td>
                  <td>
                    <span className={`badge ${p.difficulty}`}>{p.difficulty}</span>
                  </td>
                  <td>
                    <div className="row-actions">
                      <button className="btn secondary" onClick={() => startEdit(p.id)}>
                        Edit
                      </button>
                      <button className="btn danger" onClick={() => onDelete(p.id)}>
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {editing && (
        <form className="panel" onSubmit={onSave}>
          <h2>{editing.id ? 'Edit problem' : 'New problem'}</h2>
          <div className="grid-2">
            <div className="field">
              <label>Title</label>
              <input value={editing.title} onChange={(e) => setEditing({ ...editing, title: e.target.value })} required />
            </div>
            <div className="field">
              <label>Slug</label>
              <input value={editing.slug} onChange={(e) => setEditing({ ...editing, slug: e.target.value })} required />
            </div>
            <div className="field">
              <label>Difficulty</label>
              <select value={editing.difficulty} onChange={(e) => setEditing({ ...editing, difficulty: e.target.value })}>
                <option value="easy">easy</option>
                <option value="medium">medium</option>
                <option value="hard">hard</option>
              </select>
            </div>
            <div className="field">
              <label>Order</label>
              <input
                type="number"
                value={editing.order_index}
                onChange={(e) => setEditing({ ...editing, order_index: Number(e.target.value) })}
              />
            </div>
            <div className="field">
              <label>Time limit (ms)</label>
              <input
                type="number"
                value={editing.time_limit_ms}
                onChange={(e) => setEditing({ ...editing, time_limit_ms: Number(e.target.value) })}
              />
            </div>
            <div className="field">
              <label>Memory (KB)</label>
              <input
                type="number"
                value={editing.memory_limit_kb}
                onChange={(e) => setEditing({ ...editing, memory_limit_kb: Number(e.target.value) })}
              />
            </div>
          </div>
          <div className="field">
            <label>Statement (Markdown)</label>
            <textarea
              rows={12}
              value={editing.statement_md}
              onChange={(e) => setEditing({ ...editing, statement_md: e.target.value })}
            />
          </div>
          <div className="field">
            <label>Starter code (shown to students)</label>
            <textarea
              rows={8}
              className="mono"
              value={editing.starter_code}
              onChange={(e) => setEditing({ ...editing, starter_code: e.target.value })}
            />
          </div>
          <div className="field">
            <label>Official solution (hidden — used to check custom input)</label>
            <textarea
              rows={8}
              className="mono"
              value={editing.reference_code || ''}
              onChange={(e) => setEditing({ ...editing, reference_code: e.target.value })}
              placeholder="Full working solution. Runs against student custom stdin; outputs are compared."
            />
          </div>

          <h3>Test cases</h3>
          {cases.map((tc, i) => (
            <div key={i} className="panel" style={{ background: 'var(--bg)' }}>
              <div className="row-actions">
                <label>
                  <input
                    type="checkbox"
                    checked={tc.is_sample}
                    onChange={(e) => {
                      const next = [...cases]
                      next[i] = { ...tc, is_sample: e.target.checked, points: e.target.checked ? 0 : tc.points || 10 }
                      setCases(next)
                    }}
                  />{' '}
                  Sample
                </label>
                <div className="field" style={{ margin: 0, width: 120 }}>
                  <label>Points</label>
                  <input
                    type="number"
                    value={tc.points}
                    disabled={tc.is_sample}
                    onChange={(e) => {
                      const next = [...cases]
                      next[i] = { ...tc, points: Number(e.target.value) }
                      setCases(next)
                    }}
                  />
                </div>
                <button
                  type="button"
                  className="btn danger"
                  onClick={() => setCases(cases.filter((_, j) => j !== i))}
                >
                  Remove
                </button>
              </div>
              <div className="grid-2">
                <div className="field">
                  <label>Input</label>
                  <textarea
                    rows={4}
                    className="mono"
                    value={tc.input}
                    onChange={(e) => {
                      const next = [...cases]
                      next[i] = { ...tc, input: e.target.value }
                      setCases(next)
                    }}
                  />
                </div>
                <div className="field">
                  <label>Expected output</label>
                  <textarea
                    rows={4}
                    className="mono"
                    value={tc.expected_output}
                    onChange={(e) => {
                      const next = [...cases]
                      next[i] = { ...tc, expected_output: e.target.value }
                      setCases(next)
                    }}
                  />
                </div>
              </div>
            </div>
          ))}
          <div className="row-actions">
            <button
              type="button"
              className="btn secondary"
              onClick={() => setCases([...cases, { input: '', expected_output: '', is_sample: false, points: 10 }])}
            >
              Add test case
            </button>
            <button className="btn" type="submit">
              Save
            </button>
            <button className="btn ghost" type="button" onClick={() => setEditing(null)}>
              Cancel
            </button>
          </div>
        </form>
      )}
    </div>
  )
}
