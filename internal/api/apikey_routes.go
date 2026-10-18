package api

import (
	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/apiscope"
)

// apiKeyRoutes are the /api/v1 routes an API key may call, and the scope each needs. Every
// other /api/v1 route refuses API keys. A handler listed here must not assume a signed-in user:
// no user_id or role in locals (use auditWrite for the audit trail).
var apiKeyRoutes = []middleware.APIKeyRoute{
	{Method: "GET", Path: "/api/v1/leads/search", Scope: apiscope.LeadsRead},
	{Method: "GET", Path: "/api/v1/leads/:id", Scope: apiscope.LeadsRead},
	{Method: "POST", Path: "/api/v1/leads", Scope: apiscope.LeadsWrite},
	{Method: "PATCH", Path: "/api/v1/leads/:id/stage", Scope: apiscope.LeadsWrite},
	{Method: "PATCH", Path: "/api/v1/leads/:id/notes", Scope: apiscope.LeadsWrite},

	{Method: "GET", Path: "/api/v1/contacts", Scope: apiscope.ContactsRead},
	{Method: "GET", Path: "/api/v1/contacts/:id", Scope: apiscope.ContactsRead},
	{Method: "POST", Path: "/api/v1/contacts", Scope: apiscope.ContactsWrite},
	{Method: "PATCH", Path: "/api/v1/contacts/:id", Scope: apiscope.ContactsWrite},

	{Method: "GET", Path: "/api/v1/listings", Scope: apiscope.ListingsRead},
	{Method: "GET", Path: "/api/v1/listings/:id", Scope: apiscope.ListingsRead},

	{Method: "GET", Path: "/api/v1/settings/webhooks", Scope: apiscope.WebhooksManage},
	{Method: "POST", Path: "/api/v1/settings/webhooks", Scope: apiscope.WebhooksManage},
	{Method: "DELETE", Path: "/api/v1/settings/webhooks/:id", Scope: apiscope.WebhooksManage},
}
