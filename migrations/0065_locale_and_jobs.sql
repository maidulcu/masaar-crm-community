-- +goose Up
-- Country rules per company (see internal/jurisdiction). Existing companies are UAE.
ALTER TABLE company_settings
    ADD COLUMN IF NOT EXISTS country  CHAR(2)     NOT NULL DEFAULT 'AE',
    ADD COLUMN IF NOT EXISTS region   VARCHAR(10) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS currency CHAR(3)     NOT NULL DEFAULT 'AED',
    ADD COLUMN IF NOT EXISTS timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Dubai';

-- Background jobs: one row per job. A replica runs a job only after claiming it with a
-- conditional UPDATE, so each job runs once per interval however many replicas are up.
CREATE TABLE job_schedule (
    job_name        TEXT        PRIMARY KEY,
    last_started_at TIMESTAMPTZ NOT NULL DEFAULT 'epoch',
    locked_until    TIMESTAMPTZ NOT NULL DEFAULT 'epoch'
);

-- Run log, per company. Only runs that did something or failed are recorded.
CREATE TABLE job_runs (
    id          BIGSERIAL   PRIMARY KEY,
    job_name    TEXT        NOT NULL,
    company_id  UUID        REFERENCES companies(id) ON DELETE CASCADE,
    started_at  TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    status      TEXT        NOT NULL CHECK (status IN ('ok','error')),
    items       INTEGER     NOT NULL DEFAULT 0,
    error       TEXT        NOT NULL DEFAULT ''
);
CREATE INDEX idx_job_runs_company_started ON job_runs(company_id, started_at DESC);
CREATE INDEX idx_job_runs_started ON job_runs(started_at);

-- +goose Down
DROP TABLE IF EXISTS job_runs;
DROP TABLE IF EXISTS job_schedule;
ALTER TABLE company_settings
    DROP COLUMN IF EXISTS timezone,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS region,
    DROP COLUMN IF EXISTS country;
