package db

import (
    "context"
    "database/sql"
    _ "embed"
    "fmt"
    "log"
    "os"
    "time"

    _ "github.com/lib/pq"
)

var DB *sql.DB

//go:embed project_schema.sql
var projectSchema string

func InitDB() error {
    // Get database URL from environment (Render sets this)
    dbURL := os.Getenv("DATABASE_URL")
    
    if dbURL == "" {
        // Fallback to local PostgreSQL for development
        dbURL = "host=localhost port=5432 user=studylink dbname=l2e_studylink sslmode=disable password=studylink123"
    }
    
    var err error
    DB, err = sql.Open("postgres", dbURL)
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }

    if err = DB.Ping(); err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
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

    log.Println("Database connected successfully")
    return nil
}

func CloseDB() {
    if DB != nil {
        DB.Close()
    }
}
