package report

// This file is the machine-readable projection of design §4: it serialises the
// payload Build produced to one injected writer, so the report package owns both
// projections of one run and the harness that wires them writes nothing itself.
//
// It is a projection and nothing else. WriteJSON measures nothing, reads no
// file, consults no clock and writes to nothing but the writer it is handed: it
// copies the payload's own fields into their compact JSON form, and the only
// errors it can produce are the serialisation's — practically impossible for the
// payload's closed field set — and the writer's, which it returns so the caller
// can exit 2 instead of pretending the document was written.
//
// The bytes are part of the contract. The serialisation is `encoding/json`'s
// compact output of the Payload's own fields plus exactly one newline, which is
// what doctor.Run wrote before the projection moved here: moving it changed no
// whitespace, no field order and no terminator.

import (
	"encoding/json"
	"fmt"
	"io"
)

// WriteJSON renders one payload as the machine-readable document of design §3.3
// and writes it to w. It is WriteHuman's sibling — writer first, payload second,
// the writer's error returned rather than swallowed — and it is the only place
// the document is serialised: the harness calls it for the machine-readable half
// exactly as it calls WriteHuman for the human half.
//
// The document is built completely before the single write, so a failed write
// leaves the caller with the error and nothing else: the writer sees one
// document plus its newline terminator, never a partial value beside human text.
func WriteJSON(w io.Writer, payload Payload) error {
	document, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("serialising the machine-readable document: %w", err)
	}
	// One write of one document plus its terminator: a writer failure cannot
	// leave a second value or human text beside it, and the newline keeps the
	// document line-oriented without becoming a second write of its own.
	document = append(document, '\n')
	if _, err := w.Write(document); err != nil {
		return fmt.Errorf("writing the machine-readable document: %w", err)
	}
	return nil
}
