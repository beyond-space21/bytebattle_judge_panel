-- Admins
CREATE TABLE IF NOT EXISTS admins (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Single-row contest configuration
CREATE TABLE IF NOT EXISTS contest_config (
    id               INT PRIMARY KEY CHECK (id = 1),
    title            TEXT NOT NULL DEFAULT 'Byte Battle',
    duration_minutes INT NOT NULL DEFAULT 120 CHECK (duration_minutes > 0),
    is_active        BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO contest_config (id, title, duration_minutes, is_active)
VALUES (1, 'Byte Battle', 120, FALSE)
ON CONFLICT (id) DO NOTHING;

-- One-time secret IDs for students
CREATE TABLE IF NOT EXISTS secrets (
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(6) NOT NULL UNIQUE,
    claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS students (
    id         BIGSERIAL PRIMARY KEY,
    secret_id  BIGINT NOT NULL UNIQUE REFERENCES secrets(id),
    name       TEXT NOT NULL,
    started_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS student_sessions (
    token      TEXT PRIMARY KEY,
    student_id BIGINT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS problems (
    id               BIGSERIAL PRIMARY KEY,
    slug             TEXT NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    statement_md     TEXT NOT NULL DEFAULT '',
    difficulty       TEXT NOT NULL DEFAULT 'medium',
    order_index      INT NOT NULL DEFAULT 0,
    starter_code     TEXT NOT NULL DEFAULT 'def solve():\n    pass\n',
    reference_code   TEXT NOT NULL DEFAULT '',
    time_limit_ms    INT NOT NULL DEFAULT 2000,
    memory_limit_kb  INT NOT NULL DEFAULT 256000,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS test_cases (
    id              BIGSERIAL PRIMARY KEY,
    problem_id      BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    input           TEXT NOT NULL DEFAULT '',
    expected_output TEXT NOT NULL DEFAULT '',
    is_sample       BOOLEAN NOT NULL DEFAULT FALSE,
    points          INT NOT NULL DEFAULT 1 CHECK (points >= 0),
    order_index     INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_test_cases_problem ON test_cases(problem_id);

CREATE TABLE IF NOT EXISTS submissions (
    id             BIGSERIAL PRIMARY KEY,
    student_id     BIGINT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    problem_id     BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    kind           TEXT NOT NULL CHECK (kind IN ('run', 'submit')),
    source         TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'pending',
    score          INT NOT NULL DEFAULT 0,
    max_score      INT NOT NULL DEFAULT 0,
    verdict_detail JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_submissions_student ON submissions(student_id);
CREATE INDEX IF NOT EXISTS idx_submissions_problem ON submissions(problem_id);
CREATE INDEX IF NOT EXISTS idx_submissions_kind ON submissions(student_id, problem_id, kind);

CREATE TABLE IF NOT EXISTS audit_events (
    id         BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    payload    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_created ON audit_events(created_at DESC);
