-- Image Playground asynchronous generation tasks.

CREATE TABLE IF NOT EXISTS image_playground_tasks (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id    BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    group_id      BIGINT REFERENCES groups(id) ON DELETE SET NULL,
    endpoint      VARCHAR(64) NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'queued',
    request_json  JSONB NOT NULL,
    result_json   JSONB,
    error_code    VARCHAR(64),
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ NOT NULL,
    canceled_at   TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT image_playground_tasks_endpoint_check CHECK (
        endpoint IN ('/v1/images/generations', '/v1/images/edits')
    ),
    CONSTRAINT image_playground_tasks_status_check CHECK (
        status IN ('queued', 'running', 'succeeded', 'failed', 'canceled', 'expired')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_id_user_id
    ON api_keys(id, user_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'image_playground_tasks_api_key_owner_fkey'
    ) THEN
        ALTER TABLE image_playground_tasks
            ADD CONSTRAINT image_playground_tasks_api_key_owner_fkey
            FOREIGN KEY (api_key_id, user_id)
            REFERENCES api_keys(id, user_id)
            ON DELETE CASCADE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_image_playground_tasks_user_created
    ON image_playground_tasks(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_image_playground_tasks_api_key_created
    ON image_playground_tasks(api_key_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_image_playground_tasks_status_created
    ON image_playground_tasks(status, created_at);

CREATE INDEX IF NOT EXISTS idx_image_playground_tasks_expires_at
    ON image_playground_tasks(expires_at);
