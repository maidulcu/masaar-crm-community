// Package notice drafts the notices a landlord's agent serves on a tenant: a change of
// rent on renewal, renewal terms, or non-renewal. The drafts are starting points in English
// and Arabic for the agent to review and edit; they are not legal advice. Notice periods
// and the conditions for raising rent or ending a tenancy depend on the emirate (in Dubai,
// for example, Law No. 26 of 2007 as amended and the RERA rental index); check the current
// rules before serving a notice.
package notice

import (
	"fmt"
	"strings"
	"time"

	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
)

// Kinds of notice.
const (
	RentChange    = "rent_change"
	RenewalTerms  = "renewal_terms"
	NonRenewal    = "non_renewal"
	Other         = "other"
	dateFormat    = "2 January 2006"
	dateFormatNum = "2006-01-02"
)

// ValidKind reports whether k is a known kind.
func ValidKind(k string) bool {
	switch k {
	case RentChange, RenewalTerms, NonRenewal, Other:
		return true
	}
	return false
}

// Input is what a draft is written from.
type Input struct {
	Kind          string
	Today         time.Time
	TenantName    string
	PropertyName  string
	UnitNumber    string
	EjariNumber   string
	CompanyName   string
	EndDate       time.Time
	Currency      string
	CurrentAnnual float64
	NewAnnual     float64
	EffectiveDate time.Time // defaults to the day after the lease ends
	NoticeDays    int
}

func money(cur string, v float64) string {
	d, ok := jurisdiction.CurrencyDecimals(cur)
	if !ok {
		d = 2
	}
	return fmt.Sprintf("%s %s", cur, commas(fmt.Sprintf("%.*f", d, v)))
}

func commas(s string) string {
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String() + frac
}

// Draft returns the English and Arabic text of a notice.
func Draft(in Input) (en, ar string) {
	if in.EffectiveDate.IsZero() {
		in.EffectiveDate = in.EndDate.AddDate(0, 0, 1)
	}
	prop := in.PropertyName
	if in.UnitNumber != "" {
		prop += ", unit " + in.UnitNumber
	}
	propAR := in.PropertyName
	if in.UnitNumber != "" {
		propAR += "، وحدة " + in.UnitNumber
	}
	ejari, ejariAR := "", ""
	if in.EjariNumber != "" {
		ejari = "\nEjari / tenancy registration no.: " + in.EjariNumber
		ejariAR = "\nرقم تسجيل عقد الإيجار (إيجاري): " + in.EjariNumber
	}
	end := in.EndDate.Format(dateFormat)
	endNum := in.EndDate.Format(dateFormatNum)
	eff := in.EffectiveDate.Format(dateFormat)
	effNum := in.EffectiveDate.Format(dateFormatNum)

	headEN := fmt.Sprintf("Date: %s\nTo: %s\nProperty: %s%s\n\n", in.Today.Format(dateFormat), in.TenantName, prop, ejari)
	headAR := fmt.Sprintf("التاريخ: %s\nإلى: %s\nالعقار: %s%s\n\n", in.Today.Format(dateFormatNum), in.TenantName, propAR, ejariAR)
	signEN := "\n\nYours sincerely,\n" + in.CompanyName
	signAR := "\n\nوتفضلوا بقبول فائق الاحترام،\n" + in.CompanyName

	switch in.Kind {
	case RentChange:
		en = "NOTICE OF CHANGE OF RENT\n\n" + headEN + fmt.Sprintf(
			"Your tenancy contract for the above property ends on %s. We hereby notify you that, on renewal, the annual rent will change from %s to %s, with effect from %s.\n\n"+
				"This notice is given at least %d days before the end of the contract. Please contact us to arrange the renewal.",
			end, money(in.Currency, in.CurrentAnnual), money(in.Currency, in.NewAnnual), eff, in.NoticeDays) + signEN
		ar = "إشعار بتعديل القيمة الإيجارية\n\n" + headAR + fmt.Sprintf(
			"ينتهي عقد إيجار العقار المذكور أعلاه بتاريخ %s. نحيطكم علماً بأنه عند التجديد ستتغير القيمة الإيجارية السنوية من %s إلى %s اعتباراً من %s.\n\n"+
				"يُقدَّم هذا الإشعار قبل %d يوماً على الأقل من انتهاء العقد. يرجى التواصل معنا لترتيب التجديد.",
			endNum, money(in.Currency, in.CurrentAnnual), money(in.Currency, in.NewAnnual), effNum, in.NoticeDays) + signAR
	case RenewalTerms:
		en = "NOTICE OF RENEWAL TERMS\n\n" + headEN + fmt.Sprintf(
			"Your tenancy contract for the above property ends on %s. We would be pleased to renew it from %s at an annual rent of %s, on the same terms unless agreed otherwise.\n\n"+
				"Please let us know whether you wish to renew.", end, eff, money(in.Currency, in.NewAnnual)) + signEN
		ar = "إشعار بشروط تجديد العقد\n\n" + headAR + fmt.Sprintf(
			"ينتهي عقد إيجار العقار المذكور أعلاه بتاريخ %s. يسرنا تجديد العقد اعتباراً من %s بقيمة إيجارية سنوية قدرها %s، وبالشروط نفسها ما لم يُتفق على غير ذلك.\n\n"+
				"يرجى إعلامنا برغبتكم في التجديد.", endNum, effNum, money(in.Currency, in.NewAnnual)) + signAR
	case NonRenewal:
		en = "NOTICE OF NON-RENEWAL\n\n" + headEN + fmt.Sprintf(
			"Your tenancy contract for the above property ends on %s. We hereby notify you that the contract will not be renewed, and ask you to hand over the property on or before that date.\n\n"+
				"[State the reason for non-renewal, as required by the applicable law.]", end) + signEN
		ar = "إشعار بعدم تجديد العقد\n\n" + headAR + fmt.Sprintf(
			"ينتهي عقد إيجار العقار المذكور أعلاه بتاريخ %s. نحيطكم علماً بأن العقد لن يُجدَّد، ونرجو تسليم العقار في موعد أقصاه هذا التاريخ.\n\n"+
				"[اذكر سبب عدم التجديد وفق القانون المعمول به.]", endNum) + signAR
	default:
		en = headEN + "[Text of the notice]" + signEN
		ar = headAR + "[نص الإشعار]" + signAR
	}
	return en, ar
}
