package transport

// This file is the single home of the cloudflared pin statement (R-HR-16, design §5.4). The
// statement is report-only knowledge: it repeats what the upstream research record establishes,
// advises the reader to pin and verify, and claims no action of this run's own.
//
// Nothing here installs, verifies or pins anything, and nothing outside this file may restate the
// wording: PinNote is the one source the cloudflare-tunnel feasibility notes read, and the human
// projection (PR 17) renders those same notes, so the wording cannot drift between the two.

// PinNote returns the cloudflared pin statement every projection of this run carries.
//
// The text is the single home of the wording (R-HR-16, design §5.4): the cloudflare-tunnel
// Feasibility notes render it, and the human projection renders the same notes, so a second copy
// would be a second home rather than a projection. The statement stays inside the research record —
// the upstream issue and its state, the one version the report names, the unknown behaviour of
// later releases, and the published checksums the advice points at — and it is worded as the
// tool's own report, not as a copy of PRD.md's or README.md's prose about the upstream issue.
func PinNote() string { return pinNote }

// pinNote is the statement itself. Each clause is a fact the research record establishes, and the
// advice is addressed to the reader ("pin the client side...", "verify its published checksum
// before installing"); the final sentence disclaims any action of this run's own, because nothing
// in this slice installs, verifies or pins anything.
//
// The statement names version 2026.6.0 as the version the open report implicates and release
// 2026.5.1 as the release whose published SHA256 checksums the reader is told to verify. No range
// is established, so none is stated; no release note documents a fix, so none is claimed.
const pinNote = "The cloudflared client is reported to ignore service tokens for the access ssh and access tcp paths in version 2026.6.0: upstream issue #1673 is open and uncommented, and it names that version only, so the behaviour of later releases is unknown. Pin the client side to the last release the report does not implicate — release 2026.5.1 publishes SHA256 checksums — and verify its published checksum before installing. This report states that research record and does not install, verify or pin anything on this machine."
