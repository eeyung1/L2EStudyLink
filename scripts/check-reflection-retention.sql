-- Read-only. Run against the deployment's own database, not a different project.
SELECT table_schema,column_name,is_nullable,column_default
FROM information_schema.columns
WHERE table_name='activity_logs' AND column_name IN ('timetable_id','activity_name','audio_url')
ORDER BY table_schema,column_name;
SELECT n.nspname AS schema_name,c.conname,pg_get_constraintdef(c.oid) AS definition,c.convalidated
FROM pg_constraint c JOIN pg_namespace n ON n.oid=c.connamespace
WHERE c.conrelid=to_regclass('activity_logs') AND c.contype='f';
SELECT schemaname,tablename,rowsecurity FROM pg_tables WHERE tablename='activity_logs';
