package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// The public signing page gets what it shows and nothing internal; a suspended company's
// signing links stop working.
func TestPublicGetSignature_ShowsOnlyWhatThePageNeeds(t *testing.T) {
	e := newLeadEnv(t)
	ctx := context.Background()
	docs := repo.NewDocumentRepo(e.pool)
	doc := &domain.Document{DocumentType: domain.DocTypeOffer, DocumentTitle: "Offer", RelatedEntityType: "deal",
		FileURL: "https://files.example.com/offer.pdf", SignatureStatus: domain.SignaturePending, CreatedBy: e.admin,
		DataClassification: domain.ClassConfidential}
	if err := docs.CreateDocument(tenant.With(ctx, e.a), doc); err != nil {
		t.Fatal(err)
	}
	docID, sigID := doc.ID, uuid.New()
	if _, err := e.pool.Exec(ctx, `INSERT INTO document_signatures (id, document_id, signer_name, signer_email, signature_field_name, signature_image_url, signature_status, ip_address, user_agent)
		VALUES ($1, $2, 'S', 's@example.com', 'buyer', '', 'pending', '10.0.0.1', 'sender-ua')`, sigID, docID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = e.pool.Exec(ctx, `DELETE FROM document_signatures WHERE document_id = $1`, docID)
		_, _ = e.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, docID)
		_, _ = e.pool.Exec(ctx, `UPDATE companies SET is_active = true WHERE id = $1`, e.a)
	})

	h := NewDocumentHandler(docs, repo.NewAuditLogRepo(e.pool), nil)
	app := fiber.New()
	app.Get("/sign/:id", h.PublicGetSignature)
	get := func() (int, map[string]map[string]any) {
		resp, err := app.Test(httptest.NewRequest("GET", "/sign/"+sigID.String(), nil), -1)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	code, out := get()
	if code != 200 || out["document"]["document_title"] != "Offer" || out["signature"]["signer_name"] != "S" {
		t.Fatalf("get = %d %v", code, out)
	}
	for part, keys := range map[string][]string{
		"document":  {"company_id", "created_by", "related_entity_id", "data_classification", "retention_until", "content_hash"},
		"signature": {"ip_address", "user_agent", "envelope_id"},
	} {
		for _, k := range keys {
			if _, ok := out[part][k]; ok {
				t.Errorf("public %s exposes %s", part, k)
			}
		}
	}

	if _, err := e.pool.Exec(ctx, `UPDATE companies SET is_active = false WHERE id = $1`, e.a); err != nil {
		t.Fatal(err)
	}
	if code, _ := get(); code != 404 {
		t.Errorf("suspended company's signing link = %d, want 404", code)
	}
}
