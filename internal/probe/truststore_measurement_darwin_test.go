//go:build darwin

package probe_test

// This file is the measurement issue #81 asks for. It is a measurement and
// nothing else: it changes no probe, no contract and no wording. It is gated on
// darwin because the facts it records are darwin facts, and it is deliberately
// in the external test package because it needs no access to the package's
// internals.
//
// The question behind it is one sentence in internal/probe/tls.go:
//
//	Go cannot enumerate macOS system roots, and keychain trust is only visible
//	through the platform verifier when no explicit root pool is supplied
//
// No measurement of that sentence is recorded in this repository, and issue #81
// asks for one on a real macOS runner with the build flags the release uses
// (CGO_ENABLED=0; see the release workflow's "Build the five targets" step).
// This file produces that measurement: the CI job runs it twice, once with the
// toolchain's default flags and once with CGO_ENABLED=0, and both answers land
// in the job log.
//
// Every fact is reported on its own line, prefixed so a log is greppable:
//
//   - `variant=` — which build variant this test binary was compiled in, `cgo`
//     or `nocgo`. The value comes from the cgo build tag through the two files
//     beside this one (truststore_variant_cgo_test.go and
//     truststore_variant_nocgo_test.go) and is never guessed from the
//     environment, the machine or the toolchain's defaults;
//   - `x509.SystemCertPool:` — what the standard library's own loader answers on
//     this machine: the number of certificate subjects it returned, or its error
//     verbatim;
//   - `file /etc/ssl/cert.pem:` — whether the fallback path named in issue #81
//     exists, reported as a fact and not interpreted;
//   - `root ...:` — whether a couple of well-known roots appear among the
//     subjects the pool returned. This is the fact that distinguishes a pool
//     that enumerated a trust store from one that read a bundle file; the
//     measurement reports it and never asserts it.
//
// What this file asserts is only what a measurement can be trusted to say:
// that it ran and reported every fact it declares; that the variant it reports
// is one of the two values the build tags can produce; that the loader answered
// with a pool or with an error and never with silence; and that a second call
// answered the same way. It deliberately does not assert that the limitation
// sentence quoted above is true or false: contradicting that sentence is the
// deliverable issue #81 asks for, and a test that failed on the contradiction
// would block the very change that produces the measurement. The decision that
// follows the numbers belongs to the maintainer, not to this file.

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	// trustStoreVariantCgo and trustStoreVariantNoCgo are the two values the
	// variant report is allowed to take. They are declared here, beside the
	// assertion that uses them, so the build-tagged files that set the value and
	// the measurement that checks it are compared against one declaration.
	trustStoreVariantCgo   = "cgo"
	trustStoreVariantNoCgo = "nocgo"

	// trustStoreBundlePath is the fallback path issue #81 names: the file a
	// non-cgo root loader was claimed to read. This file records whether it
	// exists; it makes no claim about whether anything reads it.
	trustStoreBundlePath = "/etc/ssl/cert.pem"
)

// trustStoreKnownRoots are the well-known roots the measurement looks for among
// the subjects the pool returned, so "the pool enumerated a trust store" can be
// distinguished from "the pool read a bundle file, or nothing at all". They are
// roots a complete macOS trust store or CA bundle carries, and one of them
// (ISRG Root X1) is the anchor family behind the declared expected issuer of
// the probe this measurement exists for.
var trustStoreKnownRoots = []string{
	"ISRG Root X1",
	"DigiCert Global Root G2",
}

// TestDarwinTrustStoreMeasurement reports the facts issue #81 asks for and
// asserts only that the measurement itself is well formed. See the file comment
// for the contract.
func TestDarwinTrustStoreMeasurement(t *testing.T) {
	var (
		reportedVariant   bool
		reportedPool      bool
		reportedBundle    bool
		rootFactsReported int
	)
	report := func(format string, args ...any) {
		t.Log("truststore measurement: " + fmt.Sprintf(format, args...))
	}

	// The variant is the build tag's answer, read from the constant the variant
	// file defined; nothing here inspects the environment.
	variant := trustStoreBuildVariant
	report("variant=%s", variant)
	reportedVariant = true

	// The loader is asked twice, so "the same answer" is a measurement rather
	// than an assumption.
	first, firstErr := x509.SystemCertPool()
	second, secondErr := x509.SystemCertPool()

	var subjects [][]byte
	poolAnswered := firstErr == nil && first != nil
	if poolAnswered {
		subjects = first.Subjects()
	}
	switch {
	case firstErr != nil:
		report("x509.SystemCertPool: error=%q", firstErr.Error())
	case first == nil:
		report("x509.SystemCertPool: neither a pool nor an error was returned")
	default:
		report("x509.SystemCertPool: subjects=%d", len(subjects))
	}
	reportedPool = true

	// The fallback file, as a fact: whether it exists, and the verbatim reason
	// when it does not, so "absent" and "unreadable" stay distinguishable.
	if info, err := os.Stat(trustStoreBundlePath); err != nil {
		report("file %s: exists=false stat_error=%q", trustStoreBundlePath, err.Error())
	} else {
		report("file %s: exists=true size=%d", trustStoreBundlePath, info.Size())
	}
	reportedBundle = true

	// The roots are looked for only when the pool answered, because a pool that
	// answered with an error enumerated nothing to search.
	if poolAnswered {
		names, unparsed := trustStoreSubjectNames(subjects)
		for _, root := range trustStoreKnownRoots {
			report("root %q: observed_in_pool_subjects=%t", root, trustStoreSubjectObserved(names, root))
			rootFactsReported++
		}
		if unparsed > 0 {
			report("subject_names_unparsed=%d", unparsed)
		}
	}

	// The measurement ran and reported every fact it declares. This is the one
	// self-check that keeps a later edit from silently shrinking the log: a
	// measurement that produced silence cannot be read.
	if !reportedVariant || !reportedPool || !reportedBundle {
		t.Fatalf("the measurement did not report every fact it declares (variant=%t, pool=%t, bundle=%t)", reportedVariant, reportedPool, reportedBundle)
	}
	if poolAnswered && rootFactsReported != len(trustStoreKnownRoots) {
		t.Errorf("the measurement reported %d root facts, want %d: every declared well-known root must be reported", rootFactsReported, len(trustStoreKnownRoots))
	}

	// The variant is one of the two values the build tags can select.
	if variant != trustStoreVariantCgo && variant != trustStoreVariantNoCgo {
		t.Errorf("the variant report %q is neither %q nor %q; the build tags decide the variant and the measurement must name a known one", variant, trustStoreVariantCgo, trustStoreVariantNoCgo)
	}

	// The loader answered with a pool or with an error, never with silence.
	firstAnsweredWithPool := first != nil && firstErr == nil
	firstAnsweredWithError := first == nil && firstErr != nil
	if !firstAnsweredWithPool && !firstAnsweredWithError {
		t.Errorf("x509.SystemCertPool() answered neither a pool (%t) nor an error (%v); a measurement cannot be read from silence", first != nil, firstErr)
	}

	// The second call answered the same way.
	if (first == nil) != (second == nil) {
		t.Errorf("the two calls disagree on whether a pool was returned: first nil=%t, second nil=%t", first == nil, second == nil)
	}
	if (firstErr == nil) != (secondErr == nil) {
		t.Errorf("the two calls disagree on whether an error was returned: first %v, second %v", firstErr, secondErr)
	} else if firstErr != nil && firstErr.Error() != secondErr.Error() {
		t.Errorf("the two calls returned different errors: first %q, second %q", firstErr.Error(), secondErr.Error())
	}
	if first != nil && second != nil && !first.Equal(second) {
		t.Errorf("the two calls returned pools with different contents")
	}
}

// trustStoreSubjectNames renders each DER-encoded subject the pool returned as
// the RFC 2253-style string pkix.RDNSequence.String() produces, so a well-known
// root can be recognised by name.
//
// The parse is the ASN.1 shape a distinguished name is declared as in
// crypto/x509/pkix (the element type's name ends in SET, which is what makes
// encoding/asn1 read the SET tag), and a subject that cannot be parsed is
// counted rather than dropped in silence: "not looked for" must not read as
// "not present".
func trustStoreSubjectNames(subjects [][]byte) (names []string, unparsed int) {
	for _, raw := range subjects {
		var rdn pkix.RDNSequence
		if _, err := asn1.Unmarshal(raw, &rdn); err != nil {
			unparsed++
			continue
		}
		names = append(names, rdn.String())
	}
	return names, unparsed
}

// trustStoreSubjectObserved reports whether any rendered subject names the root.
// It is a fact about the enumeration, not a claim about the machine: a root
// absent from the returned subjects may still be trusted by the platform
// verifier the pool defers to.
func trustStoreSubjectObserved(names []string, root string) bool {
	for _, name := range names {
		if strings.Contains(name, root) {
			return true
		}
	}
	return false
}
