package doctor

// This file is the exit-code contract of design §3.4: the constants the process
// exits with, each documented with the meaning a caller reads and the evidence
// that decides it. The documented table lives in docs/diagnosis-report.md under
// "Exit codes", and internal/doctor/doctor_test.go asserts the table's code set
// and these constants are the same set in both directions — so the process exit
// and the document cannot drift apart.
//
// The harness itself never calls os.Exit: doctor.Run returns one of these codes
// and the entrypoint (cmd/herdr-reach, PR 19) exits with it. The one property the
// codes have to keep is that a run never presents a diagnosis as completed when
// it could not establish one.

const (
	// ExitOK is the code of a completed measurement: every probe either measured
	// its declared question or was never attempted, and a negative answer — no
	// viable transport, a hub address that refused the connection — is still a
	// completed measurement. Evidence: the built payload's run.completeness is
	// report.CompletenessComplete, so the code is read from the payload rather
	// than recomputed beside it.
	ExitOK = 0

	// ExitIncomplete is the code of a run in which at least one probe was
	// attempted and resolved unresolved: the run could not establish everything
	// it attempted and does not present itself as finished. Evidence: the built
	// payload's run.completeness is report.CompletenessIncomplete. A
	// not-measured observation alone never reaches this code; it appears in
	// run.not_measured instead.
	ExitIncomplete = 1

	// ExitUsage is the code of a usage or internal error: unusable run input was
	// refused before any measurement, or the run could not present its diagnosis
	// because a projection's writer failed. No diagnosis is presented as
	// completed and the output writer carries no human text. Evidence: the flag
	// surface returned a typed UsageError, or a projection write failed.
	ExitUsage = 2
)
