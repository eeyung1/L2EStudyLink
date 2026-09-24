package db

import (
    "context"
    "database/sql"
    _ "embed"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

//go:embed project_schema.sql
var projectSchema string

//go:embed review_schema.sql
var reviewSchema string

//go:embed login_schema.sql
var loginSchema string

func InitDB() error {
    // Get database URL from environment (Render sets this)
    dbURL := os.Getenv("DATABASE_URL")
    
    if dbURL == "" {
        // Fallback to local PostgreSQL for development
        dbURL = "host=localhost port=5432 user=studylink dbname=l2e_studylink sslmode=disable password=studylink123"
    }
    
    // The production connection passes through a transaction pooler. Avoid
    // server-side prepared statements and the extended protocol's unnamed
    // statement state, which the pooler does not preserve across requests.
    config, err := pgx.ParseConfig(dbURL)
    if err != nil {
        return fmt.Errorf("failed to parse database URL: %w", err)
    }
    config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
    DB = stdlib.OpenDB(*config)

    // Bound concurrent connections while retaining a small warm pool for
    // independent timetable and reflection reads.
    DB.SetMaxOpenConns(8)
    DB.SetMaxIdleConns(3)
    DB.SetConnMaxIdleTime(5 * time.Minute)
    DB.SetConnMaxLifetime(30 * time.Minute)

    if err = DB.Ping(); err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
    }

    // Existing accounts remain unsubscribed until they choose updates.
    if _, err = DB.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS marketing_opt_in_at TIMESTAMPTZ`); err != nil {
        return fmt.Errorf("failed to add product email preference: %w", err)
    }

    // Existing production databases may predate this table. Ensure the
    // recovery feature has its storage before the server accepts requests.
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    if _, err = DB.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS password_reset_tokens (
            id SERIAL PRIMARY KEY,
            user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            token VARCHAR(255) UNIQUE NOT NULL,
            expires_at TIMESTAMP NOT NULL,
            used BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `); err != nil {
        return fmt.Errorf("failed to create password reset table: %w", err)
    }
    if _, err = DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_password_reset_token ON password_reset_tokens(token)`); err != nil {
        return fmt.Errorf("failed to create password reset index: %w", err)
    }
    if _, err = DB.ExecContext(ctx, projectSchema); err != nil {
        return fmt.Errorf("failed to create project collaboration tables: %w", err)
    }
    if _, err = DB.ExecContext(ctx, reviewSchema); err != nil {
        return fmt.Errorf("failed to create reviews table: %w", err)
    }
    if _, err = DB.ExecContext(ctx, loginSchema); err != nil {
        return fmt.Errorf("failed to create login attempts table: %w", err)
    }
    if err = RepairAvailabilitySequence(ctx); err != nil {
        return fmt.Errorf("failed to repair availability IDs: %w", err)
    }

    log.Println("Database connected successfully")
    return nil
}

// RepairAvailabilitySequence handles databases where imported availability rows
// have IDs higher than the next value of the SERIAL sequence. The table lock
// keeps concurrent application instances from inserting during this repair.
func RepairAvailabilitySequence(ctx context.Context) error {
    var exists bool
    if err := DB.QueryRowContext(ctx, `SELECT to_regclass('availability') IS NOT NULL`).Scan(&exists); err != nil { return err }
    if !exists { return nil }
    tx, err := DB.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()
    if _, err = tx.ExecContext(ctx, `LOCK TABLE availability IN SHARE ROW EXCLUSIVE MODE`); err != nil { return err }
    _, err = tx.ExecContext(ctx, `SELECT setval(pg_get_serial_sequence('availability', 'id'), COALESCE(MAX(id), 0) + 1, false) FROM availability`)
    if err != nil { return err }
    return tx.Commit()
}

func CloseDB() {
    if DB != nil {
        DB.Close()
    }
}
