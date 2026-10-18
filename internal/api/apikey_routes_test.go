package api

import (
	"os"
	"strings"
	"testing"

	"github.com/maidulcu/masaar-crm/internal/apiscope"
)

// Every route opened to API keys names a known scope and is registered in the router.
func TestAPIKeyRoutes(t *testing.T) {
	src, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatal(err)
	}
	methods := map[string]string{"GET": "Get", "POST": "Post", "PATCH": "Patch", "DELETE": "Delete"}
	for _, r := range apiKeyRoutes {
		if _, ok := apiscope.Known[r.Scope]; !ok {
			t.Errorf("%s %s: unknown scope %q", r.Method, r.Path, r.Scope)
		}
		rel, ok := strings.CutPrefix(r.Path, "/api/v1")
		if !ok {
			t.Errorf("%s %s: not under /api/v1", r.Method, r.Path)
			continue
		}
		if !strings.Contains(string(src), `v1.`+methods[r.Method]+`("`+rel+`"`) {
			t.Errorf("%s %s: no such route in router.go", r.Method, r.Path)
		}
	}
}
