// Package api provides the HTTP API for Masaar CRM.
//
// @title           Masaar CRM API
// @version         1.0
// @description     Open-source CRM for the UAE market. WhatsApp-first, RTL-native, self-hosted.
// @termsOfService  https://masaar.dynamicweblab.com/terms
//
// @contact.name   Dynamic Web Lab FZE LLC
// @contact.email  info@dynamicweblab.com
// @contact.url    https://dynamicweblab.com
//
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
//
// @host      localhost:8080
// @BasePath  /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                JWT Bearer token. Format: "Bearer <token>"
//
// @tag.name Auth
// @tag.description JWT login, token refresh, logout
//
// @tag.name Contacts
// @tag.description Unified contact profiles linked to WhatsApp
//
// @tag.name Leads
// @tag.description Sales pipeline — Kanban board with drag-drop stage management
//
// @tag.name WhatsApp
// @tag.description WhatsApp inbox — threads and messages (read-only, free tier)
//
// @tag.name Deals
// @tag.description Deal management linked to leads
//
// @tag.name Invoices
// @tag.description VAT invoices (5% UAE VAT) — draft → sent → paid lifecycle
//
// @tag.name AI
// @tag.description Manual AI actions powered by Ollama (local LLM)
//
// @tag.name Notifications
// @tag.description Personal real-time notifications
package api
