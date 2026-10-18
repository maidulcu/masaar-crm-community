package jurisdiction

import "time"

// PermitREGA is the advertising licence the Saudi Real Estate General Authority (REGA)
// requires on a property advertisement.
const PermitREGA = "rega"

// Saudi Arabia: 15% VAT (ZATCA), SAR, Friday–Saturday weekend, 13 provinces. Every
// province requires a REGA advertising licence on listings. Check the current rules with
// ZATCA and REGA before relying on these values.
func init() {
	registry["SA"] = Rules{
		Country:          "SA",
		NameEN:           "Saudi Arabia",
		NameAR:           "المملكة العربية السعودية",
		Currency:         "SAR",
		CurrencyDecimals: 2,
		Timezone:         "Asia/Riyadh",
		Weekend:          []time.Weekday{time.Friday, time.Saturday},
		TaxIDDigits:      15,
		VATRates: map[VATCategory]float64{
			VATStandard:   0.15,
			VATZeroRated:  0,
			VATExempt:     0,
			VATOutOfScope: 0,
		},
		Regions: []Region{
			{Code: "SA-01", NameEN: "Riyadh", NameAR: "الرياض", ListingPermit: PermitREGA},
			{Code: "SA-02", NameEN: "Makkah", NameAR: "مكة المكرمة", ListingPermit: PermitREGA},
			{Code: "SA-03", NameEN: "Madinah", NameAR: "المدينة المنورة", ListingPermit: PermitREGA},
			{Code: "SA-04", NameEN: "Eastern Province", NameAR: "المنطقة الشرقية", ListingPermit: PermitREGA},
			{Code: "SA-05", NameEN: "Al-Qassim", NameAR: "القصيم", ListingPermit: PermitREGA},
			{Code: "SA-06", NameEN: "Ha'il", NameAR: "حائل", ListingPermit: PermitREGA},
			{Code: "SA-07", NameEN: "Tabuk", NameAR: "تبوك", ListingPermit: PermitREGA},
			{Code: "SA-08", NameEN: "Northern Borders", NameAR: "الحدود الشمالية", ListingPermit: PermitREGA},
			{Code: "SA-09", NameEN: "Jazan", NameAR: "جازان", ListingPermit: PermitREGA},
			{Code: "SA-10", NameEN: "Najran", NameAR: "نجران", ListingPermit: PermitREGA},
			{Code: "SA-11", NameEN: "Al-Bahah", NameAR: "الباحة", ListingPermit: PermitREGA},
			{Code: "SA-12", NameEN: "Al-Jawf", NameAR: "الجوف", ListingPermit: PermitREGA},
			{Code: "SA-14", NameEN: "Asir", NameAR: "عسير", ListingPermit: PermitREGA},
		},
	}
}
