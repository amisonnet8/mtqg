package journal

import (
	"bytes"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"unicode/utf8"
)

// The operations of the format (docs/reference/schema.md). A new operation is a
// change of the format, so the set is closed.
const (
	OpCreate = "create"
	OpStatus = "status"
	OpEdit   = "edit"
	OpDelete = "delete"
)

// The kinds of author (docs/reference/schema.md).
const (
	AuthorHuman = "human"
	AuthorAI    = "ai"
)

// Event is one line of journal.jsonl. The journal layer does not know what an
// event means; it only reads and writes the format.
//
// The fields are declared in the key order of the format, which is the order
// they are written in: id, op, type, re, from, status, word, text, at, v, ts,
// author, tty. Fields without a value are left out, except v, which is written
// even when it is 0.
type Event struct {
	ID     string `json:"id"`
	Op     string `json:"op"`
	Type   string `json:"type,omitempty"`
	Re     string `json:"re,omitempty"`
	From   string `json:"from,omitempty"`
	Status string `json:"status,omitempty"`
	Word   string `json:"word,omitempty"`
	Text   string `json:"text,omitempty"`
	At     *At    `json:"at,omitempty"`
	V      int    `json:"v"`
	TS     string `json:"ts"`
	Author Author `json:"author"`
	TTY    string `json:"tty,omitempty"`
}

// Author is who is responsible for the content of an event.
type Author struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// At records where in the project a record was written about. It is a fact at
// writing time and is never updated.
type At struct {
	Path string `json:"path,omitempty"`
	Line int    `json:"line,omitempty"`
	Head string `json:"head,omitempty"`
}

// This file is the only place that touches JSON, so that the written bytes are
// defined in one spot and pinned by the golden tests.
var (
	// The escaping is spelled out instead of relying on the defaults: the format
	// escapes only what the JSON grammar requires (no <, >, & or U+2028, U+2029
	// escapes), so that tools writing by the rules produce identical lines.
	marshalOptions = jsonv2.JoinOptions(
		jsontext.EscapeForHTML(false),
		jsontext.EscapeForJS(false),
	)

	// A key that appears twice in one line is valid JSON; the last value wins.
	unmarshalOptions = jsonv2.JoinOptions(jsontext.AllowDuplicateNames(true))
)

// Why a line could not be read as an event. The caller turns these into
// warnings; the messages are for tests only.
var (
	errInvalidJSON  = errors.New("not a valid JSON object")
	errInvalidUTF8  = errors.New("not valid UTF-8")
	errMissingField = errors.New("no id or no op")
)

// encodeLine returns ev as one line of the format, ending with LF. It refuses
// text that is not valid UTF-8 instead of altering it.
func encodeLine(ev Event) ([]byte, error) {
	if err := ev.checkUTF8(); err != nil {
		return nil, err
	}
	b, err := jsonv2.Marshal(ev, marshalOptions)
	if err != nil {
		return nil, fmt.Errorf("journal: encode event: %w", err)
	}
	return append(b, '\n'), nil
}

// parseEvent reads one line, without its line ending, as an event. Any valid
// JSON object is accepted, whatever its key order, spacing or escaping. Fields
// it does not know are ignored.
func parseEvent(raw []byte) (Event, error) {
	if !utf8.Valid(raw) {
		return Event{}, errInvalidUTF8
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return Event{}, errInvalidJSON
	}
	var ev Event
	if err := jsonv2.Unmarshal(trimmed, &ev, unmarshalOptions); err != nil {
		return Event{}, fmt.Errorf("%w: %w", errInvalidJSON, err)
	}
	if ev.ID == "" || ev.Op == "" {
		return Event{}, errMissingField
	}
	return ev, nil
}

// checkUTF8 rejects an event with text that is not valid UTF-8. Encoding would
// otherwise fail or, in other JSON packages, silently replace the bytes.
func (ev Event) checkUTF8() error {
	fields := []struct{ name, value string }{
		{"id", ev.ID}, {"op", ev.Op}, {"type", ev.Type}, {"re", ev.Re},
		{"from", ev.From}, {"status", ev.Status}, {"word", ev.Word}, {"text", ev.Text},
		{"ts", ev.TS}, {"author.kind", ev.Author.Kind}, {"author.name", ev.Author.Name},
		{"tty", ev.TTY},
	}
	if ev.At != nil {
		fields = append(fields,
			struct{ name, value string }{"at.path", ev.At.Path},
			struct{ name, value string }{"at.head", ev.At.Head},
		)
	}
	for _, f := range fields {
		if !utf8.ValidString(f.value) {
			return &InvalidEventError{Field: f.name, Reason: "not valid UTF-8"}
		}
	}
	return nil
}
