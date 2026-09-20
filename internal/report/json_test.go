package report_test

// This file is the machine-readable projection's suite (design §4): the document
// WriteJSON produces parses back to the payload it came from, the zero payload
// still produces a valid document, and a writer that refuses its write is
// reported to the caller rather than swallowed.
//
// It is the sibling of human_test.go, and deliberately so: the report package
// owns both projections of one payload, so each projection is driven through its
// shipped function into an injected writer. The bytes are pinned against
// encoding/json's own compact output plus the one terminator, not against a
// golden string that could drift from the payload it serialises.

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/report"
)

// TestJSONDocumentRoundTripsThePayload asserts the document is the payload's own
// compact serialisation plus one newline — the exact bytes the harness wrote
// before the projection moved here — and that it parses back to the payload it
// came from.
func TestJSONDocumentRoundTripsThePayload(t *testing.T) {
	payload := report.Build(fullInput())

	var buffer bytes.Buffer
	if err := report.WriteJSON(&buffer, payload); err != nil {
		t.Fatalf("WriteJSON returned an error: %v", err)
	}
	document := buffer.Bytes()

	want, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshalling the payload for comparison failed: %v", err)
	}
	want = append(want, '\n')
	if !bytes.Equal(document, want) {
		t.Errorf("WriteJSON wrote:\n%s\nwant the payload's compact document plus one newline:\n%s", document, want)
	}
	if count := bytes.Count(document, []byte("\n")); count != 1 {
		t.Errorf("the document carries %d newlines, want exactly the one terminator: %q", count, document)
	}

	var decoded report.Payload
	if err := json.Unmarshal(document, &decoded); err != nil {
		t.Fatalf("the document does not parse back: %v", err)
	}
	if !reflect.DeepEqual(decoded, payload) {
		t.Errorf("the document parsed back to %+v, want the payload it came from %+v", decoded, payload)
	}
}

// TestJSONEmptyPayloadIsAValidDocument asserts the zero payload still produces a
// valid, parseable document: the projection never omits the shape it promises,
// whatever the run carried.
func TestJSONEmptyPayloadIsAValidDocument(t *testing.T) {
	var buffer bytes.Buffer
	if err := report.WriteJSON(&buffer, report.Payload{}); err != nil {
		t.Fatalf("WriteJSON returned an error for the zero payload: %v", err)
	}
	document := buffer.Bytes()
	if !json.Valid(document) {
		t.Fatalf("the zero payload produced an invalid document: %q", document)
	}
	if !bytes.HasSuffix(document, []byte("\n")) {
		t.Errorf("the document does not end in the single newline terminator: %q", document)
	}
	var decoded report.Payload
	if err := json.Unmarshal(document, &decoded); err != nil {
		t.Errorf("the zero payload's document does not parse back: %v", err)
	}
}

// TestJSONWriterErrorsArePropagated asserts a refusing writer is reported to the
// caller rather than swallowed, exactly as the human projection's writer is: the
// harness maps the error to its usage exit, so a silent failure would present an
// unwritten document as a written one.
func TestJSONWriterErrorsArePropagated(t *testing.T) {
	wantErr := errors.New("the writer refused the write")
	err := report.WriteJSON(failingWriter{err: wantErr}, report.Build(fullInput()))
	if !errors.Is(err, wantErr) {
		t.Errorf("WriteJSON error = %v, want it to carry %v: a writer failure must not be reported as success", err, wantErr)
	}
}
