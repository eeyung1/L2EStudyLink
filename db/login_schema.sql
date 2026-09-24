CREATE TABLE IF NOT EXISTS login_attempts (
 subject_key TEXT PRIMARY KEY,
 attempts INT NOT NULL DEFAULT 0,
 window_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 locked_until TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_login_attempts_window ON login_attempts(window_started_at);
