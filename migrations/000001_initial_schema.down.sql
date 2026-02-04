-- +migrate Down
DROP TRIGGER IF EXISTS update_job_executions_updated_at ON job_executions;
DROP TRIGGER IF EXISTS update_jobs_updated_at ON jobs;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS job_histories;
DROP TABLE IF EXISTS job_executions;
DROP TABLE IF EXISTS jobs;
