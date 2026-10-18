// Package rera describes the RERA (Dubai Real Estate Regulatory Agency) brokerage forms
// Masaar can pre-fill: Form A (seller/landlord–broker), Form B (buyer/tenant–broker),
// Form I (broker–broker) and Form F (memorandum of understanding between seller and buyer).
//
// Masaar produces a pre-filled agreement record with each form's key fields, for the
// parties to review and sign. It does not submit anything to the Dubai Land Department and
// does not replace the official forms or the DLD's own systems; check the current official
// templates before relying on these.
package rera

import "fmt"

// Field is one entry on a form.
type Field struct {
	Key string `json:"key"`
	EN  string `json:"en"`
	AR  string `json:"ar"`
	// Long fields are shown as multi-line text.
	Long bool `json:"long,omitempty"`
}

// Section groups fields under a heading.
type Section struct {
	EN     string  `json:"en"`
	AR     string  `json:"ar"`
	Fields []Field `json:"fields"`
}

// Form is a form's definition.
type Form struct {
	Type     string    `json:"type"` // A | B | I | F
	TitleEN  string    `json:"title_en"`
	TitleAR  string    `json:"title_ar"`
	Sections []Section `json:"sections"`
	// Signers names the parties expected to sign, for the signature panel.
	Signers []string `json:"signers"`
}

var brokerSection = Section{EN: "Broker", AR: "الوسيط", Fields: []Field{
	{Key: "broker_company", EN: "Brokerage", AR: "مكتب الوساطة"},
	{Key: "broker_orn", EN: "Office registration no. (ORN)", AR: "رقم تسجيل المكتب (ORN)"},
	{Key: "broker_address", EN: "Office address", AR: "عنوان المكتب"},
	{Key: "broker_phone", EN: "Office phone", AR: "هاتف المكتب"},
	{Key: "agent_name", EN: "Agent", AR: "الوسيط العقاري"},
	{Key: "agent_brn", EN: "Broker registration no. (BRN)", AR: "رقم تسجيل الوسيط (BRN)"},
}}

var propertySection = Section{EN: "Property", AR: "العقار", Fields: []Field{
	{Key: "property_title", EN: "Property", AR: "العقار"},
	{Key: "property_type", EN: "Type", AR: "النوع"},
	{Key: "emirate", EN: "Emirate", AR: "الإمارة"},
	{Key: "community", EN: "Community / area", AR: "المجتمع / المنطقة"},
	{Key: "unit_reference", EN: "Unit / reference", AR: "الوحدة / المرجع"},
	{Key: "size_sqft", EN: "Size (sq ft)", AR: "المساحة (قدم مربع)"},
	{Key: "permit_number", EN: "Advertising permit no.", AR: "رقم تصريح الإعلان"},
}}

func termsSection(extra ...Field) Section {
	return Section{EN: "Terms", AR: "الشروط", Fields: append(extra,
		Field{Key: "start_date", EN: "Start date", AR: "تاريخ البدء"},
		Field{Key: "end_date", EN: "End date", AR: "تاريخ الانتهاء"},
		Field{Key: "special_conditions", EN: "Special conditions", AR: "شروط خاصة", Long: true},
	)}
}

// Forms holds every supported form, by type.
var Forms = map[string]Form{
	"A": {
		Type: "A", TitleEN: "Form A — Agreement between Seller/Landlord and Broker",
		TitleAR: "النموذج (أ) — اتفاقية بين البائع/المؤجر والوسيط",
		Signers: []string{"owner", "broker"},
		Sections: []Section{
			{EN: "Seller / landlord", AR: "البائع / المؤجر", Fields: []Field{
				{Key: "owner_name", EN: "Name", AR: "الاسم"},
				{Key: "owner_phone", EN: "Phone", AR: "الهاتف"},
				{Key: "owner_email", EN: "Email", AR: "البريد الإلكتروني"},
				{Key: "owner_id_no", EN: "Emirates ID / passport no.", AR: "رقم الهوية / جواز السفر"},
			}},
			propertySection,
			brokerSection,
			termsSection(
				Field{Key: "listing_type", EN: "For sale / rent", AR: "للبيع / للإيجار"},
				Field{Key: "asking_price", EN: "Asking price", AR: "السعر المطلوب"},
				Field{Key: "commission_percent", EN: "Commission (%)", AR: "العمولة (%)"},
				Field{Key: "exclusive", EN: "Exclusive", AR: "حصري"},
			),
		},
	},
	"B": {
		Type: "B", TitleEN: "Form B — Agreement between Buyer/Tenant and Broker",
		TitleAR: "النموذج (ب) — اتفاقية بين المشتري/المستأجر والوسيط",
		Signers: []string{"buyer", "broker"},
		Sections: []Section{
			{EN: "Buyer / tenant", AR: "المشتري / المستأجر", Fields: []Field{
				{Key: "buyer_name", EN: "Name", AR: "الاسم"},
				{Key: "buyer_phone", EN: "Phone", AR: "الهاتف"},
				{Key: "buyer_email", EN: "Email", AR: "البريد الإلكتروني"},
				{Key: "buyer_id_no", EN: "Emirates ID / passport no.", AR: "رقم الهوية / جواز السفر"},
				{Key: "buyer_nationality", EN: "Nationality", AR: "الجنسية"},
			}},
			{EN: "Requirement", AR: "المتطلبات", Fields: []Field{
				{Key: "requirement_type", EN: "Buy / rent", AR: "شراء / إيجار"},
				{Key: "property_type", EN: "Property type", AR: "نوع العقار"},
				{Key: "preferred_areas", EN: "Preferred areas", AR: "المناطق المفضلة"},
				{Key: "budget", EN: "Budget", AR: "الميزانية"},
			}},
			brokerSection,
			termsSection(Field{Key: "commission_percent", EN: "Commission (%)", AR: "العمولة (%)"}),
		},
	},
	"I": {
		Type: "I", TitleEN: "Form I — Agreement between Brokers",
		TitleAR: "النموذج (I) — اتفاقية بين الوسطاء",
		Signers: []string{"listing_broker", "buyer_broker"},
		Sections: []Section{
			{EN: "Listing broker", AR: "وسيط البائع", Fields: brokerSection.Fields},
			{EN: "Buyer's broker", AR: "وسيط المشتري", Fields: []Field{
				{Key: "other_broker_company", EN: "Brokerage", AR: "مكتب الوساطة"},
				{Key: "other_broker_orn", EN: "Office registration no. (ORN)", AR: "رقم تسجيل المكتب (ORN)"},
				{Key: "other_agent_name", EN: "Agent", AR: "الوسيط العقاري"},
				{Key: "other_agent_brn", EN: "Broker registration no. (BRN)", AR: "رقم تسجيل الوسيط (BRN)"},
				{Key: "buyer_name", EN: "Buyer / tenant", AR: "المشتري / المستأجر"},
			}},
			propertySection,
			termsSection(
				Field{Key: "asking_price", EN: "Price", AR: "السعر"},
				Field{Key: "commission_split_listing", EN: "Listing broker's share (%)", AR: "حصة وسيط البائع (%)"},
				Field{Key: "commission_split_buyer", EN: "Buyer's broker share (%)", AR: "حصة وسيط المشتري (%)"},
			),
		},
	},
	"F": {
		Type: "F", TitleEN: "Form F — Memorandum of Understanding between Seller and Buyer",
		TitleAR: "النموذج (و) — مذكرة تفاهم بين البائع والمشتري",
		Signers: []string{"seller", "buyer"},
		Sections: []Section{
			{EN: "Seller", AR: "البائع", Fields: []Field{
				{Key: "owner_name", EN: "Name", AR: "الاسم"},
				{Key: "owner_phone", EN: "Phone", AR: "الهاتف"},
				{Key: "owner_id_no", EN: "Emirates ID / passport no.", AR: "رقم الهوية / جواز السفر"},
			}},
			{EN: "Buyer", AR: "المشتري", Fields: []Field{
				{Key: "buyer_name", EN: "Name", AR: "الاسم"},
				{Key: "buyer_phone", EN: "Phone", AR: "الهاتف"},
				{Key: "buyer_id_no", EN: "Emirates ID / passport no.", AR: "رقم الهوية / جواز السفر"},
			}},
			propertySection,
			{EN: "Sale terms", AR: "شروط البيع", Fields: []Field{
				{Key: "sale_price", EN: "Sale price", AR: "سعر البيع"},
				{Key: "currency", EN: "Currency", AR: "العملة"},
				{Key: "deposit_amount", EN: "Deposit (security cheque)", AR: "العربون (شيك ضمان)"},
				{Key: "transfer_date", EN: "Transfer date at DLD", AR: "تاريخ النقل في دائرة الأراضي"},
				{Key: "mortgage", EN: "Buyer financing by mortgage", AR: "تمويل المشتري برهن عقاري"},
				{Key: "special_conditions", EN: "Special conditions", AR: "شروط خاصة", Long: true},
			}},
			brokerSection,
		},
	},
}

// Lookup returns the form definition for a type.
func Lookup(formType string) (Form, error) {
	f, ok := Forms[formType]
	if !ok {
		return Form{}, fmt.Errorf("unknown RERA form %q", formType)
	}
	return f, nil
}

// Keys returns the set of field keys a form accepts.
func (f Form) Keys() map[string]bool {
	keys := map[string]bool{}
	for _, s := range f.Sections {
		for _, fl := range s.Fields {
			keys[fl.Key] = true
		}
	}
	return keys
}

// Merge returns prefill overridden by overrides, keeping only the form's own keys and
// bounding value length.
func (f Form) Merge(prefill, overrides map[string]string) (map[string]string, error) {
	keys := f.Keys()
	out := map[string]string{}
	for k, v := range prefill {
		if keys[k] && v != "" {
			out[k] = v
		}
	}
	for k, v := range overrides {
		if !keys[k] {
			return nil, fmt.Errorf("form %s has no field %q", f.Type, k)
		}
		if len(v) > 4000 {
			return nil, fmt.Errorf("field %q is too long", k)
		}
		out[k] = v
	}
	return out, nil
}
