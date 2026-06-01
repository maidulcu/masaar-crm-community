package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

type Config struct {
	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
	// Azure Communication Services
	AzureEndpoint   string
	AzureKey        string
	AzureFromAddress string
}

type Service struct {
	cfg      *Config
	azureSvc *AzureService
}

func NewService(cfg *Config) *Service {
	s := &Service{cfg: cfg}
	if cfg.AzureEndpoint != "" && cfg.AzureKey != "" {
		s.azureSvc = NewAzureService(cfg.AzureEndpoint, cfg.AzureKey, cfg.AzureFromAddress)
	}
	return s
}

func (s *Service) ProviderName() string {
	if s.azureSvc != nil {
		return "azure"
	}
	return "smtp"
}

func (s *Service) IsConfigured() bool {
	if s.azureSvc != nil {
		return true
	}
	return s.cfg != nil && s.cfg.SMTPHost != "" && s.cfg.SMTPUser != ""
}

func (s *Service) Send(email *domain.EmailHistory) error {
	if s.azureSvc != nil {
		return s.azureSvc.Send(email)
	}
	return s.sendSMTP(email)
}

func (s *Service) sendSMTP(email *domain.EmailHistory) error {
	if !s.IsConfigured() {
		return fmt.Errorf("email service not configured")
	}

	if email.FromEmail == "" {
		email.FromEmail = fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.FromEmail)
	}

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)

	var body bytes.Buffer

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n",
		email.FromEmail,
		email.ToEmail,
		email.Subject,
	)

	if email.HTMLBody != "" {
		headers += "MIME-Version: 1.0\r\nContent-Type: text/html; charset=\"utf-8\"\r\n"
	} else {
		headers += "MIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n"
	}

	body.WriteString(headers + "\r\n")

	if email.HTMLBody != "" {
		body.WriteString(email.HTMLBody)
	} else {
		body.WriteString(email.Body)
	}

	err := smtp.SendMail(addr, auth, s.cfg.FromEmail, []string{email.ToEmail}, body.Bytes())
	return err
}

// ─── Email Templates ──────────────────────────────────────────────────────────

type InvoiceData struct {
	InvoiceNo   string
	DealTitle   string
	Amount      float64
	VATAmount   float64
	Total       float64
	IssuedDate  string
	DueDate     string
	ContactName string
	Lang string
}

func (d InvoiceData) IsArabic() bool { return d.Lang == "ar" }

func (s *Service) RenderInvoiceTemplate(data InvoiceData) (string, error) {
	const tmpl = `
<!DOCTYPE html>
<html lang="{{if .IsArabic}}ar{{else}}en{{end}}" dir="{{if .IsArabic}}rtl{{else}}ltr{{end}}">
<head>
<meta charset="utf-8">
<style>
  @import url('https://fonts.googleapis.com/css2?family=Cairo:wght@400;600;700&family=Inter:wght@400;600;700&display=swap');
  body {
    font-family: {{if .IsArabic}}'Cairo', Arial{{else}}'Inter', Arial{{end}}, sans-serif;
    color: #333; margin: 0; padding: 0; background: #f9f9f9;
  }
  .container { max-width: 680px; margin: 32px auto; background: #fff; border-radius: 8px;
    box-shadow: 0 1px 4px rgba(0,0,0,.08); padding: 40px; }
  .header { display: flex; justify-content: space-between; align-items: center;
    border-bottom: 2px solid #0066cc; padding-bottom: 20px; margin-bottom: 28px; }
  .logo { font-size: 22px; font-weight: 700; color: #0066cc; }
  .invoice-tag { font-size: 13px; color: #888; }
  .meta { display: grid; grid-template-columns: 1fr 1fr; gap: 6px 24px; margin-bottom: 28px; font-size: 14px; }
  .meta-label { color: #888; }
  .meta-value { font-weight: 600; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 24px; font-size: 14px; }
  th { background: #f5f7fa; padding: 10px 14px; font-weight: 600;
    text-align: {{if .IsArabic}}right{{else}}left{{end}}; border-bottom: 1px solid #e0e0e0; }
  td { padding: 10px 14px; border-bottom: 1px solid #f0f0f0;
    text-align: {{if .IsArabic}}right{{else}}left{{end}}; }
  .amount { text-align: {{if .IsArabic}}left{{else}}right{{end}} !important; }
  .row-vat td { color: #0066cc; font-weight: 600; }
  .row-total td { background: #f5f7fa; font-weight: 700; font-size: 15px; }
  .footer { margin-top: 36px; padding-top: 20px; border-top: 1px solid #eee;
    font-size: 12px; color: #999; text-align: center; }
</style>
</head>
<body>
<div class="container">

  <div class="header">
    <div class="logo">{{if .IsArabic}}مسار CRM{{else}}Masaar CRM{{end}}</div>
    <div class="invoice-tag">{{if .IsArabic}}فاتورة ضريبية{{else}}TAX INVOICE{{end}}</div>
  </div>

  <div class="meta">
    <span class="meta-label">{{if .IsArabic}}رقم الفاتورة{{else}}Invoice No.{{end}}</span>
    <span class="meta-value">{{.InvoiceNo}}</span>

    <span class="meta-label">{{if .IsArabic}}العميل{{else}}Client{{end}}</span>
    <span class="meta-value">{{.ContactName}}</span>

    <span class="meta-label">{{if .IsArabic}}تاريخ الإصدار{{else}}Date{{end}}</span>
    <span class="meta-value">{{.IssuedDate}}</span>

    <span class="meta-label">{{if .IsArabic}}تاريخ الاستحقاق{{else}}Due Date{{end}}</span>
    <span class="meta-value">{{.DueDate}}</span>
  </div>

  <table>
    <thead>
      <tr>
        <th>{{if .IsArabic}}الوصف{{else}}Description{{end}}</th>
        <th class="amount">{{if .IsArabic}}المبلغ (درهم){{else}}Amount (AED){{end}}</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td>{{.DealTitle}}</td>
        <td class="amount">{{printf "%.2f" .Amount}}</td>
      </tr>
      <tr class="row-vat">
        <td>{{if .IsArabic}}ضريبة القيمة المضافة (5٪){{else}}VAT (5%){{end}}</td>
        <td class="amount">{{printf "%.2f" .VATAmount}}</td>
      </tr>
      <tr class="row-total">
        <td>{{if .IsArabic}}الإجمالي{{else}}Total{{end}}</td>
        <td class="amount">{{printf "%.2f" .Total}}</td>
      </tr>
    </tbody>
  </table>

  <div class="footer">
    {{if .IsArabic}}
      <p>شكراً لتعاملكم معنا. يُرجى الاحتفاظ بهذه الفاتورة لسجلاتكم.</p>
      <p>هذه الفاتورة صادرة وفق أنظمة ضريبة القيمة المضافة الإماراتية.</p>
    {{else}}
      <p>Thank you for your business. Please retain this invoice for your records.</p>
      <p>This invoice is issued in compliance with UAE VAT regulations.</p>
    {{end}}
    <p>&copy; {{.IssuedDate}} Masaar CRM</p>
  </div>

</div>
</body>
</html>
`

	t, err := template.New("invoice").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ─── Magic Link Email Template ───────────────────────────────────────

type MagicLinkData struct {
	LoginURL string
	ExpiryMin int
	Lang string
}

func (d MagicLinkData) IsArabic() bool { return d.Lang == "ar" }

func (s *Service) RenderMagicLinkTemplate(data MagicLinkData) (string, error) {
	const tmpl = `
<!DOCTYPE html>
<html lang="{{if .IsArabic}}ar{{else}}en{{end}}" dir="{{if .IsArabic}}rtl{{else}}ltr{{end}}">
<head>
<meta charset="utf-8">
<style>
  @import url('https://fonts.googleapis.com/css2?family=Cairo:wght@400;600;700&family=Inter:wght@400;600;700&display=swap');
  body {
    font-family: {{if .IsArabic}}'Cairo', Arial{{else}}'Inter', Arial{{end}}, sans-serif;
    color: #333; margin: 0; padding: 0; background: #f9f9f9;
  }
  .container { max-width: 600px; margin: 32px auto; background: #fff; border-radius: 8px;
    box-shadow: 0 1px 4px rgba(0,0,0,.08); padding: 40px; }
  .header { text-align: center; border-bottom: 2px solid #0066cc; padding-bottom: 20px; margin-bottom: 28px; }
  .logo { font-size: 24px; font-weight: 700; color: #0066cc; }
  .title { font-size: 20px; font-weight: 600; margin: 24px 0 16px; }
  .button { display: inline-block; padding: 14px 32px; background: #0066cc; color: #fff;
    text-decoration: none; border-radius: 6px; font-weight: 600; margin: 20px 0; }
  .button:hover { background: #0052a3; }
  .footer { margin-top: 36px; padding-top: 20px; border-top: 1px solid #eee;
    font-size: 12px; color: #999; text-align: center; }
  .warning { background: #fff3cd; border: 1px solid #ffc107; border-radius: 6px;
    padding: 12px 16px; margin: 20px 0; font-size: 14px; color: #856404; }
  .link-fallback { word-break: break-all; font-size: 12px; color: #666; margin-top: 16px; }
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <div class="logo">{{if .IsArabic}}مسار CRM{{else}}Masaar CRM{{end}}</div>
  </div>

  <div class="title">
    {{if .IsArabic}}رابط تسجيل الدخول{{else}}Your Login Link{{end}}
  </div>

  <p>
    {{if .IsArabic}}
      انقر على الزر أدناه لتسجيل الدخول إلى حسابك. هذا الرابط صالح لمدة {{.ExpiryMin}} دقيقة.
    {{else}}
      Click the button below to log in to your account. This link is valid for {{.ExpiryMin}} minutes.
    {{end}}
  </p>

  <div style="text-align: center;">
    <a href="{{.LoginURL}}" class="button">
      {{if .IsArabic}}تسجيل الدخول{{else}}Log In{{end}}
    </a>
  </div>

  <div class="warning">
    {{if .IsArabic}}
      <strong>تنبيه:</strong> إذا لم تطلب هذا الرابط، يرجى تجاهل هذا البريد الإلكتروني.
    {{else}}
      <strong>Security Notice:</strong> If you didn't request this link, please ignore this email.
    {{end}}
  </div>

  <div class="link-fallback">
    {{if .IsArabic}}أو انسخ الرابط التالي:{{else}}Or copy this link:{{end}}<br>
    <a href="{{.LoginURL}}">{{.LoginURL}}</a>
  </div>

  <div class="footer">
    {{if .IsArabic}}
      <p>هذا بريد إلكتروني آلي، يرجى عدم الرد عليه.</p>
      <p>© {{.ExpiryMin}} مسار CRM - الإمارات العربية المتحدة</p>
    {{else}}
      <p>This is an automated email, please do not reply.</p>
      <p>© {{.ExpiryMin}} Masaar CRM - UAE</p>
    {{end}}
  </div>
</div>
</body>
</html>
`

	t, err := template.New("magic_link").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
	}
