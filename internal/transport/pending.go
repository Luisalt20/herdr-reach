package transport

// This file holds the three placeholder types the plan and verification members need before R2/R3
// and R5/R6 define the real shapes, plus the typed error those members fail with today.
//
// It is the single migration point when those slices land: the three types are replaced here, the
// three member signatures in contract.go change, and every adapter keeps compiling except for the
// call sites the compiler points at. The placeholders exist so the members are honest — a
// plan-producing member must exist and must fail loudly — not so a caller can receive an empty
// plan or evidence set (RG-9).

import (
	"errors"
	"fmt"
)

// PairingBundle is the placeholder for the pair's inputs: the node, the hub and the keys a pairing
// would be built from. R2 defines the real type; until then no value of this type carries anything,
// because a placeholder that carried half the real shape would be a plan in disguise.
type PairingBundle struct{}

// Step is the placeholder for one inert step of a pairing plan. R2/R3 define the real type; until
// then the only contract that matters is that a Step a caller could apply does not exist, so no
// adapter can hand one out.
type Step struct{}

// Handle is the placeholder for the verification handle a Verify call would act on. R5/R6 define
// the real type; until then no handle is accepted or returned by any code path, so verification
// cannot be half-built by accident.
type Handle struct{}

// ErrNotImplementedInThisPhase is the sentinel every not-implemented member wraps. It is exported
// so a caller can match the condition with errors.Is without knowing which member failed, which is
// what lets a future slice replace one member without changing a caller's error handling.
var ErrNotImplementedInThisPhase = errors.New("transport member not implemented in this phase")

// NotImplementedError is the typed error a plan- or verification-producing member returns in this
// slice. It names the member and the slice that owns it, so a caller reads which part of the
// product is missing and who will land it.
type NotImplementedError struct {
	// Member is the interface member that failed, e.g. "PlanHub".
	Member string
	// Owner is the slice that will implement it: R3 for the plan members, R5/R6 for verification.
	Owner string
}

// Error reports the member and the owning slice. The message deliberately does not describe a
// partial or empty result: there is no result beside this error.
func (e *NotImplementedError) Error() string {
	return fmt.Sprintf("transport.%s is not implemented in this phase: the %s slice owns it", e.Member, e.Owner)
}

// Unwrap returns the sentinel, so errors.Is matches every member's failure with one check.
func (e *NotImplementedError) Unwrap() error { return ErrNotImplementedInThisPhase }

// notImplemented is the single constructor of the typed error. Every adapter builds its failure
// through it, so the four adapters cannot spell the same condition two ways and a later slice
// changes the wording in one place.
func notImplemented(member, owner string) error {
	return &NotImplementedError{Member: member, Owner: owner}
}
