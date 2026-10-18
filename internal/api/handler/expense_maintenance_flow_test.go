package handler

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/repo"
)

func dataID(t *testing.T, out map[string]any) string {
	t.Helper()
	d, ok := out["data"].(map[string]any)
	if !ok {
		t.Fatalf("no data in %v", out)
	}
	return d["id"].(string)
}

func TestExpense_WebFormPayloadValidationAndApproval(t *testing.T) {
	e := newTLEnv(t)
	code, out := e.req(t, "POST", "/a/expense-categories", fiber.Map{"category_name": "Power", "category_type": "utilities"})
	if code != 201 {
		t.Fatalf("category = %d %v", code, out)
	}
	cat := dataID(t, out)
	if code, _ := e.req(t, "POST", "/a/expense-categories", fiber.Map{"category_name": "Power", "category_type": "utilities"}); code != 409 {
		t.Fatalf("duplicate category = %d, want 409", code)
	}
	if code, _ := e.req(t, "POST", "/a/expense-categories", fiber.Map{"category_name": "X", "category_type": "bogus"}); code != 400 {
		t.Fatalf("bad category type = %d, want 400", code)
	}

	// exactly what the web form posts: date-only expense_date and "" for no property
	code, out = e.req(t, "POST", "/a/expenses", fiber.Map{
		"category_id": cat, "property_id": "", "amount": 120.5, "expense_date": "2026-03-01",
		"description": "Electricity", "payment_method": "bank_transfer", "receipt_url": "",
	})
	if code != 201 {
		t.Fatalf("web-form expense = %d %v (was always 400)", code, out)
	}
	id := dataID(t, out)

	for name, body := range map[string]fiber.Map{
		"no category":    {"amount": 1, "expense_date": "2026-03-01", "description": "x"},
		"zero amount":    {"category_id": cat, "amount": 0, "expense_date": "2026-03-01", "description": "x"},
		"no date":        {"category_id": cat, "amount": 1, "description": "x"},
		"no description": {"category_id": cat, "amount": 1, "expense_date": "2026-03-01"},
		"bad method":     {"category_id": cat, "amount": 1, "expense_date": "2026-03-01", "description": "x", "payment_method": "gold"},
		"js receipt":     {"category_id": cat, "amount": 1, "expense_date": "2026-03-01", "description": "x", "receipt_url": "javascript:alert(1)"},
	} {
		if code, out := e.req(t, "POST", "/a/expenses", body); code != 400 {
			t.Errorf("%s = %d %v, want 400", name, code, out)
		}
	}
	if code, _ := e.req(t, "POST", "/a/expenses", fiber.Map{"category_id": uuid.NewString(), "amount": 1, "expense_date": "2026-03-01", "description": "x"}); code != 422 {
		t.Errorf("unknown category = %d, want 422 (was 500)", code)
	}
	if code, _ := e.req(t, "POST", "/b/expenses", fiber.Map{"category_id": cat, "amount": 1, "expense_date": "2026-03-01", "description": "x"}); code != 422 {
		t.Errorf("another company's category = %d, want 422", code)
	}

	// edit: more than amount/description/status now, validated; missing = 404
	if code, out := e.req(t, "PATCH", "/a/expenses/"+id, fiber.Map{"vendor_name": "DEWA", "expense_date": "2026-03-05", "payment_status": "paid"}); code != 200 {
		t.Fatalf("update = %d %v", code, out)
	}
	if v := e.val(t, `SELECT vendor_name || ' ' || expense_date::text || ' ' || payment_status FROM expenses WHERE id=$1`, id); v != "DEWA 2026-03-05 paid" {
		t.Fatalf("update not persisted: %q", v)
	}
	if code, _ := e.req(t, "PATCH", "/a/expenses/"+id, fiber.Map{"payment_status": "lost"}); code != 400 {
		t.Fatalf("bad status = %d, want 400", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/expenses/"+uuid.NewString(), fiber.Map{"amount": 5}); code != 404 {
		t.Fatalf("missing update = %d, want 404", code)
	}

	// approval: no body needed, once only, locks the amount
	if code, _ := e.req(t, "POST", "/a/expenses/"+uuid.NewString()+"/approve", nil); code != 404 {
		t.Fatalf("approve missing = %d, want 404 (was 500)", code)
	}
	if code, out := e.req(t, "POST", "/a/expenses/"+id+"/approve", nil); code != 200 {
		t.Fatalf("approve with no body = %d %v", code, out)
	}
	if code, _ := e.req(t, "POST", "/a/expenses/"+id+"/approve", nil); code != 409 {
		t.Fatalf("second approval = %d, want 409", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/expenses/"+id, fiber.Map{"amount": 9999}); code != 409 {
		t.Fatalf("amount change after approval = %d, want 409", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/expenses/"+id, fiber.Map{"notes": "ok"}); code != 200 {
		t.Fatalf("note after approval = %d, want 200", code)
	}
	if code, _ := e.req(t, "DELETE", "/b/expenses/"+id, nil); code != 404 {
		t.Fatalf("cross-company delete = %d, want 404", code)
	}
	if code, _ := e.req(t, "DELETE", "/a/expenses/"+id, nil); code != 204 {
		t.Fatalf("delete = %d, want 204", code)
	}
	if code, _ := e.req(t, "DELETE", "/a/expenses/"+id, nil); code != 404 {
		t.Fatalf("second delete = %d, want 404 (was 204)", code)
	}
}

func TestMaintenance_WebPayloadsLifecyclePhotosAndPropertyGuard(t *testing.T) {
	e := newTLEnv(t)
	prop := e.property(t)

	for name, body := range map[string]fiber.Map{
		"no property":    {"description": "leak"},
		"no description": {"property_id": prop},
		"bad type":       {"property_id": prop, "description": "x", "maintenance_type": "magic"},
		"bad priority":   {"property_id": prop, "description": "x", "priority": "asap"},
		"due < sched":    {"property_id": prop, "description": "x", "scheduled_date": "2026-05-10", "due_date": "2026-05-01"},
		"negative cost":  {"property_id": prop, "description": "x", "estimated_cost": -5},
	} {
		if code, out := e.req(t, "POST", "/a/maintenance-tasks", body); code != 400 {
			t.Errorf("%s = %d %v, want 400", name, code, out)
		}
	}
	if code, _ := e.req(t, "POST", "/a/maintenance-tasks", fiber.Map{"property_id": uuid.NewString(), "description": "x"}); code != 422 {
		t.Errorf("unknown property = %d, want 422 (was 500)", code)
	}
	if code, _ := e.req(t, "POST", "/b/maintenance-tasks", fiber.Map{"property_id": prop, "description": "x"}); code != 422 {
		t.Errorf("another company's property = %d, want 422", code)
	}

	// date-only dates and "" for unset optional fields (what a form sends)
	code, out := e.req(t, "POST", "/a/maintenance-tasks", fiber.Map{
		"property_id": prop, "description": "Leaking tap", "priority": "urgent", "maintenance_type": "plumbing",
		"scheduled_date": "2026-05-01", "due_date": "2026-05-03", "assigned_to": "", "inspection_id": "",
	})
	if code != 201 {
		t.Fatalf("create = %d %v (date-only dates used to be a 400)", code, out)
	}
	id := dataID(t, out)
	e.mk2(t, "/a/maintenance-tasks", fiber.Map{"property_id": prop, "description": "Paint hall", "priority": "low", "due_date": "2026-05-02"})
	e.mk2(t, "/a/maintenance-tasks", fiber.Map{"property_id": prop, "description": "Fix AC", "priority": "high", "due_date": "2026-05-02"})

	// same due date: high before low (it used to sort the words alphabetically: low before high)
	_, list := e.req(t, "GET", "/a/maintenance-tasks?limit=10", nil)
	d := list["data"].([]any)
	if len(d) != 3 || d[0].(map[string]any)["priority"] != "high" || d[1].(map[string]any)["priority"] != "low" {
		t.Fatalf("order = %v, want high, low (same due day), then the later urgent one", d)
	}
	if code, _ := e.req(t, "GET", "/a/maintenance-tasks?status=bogus", nil); code != 400 {
		t.Fatalf("bad status filter = %d, want 400", code)
	}

	// status through PATCH keeps the completion date consistent
	if code, out := e.req(t, "PATCH", "/a/maintenance-tasks/"+id, fiber.Map{"status": "completed", "actual_cost": 80}); code != 200 {
		t.Fatalf("complete via patch = %d %v", code, out)
	}
	if v := e.val(t, `SELECT completion_date::text FROM maintenance_tasks WHERE id=$1`, id); v == "" {
		t.Fatal("completed via PATCH but no completion_date")
	}
	if code, _ := e.req(t, "PATCH", "/a/maintenance-tasks/"+id, fiber.Map{"description": "edited"}); code != 409 {
		t.Fatalf("editing a closed task = %d, want 409", code)
	}
	if code, _ := e.req(t, "POST", "/a/maintenance-tasks/"+id+"/complete", nil); code != 409 {
		t.Fatalf("completing twice = %d, want 409", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/maintenance-tasks/"+id, fiber.Map{"status": "in_progress"}); code != 200 {
		t.Fatalf("re-open = %d, want 200", code)
	}
	if v := e.val(t, `SELECT completion_date::text FROM maintenance_tasks WHERE id=$1`, id); v != "" {
		t.Fatalf("re-opened task keeps completion_date %s", v)
	}
	// complete with no body (what the web client sends when no cost is typed)
	if code, out := e.req(t, "POST", "/a/maintenance-tasks/"+id+"/complete", nil); code != 200 {
		t.Fatalf("complete with no body = %d %v (was 400)", code, out)
	}

	// photos: safe URLs only, stage validated, existing task only
	for name, body := range map[string]fiber.Map{
		"javascript url": {"photo_url": "javascript:alert(1)", "photo_stage": "after"},
		"data url":       {"photo_url": "data:text/html;base64,AAAA", "photo_stage": "after"},
		"empty url":      {"photo_url": ""},
		"bad stage":      {"photo_url": "https://x.test/a.jpg", "photo_stage": "someday"},
	} {
		if code, _ := e.req(t, "POST", "/a/maintenance-tasks/"+id+"/photos", body); code != 400 {
			t.Errorf("photo %s = %d, want 400", name, code)
		}
	}
	if code, _ := e.req(t, "POST", "/a/maintenance-tasks/"+id+"/photos", fiber.Map{"photo_url": "https://x.test/a.jpg", "photo_stage": "before"}); code != 201 {
		t.Fatalf("good photo = %d", code)
	}
	if code, _ := e.req(t, "POST", "/a/maintenance-tasks/"+uuid.NewString()+"/photos", fiber.Map{"photo_url": "https://x.test/a.jpg"}); code != 404 {
		t.Fatalf("photo on a missing task = %d, want 404", code)
	}
	if code, _ := e.req(t, "GET", "/b/maintenance-tasks/"+id+"/photos", nil); code != 404 {
		t.Fatalf("another company's photos = %d, want 404", code)
	}

	// a property with open work cannot be deleted (no FK protects it)
	if code, _ := e.req(t, "DELETE", "/a/rental-properties/"+prop, nil); code != 409 {
		t.Fatalf("delete property with tasks = %d, want 409", code)
	}
	if code, _ := e.req(t, "DELETE", "/b/maintenance-tasks/"+id, nil); code != 404 {
		t.Fatalf("cross-company delete = %d, want 404", code)
	}
	if code, _ := e.req(t, "DELETE", "/a/maintenance-tasks/"+uuid.NewString(), nil); code != 404 {
		t.Fatalf("missing delete = %d, want 404 (was 204)", code)
	}
}

func TestMaintenanceAnalytics_CountsAreInTheRightColumns(t *testing.T) {
	e := newTLEnv(t)
	prop := e.property(t)
	open := e.mk2(t, "/a/maintenance-tasks", fiber.Map{"property_id": prop, "description": "a", "priority": "urgent"})
	done := e.mk2(t, "/a/maintenance-tasks", fiber.Map{"property_id": prop, "description": "b", "priority": "high"})
	_ = open
	if code, _ := e.req(t, "POST", "/a/maintenance-tasks/"+done+"/complete", nil); code != 200 {
		t.Fatal("complete failed")
	}
	e.mk2(t, "/a/maintenance-tasks", fiber.Map{"property_id": prop, "description": "c", "priority": "low"})

	a, err := repo.NewAnalyticsRepository(e.pool, nil).GetMaintenanceAnalytics(ctxBG(), e.a)
	if err != nil {
		t.Fatalf("analytics: %v", err)
	}
	if a.TotalTasks != 3 || a.CompletedTasks != 1 || a.PendingTasks != 2 || a.HighPriorityTasks != 1 {
		t.Fatalf("analytics = %+v, want total 3, completed 1, pending 2, open high-priority 1", a)
	}
}

func TestInspection_WebPayloadsValidationAndCompletion(t *testing.T) {
	e := newTLEnv(t)
	prop := e.property(t)

	if code, _ := e.req(t, "POST", "/a/inspection-templates", fiber.Map{"template_name": "Move-out", "inspection_type": "end_lease", "estimated_duration_minutes": 90, "checklist_items": []fiber.Map{{"id": "1", "description": "Walls"}}}); code != 201 {
		t.Fatalf("template = %d", code)
	}
	if code, _ := e.req(t, "POST", "/a/inspection-templates", fiber.Map{"template_name": "Move-out"}); code != 409 {
		t.Fatalf("duplicate template = %d, want 409 (was 500)", code)
	}
	if code, _ := e.req(t, "POST", "/a/inspection-templates", fiber.Map{"template_name": "Z", "inspection_type": "bogus"}); code != 400 {
		t.Fatalf("bad template type = %d, want 400", code)
	}
	if code, _ := e.req(t, "POST", "/a/inspection-templates", fiber.Map{"template_name": "Z", "checklist_items": []fiber.Map{{"id": "1", "description": ""}}}); code != 400 {
		t.Fatalf("empty checklist item = %d, want 400", code)
	}

	if code, _ := e.req(t, "POST", "/a/inspections", fiber.Map{"inspection_type": "general", "scheduled_date": "2026-06-01"}); code != 400 {
		t.Fatalf("no property = %d, want 400", code)
	}
	if code, _ := e.req(t, "POST", "/a/inspections", fiber.Map{"property_id": prop, "inspection_type": "weird", "scheduled_date": "2026-06-01"}); code != 400 {
		t.Fatalf("bad type = %d, want 400", code)
	}
	if code, _ := e.req(t, "POST", "/a/inspections", fiber.Map{"property_id": prop}); code != 400 {
		t.Fatalf("no date = %d, want 400", code)
	}
	if code, _ := e.req(t, "POST", "/a/inspections", fiber.Map{"property_id": uuid.NewString(), "scheduled_date": "2026-06-01"}); code != 422 {
		t.Fatalf("unknown property = %d, want 422 (was 500)", code)
	}
	// date-only date and "" references, as a form sends them
	code, out := e.req(t, "POST", "/a/inspections", fiber.Map{"property_id": prop, "inspection_type": "safety", "scheduled_date": "2026-06-01", "template_id": "", "inspector_id": "", "tenant_id": ""})
	if code != 201 {
		t.Fatalf("create = %d %v", code, out)
	}
	id := dataID(t, out)

	if code, _ := e.req(t, "PATCH", "/a/inspections/"+id, fiber.Map{"severity_level": "purple"}); code != 400 {
		t.Fatalf("bad severity = %d, want 400", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/inspections/"+id, fiber.Map{"photos_urls": []string{"javascript:alert(1)"}}); code != 400 {
		t.Fatalf("unsafe photo url = %d, want 400", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/inspections/"+id, fiber.Map{"photos_urls": []string{"https://x.test/1.jpg"}}); code != 200 {
		t.Fatalf("photos = %d", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/inspections/"+id, fiber.Map{"photos_urls": []string{}}); code != 200 {
		t.Fatalf("clear photos = %d", code)
	}
	if v := e.val(t, `SELECT COALESCE(array_length(photos_urls,1),0)::text FROM inspections WHERE id=$1`, id); v != "0" {
		t.Fatalf("photos not cleared: %s", v)
	}
	if code, _ := e.req(t, "PATCH", "/a/inspections/"+uuid.NewString(), fiber.Map{"findings": "x"}); code != 404 {
		t.Fatalf("missing update = %d, want 404 (was 500)", code)
	}
	if code, _ := e.req(t, "POST", "/a/inspections/"+id+"/complete", nil); code != 200 {
		t.Fatalf("complete = %d", code)
	}
	first := e.val(t, `SELECT completed_date::text FROM inspections WHERE id=$1`, id)
	if code, _ := e.req(t, "POST", "/a/inspections/"+id+"/complete", nil); code != 409 {
		t.Fatalf("complete twice = %d, want 409", code)
	}
	if v := e.val(t, `SELECT completed_date::text FROM inspections WHERE id=$1`, id); v != first {
		t.Fatalf("completion time rewritten: %s -> %s", first, v)
	}
	if code, _ := e.req(t, "GET", "/b/inspections/"+id, nil); code != 404 {
		t.Fatalf("cross-company get = %d, want 404", code)
	}
	if code, _ := e.req(t, "DELETE", "/a/rental-properties/"+prop, nil); code != 409 {
		t.Fatalf("delete property with inspections = %d, want 409", code)
	}
}
