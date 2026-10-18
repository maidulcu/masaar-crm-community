package einvoice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
)

// Statuses of a submitted e-invoice.
const (
	StatusSubmitted = "submitted"
	StatusAccepted  = "accepted"
	StatusRejected  = "rejected"
	StatusError     = "error" // could not be submitted at all
)

// Submission is what is handed to a service provider.
type Submission struct {
	Kind   string // invoice | credit_note
	Number string
	XML    []byte
}

// Result is a provider's answer about a submitted document.
type Result struct {
	Status string // submitted (still processing) | accepted | rejected
	Reason string // why it was rejected
}

// Connector submits e-invoices to an accredited service provider (ASP) and reports their
// status. Masaar bundles no real provider: an integration implements this interface and
// registers itself with Register; companies choose it in their settings.
type Connector interface {
	Name() string
	Submit(ctx context.Context, s Submission) (ref string, err error)
	Status(ctx context.Context, ref string) (Result, error)
}

var (
	mu         sync.RWMutex
	connectors = map[string]Connector{"mock": Mock{}}
)

// Register makes a connector available under its name.
func Register(c Connector) {
	mu.Lock()
	defer mu.Unlock()
	connectors[c.Name()] = c
}

// Lookup returns the connector with that name.
func Lookup(name string) (Connector, bool) {
	mu.RLock()
	defer mu.RUnlock()
	c, ok := connectors[name]
	return c, ok
}

// Providers lists the registered connector names.
func Providers() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(connectors))
	for n := range connectors {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// ErrUnknownProvider is returned for a provider name with no registered connector.
var ErrUnknownProvider = errors.New("unknown e-invoicing provider")

// RejectMarker in an invoice's notes makes the mock provider reject it, for testing the
// rejection path.
const RejectMarker = "#mock-reject"

// Mock is a stand-in provider for trying the workflow and for tests. It accepts every
// document except those whose notes contain RejectMarker. It keeps no state: the outcome
// is encoded in the reference.
type Mock struct{}

func (Mock) Name() string { return "mock" }

func (Mock) Submit(_ context.Context, s Submission) (string, error) {
	sum := sha256.Sum256(s.XML)
	outcome := "ok"
	if bytes.Contains(s.XML, []byte(RejectMarker)) {
		outcome = "reject"
	}
	return "mock-" + outcome + "-" + hex.EncodeToString(sum[:8]), nil
}

func (Mock) Status(_ context.Context, ref string) (Result, error) {
	switch {
	case strings.HasPrefix(ref, "mock-ok-"):
		return Result{Status: StatusAccepted}, nil
	case strings.HasPrefix(ref, "mock-reject-"):
		return Result{Status: StatusRejected, Reason: "Rejected by the mock provider (test marker in the notes)"}, nil
	}
	return Result{}, errors.New("unknown mock reference")
}
