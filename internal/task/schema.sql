CREATE TYPE task_status AS ENUM ('INBOX', 'TO_DO', 'IN_PROGRESS', 'DONE');

CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL NOT NULL PRIMARY KEY,
    labels TEXT[] DEFAULT ARRAY[]::TEXT[],
    title TEXT NOT NULL,
    description TEXT default '',
    status task_status NOT NULL DEFAULT 'INBOX',
    due_date TIMESTAMPTZ default now() + INTERVAL '7 days',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);