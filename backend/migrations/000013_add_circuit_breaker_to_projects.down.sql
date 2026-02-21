ALTER TABLE projects
    DROP COLUMN IF EXISTS circuit_breaker_max,
    DROP COLUMN IF EXISTS circuit_breaker_active,
    DROP COLUMN IF EXISTS circuit_breaker_count;
