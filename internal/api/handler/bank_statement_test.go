package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/mediastore"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

func TestContentMatchesFormat(t *testing.T) {
	cases := []struct {
		name   string
		format domain.FileFormat
		head   string
		want   bool
	}{
		{"pdf ok", domain.FormatPDF, "%PDF-1.7\n...", true},
		{"pdf wrong", domain.FormatPDF, "<html>", false},
		{"xlsx ok", domain.FormatXLSX, "PK\x03\x04rest", true},
		{"xlsx wrong", domain.FormatXLSX, "%PDF-1.7", false},
		{"csv ok", domain.FormatCSV, "date,amount\n2025-01-01,10\n", true},
		{"csv arabic", domain.FormatCSV, "التاريخ,المبلغ\n", true},
		{"csv truncated rune at 512 boundary", domain.FormatCSV, "a,b\n" + string([]byte("م")[:1]), true},
		{"csv binary", domain.FormatCSV, "abc\x00def", false},
		{"csv empty", domain.FormatCSV, "", false},
		{"csv invalid utf8", domain.FormatCSV, "a,\xff\xfe\xfd,b", false},
	}
	for _, tc := range cases {
		if got := contentMatchesFormat(tc.format, []byte(tc.head)); got != tc.want {
			t.Errorf("%s: contentMatchesFormat = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestBankStatementUploadStoresFile(t *testing.T) {
	e := newWAEnv(t) // provides migrated DB, two companies and a user
	ctx := context.Background()

	dir := filepath.Join(t.TempDir(), "statements")
	store, err := mediastore.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	integration := uuid.New()
	if _, err := e.pool.Exec(ctx, `INSERT INTO bank_integrations (id, company_id, bank_name) VALUES ($1, $2, 'Test Bank')`, integration, e.companyID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = e.pool.Exec(ctx, `DELETE FROM bank_statements WHERE company_id = $1`, e.companyID)
		_, _ = e.pool.Exec(ctx, `DELETE FROM bank_integrations WHERE company_id = $1`, e.companyID)
	})

	h := NewBankStatementHandler(repo.NewBankStatementRepo(e.pool), store, 1)
	app := fiber.New(fiber.Config{BodyLimit: 10 * 1024 * 1024})
	for prefix, company := range map[string]uuid.UUID{"/a": e.companyID, "/b": e.otherCo} {
		company := company
		g := app.Group(prefix, func(c *fiber.Ctx) error {
			c.Locals("user_id", e.userID)
			c.Locals("company_id", company.String())
			return c.Next()
		})
		g.Post("/bank-statements/upload", h.Upload)
		g.Get("/bank-statements/:id/download", h.Download)
		g.Delete("/bank-statements/:id", h.Delete)
	}

	upload := func(prefix, name string, content []byte) (int, map[string]any) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", name)
		_, _ = fw.Write(content)
		_ = mw.WriteField("bank_integration_id", integration.String())
		_ = mw.Close()
		req := httptest.NewRequest("POST", prefix+"/bank-statements/upload", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	storedFiles := func() int {
		entries, _ := os.ReadDir(dir)
		return len(entries)
	}

	// content that is not the declared format is refused and nothing is stored
	if code, _ := upload("/a", "evil.pdf", []byte("<script>alert(1)</script>")); code != 422 {
		t.Fatalf("fake pdf = %d, want 422", code)
	}
	// a foreign company's integration id is refused and the stored file is cleaned up
	if code, _ := upload("/b", "ok.csv", []byte("date,amount\n")); code == 201 {
		t.Fatal("company B attached a statement to company A's bank integration")
	}
	if n := storedFiles(); n != 0 {
		t.Fatalf("%d files left behind after rejected uploads", n)
	}

	// valid upload
	csv := []byte("date,amount\n2025-01-01,100.00\n")
	code, st := upload("/a", "../../jan.csv", csv)
	if code != 201 {
		t.Fatalf("upload = %d %v", code, st)
	}
	id, _ := st["id"].(string)
	if st["file_name"] != "jan.csv" {
		t.Errorf("file name = %v, want sanitised jan.csv", st["file_name"])
	}
	if want := "/api/v1/bank-statements/" + id + "/download"; st["file_url"] != want {
		t.Errorf("file_url = %v, want %s", st["file_url"], want)
	}
	if _, leaked := st["StorageKey"]; leaked {
		t.Error("storage key leaked in JSON")
	}
	if n := storedFiles(); n != 1 {
		t.Fatalf("%d stored files, want 1", n)
	}
	var date string
	if err := e.pool.QueryRow(ctx, `SELECT upload_date::text FROM bank_statements WHERE id = $1`, id).Scan(&date); err != nil || date[:4] == "0001" {
		t.Fatalf("upload_date = %q (%v), want the real upload time", date, err)
	}

	// download returns the same bytes as a nosniff attachment; another company cannot
	resp, err := app.Test(httptest.NewRequest("GET", "/a/bank-statements/"+id+"/download", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !bytes.Equal(got, csv) {
		t.Fatalf("download = %d %q", resp.StatusCode, got)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("Content-Disposition") == "" {
		t.Errorf("missing hardening headers: %v", resp.Header)
	}
	if resp, _ := app.Test(httptest.NewRequest("GET", "/b/bank-statements/"+id+"/download", nil), -1); resp.StatusCode != 404 {
		t.Errorf("company B download = %d, want 404", resp.StatusCode)
	}

	// B cannot delete A's statement; A can, and the file goes with it
	if resp, _ := app.Test(httptest.NewRequest("DELETE", "/b/bank-statements/"+id, nil), -1); resp.StatusCode != 404 {
		t.Errorf("company B delete = %d, want 404", resp.StatusCode)
	}
	if n := storedFiles(); n != 1 {
		t.Fatalf("file removed by another company's delete (%d files)", n)
	}
	if resp, _ := app.Test(httptest.NewRequest("DELETE", "/a/bank-statements/"+id, nil), -1); resp.StatusCode != 204 {
		t.Fatalf("delete = %d, want 204", resp.StatusCode)
	}
	if n := storedFiles(); n != 0 {
		t.Errorf("%d files remain after delete", n)
	}
}
