package partner

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
)

func hashOf(s string) string { sum := sha256.Sum256([]byte(s)); return hex.EncodeToString(sum[:]) }

func TestParseAndAuthenticate(t *testing.T) {
	cs, err := Parse(" portal:" + hashOf("s3cret") + ", other:" + hashOf("x") + ":partner:login ")
	if err != nil {
		t.Fatal(err)
	}
	c, ok := cs.Authenticate("portal", "s3cret")
	if !ok || !reflect.DeepEqual(c.Scopes, []string{"partner:login", "partner:provision"}) {
		t.Fatalf("portal: %v %v", c, ok)
	}
	if _, ok := cs.Authenticate("portal", "wrong"); ok {
		t.Error("wrong secret accepted")
	}
	if _, ok := cs.Authenticate("nobody", "s3cret"); ok {
		t.Error("unknown client accepted")
	}
	if o := cs["other"]; !reflect.DeepEqual(o.Scopes, []string{"partner:login"}) {
		t.Errorf("other scopes = %v", o.Scopes)
	}
	if empty, err := Parse(""); err != nil || len(empty) != 0 {
		t.Errorf("empty config: %v %v", empty, err)
	}
	for _, bad := range []string{"portal", "portal:abc", "Bad Id:" + hashOf("x"), "p:" + hashOf("x") + ":admin", "p:" + hashOf("a") + ",p:" + hashOf("b")} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}
}

func TestGrant(t *testing.T) {
	c := Client{ID: "p", Scopes: []string{"partner:login", "partner:provision"}}
	if got, ok := c.Grant(""); !ok || len(got) != 2 {
		t.Errorf("default grant = %v %v", got, ok)
	}
	if got, ok := c.Grant("partner:provision partner:provision"); !ok || !reflect.DeepEqual(got, []string{"partner:provision"}) {
		t.Errorf("grant = %v %v", got, ok)
	}
	limited := Client{ID: "q", Scopes: []string{"partner:login"}}
	if _, ok := limited.Grant("partner:provision"); ok {
		t.Error("granted a scope the client does not hold")
	}
}
