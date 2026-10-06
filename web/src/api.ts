export type ContestConfig = {
  title: string
  duration_minutes: number
  is_active: boolean
  updated_at?: string
}

export type Student = {
  id: number
  name: string
  secret_code?: string
  started_at?: string | null
}

export type ProblemSummary = {
  id: number
  slug: string
  title: string
  difficulty: string
  order_index: number
  best_score: number
  max_score: number
  submit_count: number
  submits_left: number
}

export type Sample = {
  id: number
  input: string
  expected_output: string
  points: number
  is_sample: boolean
}

export type ProblemDetail = {
  id: number
  slug: string
  title: string
  statement_md: string
  difficulty: string
  starter_code: string
  time_limit_ms: number
  memory_limit_kb: number
  samples: Sample[]
  progress?: {
    best_score: number
    max_score: number
    submit_count: number
    submits_left: number
  }
}

export type DashboardProblem = {
  id: number
  title: string
  difficulty: string
  order_index: number
  max_score: number
  attempted: number
  solved: number
  submits: number
  runs: number
  avg_best_pct: number
}

export type Dashboard = {
  contest: { title: string; duration_minutes: number; is_active: boolean }
  participants: {
    secrets_total: number
    secrets_claimed: number
    registered: number
    not_started: number
    in_progress: number
    finished: number
    active_last_5min: number
  }
  submissions: {
    total: number
    accepted: number
    partial: number
    failed: number
    errored: number
    runs: number
    last_15min: number
    runs_last_15min: number
    acceptance_rate: number
  }
  scoring: {
    max_possible: number
    avg_total: number
    median_total: number
    top_total: number
    full_solvers: number
    zero_scorers: number
  }
  problems: DashboardProblem[]
  top: LeaderboardEntry[]
}

export type LeaderboardEntry = {
  student_id: number
  name: string
  secret_code: string
  total_score: number
  problem_scores: Record<string, number>
  last_improve_at?: string
  started_at?: string
}

const STUDENT_KEY = 'student_token'
const ADMIN_KEY = 'admin_token'

export function getStudentToken() {
  return localStorage.getItem(STUDENT_KEY)
}
export function setStudentToken(t: string | null) {
  if (t) localStorage.setItem(STUDENT_KEY, t)
  else localStorage.removeItem(STUDENT_KEY)
}
export function getAdminToken() {
  return localStorage.getItem(ADMIN_KEY)
}
export function setAdminToken(t: string | null) {
  if (t) localStorage.setItem(ADMIN_KEY, t)
  else localStorage.removeItem(ADMIN_KEY)
}

export function asArray<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : []
}

async function request<T>(path: string, opts: RequestInit = {}, kind: 'student' | 'admin' | 'none' = 'none'): Promise<T> {
  const headers = new Headers(opts.headers)
  if (!headers.has('Content-Type') && opts.body) headers.set('Content-Type', 'application/json')
  if (kind === 'student') {
    const t = getStudentToken()
    if (t) headers.set('X-Student-Token', t)
  }
  if (kind === 'admin') {
    const t = getAdminToken()
    if (t) headers.set('Authorization', `Bearer ${t}`)
  }
  const res = await fetch(path, { ...opts, headers })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(data.error || res.statusText)
  return data as T
}

export const api = {
  studentLogin: (secret: string, name: string) =>
    request<{ token: string; student: Student; config: ContestConfig; remaining_seconds: number }>(
      '/api/student/login',
      { method: 'POST', body: JSON.stringify({ secret, name }) },
    ),
  studentMe: () =>
    request<{
      student: Student
      config: ContestConfig
      remaining_seconds: number
      expired: boolean
      progress: unknown[]
    }>('/api/student/me', {}, 'student'),
  studentStart: () =>
    request<{ student: Student; remaining_seconds: number }>('/api/student/start', { method: 'POST' }, 'student'),
  studentProblems: () => request<ProblemSummary[]>('/api/student/problems', {}, 'student'),
  studentProblem: (id: number) => request<ProblemDetail>(`/api/student/problems/${id}`, {}, 'student'),
  studentRun: (id: number, source: string, custom_input?: string) =>
    request<any>(
      `/api/student/problems/${id}/run`,
      {
        method: 'POST',
        body: JSON.stringify(
          custom_input !== undefined ? { source, custom_input } : { source },
        ),
      },
      'student',
    ),
  studentSubmit: (id: number, source: string) =>
    request<any>(`/api/student/problems/${id}/submit`, { method: 'POST', body: JSON.stringify({ source }) }, 'student'),
  studentSubmissions: () => request<any[]>('/api/student/submissions', {}, 'student'),

  adminLogin: (username: string, password: string) =>
    request<{ token: string; username: string }>('/api/admin/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  adminConfig: () => request<ContestConfig>('/api/admin/config', {}, 'admin'),
  adminUpdateConfig: (body: Partial<ContestConfig> & { title: string; duration_minutes: number; is_active: boolean }) =>
    request<ContestConfig>('/api/admin/config', { method: 'PUT', body: JSON.stringify(body) }, 'admin'),
  adminGenerateSecrets: (count: number) =>
    request<any[]>('/api/admin/secrets/generate', { method: 'POST', body: JSON.stringify({ count }) }, 'admin'),
  adminSecrets: () => request<any[]>('/api/admin/secrets', {}, 'admin'),
  adminDeleteAllSecrets: () =>
    request<{ secrets_deleted: number; students_deleted: number }>('/api/admin/secrets', { method: 'DELETE' }, 'admin'),
  adminResetContest: () =>
    request<{
      students_deleted: number
      secrets_deleted: number
      submissions_deleted: number
      events_deleted: number
    }>('/api/admin/reset', { method: 'POST' }, 'admin'),
  adminProblems: () => request<any[]>('/api/admin/problems', {}, 'admin'),
  adminProblem: (id: number) => request<any>(`/api/admin/problems/${id}`, {}, 'admin'),
  adminCreateProblem: (body: any) =>
    request<any>('/api/admin/problems', { method: 'POST', body: JSON.stringify(body) }, 'admin'),
  adminUpdateProblem: (id: number, body: any) =>
    request<any>(`/api/admin/problems/${id}`, { method: 'PUT', body: JSON.stringify(body) }, 'admin'),
  adminDeleteProblem: (id: number) =>
    request<any>(`/api/admin/problems/${id}`, { method: 'DELETE' }, 'admin'),
  adminReplaceTestCases: (id: number, test_cases: any[]) =>
    request<any>(`/api/admin/problems/${id}/test-cases`, {
      method: 'PUT',
      body: JSON.stringify({ test_cases }),
    }, 'admin'),
  adminStudents: () => request<any[]>('/api/admin/students', {}, 'admin'),
  adminSubmissions: () => request<any[]>('/api/admin/submissions', {}, 'admin'),
  adminLeaderboard: () => request<LeaderboardEntry[]>('/api/admin/leaderboard', {}, 'admin'),
  adminActivity: () => request<any[]>('/api/admin/activity', {}, 'admin'),
  adminStats: () => request<Record<string, number>>('/api/admin/stats', {}, 'admin'),
  adminDashboard: () => request<Dashboard>('/api/admin/dashboard', {}, 'admin'),
}

export function adminWsUrl() {
  const token = getAdminToken()
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  return `${proto}://${location.host}/api/admin/ws?token=${encodeURIComponent(token || '')}`
}

export function formatTime(seconds: number) {
  const s = Math.max(0, Math.floor(seconds))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
  return `${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
}
