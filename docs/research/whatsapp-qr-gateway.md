# WhatsApp Connectivity Strategy: QR Gateway vs the Official Partner Path

**Research brief for Masaar CRM**  
**Date:** 25 October 2026  
**Revision 2:** 26 October 2026 — adds the partner/vendor analysis (§9) and the new WhatsApp account model. The recommendation in §11 now leads with official Embedded Signup through a provider; the QR gateway is demoted to an optional, unsupported community plugin.  
**Status:** Research / recommendation, not an approved decision  
**Scope:** Product, technical, security, legal, economic, and partner-strategy evaluation of how Masaar should connect companies to WhatsApp, including a comparison of a QR-session gateway against the official Cloud API path.

> **Public document.** This file is published in Masaar's public repository. It intentionally contains no confidential vendor terms, customer data, credentials, or internal financial projections. Vendor prices quoted here are publicly listed prices, captured on the date shown, and may change. Vendor names are for evaluation only and imply no endorsement or partnership. Nothing in this document is legal advice; obtain UAE/KSA counsel before relying on the compliance analysis.

**Change log**

| Rev | Date | Change |
|---|---|---|
| 1 | 25 Oct 2026 | Initial brief: QR gateway evaluation. |
| 2 | 26 Oct 2026 | Added §9 (partner strategy, billing roles, provider shortlist) and the 2026 account-model implications. Rewrote §11 to an official-first phased plan. Added §5.7/§5.8 and appendix updates. |

---

## 1. Executive summary

The idea is directionally right, but the implementation and positioning decide whether it is an asset or a liability.

**Verdict (rev 2):** Do **not** build the QR gateway first. The onboarding friction it solves is now solvable legally, in about the same number of clicks, by shipping Meta **Embedded Signup** through a partner who already holds the Meta approval. Start as an ISV/reseller under a Solution Partner (recommended: **360dialog**; alternatives: Twilio, Infobip, Vonage), then climb to **Tech Provider** only when volume justifies owning the Meta app.

**Verdict on the original question (rev 1, retained):** If a gateway is built at all, it belongs in a **separate community plugin**, self-hosted, behind the same provider abstraction. Never as the default, never in a hosted multi-tenant Masaar Cloud, and never with one shared Meta app secret. If it is built, use **WAHA** (Apache-2.0, `GOWS`/whatsmeow engine), not an embedded protocol client.

**Why the revision:** two facts changed the calculus. First, Embedded Signup v5 plus coexistence means the client keeps their number and their Business app, and the only remaining friction is *Masaar's* legal standing — which a partner supplies for €250/month instead of months of engineering. Second, Meta's new account model (GA 23 September 2026) lets a client share one number with several partners, each with its own isolated Messaging account and billing. That removes the switching barrier in both directions, which means the gateway no longer buys lock-in, and it makes "Masaar pays on the client's behalf" a supported structure rather than a workaround.

**Why not build the protocol client into Masaar core first:** it adds a permanent protocol-maintenance burden, puts high-value session credentials in the main application's blast radius, and provides little advantage over an isolated sidecar. WAHA already exposes sessions, QR/pairing, webhooks, media, and multiple engines; Masaar can integrate it in weeks, not months.

**Why not make it the default:** unofficial WhatsApp Web clients violate WhatsApp's Terms of Service. Meta explicitly prohibits reverse engineering, non-personal use, and third-party APIs that replicate its services. Enforcement is opaque and can kill the customer's entire WhatsApp account, not just an API token. A ban is a business-continuity incident for an SME whose customer relationships live on that number.

**Why the original idea looked attractive:** Masaar's current Cloud API setup requires a Meta app, app secret, phone-number ID, permanent token, webhook URL, and verify token. That is not an SME-owner task. A QR link genuinely gets a usable inbox in under a minute. The mistake in the original framing was assuming that only a gateway could deliver that experience; Embedded Signup through a partner delivers it legally, and the client still never sees a developer dashboard.

**Most important recent changes:** Meta has made the official path materially easier. Embedded Signup v5 (released 20 October 2026) enables WhatsApp Business app coexistence by default, so an SME can keep their existing number and app while using the Cloud API. The remaining friction is Masaar's status as a Tech Provider or its partnership with a Solution Partner — not the SME's developer setup. In parallel, the new account model (GA 23 September 2026) decoupled phone numbers from billing, so a client can add Masaar as an additional partner on the same number without migrating it, and each partner is billed through its own Messaging account. Together these make the official path the cheaper, faster, and safer route to the same activation outcome.

---

## 2. What Masaar does today, and where the friction is

The current implementation is a deployment-wide Cloud API integration, not a per-company one:

- `internal/config/config.go` reads `WA_PHONE_NUMBER_ID`, `WA_ACCESS_TOKEN`, `WA_APP_SECRET`, `WA_VERIFY_TOKEN`, `WA_BASE_URL`, and media settings from environment variables.
- `cmd/server/main.go` constructs one `whatsapp.Sender` for the whole deployment.
- Inbound webhooks are verified with `X-Hub-Signature-256`, then routed to the company in `APP_COMPANY_ID` (`internal/api/handler/whatsapp.go`).
- `whatsapp_threads.wa_account_id` stores Meta's `phone_number_id` as text. There is no `wa_accounts` table and no per-company number.
- Broadcasts are template-only (`migrations/0077_whatsapp_broadcasts.sql`), and the inbox settings UI already exposes business hours, auto-reply, assignment, and a 24-hour-window indicator.

For a single-company self-hosted deployment this is workable. For the SME owner it still means:

1. Create/verify a Meta business portfolio.
2. Create a Meta app with the WhatsApp use case.
3. Add and verify a phone number, or onboard an existing WhatsApp Business app number.
4. Generate a permanent/system-user token and find the phone-number ID.
5. Expose a public HTTPS webhook, configure the callback URL and verify token.
6. Set the app secret so inbound payloads are authenticated.
7. Add a payment method before messaging limits or paid messages apply.
8. Approve message templates for anything outside the 24-hour service window.

That is a developer/IT-provider task. Masaar's README already documents this as deployment-wide environment configuration, which confirms the gap rather than solving it.

Meta's official answer is **Embedded Signup**: the SME clicks a button inside Masaar, logs in with Facebook, selects or creates a business portfolio, verifies the number, and grants the app access. The app receives `waba_id`/`messaging_account_ids`, `phone_number_id`, and an exchangeable token code. With a partner-hosted flow, the partner performs the token exchange and Masaar never handles Meta's app secret.

The catch for an open-source, self-hosted product: code exchange requires the app secret and a server-to-server call. You cannot safely ship one shared app secret in every self-hosted instance. The realistic options are:

- **Use a partner's hosted Embedded Signup (recommended).** Masaar is an ISV under a Solution Partner; the partner owns the Meta app and hosts the flow. The self-hosted instance holds only a scoped partner credential, or the deployer supplies their own. See §9.
- Masaar becomes a Tech Provider and runs a central token-exchange/onboarding service (trust and hosting implications; only for the managed tier).
- Each self-hoster creates their own Meta app (friction remains; keep as the "Advanced" path).
- Masaar Cloud only offers Embedded Signup; self-hosted keeps manual configuration.

This constraint is why the partner route matters more than the gateway: it delivers the same zero-setup experience without Masaar shipping a Meta secret, and without violating WhatsApp's terms.

---

## 3. What a QR gateway actually is

A QR link makes Masaar a **linked device** of the customer's WhatsApp account, using the WhatsApp Web multi-device protocol. The customer scans a QR code from their phone (or types an 8-character pairing code if they are on the same phone), and the session receives messages, sends as the account, and may receive recent chat history.

It is not the WhatsApp Business Platform. It is the same mechanism as WhatsApp Web/Desktop, implemented by an unauthorized client.

### 3.1 Libraries and services

| Option | Transport | Notes | License |
|---|---|---|---|
| **whatsmeow** (`go.mau.fi/whatsmeow`) | WebSocket, Go | Native Go implementation of the multi-device protocol; used by many production gateways; no browser | MPL-2.0 |
| **Baileys** (`@whiskeysockets/baileys`) | WebSocket, TypeScript/Node | Large community; v7 still in release-candidate line as of this research; Node sidecar required | MIT |
| **whatsapp-web.js** | Puppeteer/Chromium | Browser per session; highest memory; brittle to UI changes | Apache-2.0 |
| **WPPConnect** | Puppeteer/Chromium | Browser automation with a large API surface; same memory/brittleness profile | Custom/Apache-style |
| **WAHA** | REST + webhooks, multi-engine | Wraps `GOWS` (whatsmeow), `NOWEB` (Baileys), `WEBJS`, `WPP`; sessions, QR, media, HMAC webhooks; free/unlimited since 2026.6.1 | Apache-2.0 |
| **Wuzapi** | REST, Go | Thin REST wrapper around whatsmeow; MIT; smaller project | MIT |
| **Evolution API** | REST, Node | Popular in Brazil; Apache-2.0 plus additional notice/branding conditions | Apache-2.0 + conditions |

**Important licensing note:** Evolution API's additional conditions require a visible notification that Evolution API is used and restrict removal of its branding in frontend components. That is incompatible with a clean MIT self-hosted story unless the conditions are met or a commercial license is obtained. WAHA and Wuzapi do not have that problem.

### 3.2 What the link can and cannot do

| Capability | QR gateway | Cloud API |
|---|---|---|
| 1:1 text and media | Yes | Yes |
| Receive inbound messages | Yes | Yes |
| Send outside a 24-hour window | Technically yes, but this is exactly the behavior that triggers bans and violates policy | No; template required |
| Message templates / approval | No | Yes |
| Broadcasts to consenting segments | Technically possible, very high ban risk; should be disabled | Yes, designed for it |
| Groups | Yes technically; should be disabled for CRM use | No |
| Chat history sync | Recent history, commonly up to ~1 year (varies) | Only via coexistence onboarding, 180 days |
| Messages sent from the phone/app mirrored | Multi-device sync generally delivers them; dedupe required | Only with coexistence (`smb_message_echoes`) |
| Official business profile / green tick | No | Yes, subject to eligibility |
| Click-to-WhatsApp ads attribution | No | Yes |
| Quality rating, messaging tiers, appeals | No | Yes |
| Delivery guarantees / support | No | Platform SLA and support channels |
| Number safety | Account can be banned, including the phone's WhatsApp account | Enforcement typically targets the business messaging asset; still can restrict |

### 3.3 The "30 seconds" claim

The link itself can be fast:

- Open the connect modal, scan the QR with the phone, approve the link: roughly **10-30 seconds**.
- On a mobile browser where scanning your own screen is awkward, use the **pairing code** flow: display an 8-character code, enter it on the phone.
- The inbox can usually send/receive immediately; history and media continue syncing in the background and can take minutes.

So "30 seconds to link" is credible. "30 seconds to a fully synced CRM" is not. Also, the customer must:

- have WhatsApp installed and the number active;
- be under the four-linked-device limit, or unlink an old WhatsApp Web/Desktop session;
- keep the phone reachable for the initial link;
- accept that the CRM now has a linked-device session (visible in WhatsApp's Linked Devices screen and revocable at any time).

Operational realities:

- WhatsApp logs out linked devices if the **phone** is unused for more than about 14 days.
- A user can unlink the CRM at any time; the CRM must detect it and prompt re-link.
- WhatsApp can force a re-link after protocol/client changes.
- New-contact outreach may hit a **reachout timelock** (error 463), where messages to people who have not saved the business number fail even though the session is connected. This is a 2026 enforcement pattern reported widely among unofficial-client operators.
- Idle sessions have received "your account may be at risk / unauthorized tools" warnings, so detection is not purely volume-based.

---

## 4. Build options for Masaar

### 4.1 Option comparison

| Option | Effort | Ongoing maintenance | Isolation / blast radius | Fit for Masaar |
|---|---|---|---|---|
| **BSP/Solution Partner Embedded Signup** | Medium: partner API, hosted or embedded onboarding, webhook per tenant | Low: Meta owns the protocol; partner owns the approval | Official | **Recommended first step** |
| **WAHA sidecar (GOWS)** | Low-medium: REST client, webhook receiver, session mapping, UI | Pin version, watch protocol changes | Strong: separate container and session store; crash does not take down CRM | Optional community plugin only |
| **whatsmeow embedded in Go server** | Medium-high: session manager, store, reconnection, media, per-session isolation | High: protocol breakage affects the main binary | Weak: credentials and sessions share the app process | Do not build |
| **Baileys sidecar** | Medium: second Node runtime, session files, webhook bridge | High: Node dependency, protocol changes, memory leaks reported at scale | Medium | Avoid for a Go codebase |
| **whatsapp-web.js / WPPConnect** | Medium: Chromium per session, screenshots/QR, browser lifecycle | Very high: browser updates, 150-300 MB+ RAM per session, crashes | Medium | Avoid |
| **Managed gateway vendor** | Low integration | Vendor-dependent | Low control; customer data leaves the deployment | Only for a hosted pilot, with disclosure |

WAHA's own scaling guidance (2024, still directionally useful) puts the browser engine at roughly 10 sessions per 2.5 GB / 2.7 CPU and the websocket engine at 50 sessions per 4 GB / 1.5 CPU, or 500 sessions per 30 GB / 3 CPU. That is roughly 60-80 MB per websocket session, versus 200-300 MB for Chromium. For a self-hosted single-company deployment, one session is trivial.

### 4.2 Why WAHA if a gateway is ever built

These points apply only if a gateway is built as a separate community plugin. They are not a reason to prefer a gateway over the official path.

- It is a **separate service**: Masaar's main database does not have to hold WhatsApp session keys; a compromise of the CRM does not automatically hand over every linked WhatsApp account.
- It supports **GOWS** (whatsmeow), so the runtime is the Go implementation without adding Node to Masaar's build.
- It provides the unglamorous parts: session lifecycle, QR/pairing, reconnection, media, webhooks, HMAC, dashboard, multiple engines.
- It is Apache-2.0 and, since 2026.6.1, fully free including multiple sessions.
- It can run as an optional Docker Compose profile, so the default self-hosted install is unchanged.
- If WAHA proves unsuitable, the same Masaar provider interface can later point at a native whatsmeow module or another gateway.

Recommended deployment shape:

```text
Masaar (Go)  ──REST──▶  WAHA (GOWS/whatsmeow)  ──WebSocket──▶  WhatsApp
     ▲                        │
     └──── HMAC webhook ──────┘
```

Put WAHA on a private Docker network, expose only Masaar's webhook path, and never publish its API/dashboard to the internet. Store the WAHA API key as a deployment secret. Use one WAHA session per company.

### 4.3 What not to do

- Do not embed Baileys in the Next.js app or the Go server just to avoid a sidecar. You will inherit a second runtime or a protocol dependency with no isolation.
- Do not use Chromium-based engines for a CRM inbox. The memory cost and browser fragility are not worth the extra features.
- Do not build a native whatsmeow session manager before validating demand. The maintenance cost is permanent; the differentiation over WAHA is small.
- Do not offer gateway broadcasts, bulk sends, group messaging, or status automation. These are the highest-ban-risk behaviors and the hardest to support.

---

## 5. How it fits Masaar's architecture

### 5.1 The real gap: accounts are deployment-wide

The current model cannot support per-company numbers. Before any gateway work, Masaar needs a **WhatsApp account** abstraction, even for Cloud API:

```sql
CREATE TABLE wa_accounts (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id            UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    provider              TEXT NOT NULL CHECK (provider IN ('cloud_api','bsp','waha')),
    bsp_slug              TEXT,          -- e.g. partner identifier for provider='bsp'
    bsp_client_ref        TEXT,          -- partner-side client/account reference
    status                TEXT NOT NULL CHECK (status IN ('pending','linking','connected','degraded','logged_out','banned')),
    phone_e164            TEXT,
    display_name          TEXT,
    waba_id               TEXT,          -- Messaging account ID under the 2026 model
    messaging_account_id  TEXT,          -- required for multi-account sends from H1 2027
    waac_id               TEXT,          -- replaces phone_number_id in paths by H1 2028
    phone_number_id       TEXT,          -- legacy identifier; not the account key
    external_id           TEXT,          -- provider-side account or session identifier
    session_ref           TEXT,          -- gateway session id, if any
    billing_mode          TEXT CHECK (billing_mode IN ('client_card','masaar_credit_line')),
    linked_by             UUID REFERENCES users(id) ON DELETE SET NULL,
    linked_at             TIMESTAMPTZ,
    last_seen_at          TIMESTAMPTZ,
    last_error            TEXT,
    risk_flags            JSONB NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Then add `whatsapp_threads.account_id UUID REFERENCES wa_accounts(id)` (keeping `wa_account_id` for compatibility during migration) and backfill the existing deployment-wide account from `WA_PHONE_NUMBER_ID`. This is also the foundation for multi-company Cloud API support, which the current design lacks.

### 5.2 Provider abstraction

Introduce a small interface rather than branching on provider throughout the handlers:

```go
type Provider interface {
    SendText(ctx context.Context, account *domain.WAAccount, to, body string) (string, error)
    SendMedia(ctx context.Context, account *domain.WAAccount, to string, media Media) (string, error)
    MarkRead(ctx context.Context, account *domain.WAAccount, messageID string) error
    Status(ctx context.Context, account *domain.WAAccount) (AccountStatus, error)
}
```

Implementations:

- `CloudAPIProvider` wraps the existing `internal/whatsapp.Sender` (direct Meta credentials).
- `BSPProvider` calls the partner's API for send, media, and status.
- `WAHAProvider` calls WAHA's REST API (community plugin only).
- Inbound payloads are normalized into one `InboundEvent` before reaching `WhatsAppRepo` and the WebSocket hub.

Onboarding is a separate concern from sending, because it differs per provider and must never expose a server secret to the browser:

```go
type Onboarding interface {
    Start(ctx context.Context, account *domain.WAAccount) (Launch, error) // URL or embedded-signup payload
    Complete(ctx context.Context, req CallbackRequest) (*domain.WAAccount, error)
}
```

Keep the existing Meta webhook handler intact and add provider-specific receivers:

```text
POST /webhooks/whatsapp/:provider        (Meta direct or BSP; signature verified per provider)
POST /webhooks/whatsapp-gateway/:accountID   (gateway plugin; HMAC, no user session)
```

Resolve the account by ID, derive the company from the account row (never from the payload), and feed the same storage/notification path as Meta messages.

### 5.3 Gateway link flow (community plugin only)

1. Admin opens **Settings → WhatsApp → Connect**.
2. `POST /api/v1/whatsapp/accounts` creates a `pending` account with `provider=waha`.
3. `POST /api/v1/whatsapp/accounts/:id/link` asks WAHA to start a session and subscribe to its QR events.
4. The browser opens an SSE stream (simpler than WebSocket for one-way QR) or reuses the existing `internal/ws` hub:
   - `GET /api/v1/whatsapp/accounts/:id/link/events`
   - Events: `qr`, `pairing_code`, `status`, `linked`, `error`, `expired`.
5. The QR is rendered client-side with a QR library. Do not store the QR payload after expiry; keep it in Redis or memory for the short link window.
6. On `WORKING`, update the account (`connected`, phone, display name, `linked_at`), audit-log the link, and emit a company-scoped WebSocket event.
7. On unlink, call WAHA logout, delete the session, purge cached data, set `logged_out`, and audit-log.

Security requirements for the link flow:

- Admin role only, plus step-up 2FA if the company requires it.
- Link session bound to the authenticated user and company; one-time use; short TTL.
- QR and pairing codes never written to application logs, analytics, or error trackers.
- TLS only; `SameSite` cookies; CSRF protection on the POST that starts the link.
- Rate-limit link attempts per company and per user.

### 5.4 Outbound and inbound behavior

- Route `send-message` and `send-media` through the provider for the thread's account.
- Disable `send-template` and broadcasts for gateway accounts at the API and UI layers, not just by hiding buttons.
- Respect opt-out (`internal/whatsapp/optout.go`) and consent for all providers.
- For gateway accounts, keep the 24-hour-window indicator as guidance, but do not pretend it is enforced by the provider. Sending outside it is a policy/ban risk.
- Deduplicate messages by provider message ID; multi-device echoes and webhook retries will produce duplicates.
- Mirror messages sent from the phone/app where the gateway delivers them, so the inbox stays coherent.
- Media: WAHA can return media URLs or base64; download into the existing `WAMediaDir` pipeline. Preserve the SSRF and size limits already used for Meta media.
- Auto-reply: warn that the WhatsApp Business app may also auto-reply, causing duplicate responses. Provide a checklist to disable app-side auto-replies.

### 5.5 UI positioning

Settings should present a clear choice, not a hidden toggle:

```text
Connect WhatsApp

[ Official — recommended ]                  [ Advanced ]
Keep your number and your Business app      Use your own Meta app
No developer setup                          For IT teams
Templates and broadcasts                    You manage the Meta credentials
Meta bills per message                      Same compliance, no intermediary
```

The official option must be the visual default. "Advanced" holds the bring-your-own-credentials path; it is not a fallback label for the gateway.

If the community gateway plugin is installed, it appears only inside Advanced, with a permanent warning label — never as a peer of the official option.

Required disclosure before a QR is shown (plugin only):

> This connects Masaar as an unofficial WhatsApp Web device. It is not endorsed by WhatsApp or Meta, violates WhatsApp's Terms of Service, and can result in your number being banned. Use a dedicated business number you can afford to lose. Masaar cannot recover a banned number or restore its chats.

Add a checkbox (not pre-checked) and a link to a full risk page. Do not use WhatsApp logos or imply partnership. "WhatsApp Web link (unofficial)" is clearer than "WhatsApp API".

For gateway accounts, show a persistent "Unofficial connection" badge in the inbox and a "Move to official" action.

### 5.6 Health, operations, and support

- Per-session state machine with backoff and jitter; cap reconnect attempts to avoid reconnect storms.
- One goroutine/process failure must not take down other sessions. If using WAHA, that isolation is mostly free; if native, recover per handler.
- Metrics: connected/disconnected, reconnect count, send failures, webhook lag, media failures, ban/warning signals, session age.
- Alerts: session logged out, banned, repeatedly failing, phone inactive.
- Runbook: re-link, unlink old devices, change number, export chats before migration, what to do after a ban.
- Support boundary: best-effort for gateway; no guarantee of delivery; no recovery of banned accounts.
- Kill switch: `WA_GATEWAY_ENABLED=false` disables new links and pauses sends without a deploy.

### 5.7 Migration to the official tier

The upgrade path must be technically real, not marketing copy.

1. Customer decides to move (prompted by a limit, a need for templates/broadcasts, a verification requirement, or a procurement policy).
2. If the number is on the **WhatsApp Business app**, use coexistence onboarding through the partner's Embedded Signup. Meta unlinks all companion devices during onboarding, including any gateway session.
3. If the number is on the consumer app, move it to the WhatsApp Business app first, or use a new number for the Cloud API. Under the 2026 account model, phone-number migration between partners is no longer supported; the client shares the WhatsApp account (WAAC) instead, which is additive and does not interrupt conversations.
4. Onboard through the partner and create a `wa_accounts` row (`provider='bsp'` or `cloud_api`) with `waba_id`/`messaging_account_id`. Re-parent threads to it, or keep a `canonical_account_id` so history survives a provider switch.
5. Keep the CRM's own message history. Do not promise that WhatsApp app history survives a ban or a provider change.
6. Provide a "dedicated number" checklist before the customer starts: number ownership, eSIM/second line, no banking OTPs, no personal chats.

Two migration details matter for expectations:

- **Templates do not follow the number.** They belong to a Messaging account. Migrating them resets quality rating and messaging tier, with roughly a 2–4 week ramp.
- **Adding Masaar does not require removing the incumbent.** Under the new model the client can keep their previous provider on the same number while Masaar is onboarded. That is a sales advantage, and it also means the client can reverse the decision just as easily.

Do not promise that a number used with an unofficial client will be accepted into coexistence. Meta's eligibility depends on account tenure and messaging quality, and the unofficial session may already have affected the account's standing. Validate with a pilot before advertising it.

Coexistence also has product limitations that must be reflected in the upgrade copy:

- Official Business Account / blue badge is not supported for coexistence numbers; Meta Verified for Business is the alternative.
- The calling API is not supported.
- Group chats are not synchronized to the API.
- Broadcast lists are disabled, and the Business app's messaging tools (catalog, labels, quick replies, etc.) are not available through the API.
- The Business app must stay installed and opened roughly every 13 days, or the account disconnects.
- All companion devices are unlinked during onboarding and must be re-linked afterward.
- Throughput for coexist numbers is fixed at 20 messages per second.
- Meta gives the partner 24 hours to synchronize contacts/history; missing that window forces an offboard/re-onboard.

The upgrade story should be "keep your number and your app, add official messaging", not "everything in the app becomes an API feature".

### 5.8 Effort estimate: shared foundation and two tracks

The account model and provider interface are shared work, estimated at 8–12 days. After that:

| Track | Additional work | Total |
|---|---|---|
| **Official: partner Embedded Signup** | Hosted signup launch, callback, webhook normalization, UI, tests | **6–8 weeks** |
| **Gateway: WAHA sidecar** | Session lifecycle, QR/pairing UI, HMAC webhook, health, kill switch | **6–8 weeks** |

Gateway workstreams, for reference:

| Workstream | Estimate (1 engineer, plus design/QA) |
|---|---|
| `wa_accounts` schema, migration, backfill | 3-5 days |
| Provider abstraction + WAHA client | 5-8 days |
| Link flow (API, SSE/WS, QR/pairing UI) | 5-8 days |
| Inbound webhook normalization + media | 5-8 days |
| Settings UI, disclosures, badges, disable broadcasts | 4-6 days |
| Health/metrics/runbook/kill switch | 3-5 days |
| Tests (tenant isolation, duplicate handling, unlink, ban simulation) | 4-6 days |
| Docs and Compose profile | 2-3 days |

Roughly **6-8 weeks** for a production-quality WAHA connector, assuming the provider interface and account model are new. A native whatsmeow module would add several weeks and permanent maintenance. The official track is comparable in build effort but has no protocol maintenance and carries no ban liability; that is the main argument for doing it first.

---

## 6. Security and privacy

This section covers the **unofficial gateway**, where a linked-device session is a full credential to the customer's WhatsApp account. The official path has a different profile: there is no session credential to protect, but message content transits Meta and, depending on the partner, a second processor — which is a contract and PDPL question, not a session-theft question.

A linked-device session is not an API key. It is a full credential to the customer's WhatsApp account:

- It can read messages and history that WhatsApp syncs to it.
- It can send as the customer.
- It can see contacts and group metadata.
- It can mark messages read and affect presence.
- If stolen, it can be used from anywhere until the user unlinks it or the session is invalidated.

### 6.1 Threat model

| Threat | Impact | Mitigation |
|---|---|---|
| CRM database or backup breach exposes session keys | Full WhatsApp account takeover, OTP/banking risk, reputational catastrophe | Keep session store out of the CRM DB (WAHA volume), encrypt at rest, restrict access, encrypt backups, rotate/delete on unlink |
| Multi-tenant isolation bug | One company's session used by another | Account-scoped webhooks, tenant derived from account row, integration tests for cross-tenant access |
| QR/link endpoint abuse | Attacker links their own number to a victim company, or captures a QR | Admin + 2FA, one-time link token, short TTL, TLS, no QR logging, bind to user/company |
| Malicious inbound payload | Message injection, spoofed company | HMAC webhook secret, resolve company from account, validate sender/number, rate-limit |
| Employee links personal number | Employer sees personal chats; PDPL/labour issues | Require company-owned number, informed consent, UI warning, admin approval, audit |
| History sync pulls unrelated personal chats | Unnecessary sensitive data, retention/DSAR burden | Disclose, minimize retention, allow deletion, consider not persisting history by default |
| Session theft via server compromise | Impersonation, data exfiltration | Network isolation, least privilege, secrets manager, no shell access to WAHA, OS hardening |
| Ban/account loss | Business continuity incident | Dedicated number, no broadcasts, no cold outreach, health monitoring, migration plan |

The session store is the highest-value asset in this feature. WAHA's local session storage is not a secrets vault; treat the whole volume as a credential. Put it on an encrypted disk, restrict it to the gateway container, exclude it from general backups, and destroy it on unlink. Masaar's existing `internal/totp.Box` (AES-GCM) is suitable for sealing small account-level secrets; it is not a substitute for encrypting the session store or for a KMS-backed design.

### 6.2 Multi-tenant blast radius

This is the decisive architectural point:

- **Self-hosted, one company per deployment:** a compromise affects one company. The risk is comparable to the company running its own WhatsApp Web. This is acceptable with disclosure.
- **Hosted multi-tenant Masaar Cloud:** a compromise affects every linked customer. The session store becomes a high-value aggregation point, and the operator becomes a processor of all customers' WhatsApp content. This is a much higher bar: per-tenant KMS keys, HSM, strict admin separation, SOC 2-style controls, incident response, cyber insurance, and likely a legal review. For a community project, it is not worth it.

Recommendation: **gateway only in self-hosted/community deployments, not in Masaar Cloud.** If a hosted pilot is unavoidable, use a separate gateway vendor and make the customer contractually the controller of the WhatsApp account, with explicit consent and no Masaar access to session keys.

### 6.3 Privacy / PDPL

Masaar markets itself as PDPL-compliant. Linking a WhatsApp account and syncing chats adds:

- A new category of personal data (message content, contacts, media) in the CRM.
- Potential employee monitoring (if a personal number is linked).
- Cross-border transfer questions if the gateway or backups are outside the UAE/KSA.
- Data subject access/deletion obligations for synced history.
- A need for a DPIA and updated privacy notice.

Minimum controls:

- Document the processing purpose and legal basis; update the privacy policy and DPA.
- Prefer company-owned numbers; require admin attestation.
- Default to minimal history retention; provide a "delete synced history" action.
- Include gateway accounts in the existing PDPL export/delete flows.
- Log link/unlink/send events in the audit log.
- Do not use synced content for AI training; keep the existing local-Ollama/PII posture.
- Have an incident response plan specifically for session compromise.

---

## 7. Legal and platform risk

This section applies to the **unofficial gateway**. The official path is governed by Meta's platform terms through the partner agreement, which shifts the analysis from "are we violating the ToS?" to "are we meeting the partner contract, the WhatsApp Business Messaging Policy, and PDPL?" — a lower but non-zero bar.

This is not a "gray area" in WhatsApp's terms. The consumer Terms of Service prohibit, among other things:

- reverse engineering, altering, or extracting code from the service;
- sending bulk or automated messages, and non-personal use unless authorized;
- distributing the service over a network where multiple devices could use it, except through authorized tools;
- creating software or APIs that function substantially the same as WhatsApp's services and offering them to third parties without authorization.

The WhatsApp Business Messaging Policy adds that businesses may only contact people with prior opt-in and must follow the 24-hour service-window/template rules. It also says WhatsApp may limit or remove access for services that message people at scale in an unauthorized manner.

Enforcement reality:

- Meta displays "your account may be at risk / recent activity indicates unauthorized tools" warnings to linked-device users, including low-volume and idle sessions (whatsmeow issue #810, also reported with Baileys and whatsapp-web.js).
- Bans can be permanent, with no meaningful appeal for unofficial clients.
- Reporting is anecdotal and sometimes vendor-motivated; there is no reliable public ban-rate dataset. Some vendors claim unofficial sessions survive only weeks; others run large fleets for years. Treat session lifetime as uncertain and measure it yourself.
- WhatsApp's August 2026 ban wave reportedly hit thousands of accounts, including legitimate users. Even if many were false positives, it demonstrates that enforcement is opaque and that a number can disappear without warning.

Legal exposure differs by distribution model:

- **Self-hosted community plugin/sidecar:** the customer runs the software on their own infrastructure and links their own account. Masaar's exposure is lower, but it still ships and documents a ToS-violating integration. Use clear disclaimers, no warranty, and no claim of WhatsApp affiliation.
- **Hosted service:** Masaar operates the gateway and benefits commercially from the integration. This is materially higher risk: potential cease-and-desist, contract claims, app-store/platform distribution issues, enterprise procurement rejection, and reputational damage. Get counsel before launching.
- **BSP/Tech Provider official path:** the lowest legal risk and the best enterprise story, but requires Meta approval and ongoing compliance.

Also consider UAE/KSA-specific rules:

- UAE PDPL (Federal Decree-Law No. 45 of 2021) and Saudi PDPL (Royal Decree M/19 of 2021, in force from 2024) impose consent, notice, security, and transfer obligations. A session store full of message content is squarely in scope.
- Unsolicited marketing and spam rules apply independently of WhatsApp's terms (TDRA/CST and consumer-protection rules).
- WhatsApp voice/video calling restrictions in the UAE are a separate issue; do not expand the gateway into calling.

Branding: use "WhatsApp Web link (unofficial)" and a clear "not affiliated with WhatsApp or Meta" notice. Do not use WhatsApp's logo or green branding in a way that implies endorsement. Review WhatsApp's Brand Guidelines.

---

## 8. Economics

Meta moved to per-message pricing in July 2025 and, on **1 October 2026**, began charging for service messages after a free tier. This is the most important economic change for this decision.

UAE rates effective 1 October 2026 (USD; AED where published):

| Category | USD per delivered message | Approx. AED |
|---|---|---|
| Marketing template | $0.0576 | AED 0.2115 |
| Utility template | $0.0157 | AED 0.0576 |
| Authentication | $0.0157 | AED 0.0576 |
| Service (non-template reply) | $0.0157, after 1,000 free per business number per month | AED 0.0576 |
| Inbound from customer | Free | Free |

Other important mechanics:

- The 1,000 free service messages are per business phone number per month, no rollover.
- Utility templates sent inside an open service window are now billed too (they were free from July 2025 to September 2026).
- Click-to-WhatsApp ad conversations can open a free entry point window (up to 7 days per Meta docs), making messages free during that window.
- No charge for messages from users to the business.

Illustrative monthly costs for a UAE SME:

| Scenario | Cloud API cost |
|---|---|
| 800 service replies | $0 (inside free tier) |
| 2,000 service replies | 1,000 × $0 + 1,000 × $0.0157 = **$15.70** |
| 5,000 service replies | 4,000 × $0.0157 = **$62.80** |
| 5,000 marketing templates | 5,000 × $0.0576 = **$288.00** |
| 5,000 utility templates | 5,000 × $0.0157 = **$78.50** |

So the gateway's cost advantage is real for high-volume service and broadcast use, but the absolute amounts are modest for a small SME. The bigger value is **activation speed and avoiding Meta setup**, not saving $15-60 per month.

The hidden costs of the gateway:

- Support for re-links, bans, media failures, and protocol changes.
- Lost business when a number is banned.
- Engineering maintenance (even with WAHA, you own the integration and version pinning).
- Legal/compliance review and incident response.
- Potential enterprise deal loss because procurement rejects unofficial integrations.

A single ban incident can easily cost more in support and customer churn than a year of Cloud API fees for a small tenant. Do not position the gateway as a way to escape WhatsApp pricing; position it as a way to start immediately, with a clear upgrade path.

---

## 9. Partner strategy: connect, become a Tech Provider, or become a Solution Partner

"Do we need to become a vendor?" conflates two different roles. Masaar is already a vendor *to its clients*. The open question is whether it becomes a **partner of Meta**, and at which tier. Meta defines the tiers, and — critically — defines who is allowed to bill for API usage.

### 9.1 The four tiers

| Role | What you get | Who bills the client for messaging | Effort | When to choose |
|---|---|---|---|---|
| **ISV / reseller under a Solution Partner** | The partner's Meta approval, app, and onboarding flow; you call their API | The partner bills you; you bill the client for software, and optionally for usage via the partner's credit line | Days to weeks; commercial agreement | **Now** |
| **Tech Provider** | Your own Meta app and Embedded Signup UX; full WhatsApp services to onboarded clients | Meta bills the client directly; you bill only for software | Business verification + App Review (2 screen recordings) + access verification | When volume or product control justifies it |
| **Tech Partner** | Tech Provider plus Meta Business Partner status and accelerator eligibility | Same as Tech Provider | Above + partner application; months | When you want Meta co-selling |
| **Solution Partner (ex-BSP)** | Credit line, direct invoicing for API usage, partner-led business verification | You | Lengthy (Meta's own wording); support org, scale, financial liability | Not a 2026 decision |

The billing distinction is the one that matters for the "buy it on their behalf" question, and Meta states it plainly:

- **Solution Partners** have credit lines "which can be extended to clients who they bring on board, thus removing the need for those clients to enter their own payment method," and "can directly invoice their clients for the WhatsApp Business Platform services."
- **Tech Providers** cannot. "Clients onboarded by Tech Providers must provide their own payment method after onboarding is complete. Meta will then bill these clients for API usage, and the Tech Provider will bill for other services."

So Masaar cannot simultaneously be a Tech Provider and the party Meta bills for its clients' messaging. To be the payer, there must be a Solution Partner underneath.

### 9.2 What the client actually has to do

| Today (deployment-wide env config) | With Embedded Signup through a partner |
|---|---|
| Create/verify a Meta business portfolio | Log in with Facebook; select or create a portfolio |
| Create a Meta app with the WhatsApp use case | — |
| Add/verify a phone number | Confirm the number and enter the SMS OTP |
| Generate a system-user token; find the phone-number ID | — |
| Expose a public HTTPS webhook; set callback + verify token | — |
| Set the app secret for inbound authentication | — |
| Add a payment method | Add a card, or use the partner's credit line |
| Get templates approved | Unchanged: template review still applies |

The developer setup disappears. What remains is **business verification** — trade licence, address, website. Top-tier Solution Partners can submit that on the client's behalf (partner-led business verification), which speeds it up, but the client still supplies the documents and consents. It is the client's identity, and it is what unlocks higher messaging limits and the verified badge.

This is the crux of the revision: the QR gateway's core selling point was "your client gets an inbox in 30 seconds without touching Meta's developer console." A partner-hosted Embedded Signup flow delivers the same experience, legally, with the client owning the number and the account.

### 9.3 The three billing models

| Model | Paid to Meta/BSP by | Invoiced to client by | Risk Masaar carries |
|---|---|---|---|
| **Pass-through** | Client's own card | Masaar (software only) | None on messaging spend |
| **Credit line + rebill** | Masaar, via the partner's credit line | Masaar (software + usage) | Credit risk, FX, collections, bad debt |
| **Direct Solution Partner** | Masaar | Masaar | Full messaging P&L and regulatory exposure |

Recommendation: **default to pass-through**, offer credit-line rebilling as a paid convenience. Do not become a Solution Partner for this.

Two practical constraints on the credit-line model:

- In the new account model, "if you attach your credit line to a client-owned Messaging account, that Messaging account cannot be shared with another partner." Using your credit line therefore removes the client's ability to add another partner — a lock-in that clients and their advisors may notice. Disclose it.
- Rebilling messaging in the UAE means VAT invoices, foreign-exchange exposure on USD rates, and collections. Price the service fee to cover it; do not rely on a per-message spread, because partners now pass Meta's rates through at zero markup.

### 9.4 The 2026 account model changes the lock-in math

Meta split the old WABA into two objects, generally available from **23 September 2026**:

- **WhatsApp account (WAAC)** — owns the phone number, business profile, and catalog. Always owned by the client. Can be shared with several partners.
- **Messaging account** — owns templates, billing, payment method, and webhook subscriptions. One per partner, with the legacy WABA ID retained.

Consequences for Masaar:

1. **Additive onboarding.** A client can add Masaar as a partner on a number that a previous agency or BSP already manages. No number migration, no downtime, no new number. This removes the biggest objection in a competitive deal.
2. **Per-partner billing isolation.** Masaar's Messaging account on that number is billed separately from every other partner's. This is what makes "Masaar pays" structurally clean.
3. **Lower lock-in, in both directions.** Number migration between partners is no longer supported because it is no longer needed — the client simply shares the WAAC. Your clients can leave you just as easily as they can join.
4. **Templates are the remaining stickiness.** Templates belong to a Messaging account and do not move automatically; the Template Migration API resets quality rating and messaging tier, with a 2–4 week ramp. That is a real switching cost, but a modest one.
5. **Engineering deadlines.** `messaging_account_id` becomes required on Messages API calls in multi-Messaging-account setups from **Meta phase 2 (H1 2027)**, and from **Meta phase 3 (H1 2028)** every endpoint that targets a phone-number ID must target the **WAAC ID** instead. Masaar currently keys `whatsapp_threads.wa_account_id` on Meta's `phone_number_id`; that assumption has a hard expiry date.

| Meta phase | Date | What changes | Masaar action |
|---|---|---|---|
| 1 — GA | From 23 Sep 2026, all businesses by mid-Oct 2026 | Automatic migration; per-partner Messaging accounts; `messaging_account_id` optional | None required; plan the schema |
| 2 — New Graph API | H1 2027 | `messaging_account_id` required when a token can reach multiple Messaging accounts; WAAC IDs appear | Store and send `messaging_account_id` |
| 3 — Mandatory | H1 2028 | Phone-number IDs no longer valid in endpoint paths | Migrate to WAAC IDs |

### 9.5 Provider shortlist and what each forces on you

| Provider | Tier | Own Meta app / Tech Provider? | Billing | Notes |
|---|---|---|---|---|
| **360dialog** | Solution Partner | Partner hosts Embedded Signup. Vendor documentation does not state a blanket rule that this removes the need for your own Meta app, so confirm contractually | Credit lines available; list prices Oct 2026: €250/mo + €49/channel, €500 + €25, €1,000 + €15 | Best fit for an ISV/reseller; markets "no 360dialog in the way" in the signup flow |
| **Twilio** | Solution Partner | **Yes** — Independent Tech Provider, App Review, then attach Twilio's Partner Solution | Per-message platform fee on top of Meta | Twilio's own docs state that Meta requires a visible "working with Twilio" notice in the flow, and that it cannot be removed |
| **Vonage** | Solution Partner | Tech Provider enrolment, but offers **Hosted Embedded Signup** so you do not build the flow | Quoted | Least frontend work of the majors |
| **Infobip** | Solution Partner | Tech Provider program documented | Quoted; enterprise pricing | Strong regional presence and support |
| **Gupshup, Bird, Telnyx** | Solution/Tech | Varies by program | Varies | Evaluate only if the shortlist fails |
| **Regional Gulf providers** | Varies | Varies | Varies | Better Arabic SLA; verify the actual Meta program status |

Due diligence, in writing, before signing:

1. Does your hosted/managed Embedded Signup require us to hold our own Meta app and Tech Provider approval?
2. Who is the bill-to, and can we extend a credit line to our clients?
3. Do you support **coexistence** today, in the UAE/KSA, and on which signup version?
4. What are the Tech Provider onboarding limits, and how do we scale past them?
5. Can a self-hosted deployment hold only a scoped credential — never a Meta app secret?
6. Whose brand appears inside the Embedded Signup flow, and can it be removed?

Verify the partner's status in Meta's Business Partners directory. Meta has no "reseller" tier; ask for the actual program status in writing.

### 9.6 Unit economics

Partner channel fees are a fixed cost per connected number, and Meta's message fees pass through at zero markup. The detailed per-tier comparison and the pricing conclusion are maintained in the internal commercial note, not in this public brief.

### 9.7 Open-core packaging

Masaar is MIT-licensed and self-hosted. A commercial dependency must not break that promise:

- **Community edition:** the BSP adapter accepts the deployer's **own** partner API key, or the `cloud_api` adapter accepts their own Meta credentials. No dependency on Masaar's commercial account, and no shared secret in the repository.
- **Paid/hosted tier:** Masaar-managed connection, one invoice, optional credit line. This is where the partner contract and margin live.
- **If Masaar later becomes a Tech Provider:** run a small token-exchange broker for the managed tier only. Self-hosted users keep the BYO path, so a broker outage cannot break their inbox.

This split is also the answer to the OSS app-secret problem: a shared Meta app secret must never ship in the binary, and a self-hosted deployment should never need one to use the official API.

### 9.8 Escalation to Tech Provider

The commercial thresholds for moving from reseller to Tech Provider are maintained internally; the technical path is in §11.5. The move is triggered by volume, a blocked capability, conversion friction, or a contractual requirement — not by prestige.

---

## 10. Competitive context

- **Infrastructure:** WAHA, Evolution API, Wuzapi, Baileys, whatsapp-web.js, WPPConnect. These are plumbing, not CRM differentiation.
- **Local/regional SaaS:** many "WhatsApp API" vendors in the Gulf and South Asia resell unofficial gateways or BSP services; SMEs often cannot tell the difference. Transparency about official vs unofficial is a trust differentiator.
- **Open-source CRMs:** Chatwoot, Odoo, EspoCRM, Twenty, and others support official WhatsApp via Cloud API or BSPs; some communities add unofficial gateways through Evolution/WAHA. Masaar's edge is Arabic-first real-estate workflow and self-hosting, not the gateway itself.
- **BSPs:** 360dialog, Twilio, Infobip, Gupshup, and others offer Embedded Signup and coexistence. They remove Meta setup friction but add fees and a dependency.

Implication: official connectivity is now **table stakes**, and a partner removes the setup friction that used to differentiate. The gateway is a niche fallback for self-hosters who accept the risk, not a moat. The moat is the real-estate workflow, Arabic UX, PDPL posture, and the quality of the upgrade path.

---

## 11. Recommendation and phased plan

This plan replaces the rev 1 plan, which put a WAHA connector first. The gateway is now a documented fallback, not the critical path. Nothing here is approved for coding; it is the proposal for review.

### 11.1 Product framing: four tiers

| Tier | Connection | Target | Promise |
|---|---|---|---|
| **Lite** | `wa.me` click-to-chat + manual logging | Zero-setup users, cautious SMEs | No ban risk, no inbox sync |
| **Connected** (default) | Cloud API via partner Embedded Signup, coexistence where eligible | Every SME | A few clicks; official; no ban risk; no developer setup |
| **Direct** | Cloud API with the deployer's own Meta app and credentials | Enterprises with data-control or procurement requirements | No intermediary; client owns the Meta relationship |
| **Community gateway** (optional, unsupported) | QR-linked WhatsApp Web via a WAHA sidecar | Self-hosters who explicitly accept the risk | Instant link; 1:1 only; unofficial; may be banned |

The differentiation is not the connector. It is the real-estate workflow, Arabic UX, PDPL posture, and the fact that a non-technical owner can be live on WhatsApp in minutes without a developer.

### 11.2 Phase 0: partner selection and validation (2 weeks)

- Hold commercial calls with **360dialog** and one alternative (Twilio or Infobip). Get written answers to the six questions in §9.5, especially the role question and the coexistence question.
- Interview 10–15 UAE/KSA SMEs and existing Masaar users. Ask where they stopped in Meta setup, whether they would pay for a managed connection, and whether they need coexistence.
- Instrument the current onboarding funnel: how many deployments configure Cloud API, how long it takes, which step they abandon.
- Run a **concierge pilot**: manually onboard 5–10 users through the chosen partner's hosted flow. Measure time per step, drop-off, first message, and support minutes.
- Do not write production code until the partner role is confirmed in writing.

**Go/no-go gates:**

- Partner confirms, in writing, which Meta role Masaar must hold for the hosted flow.
- Median time from "Connect" to first message under 5 minutes.
- First-attempt completion above 70%, with no developer assistance.
- Coexistence confirmed for UAE/KSA, or explicitly deferred with a date.
- Channel cost fits the intended plan price without a metered WhatsApp add-on.
- At least 30% of interviewed prospects say the current Meta setup is a blocker they would pay to avoid.

### 11.3 Phase 1: account model and official connect MVP (6–8 weeks)

- Add `wa_accounts`, account-scoped threads, and the provider interface (`cloud_api`, `bsp`).
- Store provider, external account IDs (`waba_id`, `messaging_account_id`, `phone_number_id`), and a nullable `waac_id`.
- Implement the hosted/embedded signup launch and callback; create the account on completion.
- Normalize BSP webhooks into the existing `InboundEvent` path; handle media.
- Settings UI: **Connect WhatsApp**, connection status, disconnect, and a clear "who pays Meta" statement.
- Capability gating: show only what the plan and provider support (templates, broadcasts, coexistence).
- Backfill the existing deployment-wide Cloud API configuration into an account row.
- Tests: tenant isolation, webhook idempotency, disconnect/reconnect, missing-permission errors.
- Docs: self-hosted BYO partner key, plus the managed path.

| Workstream | Estimate |
|---|---|
| `wa_accounts` schema, migration, backfill | 3–5 days |
| Provider interface + BSP client | 5–7 days |
| Hosted signup launch + callback | 3–4 days |
| Inbound normalization + media | 4–6 days |
| Settings UI and status | 4–6 days |
| Capability gating and disclosures | 2–3 days |
| Tests | 4–5 days |
| Docs | 2–3 days |

Roughly **6–8 weeks** — comparable to the WAHA connector in build effort, but with no protocol maintenance, no session store to protect, and no ban liability. The advantage is operational and legal, not in the initial sprint count.

### 11.4 Phase 2: coexistence and account lifecycle (3–4 weeks)

- Coexistence onboarding: companion-device unlink, 13-day Business app reminder, 24-hour history-sync window.
- Migration tooling from the legacy env-var configuration to account rows, with a rollback path.
- `messaging_account_id` stored and sent on outbound, ahead of the H1 2027 requirement.
- Template management and the 24-hour window indicator wired to the account, not the deployment.
- Billing transparency: show the client which entity charges for messages and which charges for software.
- PDPL export/delete coverage for message content, including media.

### 11.5 Phase 3: Tech Provider upgrade (only on a trigger)

Trigger criteria are in §9.8. If triggered:

- Business verification, App Review (two screen recordings), access verification.
- Central token-exchange broker for the managed tier; `appsecret_proof`; per-client business tokens.
- Keep the Solution Partner as a multi-partner solution if a credit line is still wanted.
- Reuse the same `Provider` interface; this should be an adapter, not a rewrite.

### 11.6 Optional: community gateway plugin (separate repository)

Only after the official path ships, and only if demand persists:

- WAHA as a self-hosted sidecar, 1:1 messaging only.
- Explicit "unofficial, may be banned" disclosure and no support SLA.
- No broadcasts, groups, status automation, or bulk sends.
- Never in the default Compose file; never with a shared secret.
- Keep it out of the main repository so the product's official posture is unambiguous.

### 11.7 What not to do

- Do not make a QR gateway the default for new deployments.
- Do not ship one shared Meta app secret in an open-source binary.
- Do not become a Solution Partner in 2026.
- Do not assume a partner's brand will be invisible in the activation flow; verify before signing.
- Do not advertise "official WhatsApp API", imply Meta endorsement, or use Meta/WhatsApp marks.
- Do not promise ban recovery, coexistence eligibility, or unlimited messaging.
- Do not enable broadcasts, bulk sends, groups, or status automation.
- Do not recommend the customer's primary or personal number.

---

## 12. Metrics and decision gates

**Official path (primary):**

| Metric | Why it matters | Suggested guardrail |
|---|---|---|
| Connect start → success | Core UX promise | >70% first attempt |
| Median time to first message | "Live in minutes" claim | <5 minutes |
| Drop-off step | Finds the real friction | Track the top abandoned step |
| Business verification completion | Unlocks limits and badge | >60% within 30 days |
| 7/30-day retained connected accounts | Real usage, not novelty | >60% at 30 days |
| Template approval pass rate | Onboarding quality | >80% first submission |
| Delivery failure rate | Reliability | <2% of sends |
| Support minutes per onboarding | Unit economics | <20 minutes |
| Upgrade to Direct / enterprise | Business model | Track separately |

**Community gateway plugin (only if built):**

| Metric | Why it matters | Suggested guardrail |
|---|---|---|
| Link start → success | Core UX promise | >80% first attempt |
| Median time to link | Instant-onboarding claim | <90 seconds |
| Session uptime | Reliability | >99% for connected accounts |
| Re-link rate | Support burden | <10% per month |
| Warning/ban rate | Safety | <5% per 90 days; pause rollout above |
| Support minutes per account | Unit economics | <30 min/month |
| Churn after ban | Brand damage | Interview every case |

Roll out the official path behind a feature flag, starting with a canary group of deployments. Keep a per-account kill switch for outbound sends and a global switch for the connect flow, so a partner incident can be contained without a deploy. If delivery failure or connect failure exceeds the guardrail, pause new connections, keep existing accounts working, and fall back to manual Cloud API configuration.

---

## 13. Open questions

1. **Hosted vs self-hosted roadmap:** Is Masaar Cloud planned? If yes, decide whether the managed WhatsApp connection is part of it and who contracts with the partner.
2. **Partner and role:** Which partner is chosen, and does their hosted flow require Masaar to hold its own Meta app and Tech Provider approval? This must be answered in writing before Phase 1.
3. **Billing default:** Pass-through (client card) or credit-line rebilling? The second adds VAT, FX, and collections obligations, and attaching Masaar's credit line prevents the client from sharing that Messaging account with another partner.
4. **Coexistence:** Is coexistence available for UAE/KSA businesses through the chosen partner today, on which Embedded Signup version, and with which limitations? Do not advertise it until tested.
5. **Branding:** Whose name appears inside the Embedded Signup flow, and can it be removed or co-branded? This affects conversion and the enterprise story.
6. **Community edition:** Does the self-hosted build accept a deployer's own partner key only, or also a Masaar-issued credential? The first preserves the MIT/self-hosted promise; the second creates an operational dependency.
7. **Account-model migration:** Who owns the `phone_number_id` → WAAC ID transition, and when does Masaar adopt `messaging_account_id` ahead of the H1 2027 requirement?
8. **Legal review:** UAE/KSA counsel review of the DPA, cross-border transfer disclosure, and vendor terms before launch.
9. **Data retention:** How much message content and media should Masaar store by default, and what are the export/delete guarantees?
10. **Business app conflicts:** How will the UI prevent duplicate auto-replies when the WhatsApp Business app and the CRM both respond during coexistence?
11. **Gateway plugin:** Is there sufficient demand to justify a separate, unsupported community plugin after the official path ships?

---

## 14. Sources

Accessed 25–26 October 2026 unless noted.

**Meta platform and partner program**

- Meta, Partners overview (Solution Partners, Tech Providers, Tech Partners): <https://developers.facebook.com/documentation/business-messaging/whatsapp/solution-providers/overview>
- Meta, Become a Tech Provider: <https://developers.facebook.com/documentation/business-messaging/whatsapp/solution-providers/get-started-for-tech-providers>
- Meta, Updates to WhatsApp Business accounts (new account model, GA 23 September 2026): <https://developers.facebook.com/documentation/business-messaging/whatsapp/account-model-evolution>
- Meta, Hosted Embedded Signup: <https://developers.facebook.com/documentation/business-messaging/whatsapp/embedded-signup/hosted-es>
- Meta, Migrate a phone number via Embedded Signup (new-model note): <https://developers.facebook.com/documentation/business-messaging/whatsapp/solution-providers/support/migrating-phone-numbers-among-solution-partners-via-embedded-signup>
- Meta Business Partners directory: <https://www.facebook.com/business/partner-directory>

**Vendors (public documentation and list prices)**

- 360dialog, Partner Platform and ISV program (list pricing, page dated 19 October 2026): <https://360dialog.com/partners>
- 360dialog, Partner change (WABA migration within 360dialog): <https://docs.360dialog.com/docs/hub/partner-change-with-360dialog>
- 360dialog, Embedded Signup documentation: <https://docs.360dialog.com/docs/hub/embedded-signup>
- Twilio, WhatsApp Tech Provider program integration guide: <https://www.twilio.com/docs/whatsapp/isv/tech-provider-program/integration-guide>
- Vonage, WhatsApp Hosted Embedded Signup Flow (concepts): <https://api.support.vonage.com/hc/en-us/articles/19324140062748-WhatsApp-Hosted-Embedded-Signup-Flow-Concepts>
- Infobip, Tech Provider program setup: <https://www.infobip.com/docs/whatsapp/tech-provider-program/setup-and-integration>

**WhatsApp terms, policy, and pricing**

- WhatsApp Terms of Service (effective 4 January 2021): <https://www.whatsapp.com/legal/terms-of-service>
- WhatsApp Business Messaging Policy (last updated 23 September 2026): <https://business.whatsapp.com/policy>
- Meta, WhatsApp Business Platform pricing (effective 1 October 2026): <https://developers.facebook.com/docs/whatsapp/pricing/>
- Meta, Embedded Signup overview: <https://developers.facebook.com/documentation/business-messaging/whatsapp/embedded-signup/overview>
- Meta, Embedded Signup v5 (released 20 October 2026): <https://developers.facebook.com/documentation/business-messaging/whatsapp/embedded-signup/version-5>
- Meta, Onboard WhatsApp Business app users (coexistence): <https://developers.facebook.com/documentation/business-messaging/whatsapp/embedded-signup/onboarding-business-app-users>
- 360dialog, Coexistence documentation: <https://docs.360dialog.com/docs/resources/phone-numbers/coexistence>
- 360dialog, Free vs billed messaging: <https://docs.360dialog.com/docs/prices-plans-and-payments/free-vs-billed-messages>
- Whatsable, Meta October 2026 service message pricing: <https://whatsable.app/blog/meta-october-2026-whatsapp-service-message-pricing>

**Unofficial clients and enforcement**

- whatsmeow repository and documentation: <https://github.com/tulir/whatsmeow>
- whatsmeow issue #810, "Your account may be at risk" warning: <https://github.com/tulir/whatsmeow/issues/810>
- whatsmeow discussion #979, multi-session operational experience: <https://github.com/tulir/whatsmeow/discussions/979>
- Baileys repository: <https://github.com/WhiskeySockets/Baileys>
- WAHA repository and Plus/Core licensing: <https://github.com/devlikeapro/waha>, <https://waha.devlike.pro/docs/how-to/waha-plus/>
- WAHA scaling guidance: <https://dev.to/waha/waha-scaling-how-to-handle-500-whatsapp-sessions-3fie>
- Wuzapi repository: <https://github.com/asternic/wuzapi>
- Evolution API license: <https://github.com/EvolutionAPI/evolution-api/blob/main/LICENSE>
- whatsapp-web.js repository: <https://github.com/pedroslopez/whatsapp-web.js>
- WPPConnect repository: <https://github.com/wppconnect-team/wppconnect>
- WhatsApp FAQ, linked devices: <https://faq.whatsapp.com/378279804439436>
- WhatsApp FAQ, message history on linked devices: <https://faq.whatsapp.com/653480766448040>
- Reports on the August 2026 WhatsApp ban wave: <https://origami.sa/en/blog/whatsapp-account-bans/>, <https://yavendio.com/us/blog/whatsapp-bloqueo-mi-cuenta-sin-razon>
- Error 463 / reachout timelock: <https://whasp.ai/blog/whatsapp-463-error-reachout-timelock>

**Masaar repository**

- Files reviewed: `internal/config/config.go`, `cmd/server/main.go`, `internal/whatsapp/sender.go`, `internal/api/handler/whatsapp.go`, `internal/api/handler/whatsapp_inbound.go`, `internal/repo/whatsapp.go`, `migrations/0004_create_whatsapp.sql`, `migrations/0061_whatsapp_fixes.sql`, `migrations/0077_whatsapp_broadcasts.sql`, `README.md`, `web/app/(dashboard)/settings/inbox/page.tsx`.

---

## Appendix A: Minimal schema sketch

```sql
-- Accounts (also needed for per-company Cloud API, not just gateways)
CREATE TABLE wa_accounts (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id            UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    provider              TEXT NOT NULL CHECK (provider IN ('cloud_api','bsp','waha')),
    -- For provider='bsp': which partner, and which of its clients this maps to.
    bsp_slug              TEXT,
    bsp_client_ref        TEXT,
    status                TEXT NOT NULL CHECK (status IN ('pending','linking','connected','degraded','logged_out','banned')),
    phone_e164            TEXT,
    display_name          TEXT,
    -- Meta identifiers. Keep all three: the account model is mid-migration.
    waba_id               TEXT,        -- now a Messaging account ID
    messaging_account_id  TEXT,        -- required for multi-account sends from H1 2027
    waac_id               TEXT,        -- replaces phone_number_id in paths by H1 2028
    phone_number_id       TEXT,        -- legacy; do not use as the primary key
    external_id           TEXT,        -- provider-side account/session identifier
    session_ref           TEXT,        -- gateway session id, if any
    billing_mode          TEXT CHECK (billing_mode IN ('client_card','masaar_credit_line')),
    linked_by             UUID REFERENCES users(id) ON DELETE SET NULL,
    linked_at             TIMESTAMPTZ,
    last_seen_at          TIMESTAMPTZ,
    last_error            TEXT,
    risk_flags            JSONB NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_wa_accounts_company ON wa_accounts(company_id, created_at DESC);
CREATE UNIQUE INDEX idx_wa_accounts_company_phone
    ON wa_accounts(company_id, phone_e164) WHERE phone_e164 IS NOT NULL;

-- Add to threads; keep wa_account_id text during migration for compatibility.
ALTER TABLE whatsapp_threads ADD COLUMN IF NOT EXISTS account_id UUID REFERENCES wa_accounts(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_threads_account ON whatsapp_threads(account_id);
```

Notes:

- `phone_number_id` is retained for compatibility but must not be the account's identity. Meta phase 3 (H1 2028) requires WAAC IDs in endpoint paths.
- `messaging_account_id` is optional today and required for multi-Messaging-account sends from Meta phase 2 (H1 2027). Store it at onboarding, not on first send.
- `billing_mode` is informational for the UI; the authoritative billing relationship lives in the partner contract, not the database.
- For `provider='waha'`, `session_ref` is the WAHA session name and `external_id` can mirror the linked phone number's JID.

## Appendix B: Suggested API surface

```text
GET    /api/v1/whatsapp/accounts
POST   /api/v1/whatsapp/accounts                  { provider: "bsp" | "cloud_api" | "waha" }
POST   /api/v1/whatsapp/accounts/:id/connect      → { launch_url } or embedded-signup payload
POST   /api/v1/whatsapp/connect/callback          (provider callback; idempotent)
DELETE /api/v1/whatsapp/accounts/:id
GET    /api/v1/whatsapp/accounts/:id/health

# Gateway-only (community plugin)
POST   /api/v1/whatsapp/accounts/:id/link
GET    /api/v1/whatsapp/accounts/:id/link/events  (SSE)
POST   /webhooks/whatsapp-gateway/:accountID      (HMAC, no user session)

# Inbound (Meta direct or BSP), signature verified per provider
POST   /webhooks/whatsapp/:provider
```

Design notes:

- `connect` returns a URL or payload; the browser never receives a Meta app secret or a BSP server key.
- The callback resolves the company from the pending account row, never from the request body.
- Delete is a soft delete plus a provider-side disconnect; it must be idempotent.

## Appendix C: Gateway onboarding copy checklist

- "Unofficial connection" label visible before, during, and after linking.
- "Use a dedicated number you can afford to lose" checklist.
- "Meta/WhatsApp can ban this number; Masaar cannot recover it" disclosure.
- "Do not link a personal number or a number used for banking OTPs" warning.
- "Messages sync to this CRM; review our privacy notice" consent.
- "This feature is best-effort; no delivery guarantee or support SLA" terms.
- "Move to official WhatsApp messaging" upgrade link.

## Appendix D: Official onboarding copy checklist

- "Official WhatsApp Business Platform connection" label, with no Meta endorsement implied.
- "You will need your business details for verification" (trade licence, address, website).
- "Meta charges per message; your Masaar subscription is separate" — shown before the client connects.
- "Your number stays yours. You can disconnect at any time." — true under the new account model.
- "Coexistence keeps your WhatsApp Business app" where eligible; list the limitations.
- "Template messages require Meta approval" before the first broadcast is attempted.
- "Who is billing you" — name the entity charging for messages and the entity charging for software.

## Appendix E: Reviewer checklist

This change is research only; it contains no code and no migration.

- [ ] Approve the official-first direction in §11 (partner Embedded Signup before any gateway work).
- [ ] Approve the partner shortlist and the six due-diligence questions in §9.5.
- [ ] Decide the billing default: pass-through or credit-line rebilling (§9.3).
- [ ] Accept the schema direction: account-first, with `messaging_account_id` and `waac_id` fields (§5.1, Appendix A).
- [ ] Confirm the internal commercial note under `docs/internal/` (gitignored) is the right home for the pricing and escalation detail.
- [ ] Confirm the §9.6 and §9.8 summaries are sufficient for public readers.
- [ ] Confirm that no implementation starts before the Phase 0 gates in §11.2 are met.
- [ ] Confirm the public-document statement at the top is acceptable for this repository.
