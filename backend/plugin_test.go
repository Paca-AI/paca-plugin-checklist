package main

import (
	"encoding/json"
	"testing"

	plugin "github.com/Paca-AI/plugin-sdk"
	"github.com/Paca-AI/plugin-sdk/plugintest"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

const (
	testProjectID = "project-1"
	testTaskID    = "task-1"
)

func setupPlugin(t *testing.T) *plugintest.Context {
	t.Helper()
	tc := plugintest.NewContext(t)

	// Seed the tasks table (public schema, accessed via search_path)
	tc.DB.SeedRows("tasks", []string{"id", "project_id", "deleted_at"}, [][]any{
		{testTaskID, testProjectID, nil},
	})
	// Seed empty plugin tables so INSERT/SELECT/DELETE can operate on them.
	tc.DB.SeedRows("task_checklists",
		[]string{"id", "task_id", "title", "position", "created_by", "created_at", "updated_at"},
		nil)
	tc.DB.SeedRows("task_checklist_items",
		[]string{"id", "checklist_id", "title", "is_checked", "assignee_id", "position", "created_by", "created_at", "updated_at"},
		nil)

	var p checklistPlugin
	if err := p.Init(tc.PluginContext()); err != nil {
		t.Fatal("Init failed:", err)
	}
	return tc
}

func callerReq() plugintest.Request {
	return plugintest.Request{
		Caller: plugin.CallerIdentity{
			ProjectID:  testProjectID,
			CallerID:   "member-1",
			CallerRole: "PROJECT_MEMBER",
		},
		PathParams: map[string]string{},
	}
}

func withPathParams(req plugintest.Request, params map[string]string) plugintest.Request {
	m := make(map[string]string, len(req.PathParams)+len(params))
	for k, v := range req.PathParams {
		m[k] = v
	}
	for k, v := range params {
		m[k] = v
	}
	req.PathParams = m
	return req
}

// ── Checklist CRUD tests ──────────────────────────────────────────────────────

func TestListChecklists_Empty(t *testing.T) {
	tc := setupPlugin(t)
	res := tc.Call("GET", "/tasks/:taskId/checklists",
		withPathParams(callerReq(), map[string]string{"taskId": testTaskID}))

	if res.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", res.StatusCode, res.BodyString())
	}
	var env struct {
		Success bool        `json:"success"`
		Data    []checklist `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	if !env.Success || len(env.Data) != 0 {
		t.Fatalf("expected empty list, got %+v", env.Data)
	}
}

func TestCreateAndListChecklist(t *testing.T) {
	tc := setupPlugin(t)

	// Create
	res := tc.Call("POST", "/tasks/:taskId/checklists",
		withPathParams(callerReq(), map[string]string{"taskId": testTaskID}).
			WithJSONBody(map[string]string{"title": "My Checklist"}))

	if res.StatusCode != 201 {
		t.Fatalf("expected 201, got %d: %s", res.StatusCode, res.BodyString())
	}
	var createEnv struct {
		Data checklist `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &createEnv); err != nil {
		t.Fatal(err)
	}
	if createEnv.Data.Title != "My Checklist" {
		t.Fatalf("unexpected title: %s", createEnv.Data.Title)
	}

	// List
	listRes := tc.Call("GET", "/tasks/:taskId/checklists",
		withPathParams(callerReq(), map[string]string{"taskId": testTaskID}))

	if listRes.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", listRes.StatusCode, listRes.BodyString())
	}
	var listEnv struct {
		Data []checklist `json:"data"`
	}
	if err := json.Unmarshal(listRes.Body, &listEnv); err != nil {
		t.Fatal(err)
	}
	if len(listEnv.Data) != 1 || listEnv.Data[0].Title != "My Checklist" {
		t.Fatalf("expected 1 checklist, got %+v", listEnv.Data)
	}
}

func TestCreateChecklist_MissingTitle(t *testing.T) {
	tc := setupPlugin(t)

	res := tc.Call("POST", "/tasks/:taskId/checklists",
		withPathParams(callerReq(), map[string]string{"taskId": testTaskID}).
			WithJSONBody(map[string]string{"title": ""}))

	if res.StatusCode != 400 {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestDeleteChecklist(t *testing.T) {
	tc := setupPlugin(t)

	createRes := tc.Call("POST", "/tasks/:taskId/checklists",
		withPathParams(callerReq(), map[string]string{"taskId": testTaskID}).
			WithJSONBody(map[string]string{"title": "To Delete"}))
	var env struct {
		Data checklist `json:"data"`
	}
	_ = json.Unmarshal(createRes.Body, &env)

	delRes := tc.Call("DELETE", "/tasks/:taskId/checklists/:checklistId",
		withPathParams(callerReq(), map[string]string{
			"taskId":      testTaskID,
			"checklistId": env.Data.ID,
		}))

	if delRes.StatusCode != 204 {
		t.Fatalf("expected 204, got %d: %s", delRes.StatusCode, delRes.BodyString())
	}
}

func TestUpdateChecklist(t *testing.T) {
	tc := setupPlugin(t)

	createRes := tc.Call("POST", "/tasks/:taskId/checklists",
		withPathParams(callerReq(), map[string]string{"taskId": testTaskID}).
			WithJSONBody(map[string]string{"title": "Original"}))
	if createRes.StatusCode != 201 {
		t.Fatalf("expected 201, got %d: %s", createRes.StatusCode, createRes.BodyString())
	}

	var createEnv struct {
		Data checklist `json:"data"`
	}
	_ = json.Unmarshal(createRes.Body, &createEnv)

	patchRes := tc.Call("PATCH", "/tasks/:taskId/checklists/:checklistId",
		withPathParams(callerReq(), map[string]string{
			"taskId":      testTaskID,
			"checklistId": createEnv.Data.ID,
		}).WithJSONBody(map[string]string{"title": "Renamed"}))

	if patchRes.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", patchRes.StatusCode, patchRes.BodyString())
	}

	var patchEnv struct {
		Data checklist `json:"data"`
	}
	_ = json.Unmarshal(patchRes.Body, &patchEnv)

	if patchEnv.Data.ID != createEnv.Data.ID {
		t.Fatalf("expected same checklist id, got %s", patchEnv.Data.ID)
	}
	if patchEnv.Data.Title != "Renamed" {
		t.Fatalf("expected renamed checklist, got %+v", patchEnv.Data)
	}
	if patchEnv.Data.TaskID != testTaskID {
		t.Fatalf("expected task id %s, got %s", testTaskID, patchEnv.Data.TaskID)
	}
	if patchEnv.Data.CreatedAt == "" || patchEnv.Data.UpdatedAt == "" {
		t.Fatalf("expected timestamps in response, got %+v", patchEnv.Data)
	}
	if patchEnv.Data.UpdatedAt == createEnv.Data.UpdatedAt {
		t.Fatalf("expected updated timestamp to change, got %s", patchEnv.Data.UpdatedAt)
	}

	listRes := tc.Call("GET", "/tasks/:taskId/checklists",
		withPathParams(callerReq(), map[string]string{"taskId": testTaskID}))
	if listRes.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", listRes.StatusCode, listRes.BodyString())
	}

	var listEnv struct {
		Data []checklist `json:"data"`
	}
	_ = json.Unmarshal(listRes.Body, &listEnv)
	if len(listEnv.Data) != 1 || listEnv.Data[0].Title != "Renamed" {
		t.Fatalf("expected persisted renamed checklist, got %+v", listEnv.Data)
	}
}

// ── Item CRUD tests ───────────────────────────────────────────────────────────

func TestCreateAndToggleItem(t *testing.T) {
	tc := setupPlugin(t)

	// Create checklist
	createRes := tc.Call("POST", "/tasks/:taskId/checklists",
		withPathParams(callerReq(), map[string]string{"taskId": testTaskID}).
			WithJSONBody(map[string]string{"title": "CL"}))
	var clEnv struct {
		Data checklist `json:"data"`
	}
	_ = json.Unmarshal(createRes.Body, &clEnv)
	clID := clEnv.Data.ID

	// Create item
	itemRes := tc.Call("POST", "/tasks/:taskId/checklists/:checklistId/items",
		withPathParams(callerReq(), map[string]string{
			"taskId":      testTaskID,
			"checklistId": clID,
		}).WithJSONBody(map[string]string{"title": "Step 1"}))

	if itemRes.StatusCode != 201 {
		t.Fatalf("expected 201, got %d: %s", itemRes.StatusCode, itemRes.BodyString())
	}
	var itemEnv struct {
		Data checklistItem `json:"data"`
	}
	_ = json.Unmarshal(itemRes.Body, &itemEnv)
	itemID := itemEnv.Data.ID

	if itemEnv.Data.IsChecked {
		t.Fatal("newly created item should not be checked")
	}

	// Toggle checked
	patchRes := tc.Call("PATCH", "/tasks/:taskId/checklists/:checklistId/items/:itemId",
		withPathParams(callerReq(), map[string]string{
			"taskId":      testTaskID,
			"checklistId": clID,
			"itemId":      itemID,
		}).WithJSONBody(map[string]any{"is_checked": true}))

	if patchRes.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", patchRes.StatusCode, patchRes.BodyString())
	}
	var patchEnv struct {
		Data checklistItem `json:"data"`
	}
	_ = json.Unmarshal(patchRes.Body, &patchEnv)
	if !patchEnv.Data.IsChecked {
		t.Fatal("item should be checked after patch")
	}
}

func TestDeleteItem(t *testing.T) {
	tc := setupPlugin(t)

	createRes := tc.Call("POST", "/tasks/:taskId/checklists",
		withPathParams(callerReq(), map[string]string{"taskId": testTaskID}).
			WithJSONBody(map[string]string{"title": "CL"}))
	var clEnv struct {
		Data checklist `json:"data"`
	}
	_ = json.Unmarshal(createRes.Body, &clEnv)
	clID := clEnv.Data.ID

	itemRes := tc.Call("POST", "/tasks/:taskId/checklists/:checklistId/items",
		withPathParams(callerReq(), map[string]string{
			"taskId":      testTaskID,
			"checklistId": clID,
		}).WithJSONBody(map[string]string{"title": "Step"}))
	var itemEnv struct {
		Data checklistItem `json:"data"`
	}
	_ = json.Unmarshal(itemRes.Body, &itemEnv)

	delRes := tc.Call("DELETE", "/tasks/:taskId/checklists/:checklistId/items/:itemId",
		withPathParams(callerReq(), map[string]string{
			"taskId":      testTaskID,
			"checklistId": clID,
			"itemId":      itemEnv.Data.ID,
		}))

	if delRes.StatusCode != 204 {
		t.Fatalf("expected 204, got %d: %s", delRes.StatusCode, delRes.BodyString())
	}
}

// ── 404 guard tests ───────────────────────────────────────────────────────────

func TestListChecklists_UnknownTask(t *testing.T) {
	tc := setupPlugin(t)
	res := tc.Call("GET", "/tasks/:taskId/checklists",
		withPathParams(callerReq(), map[string]string{"taskId": "nonexistent"}))

	if res.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}
