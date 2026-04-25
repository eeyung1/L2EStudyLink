package db

import (
    "database/sql"
    "log"

    _ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() error {
    var err error
    DB, err = sql.Open("sqlite3", "./studylink.db")
    if err != nil {
        return err
    }

    if err = DB.Ping(); err != nil {
        return err
    }

    log.Println("SQLite database connected successfully")
    return nil
}

func CloseDB() {
    if DB != nil {
        DB.Close()
    }
}
