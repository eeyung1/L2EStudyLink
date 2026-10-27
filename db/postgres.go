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

//go:embed learning_schema.sql
var learningSchema string

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

	// Public portfolios are opt-in: no handle, hidden by default.
	if _, err = DB.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS public_handle VARCHAR(30)`); err != nil {
		return fmt.Errorf("failed to add public portfolio handle: %w", err)
	}
	if _, err = DB.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS portfolio_public BOOLEAN NOT NULL DEFAULT FALSE`); err != nil {
		return fmt.Errorf("failed to add portfolio visibility flag: %w", err)
	}
	if _, err = DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_public_handle ON users (lower(public_handle))`); err != nil {
		return fmt.Errorf("failed to add public handle uniqueness index: %w", err)
	}

	// Ensure audio_url exists on activity_logs for voice reflections if the table is present
	if _, err = DB.Exec(`
        DO $$ BEGIN
            IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='activity_logs') THEN
                ALTER TABLE activity_logs ADD COLUMN IF NOT EXISTS audio_url TEXT;
            END IF;
        END $$;
    `); err != nil {
		log.Printf("Warning: failed to add audio_url to activity_logs: %v", err)
	}

	// Preserve historical reflections before serving timetable deletions.
	reflectionCtx, reflectionCancel := context.WithTimeout(context.Background(), 15*time.Second)
	err = MigrateReflectionHistory(reflectionCtx, DB)
	reflectionCancel()
	if err != nil {
		return fmt.Errorf("failed to preserve reflection history: %w", err)
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
	if _, err = DB.ExecContext(ctx, learningSchema); err != nil {
		return fmt.Errorf("failed to create learning hub tables: %w", err)
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
	if err := DB.QueryRowContext(ctx, `SELECT to_regclass('availability') IS NOT NULL`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return nil
	}
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `LOCK TABLE availability IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `SELECT setval(pg_get_serial_sequence('availability', 'id'), COALESCE(MAX(id), 0) + 1, false) FROM availability`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

// MigrateReflectionHistory upgrades existing PostgreSQL databases atomically.
// No-op when activity_logs is absent; fresh databases use schema.sql instead.
// The table lock serializes backfill and constraint replacement with writers
// and concurrent startup migrations. Previously deleted rows cannot be restored.
func MigrateReflectionHistory(ctx context.Context, database *sql.DB) error {
	_, err := database.ExecContext(ctx, `
        DO $$
        DECLARE
            old_fk RECORD;
            timetable_column SMALLINT;
        BEGIN
            IF to_regclass('activity_logs') IS NULL THEN
                RETURN;
            END IF;
            LOCK TABLE activity_logs IN ACCESS EXCLUSIVE MODE;
            ALTER TABLE activity_logs ADD COLUMN IF NOT EXISTS activity_name VARCHAR(255);
            UPDATE activity_logs al SET activity_name = t.activity
            FROM timetable t WHERE al.timetable_id = t.id AND al.activity_name IS NULL;
            UPDATE activity_logs SET activity_name = '' WHERE activity_name IS NULL;
            ALTER TABLE activity_logs ALTER COLUMN activity_name SET DEFAULT '';
            ALTER TABLE activity_logs ALTER COLUMN activity_name SET NOT NULL;
            ALTER TABLE activity_logs ALTER COLUMN timetable_id DROP NOT NULL;

            SELECT attnum INTO timetable_column FROM pg_attribute
            WHERE attrelid = 'activity_logs'::regclass AND attname = 'timetable_id';
            FOR old_fk IN
                SELECT conname FROM pg_constraint
                WHERE conrelid = 'activity_logs'::regclass AND contype = 'f'
                  AND conkey = ARRAY[timetable_column]
                  AND confrelid = 'timetable'::regclass AND confdeltype <> 'n'
            LOOP
                EXECUTE format('ALTER TABLE activity_logs DROP CONSTRAINT %I', old_fk.conname);
            END LOOP;
            IF NOT EXISTS (
                SELECT 1 FROM pg_constraint
                WHERE conrelid = 'activity_logs'::regclass AND contype = 'f'
                  AND conkey = ARRAY[timetable_column]
                  AND confrelid = 'timetable'::regclass AND confdeltype = 'n'
            ) THEN
                ALTER TABLE activity_logs ADD CONSTRAINT activity_logs_timetable_id_fkey
                    FOREIGN KEY (timetable_id) REFERENCES timetable(id) ON DELETE SET NULL;
            END IF;
        END $$;
    `)
	return err
}
