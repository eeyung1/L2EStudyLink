-- App-owned reflection access, matching schema.sql and MigrateReflectionHistory.
ALTER TABLE public.activity_logs ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON public.activity_logs FROM anon, authenticated;
