'use client'

import Link from 'next/link'

const Section = ({ id, title, children }: { id: string; title: string; children: React.ReactNode }) => (
  <section id={id} className="mb-16 scroll-mt-20">
    <h2 className="text-2xl font-bold text-gray-900 mb-6 pb-3 border-b border-gray-200">{title}</h2>
    {children}
  </section>
)

const Code = ({ children }: { children: React.ReactNode }) => (
  <code className="bg-gray-100 text-blue-700 px-1.5 py-0.5 rounded text-sm font-mono">{children}</code>
)

const Pre = ({ children }: { children: React.ReactNode }) => (
  <pre className="bg-gray-900 text-green-400 rounded-lg p-4 overflow-x-auto text-sm font-mono mb-4 leading-relaxed">
    {children}
  </pre>
)

const Badge = ({ color, children }: { color: string; children: React.ReactNode }) => {
  const colors: Record<string, string> = {
    green: 'bg-green-100 text-green-800',
    blue: 'bg-blue-100 text-blue-800',
    yellow: 'bg-yellow-100 text-yellow-800',
    red: 'bg-red-100 text-red-800',
    gray: 'bg-gray-100 text-gray-700',
    purple: 'bg-purple-100 text-purple-800',
  }
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${colors[color] || colors.gray}`}>
      {children}
    </span>
  )
}

const METHOD_COLORS: Record<string, string> = {
  GET: 'bg-blue-100 text-blue-700',
  POST: 'bg-green-100 text-green-700',
  PATCH: 'bg-yellow-100 text-yellow-700',
  DELETE: 'bg-red-100 text-red-700',
  PUT: 'bg-orange-100 text-orange-700',
}

const Endpoint = ({ method, path, desc, auth }: { method: string; path: string; desc: string; auth?: string }) => (
  <div className="flex items-start gap-3 py-3 border-b border-gray-100 last:border-0">
    <span className={`mt-0.5 text-xs font-bold px-2 py-1 rounded shrink-0 ${METHOD_COLORS[method] || 'bg-gray-100 text-gray-700'}`}>
      {method}
    </span>
    <div className="flex-1 min-w-0">
      <code className="text-gray-900 font-mono text-sm">{path}</code>
      <p className="text-gray-500 text-sm mt-0.5">{desc}</p>
    </div>
    {auth && <Badge color={auth === 'public' ? 'green' : auth === 'admin' ? 'red' : auth === 'agent+' ? 'yellow' : 'blue'}>{auth}</Badge>}
  </div>
)

const navLinks = [
  { href: '#auth', label: 'Authentication' },
  { href: '#api-keys', label: 'API Keys' },
  { href: '#leads', label: 'Leads' },
  { href: '#contacts', label: 'Contacts' },
  { href: '#whatsapp', label: 'WhatsApp' },
  { href: '#ai', label: 'AI Features' },
  { href: '#webhooks', label: 'Webhooks' },
  { href: '#env', label: 'Environment' },
  { href: '#integrate', label: 'Integration Guide' },
  { href: '#roadmap', label: 'Roadmap' },
]

export default function DeveloperDocsPage() {
  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="border-b border-gray-200 bg-white sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-6 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-xl font-bold text-gray-900">Masaar CRM</span>
            <Badge color="blue">Developer Docs</Badge>
          </div>
          <div className="flex items-center gap-4">
            <a href="/docs" target="_blank" className="text-sm text-gray-500 hover:text-gray-900">
              Swagger UI ↗
            </a>
            <a href="https://github.com/maidulcu/masaar-crm" target="_blank"
              className="text-sm bg-gray-900 text-white px-4 py-1.5 rounded-lg hover:bg-gray-700">
              GitHub ↗
            </a>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto flex gap-0">
        {/* Sidebar */}
        <aside className="w-56 shrink-0 hidden md:block">
          <div className="sticky top-16 p-6 pt-8">
            <p className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-3">Contents</p>
            <nav className="space-y-1">
              {navLinks.map(l => (
                <a key={l.href} href={l.href}
                  className="block text-sm text-gray-600 hover:text-blue-600 hover:bg-blue-50 px-2 py-1.5 rounded transition-colors">
                  {l.label}
                </a>
              ))}
            </nav>

            <div className="mt-8 p-3 bg-blue-50 rounded-lg">
              <p className="text-xs font-semibold text-blue-800 mb-1">Base URL</p>
              <code className="text-xs text-blue-700 font-mono break-all">http://your-host:8080</code>
            </div>
          </div>
        </aside>

        {/* Main content */}
        <main className="flex-1 min-w-0 px-8 py-10">
          {/* Hero */}
          <div className="mb-12">
            <h1 className="text-4xl font-bold text-gray-900 mb-4">Developer API</h1>
            <p className="text-lg text-gray-600 max-w-2xl">
              Masaar CRM exposes a REST API for all CRM operations, a WebSocket endpoint for real-time
              notifications, and webhook support for Meta&apos;s WhatsApp Business API.
            </p>
            <div className="flex flex-wrap gap-2 mt-4">
              <Badge color="green">Open Source</Badge>
              <Badge color="blue">REST + WebSocket</Badge>
              <Badge color="purple">UAE / PDPL</Badge>
              <Badge color="gray">Self-Hosted</Badge>
            </div>
          </div>

          {/* Auth */}
          <Section id="auth" title="Authentication">
            <p className="text-gray-600 mb-4">
              All API endpoints require a valid JWT Bearer token. Two flows are supported:
              user login (interactive) and API keys (for external integrations).
            </p>

            <h3 className="font-semibold text-gray-900 mb-2">Login</h3>
            <Pre>{`POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "admin@masaar.local",
  "password": "yourpassword"
}

// Response
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",  // valid 15 min
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...", // valid 7 days
  "user": { "id": "...", "role": "admin" }
}`}</Pre>

            <h3 className="font-semibold text-gray-900 mb-2 mt-6">Use the token</h3>
            <Pre>{`Authorization: Bearer eyJhbGciOiJIUzI1NiIs...`}</Pre>

            <h3 className="font-semibold text-gray-900 mb-2 mt-6">Refresh + Logout</h3>
            <Pre>{`POST /api/v1/auth/refresh   // pass refresh token as Bearer
DELETE /api/v1/auth/logout  // invalidates token in Redis`}</Pre>

            <div className="mt-4 overflow-x-auto">
              <table className="w-full text-sm border-collapse">
                <thead>
                  <tr className="bg-gray-50">
                    <th className="text-left p-3 font-semibold text-gray-700 border border-gray-200">Role</th>
                    <th className="text-left p-3 font-semibold text-gray-700 border border-gray-200">Permissions</th>
                  </tr>
                </thead>
                <tbody>
                  {[
                    ['admin', 'Full access — settings, delete, manage users, API keys'],
                    ['agent', 'Create & update leads, contacts, deals, send messages'],
                    ['viewer', 'Read-only access to all resources (default for new users)'],
                  ].map(([role, desc]) => (
                    <tr key={role} className="border border-gray-200">
                      <td className="p-3"><Code>{role}</Code></td>
                      <td className="p-3 text-gray-600">{desc}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Section>

          {/* API Keys */}
          <Section id="api-keys" title="API Keys">
            <p className="text-gray-600 mb-4">
              For external integrations (website forms, Zapier, automation tools) that can&apos;t go through
              the user login flow, generate API keys in Settings. Keys use the same
              <Code>Authorization: Bearer sk_live_...</Code> header format.
            </p>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
              {[
                { scope: 'lead:create', desc: 'Submit new leads from external sources' },
                { scope: 'lead:read', desc: 'Read leads data' },
                { scope: 'contact:create', desc: 'Create contacts' },
                { scope: 'contact:read', desc: 'Read contacts' },
              ].map(({ scope, desc }) => (
                <div key={scope} className="border border-gray-200 rounded-lg p-3">
                  <Code>{scope}</Code>
                  <p className="text-gray-500 text-sm mt-1">{desc}</p>
                </div>
              ))}
            </div>

            <div className="grid md:grid-cols-3 gap-4 mb-4">
              {[
                ['GET', '/api/v1/settings/api-keys', 'List active keys'],
                ['POST', '/api/v1/settings/api-keys', 'Generate new key'],
                ['DELETE', '/api/v1/settings/api-keys/:id', 'Revoke key'],
              ].map(([m, p, d]) => <Endpoint key={p} method={m} path={p} desc={d} auth="admin" />)}
            </div>

            <Pre>{`// Generate a key
POST /api/v1/settings/api-keys
{ "name": "Website Form", "scopes": "lead:create,contact:create" }

// Response (plaintext shown ONCE — save it now)
{
  "id": "uuid",
  "key_prefix": "sk_live_abcd1234",
  "plaintext": "sk_live_a1b2c3d4...",  // SAVE THIS
  "scopes": "lead:create,contact:create"
}`}</Pre>
          </Section>

          {/* Leads */}
          <Section id="leads" title="Leads">
            <div className="mb-4 divide-y divide-gray-100 border border-gray-200 rounded-lg overflow-hidden">
              {[
                ['GET', '/api/v1/leads', 'Kanban board (all stages)', 'all roles'],
                ['GET', '/api/v1/leads/:id', 'Get lead details', 'all roles'],
                ['POST', '/api/v1/leads', 'Create lead', 'agent+'],
                ['PATCH', '/api/v1/leads/:id/stage', 'Move lead stage', 'agent+'],
                ['PATCH', '/api/v1/leads/:id/notes', 'Update notes', 'agent+'],
                ['GET', '/api/v1/leads/:id/communications', 'Communication history', 'all roles'],
              ].map(([m, p, d, a]) => <Endpoint key={p} method={m} path={p} desc={d} auth={a} />)}
            </div>

            <Pre>{`POST /api/v1/leads
{
  "contact_id": "uuid",
  "stage": "new",          // new|contacted|qualified|proposal|won|lost
  "source": "web",         // whatsapp|web|referral|event
  "deal_value": 500000,
  "currency": "AED",
  "notes": "Interested in Marina 2BR"
}`}</Pre>
          </Section>

          {/* Contacts */}
          <Section id="contacts" title="Contacts">
            <div className="mb-4 divide-y divide-gray-100 border border-gray-200 rounded-lg overflow-hidden">
              {[
                ['GET', '/api/v1/contacts', 'List contacts', 'all roles'],
                ['GET', '/api/v1/contacts/:id', 'Get contact', 'all roles'],
                ['POST', '/api/v1/contacts', 'Create contact', 'agent+'],
                ['PATCH', '/api/v1/contacts/:id', 'Update contact', 'agent+'],
                ['DELETE', '/api/v1/contacts/:id', 'Delete contact', 'admin'],
              ].map(([m, p, d, a]) => <Endpoint key={p} method={m} path={p} desc={d} auth={a} />)}
            </div>

            <Pre>{`POST /api/v1/contacts
{
  "full_name": "Ahmed Al-Mansouri",
  "phone_wa": "+971501234567",  // E.164 required (+country code)
  "email": "ahmed@example.com",
  "language": "ar"              // ar | en
}`}</Pre>
          </Section>

          {/* WhatsApp */}
          <Section id="whatsapp" title="WhatsApp">
            <p className="text-gray-600 mb-4">
              Inbound messages arrive via Meta webhook. Outbound messages are sent using the
              WhatsApp Business API.
            </p>
            <div className="divide-y divide-gray-100 border border-gray-200 rounded-lg overflow-hidden mb-4">
              {[
                ['GET', '/webhooks/whatsapp', 'Meta verification (GET)', 'public'],
                ['POST', '/webhooks/whatsapp', 'Receive inbound messages', 'public'],
                ['GET', '/api/v1/threads', 'List conversations', 'all roles'],
                ['GET', '/api/v1/threads/:id/messages', 'Get messages', 'all roles'],
                ['POST', '/api/v1/threads/:id/send-message', 'Send text message', 'agent+'],
                ['POST', '/api/v1/threads/:id/send-template', 'Send template', 'agent+'],
                ['POST', '/api/v1/threads/:id/close', 'Close thread', 'agent+'],
              ].map(([m, p, d, a]) => <Endpoint key={p} method={m} path={p} desc={d} auth={a} />)}
            </div>

            <div className="bg-amber-50 border border-amber-200 rounded-lg p-4 text-sm text-amber-800">
              <strong>Webhook Security:</strong> Set <Code>WA_APP_SECRET</Code> to your Meta App Secret.
              All inbound webhooks are validated using HMAC-SHA256 (<Code>X-Hub-Signature-256</Code> header).
            </div>
          </Section>

          {/* AI */}
          <Section id="ai" title="AI Features">
            <p className="text-gray-600 mb-4">
              Supports two AI providers — Ollama (local, private) or Google Gemini (cloud).
              Configure via <Code>AI_PROVIDER</Code> env var.
            </p>
            <div className="divide-y divide-gray-100 border border-gray-200 rounded-lg overflow-hidden mb-4">
              {[
                ['POST', '/api/v1/ai/summarize/:thread_id', 'Summarize WhatsApp thread', 'agent+'],
                ['POST', '/api/v1/messages/analyze', 'Parse intent from message', 'agent+'],
                ['POST', '/api/v1/messages/suggest-action', 'Recommend next action', 'agent+'],
                ['POST', '/api/v1/messages/auto-create-lead', 'Auto-create lead from message', 'agent+'],
              ].map(([m, p, d, a]) => <Endpoint key={p} method={m} path={p} desc={d} auth={a} />)}
            </div>

            <div className="grid md:grid-cols-2 gap-4">
              <div className="border border-gray-200 rounded-lg p-4">
                <h4 className="font-semibold text-gray-900 mb-2">🏠 Ollama (Local)</h4>
                <Pre>{`AI_PROVIDER=ollama
OLLAMA_BASE_URL=http://ollama:11434
OLLAMA_MODEL=llama3`}</Pre>
                <p className="text-sm text-gray-500">Fully offline. ~5GB first download.</p>
              </div>
              <div className="border border-gray-200 rounded-lg p-4">
                <h4 className="font-semibold text-gray-900 mb-2">☁️ Google Gemini</h4>
                <Pre>{`AI_PROVIDER=gemini
GEMINI_API_KEY=your_key
GEMINI_MODEL=gemini-2.0-flash`}</Pre>
                <p className="text-sm text-gray-500">
                  Get key at{' '}
                  <a href="https://aistudio.google.com/app/apikey" target="_blank"
                    className="text-blue-600 hover:underline">aistudio.google.com</a>
                </p>
              </div>
            </div>
          </Section>

          {/* Webhooks */}
          <Section id="webhooks" title="Webhooks">
            <div className="space-y-4">
              <div className="border border-green-200 bg-green-50 rounded-lg p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Badge color="green">Live</Badge>
                  <h4 className="font-semibold text-gray-900">Inbound — WhatsApp (Meta)</h4>
                </div>
                <p className="text-sm text-gray-600 mb-2">
                  Configure in Meta Developer Portal: <Code>POST /webhooks/whatsapp</Code>
                </p>
                <Pre>{`// Meta sends signed webhook:
X-Hub-Signature-256: sha256=<hmac>
// Validated automatically when WA_APP_SECRET is set`}</Pre>
              </div>

              <div className="border border-green-200 bg-green-50 rounded-lg p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Badge color="green">Live</Badge>
                  <h4 className="font-semibold text-gray-900">Inbound — Public Lead Endpoint</h4>
                </div>
                <p className="text-sm text-gray-600 mb-2">
                  Submit leads from any external source using an API key (scope: <Code>lead:create</Code>):
                </p>
                <Pre>{`POST /webhooks/leads
Authorization: Bearer sk_live_...
Content-Type: application/json

{
  "name": "Ahmed Al-Mansouri",       // required
  "phone": "+971501234567",          // required, E.164
  "email": "ahmed@example.com",      // optional
  "language": "ar",                  // optional (ar|en, default: ar)
  "source": "web",                   // optional (web|referral|event, default: web)
  "notes": "Interested in Marina",   // optional
  "deal_value": 500000,              // optional
  "property_type": "2BR",            // optional metadata
  "area": "Marina"                   // optional metadata
}

// Response 201
{
  "lead_id": "uuid",
  "contact_id": "uuid",
  "stage": "new",
  "source": "web",
  "message": "Lead created successfully"
}`}</Pre>
                <p className="text-sm text-gray-600 mt-2">
                  Contact is automatically created or matched by phone number.
                  Rate limited at 300 req/min.
                </p>
              </div>

              <div className="border border-green-200 bg-green-50 rounded-lg p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Badge color="green">Phase 3 — Live</Badge>
                  <h4 className="font-semibold text-gray-900">Outbound — Event Webhooks</h4>
                </div>
                <p className="text-sm text-gray-600 mb-3">
                  Push signed events to your URL when things happen in the CRM.
                  Events: <code className="bg-green-100 px-1 rounded text-xs">lead.created</code>{' '}
                  <code className="bg-green-100 px-1 rounded text-xs">lead.stage_changed</code>{' '}
                  <code className="bg-green-100 px-1 rounded text-xs">lead.won</code>{' '}
                  <code className="bg-green-100 px-1 rounded text-xs">lead.lost</code>{' '}
                  <code className="bg-green-100 px-1 rounded text-xs">payment.received</code>
                </p>
                <Pre>{`POST /api/v1/settings/webhooks
Authorization: Bearer <jwt>

{"name":"My App","url":"https://...","events":"lead.created,lead.won"}
// Response includes secret — store it to verify signatures`}</Pre>
                <p className="text-xs text-gray-500 mt-2">
                  All requests carry <Code>X-Masaar-Signature: sha256=hmac</Code>. 3 retries with 1s/4s backoff.
                </p>
              </div>
            </div>
          </Section>

          {/* Env vars */}
          <Section id="env" title="Environment Variables">
            <div className="overflow-x-auto">
              <table className="w-full text-sm border-collapse">
                <thead>
                  <tr className="bg-gray-50">
                    <th className="text-left p-3 font-semibold text-gray-700 border border-gray-200">Variable</th>
                    <th className="text-left p-3 font-semibold text-gray-700 border border-gray-200">Required</th>
                    <th className="text-left p-3 font-semibold text-gray-700 border border-gray-200">Default</th>
                    <th className="text-left p-3 font-semibold text-gray-700 border border-gray-200">Description</th>
                  </tr>
                </thead>
                <tbody>
                  {[
                    ['DATABASE_URL', '✅', '—', 'PostgreSQL connection string'],
                    ['REDIS_URL', '✅', '—', 'Redis connection string'],
                    ['JWT_SECRET', '✅', '—', 'Must be 32+ random chars in production'],
                    ['ALLOWED_ORIGINS', '✅', '*', 'CORS origins — set to your frontend URL'],
                    ['WA_VERIFY_TOKEN', 'WhatsApp', '—', 'Meta webhook verification token'],
                    ['WA_APP_SECRET', 'WhatsApp', '—', 'App secret for HMAC validation'],
                    ['WA_PHONE_NUMBER_ID', 'Outbound', '—', 'Meta phone number ID'],
                    ['WA_ACCESS_TOKEN', 'Outbound', '—', 'Meta Business API token'],
                    ['AI_PROVIDER', '—', 'ollama', 'ollama or gemini'],
                    ['OLLAMA_BASE_URL', 'Ollama', 'http://ollama:11434', 'Ollama server endpoint'],
                    ['GEMINI_API_KEY', 'Gemini', '—', 'Google AI API key'],
                    ['GEMINI_MODEL', '—', 'gemini-2.0-flash', 'Google Gemini model name'],
                    ['BOS24_API_TOKEN', 'Real estate', '—', 'BuyOrSell24 data API token'],
                    ['SMTP_HOST', 'Email', '—', 'SMTP server hostname'],
                    ['SMTP_USER', 'Email', '—', 'SMTP username'],
                    ['SMTP_PASSWORD', 'Email', '—', 'SMTP password'],
                    ['APP_COMPANY_ID', '—', '00000000-...0001', 'Single-tenant company UUID'],
                  ].map(([v, r, d, desc]) => (
                    <tr key={v} className="border border-gray-200 hover:bg-gray-50">
                      <td className="p-3"><Code>{v}</Code></td>
                      <td className="p-3 text-center">{r}</td>
                      <td className="p-3 text-gray-500 font-mono text-xs">{d}</td>
                      <td className="p-3 text-gray-600">{desc}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Section>

          {/* Integration guide */}
          <Section id="integrate" title="Integration Guide">
            <div className="space-y-6">
              <div className="border border-gray-200 rounded-lg p-5">
                <h3 className="font-semibold text-gray-900 mb-3">Website Contact Form → Lead</h3>
                <p className="text-sm text-gray-600 mb-3">
                  Until Phase 2 lands, use a service account with agent role:
                </p>
                <Pre>{`// 1. Create contact
POST /api/v1/contacts
Authorization: Bearer <service_account_token>
{ "full_name": "Ahmed", "phone_wa": "+971501234567", "language": "ar" }

// 2. Create lead
POST /api/v1/leads
{ "contact_id": "<uuid_from_step_1>", "source": "web", "stage": "new" }`}</Pre>
              </div>

              <div className="border border-gray-200 rounded-lg p-5">
                <h3 className="font-semibold text-gray-900 mb-3">Zapier / Make.com</h3>
                <p className="text-sm text-gray-600 mb-3">Use the HTTP / REST module with:</p>
                <ul className="text-sm text-gray-600 space-y-1 list-disc list-inside">
                  <li>Auth type: Bearer Token</li>
                  <li>Token: your API key (<Code>sk_live_...</Code>)</li>
                  <li>Base URL: <Code>https://crm.yourcompany.ae/api/v1</Code></li>
                  <li>Create lead: <Code>POST /leads</Code></li>
                </ul>
              </div>

              <div className="border border-gray-200 rounded-lg p-5">
                <h3 className="font-semibold text-gray-900 mb-3">HubSpot / Salesforce Sync</h3>
                <p className="text-sm text-gray-600">
                  Use a middleware (e.g. n8n, custom Lambda) to listen for HubSpot webhook events
                  and push to Masaar CRM API. Outbound sync (Masaar → HubSpot) will be available
                  in Phase 3 via outbound webhooks.
                </p>
              </div>
            </div>
          </Section>

          {/* Roadmap */}
          <Section id="roadmap" title="Roadmap — What's Coming">
            <div className="space-y-4">
              {[
                {
                  phase: 'Phase 1',
                  status: 'complete',
                  title: 'API Key Management',
                  desc: 'Generate, scope, and revoke API keys for external integrations. Available now in Admin Settings.',
                },
                {
                  phase: 'Phase 2',
                  status: 'complete',
                  title: 'Public Lead Endpoint',
                  desc: 'POST /webhooks/leads — submit leads from any external source using API key auth. Auto-creates contact if not found.',
                },
                {
                  phase: 'Phase 3',
                  status: 'complete',
                  title: 'Outbound Webhooks',
                  desc: 'Push events (lead.created, lead.stage_changed, lead.won, payment.received) to registered URLs. HMAC-SHA256 signed. Manage via Admin Settings → Webhooks.',
                },
                {
                  phase: 'Phase 4',
                  status: 'planned',
                  title: 'OAuth 2.0',
                  desc: 'Native Zapier / Make.com integration with OAuth flow. No manual API key management required.',
                },
              ].map(({ phase, status, title, desc }) => (
                <div key={phase} className="flex gap-4">
                  <div className="flex flex-col items-center">
                    <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold shrink-0 ${
                      status === 'complete' ? 'bg-green-500 text-white' :
                      status === 'in_progress' ? 'bg-blue-500 text-white' :
                      'bg-gray-200 text-gray-500'
                    }`}>
                      {status === 'complete' ? '✓' : status === 'in_progress' ? '→' : '○'}
                    </div>
                    <div className="w-0.5 bg-gray-200 flex-1 mt-1" />
                  </div>
                  <div className="pb-6">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-xs font-medium text-gray-400">{phase}</span>
                      <Badge color={status === 'complete' ? 'green' : status === 'in_progress' ? 'blue' : 'gray'}>
                        {status === 'complete' ? 'Live' : status === 'in_progress' ? 'In Progress' : 'Planned'}
                      </Badge>
                    </div>
                    <h4 className="font-semibold text-gray-900">{title}</h4>
                    <p className="text-sm text-gray-600 mt-1">{desc}</p>
                  </div>
                </div>
              ))}
            </div>
          </Section>

          {/* Footer */}
          <footer className="border-t border-gray-200 pt-8 mt-8 text-sm text-gray-500">
            <div className="flex flex-wrap gap-4 justify-between items-center">
              <p>
                Masaar CRM is open source under MIT License.
                Built for UAE businesses.
              </p>
              <div className="flex gap-4">
                <a href="/docs" className="hover:text-gray-900">Swagger UI</a>
                <a href="https://github.com/maidulcu/masaar-crm" target="_blank" className="hover:text-gray-900">GitHub</a>
              </div>
            </div>
          </footer>
        </main>
      </div>
    </div>
  )
}
