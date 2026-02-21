ALTER TABLE projects
    ADD COLUMN circuit_breaker_count  INT     NOT NULL DEFAULT 0,
    ADD COLUMN circuit_breaker_active BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN circuit_breaker_max    INT     NOT NULL DEFAULT 3;
