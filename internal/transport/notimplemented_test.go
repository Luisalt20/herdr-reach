package transport_test

// This file is the registry-wide not-implemented proof of R-HR-06 and RG-9 (design D7). For every
// registered adapter and every plan- or verification-producing member, the call fails with the
// typed NotImplementedError, errors.Is matches ErrNotImplementedInThisPhase, the error names the
// member and the owning slice, and no value is returned beside it — never an empty plan or an
// empty evidence set a caller could mistake for success.
//
// The proof runs over transport.Registry() rather than over a list of adapter names, so an adapter
// that joins the registry is covered here without a second list to keep in sync. It asserts that
// the number of adapters proofed equals the registry length, so a loop that stops early fails
// instead of passing silently; the registry's own closure case asserts the exactly-four set.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/transport"
)

// TestNotImplementedMembersFailLoudlyWithoutAValue is RG-9's case for every registered adapter:
// each plan- and verification-producing member fails with the typed not-implemented error, the
// error names both the member and the owning slice, `errors.Is` matches the sentinel, and no value
// is returned beside it — the `[]Step` and the `[]probe.Result` are nil.
func TestNotImplementedMembersFailLoudlyWithoutAValue(t *testing.T) {
	registry := transport.Registry()
	if len(registry) == 0 {
		t.Fatal("the registry is empty, so the case proves nothing")
	}
	if len(registry) != len(v1TransportNames) {
		t.Fatalf("the registry holds %d adapters, want the %d V1 transports: the not-implemented proof covers the closed set", len(registry), len(v1TransportNames))
	}
	proofed := 0
	for _, candidate := range registry {
		// Candidate is the registration contract, and every R1a entry is a full Transport: the
		// type assertion below is the honesty property this loop carries, so a future detect-only
		// entry cannot silently join the shipped set without a deliberate decision. The registry's
		// closure case asserts the exactly-four set; this loop proves the members of every entry it
		// holds.
		tr, ok := candidate.(transport.Transport)
		if !ok {
			t.Fatalf("the registered candidate %q does not implement Transport: the registration contract is Candidate and every R1a entry is a full Transport, so a detect-only entry joining the shipped set must be a deliberate decision rather than a silent registry append", candidate.Name())
		}
		proofed++
		t.Run(tr.Name()+"/PlanHub", func(t *testing.T) {
			steps, err := tr.PlanHub(transport.PairingBundle{})
			assertNotImplemented(t, err, "PlanHub", "R3")
			if steps != nil {
				t.Errorf("PlanHub returned %v beside the error; no plan may be returned alongside the not-implemented error", steps)
			}
		})
		t.Run(tr.Name()+"/PlanNode", func(t *testing.T) {
			steps, err := tr.PlanNode(transport.PairingBundle{})
			assertNotImplemented(t, err, "PlanNode", "R3")
			if steps != nil {
				t.Errorf("PlanNode returned %v beside the error; no plan may be returned alongside the not-implemented error", steps)
			}
		})
		t.Run(tr.Name()+"/Verify", func(t *testing.T) {
			results, err := tr.Verify(context.Background(), transport.Handle{})
			assertNotImplemented(t, err, "Verify", "R5/R6")
			if results != nil {
				t.Errorf("Verify returned %v beside the error; no evidence may be returned alongside the not-implemented error", results)
			}
		})
	}
	if proofed != len(registry) {
		t.Errorf("proofed %d adapters for %d registered: every registered adapter must be proofed, so a loop that stops early cannot pass", proofed, len(registry))
	}
}

// assertNotImplemented asserts the loud-failure contract of one member: a non-nil typed error that
// names the member and its owner slice, unwraps to the sentinel, and matches it under errors.Is.
func assertNotImplemented(t *testing.T, err error, member, owner string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s returned a nil error: a caller could mistake that for a successful result", member)
	}
	if !errors.Is(err, transport.ErrNotImplementedInThisPhase) {
		t.Errorf("errors.Is(err, ErrNotImplementedInThisPhase) = false for %v", err)
	}
	if got := errors.Unwrap(err); got != transport.ErrNotImplementedInThisPhase {
		t.Errorf("errors.Unwrap(%v) = %v, want the sentinel", err, got)
	}
	var typed *transport.NotImplementedError
	if !errors.As(err, &typed) {
		t.Fatalf("the error %v is not the typed NotImplementedError", err)
	}
	if typed.Member != member {
		t.Errorf("the error names member %q, want %q", typed.Member, member)
	}
	if typed.Owner != owner {
		t.Errorf("the error names owner %q, want %q", typed.Owner, owner)
	}
	for _, want := range []string{member, owner} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error text %q does not name %q", err.Error(), want)
		}
	}
}
