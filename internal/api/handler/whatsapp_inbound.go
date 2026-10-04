package handler

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Types mirroring the parts of Meta's WhatsApp Cloud API webhook payload the CRM reads.

type waMedia struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	Caption  string `json:"caption"`
	Filename string `json:"filename"`
	Voice    bool   `json:"voice"`
}

type waMessage struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      struct {
		Body string `json:"body"`
	} `json:"text"`
	Image    waMedia `json:"image"`
	Video    waMedia `json:"video"`
	Audio    waMedia `json:"audio"`
	Sticker  waMedia `json:"sticker"`
	Document waMedia `json:"document"`
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Name      string  `json:"name"`
		Address   string  `json:"address"`
	} `json:"location"`
	Interactive struct {
		Type        string `json:"type"`
		ButtonReply struct {
			Title string `json:"title"`
		} `json:"button_reply"`
		ListReply struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"list_reply"`
	} `json:"interactive"`
	Button struct {
		Text string `json:"text"`
	} `json:"button"`
	Contacts []struct {
		Name struct {
			FormattedName string `json:"formatted_name"`
		} `json:"name"`
	} `json:"contacts"`
}

type waStatus struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	RecipientID string `json:"recipient_id"`
	Errors      []struct {
		Code      int    `json:"code"`
		Title     string `json:"title"`
		Message   string `json:"message"`
		ErrorData struct {
			Details string `json:"details"`
		} `json:"error_data"`
	} `json:"errors"`
}

type waChangeValue struct {
	MessagingProduct string `json:"messaging_product"`
	Metadata         struct {
		PhoneNumberID string `json:"phone_number_id"`
	} `json:"metadata"`
	Contacts []struct {
		Profile struct {
			Name string `json:"name"`
		} `json:"profile"`
		WAID string `json:"wa_id"`
	} `json:"contacts"`
	Messages []waMessage `json:"messages"`
	Statuses []waStatus  `json:"statuses"`
}

type waPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string        `json:"field"`
			Value waChangeValue `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// inboundContent is what the CRM stores for one inbound message.
type inboundContent struct {
	Body      string
	MediaID   string // Meta's id for the attached file (Meta sends an id, never a URL)
	MediaMime string
	Skip      bool // nothing worth storing (e.g. a reaction)
}

const maxInboundBody = 8192

// parseInbound turns a WhatsApp message into the text shown in the inbox. Every message type
// yields something readable; only content with no conversational meaning (reactions) is skipped.
func parseInbound(msg waMessage) inboundContent {
	withCaption := func(label, caption string) string {
		if caption = strings.TrimSpace(caption); caption != "" {
			return label + " " + caption
		}
		return label
	}

	var c inboundContent
	switch msg.Type {
	case "text":
		c.Body = msg.Text.Body
	case "image":
		c = inboundContent{Body: withCaption("[Image]", msg.Image.Caption), MediaID: msg.Image.ID, MediaMime: msg.Image.MimeType}
	case "video":
		c = inboundContent{Body: withCaption("[Video]", msg.Video.Caption), MediaID: msg.Video.ID, MediaMime: msg.Video.MimeType}
	case "audio":
		label := "[Audio]"
		if msg.Audio.Voice {
			label = "[Voice message]"
		}
		c = inboundContent{Body: label, MediaID: msg.Audio.ID, MediaMime: msg.Audio.MimeType}
	case "sticker":
		c = inboundContent{Body: "[Sticker]", MediaID: msg.Sticker.ID, MediaMime: msg.Sticker.MimeType}
	case "document":
		label := "[Document]"
		if name := strings.TrimSpace(msg.Document.Filename); name != "" {
			label = "[Document: " + name + "]"
		}
		c = inboundContent{Body: withCaption(label, msg.Document.Caption), MediaID: msg.Document.ID, MediaMime: msg.Document.MimeType}
	case "location":
		l := msg.Location
		where := strings.TrimSpace(strings.Join(nonEmpty(l.Name, l.Address), ", "))
		coords := fmt.Sprintf("%.6f,%.6f", l.Latitude, l.Longitude)
		if where != "" {
			c.Body = fmt.Sprintf("[Location] %s (%s) https://maps.google.com/?q=%s", where, coords, coords)
		} else {
			c.Body = fmt.Sprintf("[Location] %s https://maps.google.com/?q=%s", coords, coords)
		}
	case "interactive":
		switch msg.Interactive.Type {
		case "button_reply":
			c.Body = msg.Interactive.ButtonReply.Title
		case "list_reply":
			c.Body = strings.Join(nonEmpty(msg.Interactive.ListReply.Title, msg.Interactive.ListReply.Description), " — ")
		}
		if strings.TrimSpace(c.Body) == "" {
			c.Body = "[Interactive reply]"
		}
	case "button": // reply to a template's quick-reply button
		c.Body = msg.Button.Text
		if strings.TrimSpace(c.Body) == "" {
			c.Body = "[Button reply]"
		}
	case "contacts":
		var names []string
		for _, ct := range msg.Contacts {
			names = append(names, ct.Name.FormattedName)
		}
		c.Body = withCaption("[Contact card]", strings.Join(nonEmpty(names...), ", "))
	case "reaction":
		// A reaction annotates an earlier message; it is not a message of its own.
		return inboundContent{Skip: true}
	case "unsupported", "system", "unknown", "":
		c.Body = "[Unsupported message]"
	default:
		c.Body = "[" + msg.Type + "]"
	}

	c.Body = cleanBody(c.Body)
	if c.Body == "" {
		c.Body = "[" + msg.Type + "]"
	}
	return c
}

func nonEmpty(parts ...string) []string {
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// cleanBody makes text safe to store: PostgreSQL rejects NUL bytes and invalid UTF-8 (a single
// such message would otherwise fail on every redelivery), and an absurdly long body is cut.
func cleanBody(s string) string {
	s = strings.ToValidUTF8(s, "")
	s = strings.ReplaceAll(s, "\x00", "")
	if utf8.RuneCountInString(s) > maxInboundBody {
		s = string([]rune(s)[:maxInboundBody])
	}
	return strings.TrimSpace(s)
}

// statusFromMeta maps Meta's receipt names onto outbound statuses; ok=false for receipts we do
// not track (e.g. "deleted").
func statusFromMeta(s string) (status string, ok bool) {
	switch s {
	case "sent", "delivered", "read", "failed":
		return s, true
	}
	return "", false
}

// describeStatusErrors renders the error list of a failed receipt as "131047: Re-engagement
// message — details" for display next to the message.
func describeStatusErrors(st waStatus) string {
	var parts []string
	for _, e := range st.Errors {
		text := strings.Join(nonEmpty(e.Title, e.Message, e.ErrorData.Details), " — ")
		parts = append(parts, fmt.Sprintf("%d: %s", e.Code, text))
	}
	if len(parts) == 0 {
		return "delivery failed"
	}
	return cleanBody(strings.Join(parts, "; "))
}
