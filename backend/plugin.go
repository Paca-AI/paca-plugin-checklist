// Package main implements the com.paca.checklist backend WASM plugin.
//
// It provides CRUD routes for checklists and checklist items scoped to tasks,
// and handles the task.deleted event to cascade-delete orphaned data.
package main

import (
	"time"

	plugin "github.com/Paca-AI/plugin-sdk"
)

// nowStr returns the current UTC time as an RFC3339Nano string.
func nowStr() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// checklistPlugin implements plugin.Plugin.
type checklistPlugin struct {
	db  *plugin.DB
	log *plugin.Logger
}

// Init registers all routes and event handlers on the provided context.
func (p *checklistPlugin) Init(ctx *plugin.Context) error {
	p.db = ctx.DB()
	p.log = ctx.Log()

	// Event handlers
	ctx.On("task.deleted", p.handleTaskDeleted)

	// Checklist CRUD
	ctx.Route("GET", "/tasks/:taskId/checklists", p.listChecklists)
	ctx.Route("POST", "/tasks/:taskId/checklists", p.createChecklist)
	ctx.Route("PATCH", "/tasks/:taskId/checklists/:checklistId", p.updateChecklist)
	ctx.Route("DELETE", "/tasks/:taskId/checklists/:checklistId", p.deleteChecklist)

	// Checklist item CRUD
	ctx.Route("POST", "/tasks/:taskId/checklists/:checklistId/items", p.createItem)
	ctx.Route("PATCH", "/tasks/:taskId/checklists/:checklistId/items/:itemId", p.updateItem)
	ctx.Route("DELETE", "/tasks/:taskId/checklists/:checklistId/items/:itemId", p.deleteItem)

	return nil
}

// Shutdown is a no-op for this plugin.
func (p *checklistPlugin) Shutdown() {}

// envelope wraps successful API responses to match the host's standard format.
type envelope struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

func ok(res *plugin.Response, data any) {
	res.JSON(200, envelope{Success: true, Data: data})
}

func created(res *plugin.Response, data any) {
	res.JSON(201, envelope{Success: true, Data: data})
}
