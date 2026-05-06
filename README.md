# com.paca.checklist

First-party Paca plugin that adds checklist support to tasks.  
Each task can have multiple checklists, and each checklist can have multiple items that can be checked off, assigned to a project member, and reordered by position.

---

## Architecture

The plugin follows the standard two-part plugin structure:

```
checklist/
├── backend/   — Go WASM plugin (runs inside the API host)
└── frontend/  — React micro-frontend (Module Federation remote)
```

### Backend (`backend/`)

- Written in Go, compiled to `wasip1/wasm` for production.
- Registered as `com.paca.checklist` in the plugin registry.
- Owns its database schema (`plugin_data_com_paca_checklist`) and runs its own migration on startup.
- Listens on the `task.deleted` event to cascade-delete orphaned checklists.

### Frontend (`frontend/`)

- Vite + React + TanStack Query.
- Exposed as a Module Federation remote (`com_paca_checklist/ChecklistsSection`).
- Rendered by the host app via the `task.detail.section` extension point.
- Communicates with the backend through the standard plugin API path:  
  `GET|POST|PATCH|DELETE /api/v1/plugins/com.paca.checklist/projects/:projectId/tasks/:taskId/checklists/…`

---

## API Endpoints

All routes are prefixed with `/projects/:projectId` by the host.  
The caller must be an authenticated member of the project.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/tasks/:taskId/checklists` | List all checklists (with items) for a task |
| `POST` | `/tasks/:taskId/checklists` | Create a checklist |
| `PATCH` | `/tasks/:taskId/checklists/:checklistId` | Rename a checklist |
| `DELETE` | `/tasks/:taskId/checklists/:checklistId` | Delete a checklist and all its items |
| `POST` | `/tasks/:taskId/checklists/:checklistId/items` | Add an item to a checklist |
| `PATCH` | `/tasks/:taskId/checklists/:checklistId/items/:itemId` | Update an item (title / checked state / assignee) |
| `DELETE` | `/tasks/:taskId/checklists/:checklistId/items/:itemId` | Delete an item |

### Request / Response examples

**Create checklist** — `POST /tasks/:taskId/checklists`
```json
{ "title": "Definition of Done" }
```
```json
{
  "data": {
    "id": "a1b2c3d4-…",
    "task_id": "…",
    "title": "Definition of Done",
    "position": 0,
    "items": [],
    "created_at": "2026-05-05T10:00:00Z",
    "updated_at": "2026-05-05T10:00:00Z"
  }
}
```

**Toggle item** — `PATCH /tasks/:taskId/checklists/:checklistId/items/:itemId`
```json
{ "is_checked": true }
```

---

## Database Schema

Tables live in the `plugin_data_com_paca_checklist` schema and are created by `backend/migrations/0001_create_task_checklists.sql`.

```
task_checklists
  id          UUID PK
  task_id     UUID → public.tasks(id) ON DELETE CASCADE
  title       TEXT
  position    INTEGER
  created_by  UUID → public.project_members(id)
  created_at  TIMESTAMPTZ
  updated_at  TIMESTAMPTZ

task_checklist_items
  id           UUID PK
  checklist_id UUID → task_checklists(id) ON DELETE CASCADE
  title        TEXT
  is_checked   BOOLEAN
  assignee_id  UUID → public.project_members(id)
  position     INTEGER
  created_by   UUID → public.project_members(id)
  created_at   TIMESTAMPTZ
  updated_at   TIMESTAMPTZ
```

The legacy `public.task_checklists` and `public.task_checklist_items` tables are dropped by core migration `000005_migrate_checklists_to_plugin.sql`.

---

## Development

### Backend

```bash
cd backend

# Run tests
go test -v ./...

# Build WASM binary (requires Go 1.24+)
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o checklist.wasm .
```

### Frontend

```bash
cd frontend

# Install dependencies
bun install

# Development build (watch)
bun run dev

# Production build (outputs remoteEntry.js)
bun run build
```

The frontend uses the `@paca-ai/plugin-sdk-react` package (resolved locally from `plugins/sdk/frontend`).  
Shared singletons (`react`, `react-dom`, `@tanstack/react-query`) are provided by the host shell and must not be bundled.

---

## Extension Point

The frontend is mounted by the host at:

```
task.detail.section
```

Component props passed by the host:

| Prop | Type | Description |
|------|------|-------------|
| `projectId` | `string` | Current project ID |
| `taskId` | `string` | Current task ID |
| `canEdit` | `boolean` | Whether the caller has write permission |
