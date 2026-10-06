package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrForbidden     = errors.New("forbidden")
	ErrLimitExceeded = errors.New("limit exceeded")
	ErrExpired       = errors.New("expired")
	ErrInactive      = errors.New("contest inactive")
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// --- Models ---

type Admin struct {
	ID           int64
	Username     string
	PasswordHash string
}

type ContestConfig struct {
	Title            string    `json:"title"`
	DurationMinutes  int       `json:"duration_minutes"`
	IsActive         bool      `json:"is_active"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Secret struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	ClaimedAt *time.Time `json:"claimed_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type Student struct {
	ID        int64      `json:"id"`
	SecretID  int64      `json:"secret_id"`
	SecretCode string    `json:"secret_code,omitempty"`
	Name      string     `json:"name"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type Problem struct {
	ID            int64     `json:"id"`
	Slug          string    `json:"slug"`
	Title         string    `json:"title"`
	StatementMD   string    `json:"statement_md"`
	Difficulty    string    `json:"difficulty"`
	OrderIndex    int       `json:"order_index"`
	StarterCode   string    `json:"starter_code"`
	ReferenceCode string    `json:"reference_code"` // admin-only; used for custom-input validation
	TimeLimitMS   int       `json:"time_limit_ms"`
	MemoryLimitKB int       `json:"memory_limit_kb"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	TestCases     []TestCase `json:"test_cases,omitempty"`
}

type TestCase struct {
	ID             int64  `json:"id"`
	ProblemID      int64  `json:"problem_id"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	IsSample       bool   `json:"is_sample"`
	Points         int    `json:"points"`
	OrderIndex     int    `json:"order_index"`
}

type Submission struct {
	ID            int64           `json:"id"`
	StudentID     int64           `json:"student_id"`
	ProblemID     int64           `json:"problem_id"`
	Kind          string          `json:"kind"`
	Source        string          `json:"source"`
	Status        string          `json:"status"`
	Score         int             `json:"score"`
	MaxScore      int             `json:"max_score"`
	VerdictDetail json.RawMessage `json:"verdict_detail"`
	CreatedAt     time.Time       `json:"created_at"`
	StudentName   string          `json:"student_name,omitempty"`
	ProblemTitle  string          `json:"problem_title,omitempty"`
}

type AuditEvent struct {
	ID        int64           `json:"id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type LeaderboardEntry struct {
	StudentID       int64      `json:"student_id"`
	Name            string     `json:"name"`
	SecretCode      string     `json:"secret_code"`
	TotalScore      int        `json:"total_score"`
	ProblemScores   map[string]int `json:"problem_scores"`
	LastImproveAt   *time.Time `json:"last_improve_at,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
}

type ProblemProgress struct {
	ProblemID      int64 `json:"problem_id"`
	BestScore      int   `json:"best_score"`
	MaxScore       int   `json:"max_score"`
	SubmitCount    int   `json:"submit_count"`
	SubmitsLeft    int   `json:"submits_left"`
}

// --- Admins ---

func (s *Store) EnsureAdmin(ctx context.Context, username, passwordHash string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO admins (username, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (username) DO NOTHING
	`, username, passwordHash)
	return err
}

func (s *Store) GetAdminByUsername(ctx context.Context, username string) (*Admin, error) {
	var a Admin
	err := s.pool.QueryRow(ctx, `SELECT id, username, password_hash FROM admins WHERE username=$1`, username).
		Scan(&a.ID, &a.Username, &a.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &a, err
}

// --- Contest config ---

func (s *Store) GetConfig(ctx context.Context) (*ContestConfig, error) {
	var c ContestConfig
	err := s.pool.QueryRow(ctx, `
		SELECT title, duration_minutes, is_active, updated_at FROM contest_config WHERE id=1
	`).Scan(&c.Title, &c.DurationMinutes, &c.IsActive, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (s *Store) UpdateConfig(ctx context.Context, title string, durationMinutes int, isActive bool) (*ContestConfig, error) {
	var c ContestConfig
	err := s.pool.QueryRow(ctx, `
		UPDATE contest_config
		SET title=$1, duration_minutes=$2, is_active=$3, updated_at=NOW()
		WHERE id=1
		RETURNING title, duration_minutes, is_active, updated_at
	`, title, durationMinutes, isActive).Scan(&c.Title, &c.DurationMinutes, &c.IsActive, &c.UpdatedAt)
	return &c, err
}

// --- Secrets ---

func (s *Store) CreateSecrets(ctx context.Context, codes []string) ([]Secret, error) {
	out := make([]Secret, 0, len(codes))
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	for _, code := range codes {
		var sec Secret
		err := tx.QueryRow(ctx, `
			INSERT INTO secrets (code) VALUES ($1)
			RETURNING id, code, claimed_at, created_at
		`, code).Scan(&sec.ID, &sec.Code, &sec.ClaimedAt, &sec.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("insert secret %s: %w", code, err)
		}
		out = append(out, sec)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListSecrets(ctx context.Context) ([]Secret, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, code, claimed_at, created_at FROM secrets ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Secret, 0)
	for rows.Next() {
		var sec Secret
		if err := rows.Scan(&sec.ID, &sec.Code, &sec.ClaimedAt, &sec.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sec)
	}
	return out, rows.Err()
}

func (s *Store) SecretExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM secrets WHERE code=$1)`, code).Scan(&exists)
	return exists, err
}

// DeleteAllSecrets removes every secret ID and any students (sessions/submissions cascade).
func (s *Store) DeleteAllSecrets(ctx context.Context) (secretsDeleted, studentsDeleted int64, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx)

	tagStudents, err := tx.Exec(ctx, `DELETE FROM students`)
	if err != nil {
		return 0, 0, err
	}
	tagSecrets, err := tx.Exec(ctx, `DELETE FROM secrets`)
	if err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return tagSecrets.RowsAffected(), tagStudents.RowsAffected(), nil
}

// ResetContest wipes students, secret IDs, submissions, and the activity log.
// Problems, test cases, contest settings, and admin accounts are left in place.
func (s *Store) ResetContest(ctx context.Context) (studentsDeleted, secretsDeleted, submissionsDeleted, eventsDeleted int64, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer tx.Rollback(ctx)

	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM submissions`).Scan(&submissionsDeleted); err != nil {
		return 0, 0, 0, 0, err
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM students`).Scan(&studentsDeleted); err != nil {
		return 0, 0, 0, 0, err
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM secrets`).Scan(&secretsDeleted); err != nil {
		return 0, 0, 0, 0, err
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events`).Scan(&eventsDeleted); err != nil {
		return 0, 0, 0, 0, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM students`); err != nil {
		return 0, 0, 0, 0, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM secrets`); err != nil {
		return 0, 0, 0, 0, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM audit_events`); err != nil {
		return 0, 0, 0, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, 0, 0, err
	}
	return studentsDeleted, secretsDeleted, submissionsDeleted, eventsDeleted, nil
}

// --- Students / sessions ---

func (s *Store) LoginStudent(ctx context.Context, code, name string, sessionToken string, expiresAt time.Time) (*Student, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var secID int64
	var claimedAt *time.Time
	err = tx.QueryRow(ctx, `SELECT id, claimed_at FROM secrets WHERE code=$1 FOR UPDATE`, code).
		Scan(&secID, &claimedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var st Student
	err = tx.QueryRow(ctx, `
		SELECT id, secret_id, name, started_at, created_at FROM students WHERE secret_id=$1
	`, secID).Scan(&st.ID, &st.SecretID, &st.Name, &st.StartedAt, &st.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO students (secret_id, name) VALUES ($1, $2)
			RETURNING id, secret_id, name, started_at, created_at
		`, secID, name).Scan(&st.ID, &st.SecretID, &st.Name, &st.StartedAt, &st.CreatedAt)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, `UPDATE secrets SET claimed_at=NOW() WHERE id=$1 AND claimed_at IS NULL`, secID)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// Resume: update name if provided (identification only)
		if name != "" && name != st.Name {
			_, err = tx.Exec(ctx, `UPDATE students SET name=$1 WHERE id=$2`, name, st.ID)
			if err != nil {
				return nil, err
			}
			st.Name = name
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO student_sessions (token, student_id, expires_at) VALUES ($1, $2, $3)
		ON CONFLICT (token) DO UPDATE SET student_id=EXCLUDED.student_id, expires_at=EXCLUDED.expires_at
	`, sessionToken, st.ID, expiresAt)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	st.SecretCode = code
	return &st, nil
}

func (s *Store) StudentBySession(ctx context.Context, token string) (*Student, error) {
	var st Student
	err := s.pool.QueryRow(ctx, `
		SELECT st.id, st.secret_id, st.name, st.started_at, st.created_at, sec.code
		FROM student_sessions ss
		JOIN students st ON st.id = ss.student_id
		JOIN secrets sec ON sec.id = st.secret_id
		WHERE ss.token=$1 AND ss.expires_at > NOW()
	`, token).Scan(&st.ID, &st.SecretID, &st.Name, &st.StartedAt, &st.CreatedAt, &st.SecretCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &st, err
}

func (s *Store) StartStudent(ctx context.Context, studentID int64) (*Student, error) {
	var st Student
	err := s.pool.QueryRow(ctx, `
		UPDATE students SET started_at = COALESCE(started_at, NOW())
		WHERE id=$1
		RETURNING id, secret_id, name, started_at, created_at
	`, studentID).Scan(&st.ID, &st.SecretID, &st.Name, &st.StartedAt, &st.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &st, err
}

func (s *Store) ListStudents(ctx context.Context) ([]Student, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT st.id, st.secret_id, st.name, st.started_at, st.created_at, sec.code
		FROM students st
		JOIN secrets sec ON sec.id = st.secret_id
		ORDER BY st.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Student, 0)
	for rows.Next() {
		var st Student
		if err := rows.Scan(&st.ID, &st.SecretID, &st.Name, &st.StartedAt, &st.CreatedAt, &st.SecretCode); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// --- Problems ---

func (s *Store) ListProblems(ctx context.Context) ([]Problem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, slug, title, statement_md, difficulty, order_index, starter_code, reference_code,
		       time_limit_ms, memory_limit_kb, created_at, updated_at
		FROM problems ORDER BY order_index, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Problem, 0)
	for rows.Next() {
		var p Problem
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.StatementMD, &p.Difficulty, &p.OrderIndex,
			&p.StarterCode, &p.ReferenceCode, &p.TimeLimitMS, &p.MemoryLimitKB, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProblem(ctx context.Context, id int64) (*Problem, error) {
	var p Problem
	err := s.pool.QueryRow(ctx, `
		SELECT id, slug, title, statement_md, difficulty, order_index, starter_code, reference_code,
		       time_limit_ms, memory_limit_kb, created_at, updated_at
		FROM problems WHERE id=$1
	`, id).Scan(&p.ID, &p.Slug, &p.Title, &p.StatementMD, &p.Difficulty, &p.OrderIndex,
		&p.StarterCode, &p.ReferenceCode, &p.TimeLimitMS, &p.MemoryLimitKB, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	tcs, err := s.ListTestCases(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	p.TestCases = tcs
	return &p, nil
}

func (s *Store) CreateProblem(ctx context.Context, p *Problem) (*Problem, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO problems (slug, title, statement_md, difficulty, order_index, starter_code, reference_code, time_limit_ms, memory_limit_kb)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, slug, title, statement_md, difficulty, order_index, starter_code, reference_code, time_limit_ms, memory_limit_kb, created_at, updated_at
	`, p.Slug, p.Title, p.StatementMD, p.Difficulty, p.OrderIndex, p.StarterCode, p.ReferenceCode, p.TimeLimitMS, p.MemoryLimitKB).
		Scan(&p.ID, &p.Slug, &p.Title, &p.StatementMD, &p.Difficulty, &p.OrderIndex, &p.StarterCode, &p.ReferenceCode,
			&p.TimeLimitMS, &p.MemoryLimitKB, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (s *Store) UpdateProblem(ctx context.Context, p *Problem) (*Problem, error) {
	err := s.pool.QueryRow(ctx, `
		UPDATE problems SET
			slug=$2, title=$3, statement_md=$4, difficulty=$5, order_index=$6,
			starter_code=$7, reference_code=$8, time_limit_ms=$9, memory_limit_kb=$10, updated_at=NOW()
		WHERE id=$1
		RETURNING id, slug, title, statement_md, difficulty, order_index, starter_code, reference_code, time_limit_ms, memory_limit_kb, created_at, updated_at
	`, p.ID, p.Slug, p.Title, p.StatementMD, p.Difficulty, p.OrderIndex, p.StarterCode, p.ReferenceCode, p.TimeLimitMS, p.MemoryLimitKB).
		Scan(&p.ID, &p.Slug, &p.Title, &p.StatementMD, &p.Difficulty, &p.OrderIndex, &p.StarterCode, &p.ReferenceCode,
			&p.TimeLimitMS, &p.MemoryLimitKB, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (s *Store) DeleteProblem(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM problems WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListTestCases(ctx context.Context, problemID int64, sampleOnly *bool) ([]TestCase, error) {
	q := `
		SELECT id, problem_id, input, expected_output, is_sample, points, order_index
		FROM test_cases WHERE problem_id=$1
	`
	args := []any{problemID}
	if sampleOnly != nil {
		q += ` AND is_sample=$2`
		args = append(args, *sampleOnly)
	}
	q += ` ORDER BY order_index, id`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TestCase, 0)
	for rows.Next() {
		var tc TestCase
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Input, &tc.ExpectedOutput, &tc.IsSample, &tc.Points, &tc.OrderIndex); err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	return out, rows.Err()
}

func (s *Store) ReplaceTestCases(ctx context.Context, problemID int64, cases []TestCase) ([]TestCase, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM test_cases WHERE problem_id=$1`, problemID); err != nil {
		return nil, err
	}
	out := make([]TestCase, 0, len(cases))
	for i, tc := range cases {
		tc.ProblemID = problemID
		if tc.OrderIndex == 0 {
			tc.OrderIndex = i
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO test_cases (problem_id, input, expected_output, is_sample, points, order_index)
			VALUES ($1,$2,$3,$4,$5,$6)
			RETURNING id, problem_id, input, expected_output, is_sample, points, order_index
		`, problemID, tc.Input, tc.ExpectedOutput, tc.IsSample, tc.Points, tc.OrderIndex).
			Scan(&tc.ID, &tc.ProblemID, &tc.Input, &tc.ExpectedOutput, &tc.IsSample, &tc.Points, &tc.OrderIndex)
		if err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

// --- Submissions ---

func (s *Store) CountSubmits(ctx context.Context, studentID, problemID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM submissions
		WHERE student_id=$1 AND problem_id=$2 AND kind='submit'
	`, studentID, problemID).Scan(&n)
	return n, err
}

func (s *Store) CreateSubmission(ctx context.Context, sub *Submission) (*Submission, error) {
	if sub.VerdictDetail == nil {
		sub.VerdictDetail = json.RawMessage("[]")
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO submissions (student_id, problem_id, kind, source, status, score, max_score, verdict_detail)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, student_id, problem_id, kind, source, status, score, max_score, verdict_detail, created_at
	`, sub.StudentID, sub.ProblemID, sub.Kind, sub.Source, sub.Status, sub.Score, sub.MaxScore, sub.VerdictDetail).
		Scan(&sub.ID, &sub.StudentID, &sub.ProblemID, &sub.Kind, &sub.Source, &sub.Status, &sub.Score, &sub.MaxScore, &sub.VerdictDetail, &sub.CreatedAt)
	return sub, err
}

func (s *Store) UpdateSubmissionResult(ctx context.Context, id int64, status string, score, maxScore int, detail json.RawMessage) (*Submission, error) {
	var sub Submission
	err := s.pool.QueryRow(ctx, `
		UPDATE submissions SET status=$2, score=$3, max_score=$4, verdict_detail=$5
		WHERE id=$1
		RETURNING id, student_id, problem_id, kind, source, status, score, max_score, verdict_detail, created_at
	`, id, status, score, maxScore, detail).
		Scan(&sub.ID, &sub.StudentID, &sub.ProblemID, &sub.Kind, &sub.Source, &sub.Status, &sub.Score, &sub.MaxScore, &sub.VerdictDetail, &sub.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &sub, err
}

func (s *Store) ListSubmissionsByStudent(ctx context.Context, studentID int64) ([]Submission, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, student_id, problem_id, kind, source, status, score, max_score, verdict_detail, created_at
		FROM submissions WHERE student_id=$1 ORDER BY created_at DESC
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSubs(rows)
}

func (s *Store) ListRecentSubmissions(ctx context.Context, limit int) ([]Submission, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.student_id, s.problem_id, s.kind, s.source, s.status, s.score, s.max_score, s.verdict_detail, s.created_at,
		       st.name, p.title
		FROM submissions s
		JOIN students st ON st.id = s.student_id
		JOIN problems p ON p.id = s.problem_id
		ORDER BY s.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Submission, 0)
	for rows.Next() {
		var sub Submission
		if err := rows.Scan(&sub.ID, &sub.StudentID, &sub.ProblemID, &sub.Kind, &sub.Source, &sub.Status,
			&sub.Score, &sub.MaxScore, &sub.VerdictDetail, &sub.CreatedAt, &sub.StudentName, &sub.ProblemTitle); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

func scanSubs(rows pgx.Rows) ([]Submission, error) {
	out := make([]Submission, 0)
	for rows.Next() {
		var sub Submission
		if err := rows.Scan(&sub.ID, &sub.StudentID, &sub.ProblemID, &sub.Kind, &sub.Source, &sub.Status,
			&sub.Score, &sub.MaxScore, &sub.VerdictDetail, &sub.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

func (s *Store) StudentProgress(ctx context.Context, studentID int64) ([]ProblemProgress, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id,
		       COALESCE((SELECT MAX(score) FROM submissions WHERE student_id=$1 AND problem_id=p.id AND kind='submit'), 0) AS best,
		       COALESCE((SELECT SUM(points) FROM test_cases WHERE problem_id=p.id AND is_sample=FALSE), 0) AS max_score,
		       COALESCE((SELECT COUNT(*) FROM submissions WHERE student_id=$1 AND problem_id=p.id AND kind='submit'), 0) AS submits
		FROM problems p
		ORDER BY p.order_index, p.id
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ProblemProgress, 0)
	for rows.Next() {
		var pp ProblemProgress
		if err := rows.Scan(&pp.ProblemID, &pp.BestScore, &pp.MaxScore, &pp.SubmitCount); err != nil {
			return nil, err
		}
		pp.SubmitsLeft = 3 - pp.SubmitCount
		if pp.SubmitsLeft < 0 {
			pp.SubmitsLeft = 0
		}
		out = append(out, pp)
	}
	return out, rows.Err()
}

func (s *Store) Leaderboard(ctx context.Context) ([]LeaderboardEntry, error) {
	students, err := s.ListStudents(ctx)
	if err != nil {
		return nil, err
	}
	problems, err := s.ListProblems(ctx)
	if err != nil {
		return nil, err
	}

	type bestKey struct{ sid, pid int64 }
	best := map[bestKey]struct {
		score int
		at    time.Time
	}{}

	rows, err := s.pool.Query(ctx, `
		SELECT student_id, problem_id, score, created_at
		FROM submissions
		WHERE kind='submit'
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sid, pid int64
		var score int
		var at time.Time
		if err := rows.Scan(&sid, &pid, &score, &at); err != nil {
			return nil, err
		}
		k := bestKey{sid, pid}
		cur, ok := best[k]
		if !ok || score > cur.score {
			best[k] = struct {
				score int
				at    time.Time
			}{score: score, at: at}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]LeaderboardEntry, 0, len(students))
	for _, st := range students {
		entry := LeaderboardEntry{
			StudentID:     st.ID,
			Name:          st.Name,
			SecretCode:    st.SecretCode,
			ProblemScores: map[string]int{},
			StartedAt:     st.StartedAt,
		}
		var last *time.Time
		for _, p := range problems {
			k := bestKey{st.ID, p.ID}
			if b, ok := best[k]; ok {
				entry.ProblemScores[fmt.Sprintf("%d", p.ID)] = b.score
				entry.TotalScore += b.score
				t := b.at
				if last == nil || t.After(*last) {
					last = &t
				}
			} else {
				entry.ProblemScores[fmt.Sprintf("%d", p.ID)] = 0
			}
		}
		entry.LastImproveAt = last
		out = append(out, entry)
	}

	// Sort by total desc, then earliest last improve
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			swap := false
			if out[j].TotalScore > out[i].TotalScore {
				swap = true
			} else if out[j].TotalScore == out[i].TotalScore {
				ai, aj := out[i].LastImproveAt, out[j].LastImproveAt
				if ai != nil && aj != nil && aj.Before(*ai) {
					swap = true
				} else if ai == nil && aj != nil {
					swap = true
				}
			}
			if swap {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

// --- Audit ---

func (s *Store) AddAudit(ctx context.Context, eventType string, payload any) (*AuditEvent, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var ev AuditEvent
	err = s.pool.QueryRow(ctx, `
		INSERT INTO audit_events (event_type, payload) VALUES ($1, $2)
		RETURNING id, event_type, payload, created_at
	`, eventType, b).Scan(&ev.ID, &ev.EventType, &ev.Payload, &ev.CreatedAt)
	return &ev, err
}

func (s *Store) ListAudit(ctx context.Context, limit int) ([]AuditEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, event_type, payload, created_at FROM audit_events
		ORDER BY created_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AuditEvent, 0)
	for rows.Next() {
		var ev AuditEvent
		if err := rows.Scan(&ev.ID, &ev.EventType, &ev.Payload, &ev.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

type DashboardProblem struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Difficulty  string  `json:"difficulty"`
	OrderIndex  int     `json:"order_index"`
	MaxScore    int     `json:"max_score"`
	Attempted   int     `json:"attempted"`
	Solved      int     `json:"solved"`
	Submits     int     `json:"submits"`
	Runs        int     `json:"runs"`
	AvgBestPct  float64 `json:"avg_best_pct"`
}

type Dashboard struct {
	Contest struct {
		Title           string `json:"title"`
		DurationMinutes int    `json:"duration_minutes"`
		IsActive        bool   `json:"is_active"`
	} `json:"contest"`
	Participants struct {
		SecretsTotal     int `json:"secrets_total"`
		SecretsClaimed   int `json:"secrets_claimed"`
		Registered       int `json:"registered"`
		NotStarted       int `json:"not_started"`
		InProgress       int `json:"in_progress"`
		Finished         int `json:"finished"`
		ActiveLast5Min   int `json:"active_last_5min"`
	} `json:"participants"`
	Submissions struct {
		Total          int     `json:"total"`
		Accepted       int     `json:"accepted"`
		Partial        int     `json:"partial"`
		Failed         int     `json:"failed"`
		Errored        int     `json:"errored"`
		Runs           int     `json:"runs"`
		Last15Min      int     `json:"last_15min"`
		RunsLast15Min  int     `json:"runs_last_15min"`
		AcceptanceRate float64 `json:"acceptance_rate"`
	} `json:"submissions"`
	Scoring struct {
		MaxPossible   int     `json:"max_possible"`
		AvgTotal      float64 `json:"avg_total"`
		MedianTotal   float64 `json:"median_total"`
		TopTotal      int     `json:"top_total"`
		FullSolvers   int     `json:"full_solvers"`
		ZeroScorers   int     `json:"zero_scorers"`
	} `json:"scoring"`
	Problems []DashboardProblem `json:"problems"`
	Top      []LeaderboardEntry `json:"top"`
}

func (s *Store) DashboardData(ctx context.Context, now time.Time) (*Dashboard, error) {
	d := &Dashboard{Problems: []DashboardProblem{}, Top: []LeaderboardEntry{}}

	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	d.Contest.Title = cfg.Title
	d.Contest.DurationMinutes = cfg.DurationMinutes
	d.Contest.IsActive = cfg.IsActive

	if err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM secrets),
			(SELECT COUNT(*) FROM secrets WHERE claimed_at IS NOT NULL),
			(SELECT COUNT(*) FROM students),
			(SELECT COUNT(*) FROM students WHERE started_at IS NULL),
			(SELECT COUNT(*) FROM students WHERE started_at IS NOT NULL AND started_at + make_interval(mins => $1::int) > $2::timestamptz),
			(SELECT COUNT(*) FROM students WHERE started_at IS NOT NULL AND started_at + make_interval(mins => $1::int) <= $2::timestamptz),
			(SELECT COUNT(DISTINCT student_id) FROM submissions WHERE created_at > $2::timestamptz - INTERVAL '5 minutes')
	`, cfg.DurationMinutes, now).Scan(
		&d.Participants.SecretsTotal, &d.Participants.SecretsClaimed, &d.Participants.Registered,
		&d.Participants.NotStarted, &d.Participants.InProgress, &d.Participants.Finished,
		&d.Participants.ActiveLast5Min,
	); err != nil {
		return nil, err
	}

	if err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE kind='submit'),
			COUNT(*) FILTER (WHERE kind='submit' AND status='Accepted'),
			COUNT(*) FILTER (WHERE kind='submit' AND status='Partial'),
			COUNT(*) FILTER (WHERE kind='submit' AND status NOT IN ('Accepted','Partial','error','pending')),
			COUNT(*) FILTER (WHERE kind='submit' AND status='error'),
			COUNT(*) FILTER (WHERE kind='run'),
			COUNT(*) FILTER (WHERE kind='submit' AND created_at > $1::timestamptz - INTERVAL '15 minutes'),
			COUNT(*) FILTER (WHERE kind='run' AND created_at > $1::timestamptz - INTERVAL '15 minutes')
		FROM submissions
	`, now).Scan(
		&d.Submissions.Total, &d.Submissions.Accepted, &d.Submissions.Partial, &d.Submissions.Failed,
		&d.Submissions.Errored, &d.Submissions.Runs, &d.Submissions.Last15Min, &d.Submissions.RunsLast15Min,
	); err != nil {
		return nil, err
	}
	if d.Submissions.Total > 0 {
		d.Submissions.AcceptanceRate = float64(d.Submissions.Accepted) * 100 / float64(d.Submissions.Total)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.title, p.difficulty, p.order_index,
		       COALESCE((SELECT SUM(points) FROM test_cases WHERE problem_id=p.id AND is_sample=FALSE), 0) AS max_score,
		       COALESCE((SELECT COUNT(DISTINCT student_id) FROM submissions WHERE problem_id=p.id AND kind='submit'), 0) AS attempted,
		       COALESCE((SELECT COUNT(*) FROM submissions WHERE problem_id=p.id AND kind='submit'), 0) AS submits,
		       COALESCE((SELECT COUNT(*) FROM submissions WHERE problem_id=p.id AND kind='run'), 0) AS runs,
		       COALESCE((
		           SELECT COUNT(*) FROM (
		               SELECT student_id, MAX(score) AS best, MAX(max_score) AS mx
		               FROM submissions WHERE problem_id=p.id AND kind='submit'
		               GROUP BY student_id
		           ) b WHERE b.mx > 0 AND b.best >= b.mx
		       ), 0) AS solved,
		       COALESCE((
		           SELECT AVG(CASE WHEN b.mx > 0 THEN b.best::float * 100 / b.mx ELSE 0 END) FROM (
		               SELECT student_id, MAX(score) AS best, MAX(max_score) AS mx
		               FROM submissions WHERE problem_id=p.id AND kind='submit'
		               GROUP BY student_id
		           ) b
		       ), 0) AS avg_best_pct
		FROM problems p
		ORDER BY p.order_index, p.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var dp DashboardProblem
		if err := rows.Scan(&dp.ID, &dp.Title, &dp.Difficulty, &dp.OrderIndex, &dp.MaxScore,
			&dp.Attempted, &dp.Submits, &dp.Runs, &dp.Solved, &dp.AvgBestPct); err != nil {
			return nil, err
		}
		d.Scoring.MaxPossible += dp.MaxScore
		d.Problems = append(d.Problems, dp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	lb, err := s.Leaderboard(ctx)
	if err != nil {
		return nil, err
	}
	if len(lb) > 0 {
		totals := make([]int, 0, len(lb))
		sum := 0
		for _, e := range lb {
			totals = append(totals, e.TotalScore)
			sum += e.TotalScore
			if d.Scoring.MaxPossible > 0 && e.TotalScore >= d.Scoring.MaxPossible {
				d.Scoring.FullSolvers++
			}
			if e.TotalScore == 0 {
				d.Scoring.ZeroScorers++
			}
		}
		d.Scoring.TopTotal = lb[0].TotalScore
		d.Scoring.AvgTotal = float64(sum) / float64(len(lb))
		// lb is sorted desc by total; median from that order
		mid := len(totals) / 2
		if len(totals)%2 == 1 {
			d.Scoring.MedianTotal = float64(totals[mid])
		} else {
			d.Scoring.MedianTotal = float64(totals[mid-1]+totals[mid]) / 2
		}
		if len(lb) > 5 {
			lb = lb[:5]
		}
		d.Top = lb
	}
	return d, nil
}

func (s *Store) Stats(ctx context.Context) (map[string]int, error) {
	out := map[string]int{}
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM students`).Scan(&n); err != nil {
		return nil, err
	}
	out["students"] = n
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM students WHERE started_at IS NOT NULL`).Scan(&n); err != nil {
		return nil, err
	}
	out["started"] = n
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM submissions WHERE kind='submit'`).Scan(&n); err != nil {
		return nil, err
	}
	out["submissions"] = n
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM problems`).Scan(&n); err != nil {
		return nil, err
	}
	out["problems"] = n
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM secrets`).Scan(&n); err != nil {
		return nil, err
	}
	out["secrets"] = n
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM secrets WHERE claimed_at IS NOT NULL`).Scan(&n); err != nil {
		return nil, err
	}
	out["claimed_secrets"] = n
	return out, nil
}
