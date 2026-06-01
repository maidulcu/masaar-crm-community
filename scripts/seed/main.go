package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// ── Deterministic UUIDs ─────────────────────────────────────────────────────
// Same key always → same UUID → re-running the seed is safe (ON CONFLICT).
func id(parts ...string) uuid.UUID {
	k := "masaar-demo"
	for _, p := range parts {
		k += ":" + p
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(k))
}

func main() {
	_ = godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://masaar:masaar@localhost:5432/masaar?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}

	fmt.Println("🌱 Seeding Masaar CRM demo data...")
	now := time.Now().UTC()

	// ══════════════════════════════════════════════════════════════════════════
	//  1. COMPANY
	// ══════════════════════════════════════════════════════════════════════════
	companyID := id("company")
	trialEnd := now.AddDate(0, 0, 25) // 25 days remaining on trial
	exec(ctx, pool, `
		INSERT INTO companies (id,name,subdomain,plan,trial_started_at,trial_ends_at,on_trial,is_active,is_demo)
		VALUES ($1,'Masaar Properties Demo','demo','starter',$2,$3,true,true,true)
		ON CONFLICT (id) DO UPDATE SET is_demo = TRUE
	`, companyID, now.AddDate(0, 0, -65), trialEnd)
	logged("Company", "Masaar Properties Demo (is_demo=true)")

	// ══════════════════════════════════════════════════════════════════════════
	//  2. USERS
	// ══════════════════════════════════════════════════════════════════════════
	type userRow struct {
		ID       uuid.UUID
		Name     string
		Email    string
		Role     string
		Lang     string
		Phone    string
		WANumber string
	}
	users := []userRow{
		{id("user", "ahmed"), "Ahmed Al Mansoori", "ahmed@masaar.local", "admin", "en", "+971501111111", "+971501111111"},
		{id("user", "sarah"), "Sarah Johnson", "sarah@masaar.local", "agent", "en", "+971502222222", "+971502222222"},
		{id("user", "mohammed"), "Mohammed Al Rashidi", "mohammed@masaar.local", "agent", "ar", "+971503333333", "+971503333333"},
		{id("user", "lisa"), "Lisa Chen", "lisa@masaar.local", "viewer", "en", "+971504444444", "+971504444444"},
	}
	// Only the admin account gets the public demo password.
	// Supporting agent/viewer accounts get a random unguessable hash so visitors
	// cannot log in as them, but they still exist as assigned-agent references in
	// leads, deals, properties, etc., making the data look realistic.
	demoPassHash, _ := bcrypt.GenerateFromPassword([]byte("Demo@1234"), bcrypt.DefaultCost)
	randomPassHash, _ := bcrypt.GenerateFromPassword([]byte(uuid.New().String()), bcrypt.DefaultCost)
	for _, u := range users {
		hash := string(randomPassHash)
		if u.Email == "ahmed@masaar.local" {
			// Always reset admin password so vandals can't lock out other visitors.
			hash = string(demoPassHash)
		}
		exec(ctx, pool, `
			INSERT INTO users (id,company_id,name,email,password_hash,role,lang_pref,phone,wa_number,is_active)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,true)
			ON CONFLICT (email) DO UPDATE
			  SET password_hash = EXCLUDED.password_hash,
			      role           = EXCLUDED.role,
			      is_active      = TRUE
		`, u.ID, companyID, u.Name, u.Email, hash, u.Role, u.Lang, u.Phone, u.WANumber)
		logged("User", fmt.Sprintf("%s (%s)", u.Name, u.Email))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  3. COMPANY SETTINGS
	// ══════════════════════════════════════════════════════════════════════════
	pool.Exec(ctx, `
		DELETE FROM company_settings WHERE company_id=$1
	`, companyID)
	exec(ctx, pool, `
		INSERT INTO company_settings (company_id,name,vat_number,business_address,business_phone,business_email,bank_name,bank_account,bank_iban,updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, companyID, "Masaar Properties Demo", "AE123456789", "Level 14, Boulevard Plaza Tower 1, Sheikh Mohammed Bin Rashid Blvd, Downtown Dubai",
		"+971500000000", "billing@masaarproperties.ae", "Emirates NBD", "AE100123456789", "AE3802600000123456789", users[0].ID)
	logged("CompanySettings", "Masaar Properties Demo")

	// ══════════════════════════════════════════════════════════════════════════
	//  4. API SETTINGS
	// ══════════════════════════════════════════════════════════════════════════
	apiSettings := []struct{ key, val, desc string }{
		{"whatsapp_business_hours", "09:00-18:00", "Business hours for auto-reply"},
		{"default_currency", "AED", "Default currency for deals/payments"},
		{"lead_assignment_mode", "round-robin", "How leads are assigned to agents"},
		{"invoice_prefix", "INV-", "Prefix for invoice numbers"},
		{"auto_archive_threads_days", "30", "Auto-close inactive WhatsApp threads"},
	}
	for i, s := range apiSettings {
		exec(ctx, pool, `
			INSERT INTO api_settings (id,setting_key,setting_value,description,updated_by)
			VALUES ($1,$2,$3,$4,$5) ON CONFLICT (setting_key) DO NOTHING
		`, id("api-setting", s.key), s.key, s.val, s.desc, users[0].ID)
		logged("APISetting", fmt.Sprintf("%s=%s", s.key, s.val))
		_ = i
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  5. CONTACTS
	// ══════════════════════════════════════════════════════════════════════════
	type contactRow struct {
		ID       uuid.UUID
		Phone    string
		Name     string
		Email    string
		Lang     string
		Score    int
		Assigned uuid.UUID
	}
	contacts := []contactRow{
		{id("contact", "khaled"), "+971505551111", "خالد العامري", "khaled@example.ae", "ar", 85, users[1].ID},
		{id("contact", "rania"), "+971505552222", "Rania Boutros", "rania@boutros.ae", "en", 72, users[2].ID},
		{id("contact", "yousuf"), "+971505553333", "يوسف المهيري", "yousuf@alheri.ae", "ar", 91, users[1].ID},
		{id("contact", "david"), "+971505554444", "David Chen", "david@sinobizdubai.com", "en", 60, users[1].ID},
		{id("contact", "fatima"), "+971505555555", "فاطمة الزيدي", "fatima@zaidi.ae", "ar", 45, users[2].ID},
		{id("contact", "priya"), "+971505556666", "Priya Sharma", "priya@dubaifintech.io", "en", 78, users[1].ID},
		{id("contact", "abdulla"), "+971505557777", "عبدالله بن زايد", "abdulla@bz.ae", "ar", 95, users[2].ID},
		{id("contact", "lara"), "+971505558888", "Lara Hadid", "lara@realestateuae.com", "en", 67, users[1].ID},
		{id("contact", "nora"), "+971505559999", "نورة الكتبي", "nora@alketbi.ae", "ar", 82, users[2].ID},
		{id("contact", "james"), "+971505550000", "James Mitchell", "james@mitchell-consulting.com", "en", 55, users[1].ID},
	}
	for _, c := range contacts {
		exec(ctx, pool, `
			INSERT INTO contacts (id,phone_wa,full_name,email,language,lead_score,assigned_to)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (phone_wa) DO NOTHING
		`, c.ID, c.Phone, c.Name, c.Email, c.Lang, c.Score, c.Assigned)
		logged("Contact", fmt.Sprintf("%s (%s)", c.Name, c.Phone))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  6. LEADS  (Pipeline stages: new → contacted → qualified → proposal → won → lost)
	// ══════════════════════════════════════════════════════════════════════════
	type leadRow struct {
		ID          uuid.UUID
		ContactID   uuid.UUID
		Stage       string
		Source      string
		Value       float64
		Notes       string
		Assigned    uuid.UUID
		LeadScore   int
		LastContact time.Time
	}
	leads := []leadRow{
		{id("lead", "1"), contacts[0].ID, "new", "whatsapp", 85000, "Interested in 1BR apartment in Dubai Marina — budget 85K/yr", users[1].ID, 85, now.Add(-2 * time.Hour)},
		{id("lead", "2"), contacts[1].ID, "contacted", "web", 120000, "Looking for office space in DIFC 800-1000 sqft", users[2].ID, 72, now.Add(-24 * time.Hour)},
		{id("lead", "3"), contacts[2].ID, "qualified", "referral", 210000, "VIP client referred by existing tenant — wants Palm villa", users[1].ID, 91, now.Add(-3 * time.Hour)},
		{id("lead", "4"), contacts[3].ID, "proposal", "web", 15000, "Short-term rental 3 months in JLT — budget 15K/mo", users[1].ID, 60, now.Add(-12 * time.Hour)},
		{id("lead", "5"), contacts[4].ID, "won", "event", 320000, "Signed 2BR in Downtown — closing next week", users[2].ID, 45, now.Add(-48 * time.Hour)},
		{id("lead", "6"), contacts[5].ID, "contacted", "whatsapp", 78000, "Studio in Business Bay — first-time renter", users[1].ID, 78, now.Add(-6 * time.Hour)},
		{id("lead", "7"), contacts[6].ID, "qualified", "referral", 550000, "Penthouse in Palm Jumeirah — very high budget", users[2].ID, 95, now.Add(-1 * time.Hour)},
		{id("lead", "8"), contacts[7].ID, "lost", "web", 29000, "Found another property — lost to competitor", users[1].ID, 67, now.Add(-72 * time.Hour)},
		{id("lead", "9"), contacts[8].ID, "new", "whatsapp", 175000, "3BR townhouse in Arabian Ranches", users[2].ID, 82, now.Add(-8 * time.Hour)},
		{id("lead", "10"), contacts[9].ID, "proposal", "event", 450000, "Commercial space in Dubai Silicon Oasis", users[1].ID, 55, now.Add(-4 * time.Hour)},
	}
	leadIDs := make([]uuid.UUID, len(leads))
	for i, l := range leads {
		leadIDs[i] = l.ID
		lastContact := l.LastContact.Truncate(time.Second)
		exec(ctx, pool, `
			INSERT INTO leads (id,contact_id,stage,source,deal_value,currency,notes,assigned_to,lead_score,last_contacted_at)
			VALUES ($1,$2,$3,$4,$5,'AED',$6,$7,$8,$9)
			ON CONFLICT (id) DO NOTHING
		`, l.ID, l.ContactID, l.Stage, l.Source, l.Value, l.Notes, l.Assigned, l.LeadScore, lastContact)
		logged("Lead", fmt.Sprintf("%s → %s (AED %.0f)", l.Stage, l.Source, l.Value))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  7. LEAD TAGS
	// ══════════════════════════════════════════════════════════════════════════
	leadTags := []struct {
		leadID   uuid.UUID
		tag      string
		category string
		userID   uuid.UUID
	}{
		{leads[0].ID, "budget-conscious", "financial", users[1].ID},
		{leads[1].ID, "commercial", "property-type", users[2].ID},
		{leads[2].ID, "vip", "tier", users[1].ID},
		{leads[3].ID, "short-term", "lease-type", users[1].ID},
		{leads[6].ID, "premium", "tier", users[2].ID},
		{leads[8].ID, "family", "tenant-type", users[2].ID},
	}
	for _, t := range leadTags {
		exec(ctx, pool, `
			INSERT INTO lead_tags (lead_id,tag,category,created_by)
			VALUES ($1,$2,$3,$4) ON CONFLICT (lead_id,tag) DO NOTHING
		`, t.leadID, t.tag, t.category, t.userID)
		logged("LeadTag", fmt.Sprintf("%s → %s", t.tag, t.leadID.String()[:8]))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  8. COMMUNICATION HISTORY
	// ══════════════════════════════════════════════════════════════════════════
	comms := []struct {
		leadIdx int
		typ     string
		dir     string
		body    string
		from    string
		to      string
		userIdx int
	}{
		{0, "whatsapp", "inbound", "السلام عليكم، أبحث عن شقة في دبي مارينا", contacts[0].Phone, "+971500000000", 1},
		{0, "whatsapp", "outbound", "وعليكم السلام! يسعدنا مساعدتك. هل تبحث عن إيجار سنوي؟", "+971500000000", contacts[0].Phone, -1},
		{1, "email", "inbound", "Hi, could you send me the office listings in DIFC?", "rania@boutros.ae", "sales@masaarproperties.ae", 2},
		{1, "email", "outbound", "Sure! Attaching 3 options with floor plans.", "sales@masaarproperties.ae", "rania@boutros.ae", -1},
		{3, "whatsapp", "inbound", "Hi David here — is the JLT studio still available?", contacts[3].Phone, "+971500000000", 1},
		{3, "whatsapp", "outbound", "Hi David! Yes, still available. Can you come for viewing tomorrow?", "+971500000000", contacts[3].Phone, -1},
	}
	for _, c := range comms {
		var createdBy *uuid.UUID
		if c.userIdx >= 0 {
			createdBy = &users[c.userIdx].ID
		}
		exec(ctx, pool, `
			INSERT INTO communication_history (lead_id,contact_id,communication_type,direction,body,from_identifier,to_identifier,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		`, leads[c.leadIdx].ID, contacts[c.leadIdx].ID, c.typ, c.dir, c.body, c.from, c.to, createdBy)
		logged("CommHistory", fmt.Sprintf("%s %s", c.typ, c.dir))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  9. DEALS
	// ══════════════════════════════════════════════════════════════════════════
	type dealRow struct {
		ID          uuid.UUID
		LeadID      uuid.UUID
		Title       string
		Stage       string
		Amount      float64
		CloseDate   time.Time
		Probability int
		Owner       uuid.UUID
	}
	deals := []dealRow{
		{id("deal", "1"), leads[4].ID, "Downtown 2BR — Al Mansoori", "won", 320000, now.AddDate(0, 0, 5), 100, users[2].ID},
		{id("deal", "2"), leads[2].ID, "Palm Jumeirah Villa — Al Heri", "open", 550000, now.AddDate(0, 0, 45), 60, users[1].ID},
		{id("deal", "3"), leads[1].ID, "DIFC Office Space — Boutros", "open", 120000, now.AddDate(0, 0, 30), 40, users[2].ID},
		{id("deal", "4"), leads[6].ID, "Palm Penthouse — Bin Zayed", "open", 550000, now.AddDate(0, 0, 60), 75, users[2].ID},
		{id("deal", "5"), leads[7].ID, "Studio JLT — Hadid (Lost)", "lost", 29000, now.AddDate(0, 0, -5), 0, users[1].ID},
	}
	for _, d := range deals {
		exec(ctx, pool, `
			INSERT INTO deals (id,lead_id,title,stage,amount,currency,close_date,probability,owner_id)
			VALUES ($1,$2,$3,$4,$5,'AED',$6,$7,$8)
			ON CONFLICT (id) DO NOTHING
		`, d.ID, d.LeadID, d.Title, d.Stage, d.Amount, d.CloseDate, d.Probability, d.Owner)
		logged("Deal", fmt.Sprintf("%s (%s, AED %.0f)", d.Title, d.Stage, d.Amount))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  10. VAT INVOICES
	// ══════════════════════════════════════════════════════════════════════════
	invoices := []struct {
		ID       uuid.UUID
		DealID   uuid.UUID
		InvNo    string
		Subtotal float64
		VATRate  float64
		Status   string
		IssuedAt time.Time
	}{
		{id("inv", "1"), deals[0].ID, "INV-2026-001", 304761.90, 0.05, "paid", now.AddDate(0, 0, -10)},
		{id("inv", "2"), deals[1].ID, "INV-2026-002", 523809.52, 0.05, "sent", now.AddDate(0, 0, -2)},
		{id("inv", "3"), deals[2].ID, "INV-2026-003", 114285.71, 0.05, "draft", now.AddDate(0, 0, -1)},
		{id("inv", "4"), deals[3].ID, "INV-2026-004", 523809.52, 0.05, "draft", now},
	}
	for _, inv := range invoices {
		exec(ctx, pool, `
			INSERT INTO vat_invoices (id,deal_id,invoice_no,subtotal,vat_rate,status,issued_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (invoice_no) DO NOTHING
		`, inv.ID, inv.DealID, inv.InvNo, inv.Subtotal, inv.VATRate, inv.Status, inv.IssuedAt)
		logged("Invoice", fmt.Sprintf("%s (%s)", inv.InvNo, inv.Status))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  11. WHATSAPP THREADS + MESSAGES
	// ══════════════════════════════════════════════════════════════════════════
	type msgRow struct {
		dir  string
		body string
	}
	threads := []struct {
		contactIdx int
		status     string
		messages   []msgRow
	}{
		{
			0, "open", []msgRow{
				{"inbound", "السلام عليكم، أبحث عن شقة في دبي مارينا"},
				{"outbound", "وعليكم السلام! يسعدنا مساعدتك. ما هي ميزانيتك؟"},
				{"inbound", "ميزانيتي حوالي 100 ألف درهم سنوياً"},
				{"outbound", "ممتاز! لدي 3 خيارات متاحة. هل تفضل زيارة اليوم؟"},
			},
		},
		{
			1, "open", []msgRow{
				{"inbound", "Hi, I'm interested in your office spaces in DIFC"},
				{"outbound", "Hello! Great choice. What size are you looking for?"},
				{"inbound", "Around 1500 sqft for a team of 10"},
				{"outbound", "Perfect, I have 3 options ready. Shall we schedule a viewing?"},
				{"inbound", "Yes, Thursday at 11am works"},
			},
		},
		{
			2, "open", []msgRow{
				{"inbound", "مرحبا، هل لديكم عروض على الوحدات التجارية؟"},
				{"outbound", "أهلاً! نعم لدينا عروض ممتازة. متى يمكنك الحضور للمعاينة؟"},
			},
		},
		{
			5, "closed", []msgRow{
				{"inbound", "Hi, I need a studio in Business Bay"},
				{"outbound", "Sure! We have several options starting from 55K/year"},
				{"inbound", "Can you send me the list?"},
				{"outbound", "Here you go: attached brochure with 5 options"},
			},
		},
		{
			8, "open", []msgRow{
				{"inbound", "مرحبا، أبحث عن تاون هاوس في العربية"},
				{"outbound", "أهلاً نورة! لدينا خيارين ممتازين. تفضلي بالزيارة"},
			},
		},
	}
	for i, t := range threads {
		tID := id("thread", fmt.Sprintf("%d", i))
		cID := contacts[t.contactIdx].ID
		exec(ctx, pool, `
			INSERT INTO whatsapp_threads (id,contact_id,wa_account_id,thread_status,last_message_at,message_count)
			VALUES ($1,$2,'15551234567',$3,$4,$5)
			ON CONFLICT (id) DO NOTHING
		`, tID, cID, t.status, now.Add(-time.Duration(i)*time.Hour), len(t.messages))
		logged("Thread", fmt.Sprintf("contact %d (%d messages)", t.contactIdx+1, len(t.messages)))

		for j, m := range t.messages {
			exec(ctx, pool, `
				INSERT INTO whatsapp_messages (id,thread_id,direction,body,wa_message_id,sent_at)
				VALUES ($1,$2,$3,$4,$5,$6)
				ON CONFLICT (wa_message_id) DO NOTHING
			`, id("msg", fmt.Sprintf("%d", i), fmt.Sprintf("%d", j)), tID, m.dir, m.body,
				fmt.Sprintf("wa-msg-%d-%d", i, j),
				now.Add(-time.Duration(i)*time.Hour).Add(time.Duration(j)*time.Minute))
		}
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  12. NOTIFICATIONS
	// ══════════════════════════════════════════════════════════════════════════
	notifications := []struct {
		id      uuid.UUID
		userIdx int
		typ     string
		title   string
		body    string
		read    bool
	}{
		{id("notif", "1"), 0, "lead_new", "New lead assigned", "Khaled Al Ameri was assigned to you", false},
		{id("notif", "2"), 1, "deal_won", "Deal closed!", "Downtown 2BR deal marked as won", false},
		{id("notif", "3"), 0, "thread_new", "New WhatsApp message", "Rania Boutros sent a message in DIFC thread", true},
		{id("notif", "4"), 2, "lease_expiring", "Lease expiring soon", "Palm Jumeirah lease expires in 30 days", false},
		{id("notif", "5"), 0, "payment_overdue", "Payment overdue", "Tenant #204 payment is 5 days overdue", false},
		{id("notif", "6"), 1, "maintenance", "Maintenance request", "AC repair in JLT unit completed", true},
	}
	for _, n := range notifications {
		exec(ctx, pool, `
			INSERT INTO notifications (id,user_id,type,title,body,read)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (id) DO NOTHING
		`, n.id, users[n.userIdx].ID, n.typ, n.title, n.body, n.read)
		logged("Notification", n.title)
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  13. RENTAL PROPERTIES
	// ══════════════════════════════════════════════════════════════════════════
	properties := []struct {
		ID              uuid.UUID
		Name            string
		PropType        string
		City            string
		Emirate         string
		Area            string
		Bed             int
		Bath            int
		Parking         int
		Sqft            float64
		MarketValue     float64
		Status          string
		OccupancyStatus string
		Amenities       []string
	}{
		{id("prop", "1"), "Marina Heights Tower", "apartment", "Dubai", "Dubai", "Dubai Marina", 2, 2, 1, 1250, 1850000, "occupied", "occupied", []string{"pool", "gym", "security", "parking"}},
		{id("prop", "2"), "The Boulevard Central", "apartment", "Dubai", "Dubai", "Downtown Dubai", 3, 3, 2, 1800, 3200000, "occupied", "occupied", []string{"pool", "gym", "concierge", "valet"}},
		{id("prop", "3"), "JLT Cluster Y Residency", "apartment", "Dubai", "Dubai", "JLT", 1, 1, 1, 750, 950000, "occupied", "occupied", []string{"pool", "gym", "retail"}},
		{id("prop", "4"), "Palm Frond E Villa", "villa", "Dubai", "Dubai", "Palm Jumeirah", 5, 6, 3, 5200, 15000000, "vacant", "vacant", []string{"private pool", "beach access", "maid room", "garden"}},
		{id("prop", "5"), "DIFC Gate Building", "commercial", "Dubai", "Dubai", "DIFC", 0, 2, 2, 2200, 5800000, "occupied", "occupied", []string{"24hr security", "conference room", "cafeteria"}},
		{id("prop", "6"), "Arabian Ranches Townhouses", "townhouse", "Dubai", "Dubai", "Arabian Ranches", 3, 3, 2, 2100, 2800000, "vacant", "vacant", []string{"garden", "community pool", "play area"}},
		{id("prop", "7"), "Business Bay Metro Tower", "apartment", "Dubai", "Dubai", "Business Bay", 1, 1, 1, 680, 820000, "occupied", "occupied", []string{"pool", "gym", "metro access"}},
		{id("prop", "8"), "Al Reem Tower", "apartment", "Abu Dhabi", "Abu Dhabi", "Al Reem Island", 2, 2, 1, 1150, 1650000, "maintenance", "maintenance", []string{"pool", "gym", "security"}},
	}
	for _, p := range properties {
		buildingNum := "Bldg " + p.Name
		exec(ctx, pool, `
			INSERT INTO rental_properties (id,company_id,name,property_type,city,emirate,area,building_number,bedrooms,bathrooms,parking_spaces,total_sqft,market_value,currency,status,occupancy_status,amenities,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,'AED',$14,$15,$16,$17)
			ON CONFLICT (id) DO NOTHING
		`, p.ID, companyID, p.Name, p.PropType, p.City, p.Emirate, p.Area, buildingNum,
			p.Bed, p.Bath, p.Parking, p.Sqft, p.MarketValue, p.Status, p.OccupancyStatus, p.Amenities, users[0].ID)
		logged("Property", fmt.Sprintf("%s (%s)", p.Name, p.Emirate))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  14. TENANTS
	// ══════════════════════════════════════════════════════════════════════════
	type tenantRow struct {
		ID          uuid.UUID
		NameEN      string
		NameAR      string
		Email       string
		Phone       string
		PhoneWA     string
		Nationality string
		Status      string
	}
	tenants := []tenantRow{
		{id("tenant", "1"), "John Smith", "جون سميث", "john.smith@acmecorp.ae", "+971506611111", "+971506611111", "British", "active"},
		{id("tenant", "2"), "Aisha Al Maktoum", "عائشة آل مكتوم", "aisha@maktoum.ae", "+971506622222", "+971506622222", "UAE", "active"},
		{id("tenant", "3"), "Raj Patel", "راج باتيل", "raj@techdubai.com", "+971506633333", "+971506633333", "Indian", "active"},
		{id("tenant", "4"), "Mona Al Suwaidi", "منا السويدي", "mona@suwaidi.ae", "+971506644444", "+971506644444", "UAE", "active"},
		{id("tenant", "5"), "Omar Hassan", "عمر حسن", "omar@hassan.ae", "+971506655555", "+971506655555", "Egyptian", "active"},
		{id("tenant", "6"), "Emma Wilson", "إيما ويلسون", "emma@britishschool.ae", "+971506666666", "+971506666666", "British", "active"},
		{id("tenant", "7"), "Saeed Al Falasi", "سعيد الفلاسي", "saeed@alfalasi.ae", "+971506677777", "+971506677777", "UAE", "inactive"},
		{id("tenant", "8"), "Maria Garcia", "ماريا غارسيا", "maria@spainconsult.ae", "+971506688888", "+971506688888", "Spanish", "active"},
	}
	for _, t := range tenants {
		exec(ctx, pool, `
			INSERT INTO tenants (id,company_id,full_name_en,full_name_ar,email,phone,phone_wa,id_type,id_number,nationality,status,is_verified,verification_status,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,'Emirates ID','784-'||$8,$9,$10,true,'verified',$11)
			ON CONFLICT (id_number) DO NOTHING
		`, t.ID, companyID, t.NameEN, t.NameAR, t.Email, t.Phone, t.PhoneWA,
			t.ID.String()[:8], t.Nationality, t.Status, users[0].ID)
		logged("Tenant", fmt.Sprintf("%s / %s", t.NameEN, t.NameAR))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  15. LEASE TEMPLATES
	// ══════════════════════════════════════════════════════════════════════════
	leaseTpls := []struct {
		ID     uuid.UUID
		Name   string
		Desc   string
		Months int
		Notice int
		DepPct float64
		LPct   float64
	}{
		{id("l-tpl", "1"), "Standard Residential Lease", "Standard 12-month residential lease for apartments", 12, 90, 5.0, 2.0},
		{id("l-tpl", "2"), "Premium Villa Lease", "Annual lease for luxury villas with enhanced terms", 12, 60, 10.0, 3.0},
		{id("l-tpl", "3"), "Commercial Lease", "Standard commercial lease for office/retail spaces", 36, 180, 5.0, 5.0},
		{id("l-tpl", "4"), "Short-Term Rental", "Flexible 3-6 month rental agreement", 3, 30, 5.0, 5.0},
	}
	for _, t := range leaseTpls {
		exec(ctx, pool, `
			INSERT INTO lease_templates (id,company_id,name,description,is_default,default_lease_duration_months,default_notice_period_days,default_security_deposit_percent,default_late_fee_percent,payment_frequency,payment_day_of_month,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'monthly',1,$10)
			ON CONFLICT (id) DO NOTHING
		`, t.ID, companyID, t.Name, t.Desc, t.Name == "Standard Residential Lease", t.Months, t.Notice, t.DepPct, t.LPct, users[0].ID)
		logged("LeaseTemplate", t.Name)
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  16. LEASES
	// ══════════════════════════════════════════════════════════════════════════
	leases := []struct {
		ID         uuid.UUID
		PropIdx    int
		TenantIdx  int
		TplIdx     int
		Monthly    float64
		Start      time.Time
		End        time.Time
		Status     string
		Ejari      string
		NoticeDays int
		Deposit    float64
	}{
		{id("lease", "1"), 0, 0, 0, 12000, now.AddDate(0, -8, 0), now.AddDate(0, 4, 0), "active", "EJ-2023-001", 90, 60000},
		{id("lease", "2"), 1, 1, 0, 25000, now.AddDate(0, -11, 0), now.AddDate(0, 1, 0), "active", "EJ-2023-002", 90, 125000},
		{id("lease", "3"), 2, 2, 3, 6500, now.AddDate(0, -2, 0), now.AddDate(0, 1, 0), "active", "EJ-2024-003", 30, 32500},
		{id("lease", "4"), 4, 3, 2, 45000, now.AddDate(0, -18, 0), now.AddDate(0, 18, 0), "active", "EJ-2024-004", 180, 450000},
		{id("lease", "5"), 6, 4, 0, 7000, now.AddDate(0, -4, 0), now.AddDate(0, 8, 0), "active", "EJ-2025-005", 90, 35000},
		{id("lease", "6"), 7, 5, 0, 11000, now.AddDate(0, -6, 0), now.AddDate(0, 6, 0), "active", "EJ-2025-006", 90, 55000},
		{id("lease", "7"), 6, 6, 3, 7500, now.AddDate(0, -14, 0), now.AddDate(0, -2, 0), "terminated", "EJ-2024-007", 30, 37500},
	}
	for _, l := range leases {
		exec(ctx, pool, `
			INSERT INTO leases (id,company_id,property_id,tenant_id,template_id,start_date,end_date,monthly_rent,currency,security_deposit,payment_frequency,ejari_number,status,notice_period_days,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'AED',$9,'monthly',$10,$11,$12,$13)
			ON CONFLICT (id) DO NOTHING
		`, l.ID, companyID, properties[l.PropIdx].ID, tenants[l.TenantIdx].ID, leaseTpls[l.TplIdx].ID,
			l.Start, l.End, l.Monthly, l.Deposit, l.Ejari, l.Status, l.NoticeDays, users[0].ID)
		logged("Lease", fmt.Sprintf("P%d → T%d (%s)", l.PropIdx+1, l.TenantIdx+1, l.Status))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  17. PAYMENTS
	// ══════════════════════════════════════════════════════════════════════════
	payments := []struct {
		ID      uuid.UUID
		LeaseID uuid.UUID
		Amount  float64
		DueDate time.Time
		Paid    *time.Time
		Method  string
		Ref     string
		Status  string
	}{
		{id("pmt", "1"), leases[0].ID, 12000, now.AddDate(0, -1, 0), timePtr(now.AddDate(0, -1, 5)), "bank_transfer", "TRX-001", "paid"},
		{id("pmt", "2"), leases[0].ID, 12000, now.AddDate(0, 0, 0), nil, "", "", "pending"},
		{id("pmt", "3"), leases[1].ID, 25000, now.AddDate(0, 0, -10), timePtr(now.AddDate(0, 0, -8)), "cheque", "CHQ-2026-001", "paid"},
		{id("pmt", "4"), leases[1].ID, 25000, now.AddDate(0, 1, 0), nil, "", "", "pending"},
		{id("pmt", "5"), leases[2].ID, 6500, now.AddDate(0, 0, -15), timePtr(now.AddDate(0, 0, -12)), "bank_transfer", "TRX-002", "paid"},
		{id("pmt", "6"), leases[2].ID, 6500, now.AddDate(0, 0, 0), nil, "", "", "pending"},
		{id("pmt", "7"), leases[3].ID, 45000, now.AddDate(0, 0, -5), nil, "", "", "overdue"},
		{id("pmt", "8"), leases[4].ID, 7000, now.AddDate(0, 0, -3), timePtr(now.AddDate(0, 0, -1)), "card", "CARD-PAY-001", "paid"},
		{id("pmt", "9"), leases[4].ID, 7000, now.AddDate(0, 1, 0), nil, "", "", "pending"},
		{id("pmt", "10"), leases[5].ID, 11000, now.AddDate(0, 0, -7), timePtr(now.AddDate(0, 0, -5)), "bank_transfer", "TRX-003", "paid"},
		{id("pmt", "11"), leases[5].ID, 11000, now.AddDate(0, 0, 23), nil, "", "", "pending"},
		{id("pmt", "12"), leases[0].ID, 12000, now.AddDate(0, -2, 0), timePtr(now.AddDate(0, -2, 1)), "bank_transfer", "TRX-000", "paid"},
	}
	for _, p := range payments {
		exec(ctx, pool, `
			INSERT INTO payments (id,company_id,lease_id,amount,currency,due_date,paid_date,payment_method,payment_reference,status,created_by)
			VALUES ($1,$2,$3,$4,'AED',$5,$6,$7,$8,$9,$10)
			ON CONFLICT (id) DO NOTHING
		`, p.ID, companyID, p.LeaseID, p.Amount, p.DueDate, p.Paid, p.Method, p.Ref, p.Status, users[0].ID)
		logged("Payment", fmt.Sprintf("AED %.0f — %s", p.Amount, p.Status))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  18. INSPECTION TEMPLATES
	// ══════════════════════════════════════════════════════════════════════════
	insTpls := []struct {
		id       uuid.UUID
		name     string
		insType  string
		duration int
	}{
		{id("ins-tpl", "1"), "Move-In Inspection", "move-in", 60},
		{id("ins-tpl", "2"), "Move-Out Inspection", "move-out", 90},
		{id("ins-tpl", "3"), "Quarterly Maintenance Audit", "periodic", 45},
	}
	for _, t := range insTpls {
		exec(ctx, pool, `
			INSERT INTO inspection_templates (id,company_id,template_name,inspection_type,estimated_duration_minutes)
			VALUES ($1,$2,$3,$4,$5) ON CONFLICT (id) DO NOTHING
		`, t.id, companyID, t.name, t.insType, t.duration)
		logged("InspectionTemplate", t.name)
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  19. INSPECTIONS
	// ══════════════════════════════════════════════════════════════════════════
	inspections := []struct {
		id           uuid.UUID
		propIdx      int
		tplIdx       int
		insType      string
		scheduled    time.Time
		completed    *time.Time
		inspectorIdx int
		tenantIdx    int
		status       string
		findings     string
	}{
		{id("ins", "1"), 0, 1, "move-out", now.AddDate(0, 0, -20), timePtr(now.AddDate(0, 0, -18)), 1, 0, "completed", "Minor wall scratches in hallway, AC filter needs replacement"},
		{id("ins", "2"), 3, 0, "move-in", now.AddDate(0, 0, 5), nil, 0, -1, "scheduled", ""},
		{id("ins", "3"), 7, 2, "periodic", now.AddDate(0, 0, -15), timePtr(now.AddDate(0, 0, -14)), 2, 5, "completed", "All systems operational. Pool pump needs servicing."},
	}
	for _, ins := range inspections {
		var tenantID *uuid.UUID
		if ins.tenantIdx >= 0 {
			tenantID = &tenants[ins.tenantIdx].ID
		}
		exec(ctx, pool, `
			INSERT INTO inspections (id,company_id,property_id,template_id,inspection_type,scheduled_date,completed_date,inspector_id,tenant_id,status,findings,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT (id) DO NOTHING
		`, ins.id, companyID, properties[ins.propIdx].ID, insTpls[ins.tplIdx].id, ins.insType,
			ins.scheduled, ins.completed, users[ins.inspectorIdx].ID, tenantID, ins.status, ins.findings, users[0].ID)
		logged("Inspection", fmt.Sprintf("%s (%s)", ins.insType, ins.status))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  20. MAINTENANCE TASKS + PHOTOS
	// ══════════════════════════════════════════════════════════════════════════
	maintenance := []struct {
		ID          uuid.UUID
		PropIdx     int
		InsIdx      int
		Type        string
		Desc        string
		Priority    string
		Status      string
		Contractor  string
		EstCost     float64
		ActualCost  float64
		Completed   *time.Time
		AssignedIdx int
		Photos      []string
	}{
		{id("maint", "1"), 0, 0, "repair", "AC maintenance — replace filters in Marina Heights unit 204", "medium", "completed", "CoolTech AC Services", 1500, 1200, timePtr(now.AddDate(0, 0, -17)), 1, nil},
		{id("maint", "2"), 7, 2, "repair", "Pool pump servicing at Al Reem Tower", "high", "completed", "AquaTech Pools LLC", 3500, 3200, timePtr(now.AddDate(0, 0, -13)), 2, nil},
		{id("maint", "3"), 2, -1, "plumbing", "Leaking faucet in JLT apartment — kitchen", "low", "in_progress", "QuickFix Plumbing", 800, 0, nil, 1, nil},
		{id("maint", "4"), 3, -1, "painting", "Full interior repaint for Palm villa (new tenant move-in)", "medium", "pending", "Premium Painters LLC", 15000, 0, nil, 0, nil},
		{id("maint", "5"), 4, -1, "repair", "Elevator maintenance at DIFC Gate Building", "critical", "pending", "Otis Elevators Gulf", 25000, 0, nil, 2, nil},
	}
	for _, m := range maintenance {
		var insID *uuid.UUID
		if m.InsIdx >= 0 {
			insID = &inspections[m.InsIdx].id
		}
		exec(ctx, pool, `
			INSERT INTO maintenance_tasks (id,company_id,property_id,inspection_id,maintenance_type,description,priority,scheduled_date,due_date,completion_date,contractor_name,estimated_cost,actual_cost,status,assigned_to,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
			ON CONFLICT (id) DO NOTHING
		`, m.ID, companyID, properties[m.PropIdx].ID, insID, m.Type, m.Desc, m.Priority,
			now.AddDate(0, 0, -20), now.AddDate(0, 0, -18), m.Completed, m.Contractor, m.EstCost, m.ActualCost,
			m.Status, users[m.AssignedIdx].ID, users[0].ID)
		logged("MaintenanceTask", fmt.Sprintf("%s — %s", m.Type, m.Status))

		for pi, photo := range m.Photos {
			exec(ctx, pool, `
				INSERT INTO maintenance_photos (id,task_id,photo_url,photo_stage)
				VALUES ($1,$2,$3,$4) ON CONFLICT (id) DO NOTHING
			`, id("maint-photo", m.ID.String(), fmt.Sprintf("%d", pi)), m.ID, photo, "before")
		}
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  21. LEASE RENEWALS + COMMUNICATION LOG
	// ══════════════════════════════════════════════════════════════════════════
	renewals := []struct {
		ID       uuid.UUID
		LeaseIdx int
		Status   string
		Proposed float64
		Response string
		Counter  float64
	}{
		{id("renew", "1"), 1, "negotiating", 275000, "countered", 265000},
		{id("renew", "2"), 4, "pending", 84000, "", 0},
	}
	for _, r := range renewals {
		exec(ctx, pool, `
			INSERT INTO lease_renewal_workflows (id,company_id,lease_id,renewal_date,renewal_status,proposed_rent_amount,tenant_response,tenant_counter_offer,counter_offer_date)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (id) DO NOTHING
		`, r.ID, companyID, leases[r.LeaseIdx].ID,
			leases[r.LeaseIdx].End.AddDate(0, -1, 0), r.Status, r.Proposed, r.Response, r.Counter,
			now.Add(-24*time.Hour))
		logged("LeaseRenewal", fmt.Sprintf("lease %d → %s", r.LeaseIdx+1, r.Status))
	}

	// Renewal communication templates
	renewCommTpls := []struct {
		id       uuid.UUID
		name     string
		subject  string
		body     string
		whatsapp string
		lang     string
	}{
		{id("ren-tpl", "en"), "Renewal Notice (English)", "Your Lease Renewal Is Due", "Dear tenant, your lease at {{property_name}} is expiring soon...", "Hi {{tenant_name}}, your lease is up for renewal. Let's discuss the terms!", "en"},
		{id("ren-tpl", "ar"), "إشعار التجديد (عربي)", "تجديد عقد الإيجار الخاص بك", "عزيزي المستأجر، عقد الإيجار الخاص بك في {{property_name}} على وشك الانتهاء...", "مرحباً {{tenant_name}}، عقد الإيجار الخاص بك على وشك الانتهاء. دعنا نناقش الشروط!", "ar"},
	}
	for _, t := range renewCommTpls {
		exec(ctx, pool, `
			INSERT INTO renewal_communication_templates (id,company_id,template_name,email_subject,email_body,whatsapp_message,language)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING
		`, t.id, companyID, t.name, t.subject, t.body, t.whatsapp, t.lang)
		logged("RenewalCommTemplate", t.name)
	}

	// Renewal communication log
	renewLogs := []struct {
		id         uuid.UUID
		renewIdx   int
		commType   string
		tplIdx     int
		delivered  string
		response   *string
	}{
		{id("ren-log", "1"), 0, "email", 0, "delivered", strPtr("Sounds good, but can we do 265K?")},
		{id("ren-log", "2"), 0, "whatsapp", 0, "delivered", nil},
	}
	for _, l := range renewLogs {
		exec(ctx, pool, `
			INSERT INTO renewal_communication_log (id,renewal_id,communication_type,template_id,sent_date,delivery_status,response_text)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, l.id, renewals[l.renewIdx].ID, l.commType, renewCommTpls[l.tplIdx].id,
			now.Add(-48*time.Hour), l.delivered, l.response)
		logged("RenewalCommLog", l.commType)
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  22. COMMISSION STRUCTURES + AGENT COMMISSIONS
	// ══════════════════════════════════════════════════════════════════════════
	commStructures := []struct {
		id   uuid.UUID
		name string
		typ  string
	}{
		{id("comm-struct", "1"), "Standard Sales Commission", "percentage"},
		{id("comm-struct", "2"), "Leasing Commission", "percentage"},
	}
	for _, s := range commStructures {
		exec(ctx, pool, `
			INSERT INTO commission_structures (id,company_id,structure_name,commission_type,applicable_to,effective_from,effective_to)
			VALUES ($1,$2,$3,$4,'both',$5,$6)
			ON CONFLICT (id) DO NOTHING
		`, s.id, companyID, s.name, s.typ, now.AddDate(0, -6, 0), now.AddDate(1, 0, 0))
		logged("CommissionStructure", s.name)
	}

	agentComms := []struct {
		id            uuid.UUID
		agentIdx      int
		structureIdx  int
		dealsCount    int
		dealsRevenue  float64
		leasesCount   int
		leasesRevenue float64
		total         float64
		status        string
	}{
		{id("agent-comm", "1"), 1, 0, 3, 1250000, 2, 360000, 80500, "approved"},
		{id("agent-comm", "2"), 2, 0, 1, 550000, 1, 120000, 33500, "pending"},
	}
	for _, c := range agentComms {
		exec(ctx, pool, `
			INSERT INTO agent_commissions (id,company_id,agent_id,commission_period_start,commission_period_end,commission_structure_id,deals_count,deals_revenue,leases_count,leases_revenue,total_commission,status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT (id) DO NOTHING
		`, c.id, companyID, users[c.agentIdx].ID, now.AddDate(0, -1, 0), now, commStructures[c.structureIdx].id,
			c.dealsCount, c.dealsRevenue, c.leasesCount, c.leasesRevenue, c.total, c.status)
		logged("AgentCommission", fmt.Sprintf("%s — AED %.0f", users[c.agentIdx].Name, c.total))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  23. EXPENSE CATEGORIES + EXPENSES
	// ══════════════════════════════════════════════════════════════════════════
	expCatIDs := make([]uuid.UUID, 0)
	expCats := []struct {
		id   uuid.UUID
		name string
		typ  string
	}{
		{id("exp-cat", "1"), "Utilities", "operating"},
		{id("exp-cat", "2"), "Maintenance & Repairs", "operating"},
		{id("exp-cat", "3"), "Property Management", "operating"},
		{id("exp-cat", "4"), "Marketing", "operating"},
		{id("exp-cat", "5"), "Insurance", "operating"},
		{id("exp-cat", "6"), "Professional Services", "operating"},
	}
	for _, c := range expCats {
		exec(ctx, pool, `
			INSERT INTO expense_categories (id,company_id,category_name,category_type)
			VALUES ($1,$2,$3,$4) ON CONFLICT (company_id, category_name) DO NOTHING
		`, c.id, companyID, c.name, c.typ)
		expCatIDs = append(expCatIDs, c.id)
		logged("ExpenseCategory", c.name)
	}

	expenses := []struct {
		ID         uuid.UUID
		CatIdx     int
		PropIdx    int
		Amount     float64
		Date       time.Time
		Desc       string
		Vendor     string
		PayMethod  string
		PayStatus  string
	}{
		// CatIdx: 0=Utilities 1=Maintenance 2=PropertyMgmt 3=Marketing 4=Insurance 5=ProfServices
		{id("exp", "1"), 0, 0, 2450.50, now.AddDate(0, 0, -20), "DEWA electricity bill — Marina Heights", "DEWA", "bank_transfer", "paid"},
		{id("exp", "2"), 0, 1, 3800.00, now.AddDate(0, 0, -18), "DEWA bill — Downtown", "DEWA", "bank_transfer", "paid"},
		{id("exp", "3"), 3, 3, 25000.00, now.AddDate(0, 0, -30), "Property listing campaign — Palm villa", "PropertyFinder", "card", "paid"},
		{id("exp", "4"), 4, 0, 15000.00, now.AddDate(0, -1, 0), "Annual building insurance — Marina Heights", "Orient Insurance", "bank_transfer", "paid"},
		{id("exp", "5"), 2, 0, 5000.00, now.AddDate(0, 0, -10), "Property management fee — March 2026", "Masaar PM", "bank_transfer", "paid"},
		{id("exp", "6"), 0, 6, 890.75, now.AddDate(0, 0, -5), "DEWA bill — Business Bay", "DEWA", "bank_transfer", "paid"},
		{id("exp", "7"), 1, 4, 3200.00, now.AddDate(0, 0, -12), "Elevator maintenance prepayment", "Otis Elevators", "bank_transfer", "pending"},
		{id("exp", "8"), 3, 2, 3500.00, now.AddDate(0, 0, -3), "Google Ads campaign — Q2 2026", "Google Ads", "card", "paid"},
		{id("exp", "9"), 5, -1, 8500.00, now.AddDate(0, 0, -15), "Legal retainer — lease dispute consultation", "Al Suwaidi & Associates", "bank_transfer", "paid"},
	}
	for _, e := range expenses {
		var propID *uuid.UUID
		if e.PropIdx >= 0 {
			propID = &properties[e.PropIdx].ID
		}
		exec(ctx, pool, `
			INSERT INTO expenses (id,company_id,category_id,property_id,amount,currency,expense_date,description,vendor_name,payment_method,payment_status,created_by)
			VALUES ($1,$2,$3,$4,$5,'AED',$6,$7,$8,$9,$10,$11)
			ON CONFLICT (id) DO NOTHING
		`, e.ID, companyID, expCatIDs[e.CatIdx], propID, e.Amount, e.Date, e.Desc, e.Vendor, e.PayMethod, e.PayStatus, users[0].ID)
		preview := e.Desc
		if len(preview) > 40 {
			preview = preview[:40]
		}
		logged("Expense", fmt.Sprintf("AED %.2f — %s", e.Amount, preview))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  24. DOCUMENT TEMPLATES + DOCUMENTS + SIGNATURES
	// ══════════════════════════════════════════════════════════════════════════
	docTpls := []struct {
		id          uuid.UUID
		name        string
		docType     string
		lang        string
		sigRequired bool
	}{
		{id("doc-tpl", "1"), "Standard Lease Agreement (EN)", "lease_agreement", "en", true},
		{id("doc-tpl", "2"), "عقد الإيجار القياسي (AR)", "lease_agreement", "ar", true},
		{id("doc-tpl", "3"), "Maintenance Request Form", "maintenance_request", "en", false},
	}
	for _, t := range docTpls {
		exec(ctx, pool, `
			INSERT INTO document_templates (id,company_id,template_name,document_type,language,signature_required,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING
		`, t.id, companyID, t.name, t.docType, t.lang, t.sigRequired, users[0].ID)
		logged("DocumentTemplate", t.name)
	}

	documents := []struct {
		ID             uuid.UUID
		TplIdx         int
		RelatedEntity  string
		RelatedID      uuid.UUID
		Title          string
		SigStatus      string
	}{
		{id("doc", "1"), 0, "lease", leases[0].ID, "Lease Agreement — Marina Heights #204", "signed"},
		{id("doc", "2"), 0, "lease", leases[1].ID, "Lease Agreement — Downtown #1201", "pending"},
		{id("doc", "3"), 2, "maintenance", maintenance[2].ID, "Maintenance Request — JLT Kitchen Faucet", "not_required"},
	}
	for _, d := range documents {
		exec(ctx, pool, `
			INSERT INTO documents (id,company_id,document_type,original_template_id,related_entity_type,related_entity_id,document_title,signature_status,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (id) DO NOTHING
		`, d.ID, companyID, docTpls[d.TplIdx].docType, docTpls[d.TplIdx].id, d.RelatedEntity, d.RelatedID,
			d.Title, d.SigStatus, users[0].ID)
		logged("Document", d.Title)

		if d.SigStatus == "signed" {
			exec(ctx, pool, `
				INSERT INTO document_signatures (id,document_id,signer_name,signer_email,signature_field_name,signature_status,signed_at)
				VALUES ($1,$2,$3,$4,$5,'signed',$6) ON CONFLICT (id) DO NOTHING
			`, id("doc-sig", d.ID.String()), d.ID, "John Smith", "john.smith@acmecorp.ae", "tenant_signature", now.AddDate(0, 0, -20))
			logged("DocumentSignature", "John Smith")

			exec(ctx, pool, `
				INSERT INTO document_audit_log (id,document_id,action,actor_id,actor_name)
				VALUES ($1,$2,'signed',$3,'Ahmed Al Mansoori') ON CONFLICT (id) DO NOTHING
			`, id("doc-audit", d.ID.String()), d.ID, users[0].ID)
		}
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  25. MESSAGE TEMPLATES
	// ══════════════════════════════════════════════════════════════════════════
	msgTpls := []struct {
		id       uuid.UUID
		name     string
		body     string
		category string
		active   bool
	}{
		{id("msg-tpl", "1"), "Welcome Message (AR)", "مرحباً بك في مسار! شكراً لتواصلك معنا. كيف يمكننا مساعدتك اليوم؟", "welcome", true},
		{id("msg-tpl", "2"), "Welcome Message (EN)", "Welcome to Masaar! Thank you for reaching out. How can we help you today?", "welcome", true},
		{id("msg-tpl", "3"), "Viewing Confirmation", "Dear {{name}}, your property viewing at {{property}} is confirmed for {{date}} at {{time}}. See you there!", "viewing", true},
		{id("msg-tpl", "4"), "Payment Reminder", "Dear {{name}}, your rent payment of AED {{amount}} is due on {{date}}. Please arrange payment to avoid late fees.", "payment", true},
		{id("msg-tpl", "5"), "Offer Follow-Up", "Hi {{name}}, following up on your offer for {{property}}. Have you had a chance to discuss? Happy to answer any questions.", "sales", true},
	}
	for _, m := range msgTpls {
		exec(ctx, pool, `
			INSERT INTO message_templates (id,company_id,name,body,category,is_active,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING
		`, m.id, companyID, m.name, m.body, m.category, m.active, users[0].ID)
		logged("MessageTemplate", m.name)
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  26. WHATSAPP OUTBOUND
	// ══════════════════════════════════════════════════════════════════════════
	outbounds := []struct {
		threadIdx int
		toNumber  string
		body      string
		status    string
		userIdx   int
	}{
		{0, contacts[0].Phone, "ممتاز! لدي 3 خيارات متاحة. هل تفضل زيارة اليوم؟", "sent", 1},
		{1, contacts[1].Phone, "Perfect, I have 3 options ready. Shall we schedule a viewing?", "sent", 2},
		{4, contacts[8].Phone, "أهلاً نورة! لدينا خيارين ممتازين. تفضلي بالزيارة", "sent", 2},
	}
	for _, o := range outbounds {
		exec(ctx, pool, `
			INSERT INTO whatsapp_outbound (thread_id,to_number,message_body,wa_message_id,status,created_by)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, id("thread", fmt.Sprintf("%d", o.threadIdx)), o.toNumber, o.body,
			"wa-out-"+id("out", fmt.Sprintf("%d", o.threadIdx)).String(), o.status, users[o.userIdx].ID)
		logged("WhatsAppOutbound", fmt.Sprintf("→ %s", o.toNumber))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  27. CUSTOM FIELD DEFINITIONS + VALUES
	// ══════════════════════════════════════════════════════════════════════════
	customFields := []struct {
		id         uuid.UUID
		entityType string
		fieldName  string
		fieldLabel string
		fieldType  string
		required   bool
		order      int
	}{
		{id("cf", "1"), "contact", "preferred_contact_time", "Preferred Contact Time", "text", false, 1},
		{id("cf", "2"), "lead", "budget_range", "Budget Range", "text", false, 1},
		{id("cf", "3"), "property", "pet_friendly", "Pet Friendly", "boolean", false, 1},
	}
	for _, f := range customFields {
		exec(ctx, pool, `
			INSERT INTO custom_field_definitions (id,company_id,entity_type,field_name,field_label,field_type,is_required,display_order)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING
		`, f.id, companyID, f.entityType, f.fieldName, f.fieldLabel, f.fieldType, f.required, f.order)
		logged("CustomFieldDef", f.fieldLabel)

		// Seed a few values
		if f.entityType == "contact" {
			exec(ctx, pool, `
				INSERT INTO custom_field_values (id,entity_id,entity_type,field_id,value)
				VALUES ($1,$2,$3,$4,$5) ON CONFLICT (id) DO NOTHING
			`, id("cfv", "1"), contacts[0].ID, "contact", f.id, "Evening after 6pm")
		}
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  28. API KEYS
	// ══════════════════════════════════════════════════════════════════════════
	apiKeys := []struct {
		id     uuid.UUID
		name   string
		prefix string
		scopes string
	}{
		{id("apikey", "1"), "Production API Key", "msr_prod", "contacts:read,leads:read,deals:read"},
		{id("apikey", "2"), "Webhook Integration", "msr_webhook", "contacts:read,leads:write"},
	}
	for _, k := range apiKeys {
		exec(ctx, pool, `
			INSERT INTO api_keys (id,company_id,name,key_hash,key_prefix,scopes)
			VALUES ($1,$2,$3,'hashed_'||$4,$4,$5) ON CONFLICT (company_id, name) DO NOTHING
		`, k.id, companyID, k.name, k.prefix, k.scopes)
		logged("APIKey", k.name)
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  29. WEBHOOK SUBSCRIPTIONS + DELIVERIES
	// ══════════════════════════════════════════════════════════════════════════
	webhooks := []struct {
		id     uuid.UUID
		name   string
		url    string
		events string
		active bool
	}{
		{id("wh", "1"), "Lead Sync — HubSpot", "https://hooks.example.com/hubspot/leads", "lead.created,lead.updated", true},
		{id("wh", "2"), "Payment Notifications", "https://hooks.example.com/payments", "payment.received,payment.overdue", true},
	}
	for _, w := range webhooks {
		exec(ctx, pool, `
			INSERT INTO webhook_subscriptions (id,company_id,name,url,events,active)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (id) DO NOTHING
		`, w.id, companyID, w.name, w.url, w.events, w.active)
		logged("WebhookSub", w.name)

		exec(ctx, pool, `
			INSERT INTO webhook_deliveries (id,subscription_id,event,payload,status,response_code)
			VALUES ($1,$2,'lead.created','{}','delivered',200) ON CONFLICT (id) DO NOTHING
		`, id("wh-del", w.id.String()), w.id)
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  30. BANK INTEGRATIONS + TRANSACTIONS + STATEMENTS
	// ══════════════════════════════════════════════════════════════════════════
	bankID := id("bank", "enb")
	exec(ctx, pool, `
		INSERT INTO bank_integrations (id,company_id,bank_name,bank_code,account_number,account_name,iban,integration_type,status,is_connected,created_by)
		VALUES ($1,$2,'Emirates NBD','ENB','AE100123456789','Masaar Properties LLC','AE3802600000123456789','manual','connected',true,$3)
		ON CONFLICT (id) DO NOTHING
	`, bankID, companyID, users[0].ID)
	logged("BankIntegration", "Emirates NBD")

	bankTxns := []struct {
		id       uuid.UUID
		date     time.Time
		amount   float64
		fromName string
		toName   string
		ref      string
	}{
		{id("bank-txn", "1"), now.AddDate(0, 0, -5), 12000, "John Smith", "Masaar Properties LLC", "Rent Payment — Marina Heights"},
		{id("bank-txn", "2"), now.AddDate(0, 0, -8), 25000, "Aisha Al Maktoum", "Masaar Properties LLC", "Rent Payment — Downtown"},
		{id("bank-txn", "3"), now.AddDate(0, 0, -12), 6500, "Raj Patel", "Masaar Properties LLC", "Rent Payment — JLT"},
		{id("bank-txn", "4"), now.AddDate(0, 0, -1), 7000, "Omar Hassan", "Masaar Properties LLC", "Rent Payment — Business Bay"},
	}
	for _, t := range bankTxns {
		exec(ctx, pool, `
			INSERT INTO bank_transactions (id,company_id,bank_integration_id,external_id,transaction_date,amount,currency,from_name,to_name,reference,transaction_type,status)
			VALUES ($1,$2,$3,$4,$5,$6,'AED',$7,$8,$9,'credit','posted')
			ON CONFLICT (external_id) DO NOTHING
		`, t.id, companyID, bankID, "ext-"+t.id.String(), t.date, t.amount, t.fromName, t.toName, t.ref)
		logged("BankTransaction", fmt.Sprintf("AED %.0f from %s", t.amount, t.fromName))
	}

	exec(ctx, pool, `
		INSERT INTO bank_statements (id,company_id,bank_integration_id,file_name,file_size_bytes,file_format,processing_status,transactions_imported,created_by)
		VALUES ($1,$2,$3,'statement_mar2026.csv',24580,'csv','processed',4,$4)
		ON CONFLICT (id) DO NOTHING
	`, id("bank-stmt", "1"), companyID, bankID, users[0].ID)
	logged("BankStatement", "statement_mar2026.csv")

	// ══════════════════════════════════════════════════════════════════════════
	//  31. PAYMENT CONFIRMATIONS
	// ══════════════════════════════════════════════════════════════════════════
	exec(ctx, pool, `
		INSERT INTO payment_confirmations (id,company_id,payment_id,confirmation_number,tenant_email,tenant_phone,delivery_status,delivery_method)
		VALUES ($1,$2,$3,'CNF-2026-001','john.smith@acmecorp.ae','+971506611111','delivered','email')
		ON CONFLICT (confirmation_number) DO NOTHING
	`, id("pmt-conf", "1"), companyID, payments[0].ID)
	logged("PaymentConfirmation", "CNF-2026-001")

	// ══════════════════════════════════════════════════════════════════════════
	//  32. USAGE TRACKING
	// ══════════════════════════════════════════════════════════════════════════
	usageResources := []struct {
		id       uuid.UUID
		resource string
		count    int
	}{
		{id("usage", "contacts"), "contacts", 10},
		{id("usage", "leads"), "leads", 10},
		{id("usage", "deals"), "deals", 5},
		{id("usage", "properties"), "properties", 8},
		{id("usage", "tenants"), "tenants", 8},
		{id("usage", "leases"), "leases", 7},
		{id("usage", "messages"), "messages", 19},
	}
	for _, u := range usageResources {
		exec(ctx, pool, `
			INSERT INTO usage_tracking (id,company_id,resource,period,count)
			VALUES ($1,$2,$3,to_char(NOW(),'YYYY-MM'),$4)
			ON CONFLICT (company_id, resource, period) DO UPDATE SET count = EXCLUDED.count
		`, u.id, companyID, u.resource, u.count)
		logged("UsageTracking", fmt.Sprintf("%s: %d", u.resource, u.count))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  33. AUDIT LOGS
	// ══════════════════════════════════════════════════════════════════════════
	auditLogs := []struct {
		entityType string
		entityID   uuid.UUID
		action     string
		actorIdx   int
	}{
		{"user", users[0].ID, "login", 0},
		{"lead", leads[4].ID, "stage_change: proposal → won", 2},
		{"deal", deals[0].ID, "created", 2},
		{"lease", leases[0].ID, "created", 0},
		{"payment", payments[0].ID, "received", 0},
	}
	for _, a := range auditLogs {
		exec(ctx, pool, `
			INSERT INTO audit_logs (entity_type,entity_id,action,actor_id)
			VALUES ($1,$2,$3,$4)
		`, a.entityType, a.entityID, a.action, users[a.actorIdx].ID)
	}
	logged("AuditLogs", fmt.Sprintf("%d entries", len(auditLogs)))

	// ══════════════════════════════════════════════════════════════════════════
	//  34. LISTINGS  (migration 0048)
	// ══════════════════════════════════════════════════════════════════════════
	listings := []struct {
		ID       uuid.UUID
		Title    string
		PropType string
		LType    string
		Price    float64
		Area     string
		Community string
		City     string
		Emirate  string
		Bed      int
		Bath     int
		Sqft     float64
		Furn     string
		Status   string
		AgentIdx int
	}{
		{id("lst", "1"), "Stunning 2BR with Marina View", "apartment", "rent", 120000, "Dubai Marina", "Dubai Marina", "Dubai", "Dubai", 2, 2, 1250, "furnished", "published", 1},
		{id("lst", "2"), "Premium 3BR Downtown Penthouse", "apartment", "rent", 300000, "Downtown Dubai", "Downtown Dubai", "Dubai", "Dubai", 3, 3, 1800, "furnished", "published", 0},
		{id("lst", "3"), "Cozy 1BR in JLT Cluster Y", "apartment", "rent", 78000, "JLT", "JLT", "Dubai", "Dubai", 1, 1, 750, "semi-furnished", "published", 1},
		{id("lst", "4"), "5BR Beachfront Villa — Palm", "villa", "rent", 950000, "Palm Jumeirah", "Palm Jumeirah", "Dubai", "Dubai", 5, 6, 5200, "unfurnished", "published", 2},
		{id("lst", "5"), "DIFC Grade A Office 2200sqft", "commercial", "rent", 540000, "DIFC", "DIFC", "Dubai", "Dubai", 0, 2, 2200, "furnished", "published", 2},
		{id("lst", "6"), "3BR Townhouse in Arabian Ranches", "townhouse", "rent", 180000, "Arabian Ranches", "Arabian Ranches", "Dubai", "Dubai", 3, 3, 2100, "unfurnished", "draft", 1},
		{id("lst", "7"), "2BR on Al Reem Island", "apartment", "rent", 132000, "Al Reem Island", "Al Reem Island", "Abu Dhabi", "Abu Dhabi", 2, 2, 1150, "furnished", "published", 0},
		{id("lst", "8"), "One Bedroom in Business Bay", "apartment", "rent", 84000, "Business Bay", "Business Bay", "Dubai", "Dubai", 1, 1, 680, "furnished", "published", 1},
	}
	for _, l := range listings {
		exec(ctx, pool, `
			INSERT INTO listings (id,company_id,title,property_type,listing_type,price,area,community,city,emirate,bedrooms,bathrooms,total_sqft,furnishing,status,assigned_to,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
			ON CONFLICT (id) DO NOTHING
		`, l.ID, companyID, l.Title, l.PropType, l.LType, l.Price, l.Area, l.Community, l.City, l.Emirate,
			l.Bed, l.Bath, l.Sqft, l.Furn, l.Status, users[l.AgentIdx].ID, users[0].ID)
		logged("Listing", l.Title)
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  35. PIPELINE STAGES  (migration 0057 — seed defaults if not present)
	// ══════════════════════════════════════════════════════════════════════════
	defaultStages := []struct {
		name      string
		order     int
		color     string
		isWon     bool
		isLost    bool
		isDefault bool
	}{
		{"new", 1, "#6366f1", false, false, true},
		{"contacted", 2, "#f59e0b", false, false, false},
		{"qualified", 3, "#10b981", false, false, false},
		{"proposal", 4, "#3b82f6", false, false, false},
		{"won", 5, "#059669", true, false, false},
		{"lost", 6, "#ef4444", false, true, false},
	}
	for _, s := range defaultStages {
		exec(ctx, pool, `
			INSERT INTO pipeline_stages (id,company_id,entity_type,name,sort_order,color,is_won,is_lost,is_default)
			VALUES ($1,$2,'lead',$3,$4,$5,$6,$7,$8) ON CONFLICT (company_id,entity_type,name) DO NOTHING
		`, id("pstage", s.name), companyID, s.name, s.order, s.color, s.isWon, s.isLost, s.isDefault)
	}
	logged("PipelineStages", "6 default stages seeded")

	// ══════════════════════════════════════════════════════════════════════════
	//  36. OFFERS  (migration 0051)
	// ══════════════════════════════════════════════════════════════════════════
	offers := []struct {
		id          uuid.UUID
		listingIdx  int
		contactIdx  int
		agentIdx    int
		amount      float64
		status      string
	}{
		{id("offer", "1"), 0, 0, 1, 115000, "countered"},
		{id("offer", "2"), 1, 1, 2, 290000, "submitted"},
		{id("offer", "3"), 4, 3, 2, 520000, "accepted"},
	}
	for _, o := range offers {
		exec(ctx, pool, `
			INSERT INTO offers (id,listing_id,contact_id,agent_id,offer_amount,currency,status,created_by)
			VALUES ($1,$2,$3,$4,$5,'AED',$6,$7) ON CONFLICT (id) DO NOTHING
		`, o.id, listings[o.listingIdx].ID, contacts[o.contactIdx].ID, users[o.agentIdx].ID,
			o.amount, o.status, users[0].ID)
		logged("Offer", fmt.Sprintf("AED %.0f — %s", o.amount, o.status))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  37. VIEWINGS  (migration 0054)
	// ══════════════════════════════════════════════════════════════════════════
	viewings := []struct {
		id         uuid.UUID
		listingIdx int
		contactIdx int
		agentIdx   int
		status     string
		offsetDays int
	}{
		{id("view", "1"), 0, 0, 1, "completed", -5},
		{id("view", "2"), 1, 1, 2, "scheduled", 3},
		{id("view", "3"), 3, 3, 2, "confirmed", 1},
		{id("view", "4"), 7, 5, 1, "completed", -10},
	}
	for _, v := range viewings {
		scheduled := now.AddDate(0, 0, v.offsetDays)
		var checkedIn *time.Time
		if v.status == "completed" {
			checkedIn = timePtr(scheduled.Add(time.Hour))
		}
		exec(ctx, pool, `
			INSERT INTO viewings (id,listing_id,contact_id,agent_id,scheduled_at,status,checked_in_at,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING
		`, v.id, listings[v.listingIdx].ID, contacts[v.contactIdx].ID, users[v.agentIdx].ID,
			scheduled, v.status, checkedIn, users[0].ID)
		logged("Viewing", fmt.Sprintf("%s — %s", listings[v.listingIdx].Title[:30], v.status))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  38. LEAD ROTATION SETTINGS  (migration 0052)
	// ══════════════════════════════════════════════════════════════════════════
	exec(ctx, pool, `
		INSERT INTO lead_rotation_settings (id,company_id,mode,enabled,rotation_index)
		VALUES ($1,$2,'round_robin',true,0) ON CONFLICT (company_id) DO NOTHING
	`, id("rotation-settings"), companyID)
	logged("LeadRotationSettings", "round_robin enabled")

	// ══════════════════════════════════════════════════════════════════════════
	//  39. AGENT TARGETS  (migration 0053)
	// ══════════════════════════════════════════════════════════════════════════
	agentTargets := []struct {
		agentIdx int
		metric   string
		value    float64
	}{
		{1, "deals_won", 8},
		{1, "revenue", 5000000},
		{2, "deals_won", 5},
		{2, "revenue", 3000000},
	}
	period := now.Format("2006-01")
	for _, t := range agentTargets {
		exec(ctx, pool, `
			INSERT INTO agent_targets (id,agent_id,company_id,metric,target_value,period)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (agent_id,metric,period) DO NOTHING
		`, id("target", fmt.Sprintf("%d-%s", t.agentIdx, t.metric)), users[t.agentIdx].ID, companyID, t.metric, t.value, period)
		logged("AgentTarget", fmt.Sprintf("%s → %.0f (agent %d)", t.metric, t.value, t.agentIdx))
	}

	// ══════════════════════════════════════════════════════════════════════════
	//  40. APPROVAL CONFIGS  (migration 0056)
	// ══════════════════════════════════════════════════════════════════════════
	exec(ctx, pool, `
		INSERT INTO approval_configs (id,company_id,listing_approval,deal_approval_above,offer_approval_above)
		VALUES ($1,$2,true,500000,300000) ON CONFLICT (company_id) DO NOTHING
	`, id("approval-config"), companyID)
	logged("ApprovalConfig", "listing=true, deal>500K, offer>300K")

	// ══════════════════════════════════════════════════════════════════════════
	fmt.Println("\n✅ Demo data seeded successfully!")
	fmt.Println("   Demo login: ahmed@masaar.local / Demo@1234  (admin)")
	fmt.Println("   Supporting agent accounts exist as data references only (no public password).")
	fmt.Printf("   Trial ends: %s (%d days remaining)\n", trialEnd.Format("Jan 2, 2006"), 25)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func exec(ctx context.Context, pool *pgxpool.Pool, sql string, args ...interface{}) {
	if _, err := pool.Exec(ctx, sql, args...); err != nil {
		log.Printf("  ⚠ %v", err)
	}
}

func logged(entity string, msg string) {
	fmt.Printf("  ✓ %s: %s\n", entity, msg)
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func strPtr(s string) *string {
	return &s
}
