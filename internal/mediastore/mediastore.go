// Package mediastore keeps files received from or sent to customers (WhatsApp images,
// documents, voice notes) on local disk.
//
// Files are written under server-generated random names, never a name supplied by a customer,
// and every access goes through os.Root, which refuses paths that escape the storage directory
// (".." components, absolute paths, symlinks pointing outside).
package mediastore

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// ErrTooLarge is returned by Save when the data exceeds the size limit.
var ErrTooLarge = errors.New("media file too large")

type Store struct {
	root *os.Root
}

// New opens (creating it if necessary) the storage directory.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create media dir: %w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open media dir: %w", err)
	}
	return &Store{root: root}, nil
}

// Close releases the directory handle.
func (s *Store) Close() error { return s.root.Close() }

// Save stores r and returns the opaque name to pass to Open, plus the size in bytes. It reads at
// most maxBytes; a longer stream fails with ErrTooLarge and leaves nothing behind.
func (s *Store) Save(r io.Reader, maxBytes int64) (name string, size int64, err error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", 0, err
	}
	name = hex.EncodeToString(id[:])
	tmp := name + ".part"

	f, err := s.root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", 0, err
	}
	// Read one byte past the limit so an over-long stream is detected, not silently truncated.
	n, copyErr := io.Copy(f, io.LimitReader(r, maxBytes+1))
	closeErr := f.Close()
	switch {
	case copyErr != nil:
		err = copyErr
	case closeErr != nil:
		err = closeErr
	case n > maxBytes:
		err = ErrTooLarge
	}
	if err != nil {
		_ = s.root.Remove(tmp)
		return "", 0, err
	}
	if err := s.root.Rename(tmp, name); err != nil {
		_ = s.root.Remove(tmp)
		return "", 0, err
	}
	return name, n, nil
}

// Open returns the stored file. A name that is not a plain file name inside the directory is
// rejected, even if it came from the database.
func (s *Store) Open(name string) (*os.File, error) {
	if !validName(name) {
		return nil, os.ErrNotExist
	}
	return s.root.Open(name)
}

// Remove deletes a stored file; a missing file is not an error.
func (s *Store) Remove(name string) error {
	if !validName(name) {
		return nil
	}
	if err := s.root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// validName accepts only the names Save produces (32 hex characters).
func validName(name string) bool {
	if len(name) != 32 {
		return false
	}
	for _, c := range name {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
