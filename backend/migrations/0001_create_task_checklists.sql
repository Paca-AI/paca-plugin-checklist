-- 0001_create_task_checklists.sql
-- Creates the checklist tables in the plugin schema.
-- Run with search_path = plugin_data_com_paca_checklist, public.
--
-- NOTE: If the core migration (000005_migrate_checklists_to_plugin.sql) has
-- already moved the tables from the public schema, the IF NOT EXISTS guards
-- make this migration a safe no-op.

CREATE TABLE IF NOT EXISTS task_checklists (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id    UUID        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    title      TEXT        NOT NULL,
    position   INTEGER     NOT NULL DEFAULT 0,
    created_by UUID        REFERENCES project_members(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_checklists_task_id
    ON task_checklists (task_id);

CREATE TABLE IF NOT EXISTS task_checklist_items (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    checklist_id UUID        NOT NULL REFERENCES task_checklists(id) ON DELETE CASCADE,
    title        TEXT        NOT NULL,
    is_checked   BOOLEAN     NOT NULL DEFAULT FALSE,
    checked_by   UUID        REFERENCES project_members(id) ON DELETE SET NULL,
    checked_at   TIMESTAMPTZ,
    assignee_id  UUID        REFERENCES project_members(id) ON DELETE SET NULL,
    due_date     DATE,
    position     INTEGER     NOT NULL DEFAULT 0,
    created_by   UUID        REFERENCES project_members(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_checklist_items_checklist_id
    ON task_checklist_items (checklist_id, position);
