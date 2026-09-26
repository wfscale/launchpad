-- LaunchPad: схема БД продюсерского центра образовательных продуктов.
-- Логика такая: центр ищет и ведёт проекты (courses/products), у проекта
-- может быть несколько запусков (потоков), у запуска — задачи на команду,
-- к задачам команда оставляет комментарии.

CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL   PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL UNIQUE,
    role       TEXT        NOT NULL DEFAULT 'producer'
        CHECK (role IN ('producer', 'marketer', 'methodist', 'lead')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS projects (
    id          BIGSERIAL   PRIMARY KEY,
    title       TEXT        NOT NULL,
    description TEXT,
    status      TEXT        NOT NULL DEFAULT 'search'
        CHECK (status IN ('search', 'in_progress', 'launched', 'archived', 'cancelled')),
    owner_id    BIGINT      NOT NULL REFERENCES users(id),
    budget      NUMERIC(12, 2),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects(owner_id);

CREATE TABLE IF NOT EXISTS launches (
    id         BIGSERIAL   PRIMARY KEY,
    project_id BIGINT      NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title      TEXT        NOT NULL,
    status     TEXT        NOT NULL DEFAULT 'planned'
        CHECK (status IN ('planned', 'running', 'finished', 'cancelled')),
    start_date DATE,
    end_date   DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_launches_project_id ON launches(project_id);

CREATE TABLE IF NOT EXISTS tasks (
    id          BIGSERIAL   PRIMARY KEY,
    launch_id   BIGINT      NOT NULL REFERENCES launches(id) ON DELETE CASCADE,
    title       TEXT        NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'todo'
        CHECK (status IN ('todo', 'in_progress', 'done')),
    assignee_id BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    due_date    DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tasks_launch_id ON tasks(launch_id);
CREATE INDEX IF NOT EXISTS idx_tasks_assignee_id ON tasks(assignee_id);

CREATE TABLE IF NOT EXISTS comments (
    id         BIGSERIAL   PRIMARY KEY,
    task_id    BIGINT      NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id    BIGINT      NOT NULL REFERENCES users(id),
    text       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_comments_task_id ON comments(task_id);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
