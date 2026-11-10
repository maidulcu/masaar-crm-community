package repo

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestUserMutations_KeepAnActiveAdmin(t *testing.T) {
	e := setup(t)
	users := NewUserRepo(e.pool)
	co, admin := e.a.id, e.a.user
	second, agent := uuid.New(), uuid.New()
	must(t, e.pool, `INSERT INTO users (id, company_id, name, email, password_hash, role) VALUES ($1, $2, 'Second', $3, 'x', 'admin'), ($4, $2, 'Agent', $5, 'x', 'agent')`,
		second, co, second.String()+"@test.local", agent, agent.String()+"@test.local")

	// With two admins, one may be demoted; the remaining one may not be demoted, deactivated or deleted.
	if err := users.UpdateUser(e.a.ctx, co, second, "Second", domain.RoleAgent); err != nil {
		t.Fatalf("demote one of two admins: %v", err)
	}
	for name, err := range map[string]error{
		"demote":     users.UpdateUser(e.a.ctx, co, admin, "Admin", domain.RoleViewer),
		"deactivate": users.SetActive(e.a.ctx, co, admin, false),
		"delete":     users.Delete(e.a.ctx, co, admin),
	} {
		if !errors.Is(err, ErrLastAdmin) {
			t.Errorf("%s the last admin: %v, want ErrLastAdmin", name, err)
		}
	}
	var role string
	var active bool
	if err := e.pool.QueryRow(e.a.ctx, `SELECT role, is_active FROM users WHERE id = $1`, admin).Scan(&role, &active); err != nil || role != "admin" || !active {
		t.Fatalf("last admin changed anyway: role=%s active=%v err=%v", role, active, err)
	}
	// Changes to other users still work, and renaming the admin keeps the role.
	if err := users.SetActive(e.a.ctx, co, agent, false); err != nil {
		t.Errorf("deactivate an agent: %v", err)
	}
	if err := users.UpdateUser(e.a.ctx, co, admin, "Renamed", domain.RoleAdmin); err != nil {
		t.Errorf("rename the last admin: %v", err)
	}
	if err := users.SetActive(e.a.ctx, co, uuid.New(), false); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("unknown user: %v, want ErrUserNotFound", err)
	}
}
