package whatsapp

import "testing"

func TestOptKeyword(t *testing.T) {
	for body, want := range map[string][2]bool{
		"STOP":                   {true, false},
		" stop. ":                {true, false},
		"Unsubscribe!":           {true, false},
		"إيقاف":                  {true, false},
		"ايقاف":                  {true, false},
		"إلغاء الاشتراك":         {true, false},
		"START":                  {false, true},
		"اشتراك":                 {false, true},
		"please stop calling me": {false, false},
		"stop by the office?":    {false, false},
		"":                       {false, false},
		"Is the flat available?": {false, false},
	} {
		out, in := OptKeyword(body)
		if out != want[0] || in != want[1] {
			t.Errorf("%q: out=%v in=%v, want %v", body, out, in, want)
		}
	}
}
