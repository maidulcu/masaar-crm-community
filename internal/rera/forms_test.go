package rera

import "testing"

func TestFormsAreWellFormed(t *testing.T) {
	for typ, f := range Forms {
		if f.Type != typ || f.TitleEN == "" || f.TitleAR == "" || len(f.Signers) != 2 {
			t.Errorf("form %s header incomplete: %+v", typ, f)
		}
		seen := map[string]bool{}
		for _, s := range f.Sections {
			for _, fl := range s.Fields {
				if fl.EN == "" || fl.AR == "" {
					t.Errorf("form %s field %s lacks a label", typ, fl.Key)
				}
				if seen[fl.Key] {
					t.Errorf("form %s repeats field %s", typ, fl.Key)
				}
				seen[fl.Key] = true
			}
		}
	}
}

func TestMerge(t *testing.T) {
	a, _ := Lookup("A")
	got, err := a.Merge(map[string]string{"owner_name": "Ahmed", "buyer_name": "ignored", "asking_price": ""},
		map[string]string{"commission_percent": "2"})
	if err != nil {
		t.Fatal(err)
	}
	if got["owner_name"] != "Ahmed" || got["commission_percent"] != "2" || got["buyer_name"] != "" || len(got) != 2 {
		t.Fatalf("merge = %v", got)
	}
	if _, err := a.Merge(nil, map[string]string{"nope": "x"}); err == nil {
		t.Fatal("unknown override accepted")
	}
	if _, err := Lookup("Z"); err == nil {
		t.Fatal("unknown form resolved")
	}
}
