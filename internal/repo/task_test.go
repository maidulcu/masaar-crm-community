package repo

import (
	"testing"
	"time"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestTasksLifecycleAndReminders(t *testing.T) {
	e := setup(t)
	tasks := NewTaskRepo(e.pool)
	ct := e.contact(t, e.a, testPhone("97152"))
	ld := e.lead(t, e.a, ct.ID)
	lead := "lead"
	past := time.Now().Add(-time.Hour)
	soon := time.Now().Add(5 * time.Minute)
	later := time.Now().Add(48 * time.Hour)

	overdue := &domain.Task{Title: "Call back", Type: "call", Priority: "high", DueAt: &past, AssignedTo: &e.a.user,
		RelatedType: &lead, RelatedID: &ld.ID, CreatedBy: &e.a.user}
	if err := tasks.Create(e.a.ctx, overdue); err != nil {
		t.Fatal(err)
	}
	dueSoon := &domain.Task{Title: "Send brochure", Type: "todo", Priority: "normal", DueAt: &soon, AssignedTo: &e.a.user}
	future := &domain.Task{Title: "Quarterly review", Type: "meeting", Priority: "low", DueAt: &later, AssignedTo: &e.a.user}
	for _, tk := range []*domain.Task{dueSoon, future} {
		if err := tasks.Create(e.a.ctx, tk); err != nil {
			t.Fatal(err)
		}
	}

	// B cannot link a task to A's lead, assign it to A's user, or see A's tasks.
	if err := tasks.Create(e.b.ctx, &domain.Task{Title: "x", Type: "todo", Priority: "low", RelatedType: &lead, RelatedID: &ld.ID}); !isNotFound(err) {
		t.Fatalf("B linked a task to A's lead: %v", err)
	}
	if err := tasks.Create(e.b.ctx, &domain.Task{Title: "x", Type: "todo", Priority: "low", AssignedTo: &e.a.user}); !isNotFound(err) {
		t.Fatalf("B assigned a task to A's user: %v", err)
	}
	if list, total, _ := tasks.List(e.b.ctx, domain.TaskFilter{}); total != 0 || len(list) != 0 {
		t.Fatal("B listed A's tasks")
	}

	got, err := tasks.Get(e.a.ctx, overdue.ID)
	if err != nil || !got.Overdue || got.RelatedLabel != ct.FullName || got.AssigneeName == "" {
		t.Fatalf("overdue task = %+v, %v", got, err)
	}
	list, total, err := tasks.List(e.a.ctx, domain.TaskFilter{AssignedTo: &e.a.user, OverdueOnly: true})
	if err != nil || total != 1 || list[0].ID != overdue.ID {
		t.Fatalf("overdue list = %+v (%d) %v", list, total, err)
	}
	if list, _, _ := tasks.List(e.a.ctx, domain.TaskFilter{RelatedType: "lead", RelatedID: &ld.ID}); len(list) != 1 {
		t.Fatalf("tasks for the lead = %d", len(list))
	}

	claimed, err := tasks.ClaimDueReminders(e.a.ctx, time.Now().Add(15*time.Minute))
	if err != nil || len(claimed) != 2 {
		t.Fatalf("claimed %d reminders (%v), want the overdue and due-soon tasks", len(claimed), err)
	}
	if again, _ := tasks.ClaimDueReminders(e.a.ctx, time.Now().Add(15*time.Minute)); len(again) != 0 {
		t.Fatalf("reminders claimed twice: %+v", again)
	}
	// Moving the due date re-arms the reminder.
	newDue := time.Now().Add(10 * time.Minute)
	dueSoon.DueAt = &newDue
	if err := tasks.Update(e.a.ctx, dueSoon); err != nil {
		t.Fatal(err)
	}
	if again, _ := tasks.ClaimDueReminders(e.a.ctx, time.Now().Add(15*time.Minute)); len(again) != 1 {
		t.Fatalf("rescheduled task not re-armed: %+v", again)
	}

	if err := tasks.SetStatus(e.a.ctx, overdue.ID, "done"); err != nil {
		t.Fatal(err)
	}
	if got, _ := tasks.Get(e.a.ctx, overdue.ID); got.Status != "done" || got.CompletedAt == nil || got.Overdue {
		t.Fatalf("after done: %+v", got)
	}
	if err := tasks.SetStatus(e.b.ctx, future.ID, "done"); !isNotFound(err) {
		t.Fatalf("B completed A's task: %v", err)
	}
	if err := tasks.Delete(e.a.ctx, future.ID); err != nil {
		t.Fatal(err)
	}
}
