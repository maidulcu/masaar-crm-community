package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// TaskReminderLead is how long before a task is due its assignee is reminded.
const TaskReminderLead = 15 * time.Minute

type taskStore interface {
	ClaimDueReminders(ctx context.Context, until time.Time) ([]domain.Task, error)
}

// TaskReminders notifies each task's assignee once, shortly before it is due (or as soon
// as possible for a task created already overdue).
func TaskReminders(tasks taskStore, n *Notifier) Job {
	return Job{
		Name:     "task_reminders",
		Interval: time.Minute,
		PerCompany: func(ctx context.Context, co domain.CompanyLocale) (int, error) {
			due, err := tasks.ClaimDueReminders(ctx, time.Now().Add(TaskReminderLead))
			if err != nil {
				return 0, err
			}
			loc := location(co.Timezone)
			for _, t := range due {
				at := t.DueAt.In(loc).Format("2 Jan 15:04")
				overdue := t.DueAt.Before(time.Now())
				msg := Message{
					Type:    "task_due",
					TitleEN: "Task due " + at,
					TitleAR: "مهمة مستحقة " + at,
					BodyEN:  t.Title,
					BodyAR:  t.Title,
					Data:    fmt.Sprintf(`{"task_id":%q}`, t.ID.String()),
				}
				if overdue {
					msg.TitleEN, msg.TitleAR = "Overdue task (due "+at+")", "مهمة متأخرة (مستحقة "+at+")"
				}
				if err := n.Notify(ctx, *t.AssignedTo, msg); err != nil {
					return len(due), err
				}
			}
			return len(due), nil
		},
	}
}
