package handlers

import (
    "context"
    "database/sql"
    "github.com/gin-gonic/gin"
)

// Inspect the connected database, not a historical README or assumed migration version.
const reflectionRetentionSQL = `SELECT
    EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='activity_logs' AND column_name='activity_name' AND is_nullable='NO')
    AND EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='activity_logs' AND column_name='timetable_id' AND is_nullable='YES')
    AND EXISTS(SELECT 1 FROM pg_constraint c JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=ANY(c.conkey)
        WHERE c.conrelid='activity_logs'::regclass AND c.confrelid='timetable'::regclass AND c.contype='f' AND a.attname='timetable_id' AND c.confdeltype='n')
    AND NOT EXISTS(SELECT 1 FROM pg_constraint c JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=ANY(c.conkey)
        WHERE c.conrelid='activity_logs'::regclass AND c.confrelid='timetable'::regclass AND c.contype='f' AND a.attname='timetable_id' AND c.confdeltype<>'n')`

type retentionQuery interface { QueryRowContext(context.Context,string,...any) *sql.Row }
func reflectionRetentionReady(ctx context.Context, database retentionQuery) bool {
    var ready bool
    return database.QueryRowContext(ctx,reflectionRetentionSQL).Scan(&ready)==nil && ready
}

func GetReflectionRetention(c *gin.Context) {
    c.Header("Cache-Control","private, no-store")
    database:=c.MustGet("db").(*sql.DB)
    if !reflectionRetentionReady(c.Request.Context(),database){c.JSON(503,gin.H{"verified":false,"error":"Reflection retention could not be verified. Deletion is unavailable."});return}
    c.JSON(200,gin.H{"verified":true,"policy":"preserve_reflections","basis":"connected_database_constraints"})
}
