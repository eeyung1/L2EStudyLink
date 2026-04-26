package db

import (
    "database/sql"
    "fmt"
    "log"
    "os"

    _ "github.com/lib/pq"
)

var DB *sql.DB

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

    log.Println("Database connected successfully")
    return nil
}

func CloseDB() {
    if DB != nil {
        DB.Close()
    }
}
