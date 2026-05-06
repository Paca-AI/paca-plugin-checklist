package main

import (
	"fmt"

	plugin "github.com/Paca-AI/plugin-sdk"
	"github.com/google/uuid"
)

// ── Domain types ──────────────────────────────────────────────────────────────

type checklistItem struct {
	ID          string  `json:"id"`
	ChecklistID string  `json:"checklist_id"`
	Title       string  `json:"title"`
	IsChecked   bool    `json:"is_checked"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
	Position    int     `json:"position"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type checklist struct {
	ID        string          `json:"id"`
	TaskID    string          `json:"task_id"`
	Title     string          `json:"title"`
	Position  int             `json:"position"`
	Items     []checklistItem `json:"items"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

// ── Row scanner helper ────────────────────────────────────────────────────────

type scanner struct {
	idx map[string]int
	row []any
}

func newRowScanner(cols []string, row []any) *scanner {
	idx := make(map[string]int, len(cols))
	for i, c := range cols {
		idx[c] = i
	}
	return &scanner{idx: idx, row: row}
}

func (s *scanner) str(col string) string {
	i, ok := s.idx[col]
	if !ok || i >= len(s.row) || s.row[i] == nil {
		return ""
	}
	if v, ok := s.row[i].(string); ok {
		return v
	}
	return fmt.Sprintf("%v", s.row[i])
}

func (s *scanner) strPtr(col string) *string {
	i, ok := s.idx[col]
	if !ok || i >= len(s.row) || s.row[i] == nil {
		return nil
	}
	v := s.str(col)
	return &v
}

func (s *scanner) boolVal(col string) bool {
	i, ok := s.idx[col]
	if !ok || i >= len(s.row) || s.row[i] == nil {
		return false
	}
	if v, ok := s.row[i].(bool); ok {
		return v
	}
	return false
}

func (s *scanner) intVal(col string) int {
	i, ok := s.idx[col]
	if !ok || i >= len(s.row) || s.row[i] == nil {
		return 0
	}
	if v, ok := s.row[i].(float64); ok {
		return int(v)
	}
	return 0
}

// ── Checklist route handlers ──────────────────────────────────────────────────

// listChecklists handles GET /tasks/:taskId/checklists
// Returns all checklists for the task with their items embedded.
func (p *checklistPlugin) listChecklists(req *plugin.Request, res *plugin.Response) {
	taskID := req.PathParam("taskId")
	projectID := req.Caller.ProjectID

	if !p.taskBelongsToProject(taskID, projectID, res) {
		return
	}

	cls, err := p.fetchChecklistsForTask(taskID)
	if err != nil {
		p.log.Error("listChecklists: " + err.Error())
		res.Error(500, "failed to list checklists")
		return
	}
	ok(res, cls)
}

// createChecklist handles POST /tasks/:taskId/checklists
func (p *checklistPlugin) createChecklist(req *plugin.Request, res *plugin.Response) {
	taskID := req.PathParam("taskId")
	projectID := req.Caller.ProjectID

	if !p.taskBelongsToProject(taskID, projectID, res) {
		return
	}

	type body struct {
		Title string `json:"title"`
	}
	b, err := plugin.JSONBody[body](req)
	if err != nil || b.Title == "" {
		res.Error(400, "title is required")
		return
	}

	result, err := p.db.Query(
		`SELECT COALESCE(MAX(position) + 1, 0) AS next_pos
		 FROM task_checklists WHERE task_id = $1`,
		taskID,
	)
	if err != nil {
		p.log.Error("createChecklist position query: " + err.Error())
		res.Error(500, "failed to create checklist")
		return
	}
	nextPos := 0
	if len(result.Rows) > 0 {
		sc := newRowScanner(result.Columns, result.Rows[0])
		nextPos = sc.intVal("next_pos")
	}

	id := uuid.New().String()
	now := nowStr()
	_, err = p.db.Exec(
		`INSERT INTO task_checklists (id, task_id, title, position, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, taskID, b.Title, nextPos, req.Caller.CallerID, now, now,
	)
	if err != nil {
		p.log.Error("createChecklist insert: " + err.Error())
		res.Error(500, "failed to create checklist")
		return
	}
	cl := checklist{
		ID:        id,
		TaskID:    taskID,
		Title:     b.Title,
		Position:  nextPos,
		Items:     []checklistItem{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	created(res, cl)
}

// updateChecklist handles PATCH /tasks/:taskId/checklists/:checklistId
func (p *checklistPlugin) updateChecklist(req *plugin.Request, res *plugin.Response) {
	taskID := req.PathParam("taskId")
	checklistID := req.PathParam("checklistId")
	projectID := req.Caller.ProjectID

	if !p.taskBelongsToProject(taskID, projectID, res) {
		return
	}

	type body struct {
		Title *string `json:"title"`
	}
	b, err := plugin.JSONBody[body](req)
	if err != nil {
		res.Error(400, "invalid request body")
		return
	}
	if b.Title == nil || *b.Title == "" {
		res.Error(400, "title is required")
		return
	}

	current, err := p.db.Query(
		`SELECT id, task_id, position, created_at
		 FROM task_checklists
		 WHERE id = $1 AND task_id = $2`,
		checklistID, taskID,
	)
	if err != nil {
		p.log.Error("updateChecklist fetch: " + err.Error())
		res.Error(500, "failed to update checklist")
		return
	}
	if len(current.Rows) == 0 {
		res.Error(404, "checklist not found")
		return
	}
	sc := newRowScanner(current.Columns, current.Rows[0])
	now := nowStr()
	affected, err := p.db.Exec(
		`UPDATE task_checklists
		 SET title = $1, updated_at = $2
		 WHERE id = $3 AND task_id = $4`,
		*b.Title, now, checklistID, taskID,
	)
	if err != nil {
		p.log.Error("updateChecklist update: " + err.Error())
		res.Error(500, "failed to update checklist")
		return
	}
	if affected == 0 {
		res.Error(404, "checklist not found")
		return
	}
	items, _ := p.fetchItemsForChecklist(checklistID)
	cl := checklist{
		ID:        checklistID,
		TaskID:    sc.str("task_id"),
		Title:     *b.Title,
		Position:  sc.intVal("position"),
		Items:     items,
		CreatedAt: sc.str("created_at"),
		UpdatedAt: now,
	}
	ok(res, cl)
}

// deleteChecklist handles DELETE /tasks/:taskId/checklists/:checklistId
func (p *checklistPlugin) deleteChecklist(req *plugin.Request, res *plugin.Response) {
	taskID := req.PathParam("taskId")
	checklistID := req.PathParam("checklistId")
	projectID := req.Caller.ProjectID

	if !p.taskBelongsToProject(taskID, projectID, res) {
		return
	}

	affected, err := p.db.Exec(
		`DELETE FROM task_checklists WHERE id = $1 AND task_id = $2`,
		checklistID, taskID,
	)
	if err != nil {
		p.log.Error("deleteChecklist: " + err.Error())
		res.Error(500, "failed to delete checklist")
		return
	}
	if affected == 0 {
		res.Error(404, "checklist not found")
		return
	}
	res.NoContent()
}

// ── Task deleted event ────────────────────────────────────────────────────────

// handleTaskDeleted handles the task.deleted event.
// The task_checklists table has ON DELETE CASCADE from tasks, but this
// handler ensures any remaining data is cleaned up if the cascade fails.
func (p *checklistPlugin) handleTaskDeleted(evt *plugin.Event) {
	type payload struct {
		TaskID string `json:"task_id"`
	}
	ev, err := plugin.JSONPayload[payload](evt)
	if err != nil || ev.TaskID == "" {
		p.log.Warn("task.deleted: invalid payload")
		return
	}
	if _, err := p.db.Exec(
		`DELETE FROM task_checklists WHERE task_id = $1`, ev.TaskID,
	); err != nil {
		p.log.Error("task.deleted cleanup: " + err.Error())
	}
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// taskBelongsToProject validates that the given task exists in the project.
// Sets a 404 error on res and returns false when validation fails.
func (p *checklistPlugin) taskBelongsToProject(taskID, projectID string, res *plugin.Response) bool {
	result, err := p.db.Query(
		`SELECT id FROM tasks
		 WHERE id = $1 AND project_id = $2 AND deleted_at IS NULL`,
		taskID, projectID,
	)
	if err != nil {
		p.log.Error("taskBelongsToProject: " + err.Error())
		res.Error(500, "internal error")
		return false
	}
	if len(result.Rows) == 0 {
		res.Error(404, "task not found")
		return false
	}
	return true
}

// fetchChecklistsForTask returns all checklists for a task with items embedded.
func (p *checklistPlugin) fetchChecklistsForTask(taskID string) ([]checklist, error) {
	clResult, err := p.db.Query(
		`SELECT id, task_id, title, position, created_at, updated_at
		 FROM task_checklists
		 WHERE task_id = $1
		 ORDER BY position, created_at`,
		taskID,
	)
	if err != nil {
		return nil, err
	}

	checklists := make([]checklist, 0, len(clResult.Rows))
	for _, row := range clResult.Rows {
		sc := newRowScanner(clResult.Columns, row)
		id := sc.str("id")
		items, err := p.fetchItemsForChecklist(id)
		if err != nil {
			return nil, err
		}
		checklists = append(checklists, checklist{
			ID:        id,
			TaskID:    sc.str("task_id"),
			Title:     sc.str("title"),
			Position:  sc.intVal("position"),
			Items:     items,
			CreatedAt: sc.str("created_at"),
			UpdatedAt: sc.str("updated_at"),
		})
	}
	return checklists, nil
}

// fetchItemsForChecklist returns all items for a checklist ordered by position.
func (p *checklistPlugin) fetchItemsForChecklist(checklistID string) ([]checklistItem, error) {
	result, err := p.db.Query(
		`SELECT id, checklist_id, title, is_checked, assignee_id, position, created_at, updated_at
		 FROM task_checklist_items
		 WHERE checklist_id = $1
		 ORDER BY position, created_at`,
		checklistID,
	)
	if err != nil {
		return nil, err
	}

	items := make([]checklistItem, 0, len(result.Rows))
	for _, row := range result.Rows {
		sc := newRowScanner(result.Columns, row)
		items = append(items, checklistItem{
			ID:          sc.str("id"),
			ChecklistID: sc.str("checklist_id"),
			Title:       sc.str("title"),
			IsChecked:   sc.boolVal("is_checked"),
			AssigneeID:  sc.strPtr("assignee_id"),
			Position:    sc.intVal("position"),
			CreatedAt:   sc.str("created_at"),
			UpdatedAt:   sc.str("updated_at"),
		})
	}
	return items, nil
}

// checklistOwnedByTask verifies that the checklist belongs to the given task.
func (p *checklistPlugin) checklistOwnedByTask(checklistID, taskID string, res *plugin.Response) bool {
	result, err := p.db.Query(
		`SELECT id FROM task_checklists WHERE id = $1 AND task_id = $2`,
		checklistID, taskID,
	)
	if err != nil {
		p.log.Error("checklistOwnedByTask: " + err.Error())
		res.Error(500, "internal error")
		return false
	}
	if len(result.Rows) == 0 {
		res.Error(404, "checklist not found")
		return false
	}
	return true
}
