package mediastore

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "media")
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, dir
}

func TestSaveAndOpen(t *testing.T) {
	s, _ := newStore(t)
	name, size, err := s.Save(strings.NewReader("hello media"), 1024)
	if err != nil || size != 11 {
		t.Fatalf("Save = %q, %d, %v", name, size, err)
	}
	f, err := s.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if b, _ := io.ReadAll(f); string(b) != "hello media" {
		t.Errorf("read back %q", b)
	}
}

func TestSaveEnforcesLimitAndLeavesNothingBehind(t *testing.T) {
	s, dir := newStore(t)
	if _, _, err := s.Save(bytes.NewReader(make([]byte, 101)), 100); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
	if _, _, err := s.Save(bytes.NewReader(make([]byte, 100)), 100); err != nil {
		t.Errorf("exactly-at-limit file rejected: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".part") {
			t.Errorf("temporary file left behind: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("%d files stored, want only the valid one", len(entries))
	}
}

// Names come from the database; even a tampered value must not reach outside the directory.
func TestOpenRejectsTraversalAndForeignNames(t *testing.T) {
	s, dir := newStore(t)
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		"../secret.txt", "../../etc/passwd", "/etc/passwd", secret, "", ".", "..",
		strings.Repeat("a", 31), strings.Repeat("g", 32), strings.Repeat("A", 32),
		strings.Repeat("a", 31) + "/", "sub/" + strings.Repeat("a", 27),
	} {
		if f, err := s.Open(bad); err == nil {
			f.Close()
			t.Errorf("Open(%q) succeeded", bad)
		}
	}
}

func TestRemove(t *testing.T) {
	s, _ := newStore(t)
	name, _, _ := s.Save(strings.NewReader("x"), 10)
	if err := s.Remove(name); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(name); err == nil {
		t.Error("file still openable after Remove")
	}
	if err := s.Remove(name); err != nil {
		t.Errorf("removing a missing file = %v, want nil", err)
	}
	if err := s.Remove("../x"); err != nil {
		t.Errorf("removing an invalid name = %v, want nil (ignored)", err)
	}
}
