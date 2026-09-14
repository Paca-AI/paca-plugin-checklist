package main

import (
	"encoding/json"
	"testing"

	plugin "github.com/Paca-AI/plugin-sdk-go"
	"github.com/Paca-AI/plugin-sdk-go/plugintest"
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

// TestUpdateItem_OmittedAssigneeUnchanged guards against regressing to
// unconditionally overwriting assignee_id on every PATCH, even when the
// request omits it.
func TestUpdateItem_OmittedAssigneeUnchanged(t *testing.T) {
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
	itemID := itemEnv.Data.ID

	itemPath := map[string]string{"taskId": testTaskID, "checklistId": clID, "itemId": itemID}

	assignRes := tc.Call("PATCH", "/tasks/:taskId/checklists/:checklistId/items/:itemId",
		withPathParams(callerReq(), itemPath).WithJSONBody(map[string]any{"assignee_id": "user-42"}))
	if assignRes.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", assignRes.StatusCode, assignRes.BodyString())
	}
	var assignEnv struct {
		Data checklistItem `json:"data"`
	}
	_ = json.Unmarshal(assignRes.Body, &assignEnv)
	if assignEnv.Data.AssigneeID == nil || *assignEnv.Data.AssigneeID != "user-42" {
		t.Fatalf("expected assignee_id=user-42, got %+v", assignEnv.Data.AssigneeID)
	}

	// Patch only is_checked; assignee_id is absent from the body and must
	// survive the update untouched.
	patchRes := tc.Call("PATCH", "/tasks/:taskId/checklists/:checklistId/items/:itemId",
		withPathParams(callerReq(), itemPath).WithJSONBody(map[string]any{"is_checked": true}))
	if patchRes.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", patchRes.StatusCode, patchRes.BodyString())
	}
	var patchEnv struct {
		Data checklistItem `json:"data"`
	}
	_ = json.Unmarshal(patchRes.Body, &patchEnv)
	if !patchEnv.Data.IsChecked {
		t.Fatal("expected is_checked=true")
	}
	if patchEnv.Data.AssigneeID == nil || *patchEnv.Data.AssigneeID != "user-42" {
		t.Fatalf("expected assignee_id to remain user-42, got %+v", patchEnv.Data.AssigneeID)
	}
}

// TestUpdateItem_ExplicitNullClearsAssignee verifies the other half of the
// three-state contract: an explicit JSON null clears the assignee, matching
// the checklist MCP tool's documented "pass null to unassign" behavior.
func TestUpdateItem_ExplicitNullClearsAssignee(t *testing.T) {
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
	itemID := itemEnv.Data.ID

	itemPath := map[string]string{"taskId": testTaskID, "checklistId": clID, "itemId": itemID}

	assignRes := tc.Call("PATCH", "/tasks/:taskId/checklists/:checklistId/items/:itemId",
		withPathParams(callerReq(), itemPath).WithJSONBody(map[string]any{"assignee_id": "user-42"}))
	if assignRes.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", assignRes.StatusCode, assignRes.BodyString())
	}

	clearRes := tc.Call("PATCH", "/tasks/:taskId/checklists/:checklistId/items/:itemId",
		withPathParams(callerReq(), itemPath).WithJSONBody(map[string]any{"assignee_id": nil}))
	if clearRes.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", clearRes.StatusCode, clearRes.BodyString())
	}
	var clearEnv struct {
		Data checklistItem `json:"data"`
	}
	_ = json.Unmarshal(clearRes.Body, &clearEnv)
	if clearEnv.Data.AssigneeID != nil {
		t.Fatalf("expected assignee_id to be cleared, got %q", *clearEnv.Data.AssigneeID)
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

// ── Cross-project ownership guards ───────────────────────────────────────────

const (
	otherProjectID = "project-2"
	otherTaskID    = "task-2"
)

// otherProjectCallerReq mints a foreign checklist/item under otherTaskID,
// which genuinely belongs to otherProjectID — used to set up the victim
// resource for the cross-project tests below.
func otherProjectCallerReq() plugintest.Request {
	return plugintest.Request{
		Caller: plugin.CallerIdentity{
			ProjectID:  otherProjectID,
			CallerID:   "member-2",
			CallerRole: "PROJECT_MEMBER",
		},
		PathParams: map[string]string{},
	}
}

// setupForeignItem re-seeds the tasks table with both the default
// project-1/task-1 pair and a second, genuinely unrelated project-2/task-2
// pair, then creates a real checklist + item under task-2 as a project-2
// caller. Returns the foreign checklist and item IDs.
func setupForeignItem(t *testing.T, tc *plugintest.Context) (foreignChecklistID, foreignItemID string) {
	t.Helper()
	tc.DB.SeedRows("tasks", []string{"id", "project_id", "deleted_at"}, [][]any{
		{testTaskID, testProjectID, nil},
		{otherTaskID, otherProjectID, nil},
	})

	clRes := tc.Call("POST", "/tasks/:taskId/checklists",
		withPathParams(otherProjectCallerReq(), map[string]string{"taskId": otherTaskID}).
			WithJSONBody(map[string]string{"title": "Someone else's checklist"}))
	var clEnv struct {
		Data checklist `json:"data"`
	}
	if err := json.Unmarshal(clRes.Body, &clEnv); err != nil {
		t.Fatalf("failed to create foreign checklist: %s", clRes.BodyString())
	}

	itemRes := tc.Call("POST", "/tasks/:taskId/checklists/:checklistId/items",
		withPathParams(otherProjectCallerReq(), map[string]string{
			"taskId":      otherTaskID,
			"checklistId": clEnv.Data.ID,
		}).WithJSONBody(map[string]string{"title": "Someone else's item"}))
	var itemEnv struct {
		Data checklistItem `json:"data"`
	}
	if err := json.Unmarshal(itemRes.Body, &itemEnv); err != nil {
		t.Fatalf("failed to create foreign item: %s", itemRes.BodyString())
	}
	return clEnv.Data.ID, itemEnv.Data.ID
}

// TestUpdateItem_CrossProjectChecklistRejected pins the fix for a real IDOR:
// updateItem previously fetched/deleted/re-inserted the target item by bare
// id, with no check that the checklist in the URL actually owns it — a
// caller with tasks.write on their own project (testTaskID/testProjectID,
// which legitimately passes taskBelongsToProject) could hijack and
// reparent an arbitrary item from a checklist belonging to a completely
// different project, just by knowing its UUID.
func TestUpdateItem_CrossProjectChecklistRejected(t *testing.T) {
	tc := setupPlugin(t)
	foreignChecklistID, foreignItemID := setupForeignItem(t, tc)

	res := tc.Call("PATCH", "/tasks/:taskId/checklists/:checklistId/items/:itemId",
		withPathParams(callerReq(), map[string]string{
			"taskId":      testTaskID, // caller's own, legitimate task
			"checklistId": foreignChecklistID,
			"itemId":      foreignItemID,
		}).WithJSONBody(map[string]any{"title": "hijacked"}))
	if res.StatusCode != 404 {
		t.Fatalf("expected 404 (checklist belongs to a different task/project), got %d: %s", res.StatusCode, res.BodyString())
	}

	// The foreign item must be completely untouched.
	rows := tc.DB.AllRows("task_checklist_items")
	for _, row := range rows {
		if row[0] == foreignItemID && row[2] == "hijacked" {
			t.Fatal("foreign item was modified despite the 404")
		}
	}
}

// TestDeleteItem_CrossProjectChecklistRejected mirrors the update case for
// delete: deleteItem checked taskBelongsToProject but never verified the
// URL's checklistId actually belongs to that task.
func TestDeleteItem_CrossProjectChecklistRejected(t *testing.T) {
	tc := setupPlugin(t)
	foreignChecklistID, foreignItemID := setupForeignItem(t, tc)

	res := tc.Call("DELETE", "/tasks/:taskId/checklists/:checklistId/items/:itemId",
		withPathParams(callerReq(), map[string]string{
			"taskId":      testTaskID,
			"checklistId": foreignChecklistID,
			"itemId":      foreignItemID,
		}))
	if res.StatusCode != 404 {
		t.Fatalf("expected 404 (checklist belongs to a different task/project), got %d: %s", res.StatusCode, res.BodyString())
	}

	found := false
	for _, row := range tc.DB.AllRows("task_checklist_items") {
		if row[0] == foreignItemID {
			found = true
		}
	}
	if !found {
		t.Fatal("foreign item was deleted despite the 404")
	}
}
