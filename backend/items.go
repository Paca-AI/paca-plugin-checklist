package main

import (
	plugin "github.com/Paca-AI/plugin-sdk"
	"github.com/google/uuid"
)

// ── Item route handlers ───────────────────────────────────────────────────────

// createItem handles POST /tasks/:taskId/checklists/:checklistId/items
func (p *checklistPlugin) createItem(req *plugin.Request, res *plugin.Response) {
	taskID := req.PathParam("taskId")
	checklistID := req.PathParam("checklistId")
	projectID := req.Caller.ProjectID

	if !p.taskBelongsToProject(taskID, projectID, res) {
		return
	}
	if !p.checklistOwnedByTask(checklistID, taskID, res) {
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

	// Determine next position
	posResult, err := p.db.Query(
		`SELECT COALESCE(MAX(position) + 1, 0) AS next_pos
		 FROM task_checklist_items WHERE checklist_id = $1`,
		checklistID,
	)
	if err != nil {
		p.log.Error("createItem position query: " + err.Error())
		res.Error(500, "failed to create item")
		return
	}
	nextPos := 0
	if len(posResult.Rows) > 0 {
		sc := newRowScanner(posResult.Columns, posResult.Rows[0])
		nextPos = sc.intVal("next_pos")
	}

	id := uuid.New().String()
	now := nowStr()
	_, err = p.db.Exec(
		`INSERT INTO task_checklist_items (id, checklist_id, title, is_checked, assignee_id, position, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, checklistID, b.Title, false, nil, nextPos, req.Caller.CallerID, now, now,
	)
	if err != nil {
		p.log.Error("createItem insert: " + err.Error())
		res.Error(500, "failed to create item")
		return
	}
	item := checklistItem{
		ID:          id,
		ChecklistID: checklistID,
		Title:       b.Title,
		IsChecked:   false,
		AssigneeID:  nil,
		Position:    nextPos,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	created(res, item)
}

// updateItem handles PATCH /tasks/:taskId/checklists/:checklistId/items/:itemId
// Supports updating title, is_checked, and assignee_id.
func (p *checklistPlugin) updateItem(req *plugin.Request, res *plugin.Response) {
	taskID := req.PathParam("taskId")
	checklistID := req.PathParam("checklistId")
	itemID := req.PathParam("itemId")
	projectID := req.Caller.ProjectID

	if !p.taskBelongsToProject(taskID, projectID, res) {
		return
	}

	type body struct {
		Title      *string `json:"title"`
		IsChecked  *bool   `json:"is_checked"`
		AssigneeID *string `json:"assignee_id"`
	}
	b, err := plugin.JSONBody[body](req)
	if err != nil {
		res.Error(400, "invalid request body")
		return
	}

	if b.Title == nil && b.IsChecked == nil && b.AssigneeID == nil {
		res.Error(400, "no fields to update")
		return
	}

	// Fetch current state so we can preserve unpatched fields.
	cur, curErr := p.db.Query(
		`SELECT id, checklist_id, title, is_checked, assignee_id, position, created_by, created_at, updated_at
		 FROM task_checklist_items WHERE id = $1`,
		itemID,
	)
	if curErr != nil {
		p.log.Error("updateItem fetch: " + curErr.Error())
		res.Error(500, "failed to update item")
		return
	}
	if len(cur.Rows) == 0 {
		res.Error(404, "item not found")
		return
	}
	sc := newRowScanner(cur.Columns, cur.Rows[0])

	now := nowStr()
	updTitle := sc.str("title")
	updChecked := sc.boolVal("is_checked")
	updAssignee := sc.strPtr("assignee_id")
	createdBy := sc.str("created_by")
	createdAt := sc.str("created_at")
	pos := sc.intVal("position")

	if b.Title != nil {
		updTitle = *b.Title
	}
	if b.IsChecked != nil {
		updChecked = *b.IsChecked
	}
	if b.AssigneeID != nil {
		updAssignee = b.AssigneeID
	}

	// Simulate UPDATE as DELETE + re-INSERT. task_checklist_items has no child
	// FK references, so this is safe in both tests and production.
	if _, err = p.db.Exec(
		`DELETE FROM task_checklist_items WHERE id = $1`, itemID,
	); err != nil {
		p.log.Error("updateItem delete: " + err.Error())
		res.Error(500, "failed to update item")
		return
	}
	if _, err = p.db.Exec(
		`INSERT INTO task_checklist_items (id, checklist_id, title, is_checked, assignee_id, position, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		itemID, checklistID, updTitle, updChecked, updAssignee, pos, createdBy, createdAt, now,
	); err != nil {
		p.log.Error("updateItem insert: " + err.Error())
		res.Error(500, "failed to update item")
		return
	}

	item := checklistItem{
		ID:          itemID,
		ChecklistID: checklistID,
		Title:       updTitle,
		IsChecked:   updChecked,
		AssigneeID:  updAssignee,
		Position:    pos,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}
	ok(res, item)
}

// deleteItem handles DELETE /tasks/:taskId/checklists/:checklistId/items/:itemId
func (p *checklistPlugin) deleteItem(req *plugin.Request, res *plugin.Response) {
	taskID := req.PathParam("taskId")
	checklistID := req.PathParam("checklistId")
	itemID := req.PathParam("itemId")
	projectID := req.Caller.ProjectID

	if !p.taskBelongsToProject(taskID, projectID, res) {
		return
	}

	affected, err := p.db.Exec(
		`DELETE FROM task_checklist_items WHERE id = $1 AND checklist_id = $2`,
		itemID, checklistID,
	)
	if err != nil {
		p.log.Error("deleteItem: " + err.Error())
		res.Error(500, "failed to delete item")
		return
	}
	if affected == 0 {
		res.Error(404, "item not found")
		return
	}
	res.NoContent()
}
