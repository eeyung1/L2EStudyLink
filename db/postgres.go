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
    // Build connection string
    connStr := fmt.Sprintf(
        "host=localhost port=5432 user=postgres dbname=l2e_studylink sslmode=disable password=",
    )

    var err error
    DB, err = sql.Open("postgres", connStr)
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }

    // Test connection
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