package journal

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

// Line is one non-blank line of a journal file.
type Line struct {
	// Raw is the line as it was read, without its line ending. Rewriting keeps
	// these bytes as they are, so unknown fields, lines that could not be read
	// and hand-made formatting survive. It is nil for a line that is to be
	// written from Event.
	Raw []byte

	// Event is the line read as an event. It is nil when the line could not be
	// read as one; Scan has then reported a warning for it.
	Event *Event

	// Number is the 1-based number of the line in the input, counting blank
	// lines. It is 0 for a line that was not read from a file.
	Number int
}

// WarningKind says why a line was skipped or is suspect.
type WarningKind int

const (
	// WarnInvalidJSON: the line is not a valid JSON object, or a field has the
	// wrong type.
	WarnInvalidJSON WarningKind = iota + 1
	// WarnInvalidUTF8: the line contains bytes that are not valid UTF-8.
	WarnInvalidUTF8
	// WarnMissingField: the line has no id or no op.
	WarnMissingField
	// WarnConflictMarker: the line is a merge conflict marker.
	WarnConflictMarker
	// WarnNoTrailingNewline: the last line does not end with a line feed. It
	// may be a line that is still being written.
	WarnNoTrailingNewline
)

// Warning reports a line that was skipped, or that may not be complete. The
// wording is left to the CLI layer.
type Warning struct {
	Kind WarningKind
	Line int
}

// Result is the events of a journal, ready to be turned into a state.
type Result struct {
	// Events are the events in the order of the file, with lines whose content
	// is exactly the same counted once. Their order carries no meaning: the
	// order of events is decided by ts and id (docs/reference/schema.md).
	Events []Event

	Warnings []Warning
}

// Scan reads lines of the format from r. It is used for journal.jsonl and for
// any other text that holds event lines.
//
// Blank lines are skipped without a warning. Every other line is returned, so
// that a rewrite can keep it; a line that cannot be read as an event has a nil
// Event and a warning. A line ending of CRLF is read as LF. Lines can be of any
// length. The error is only for a failure of r.
func Scan(r io.Reader) ([]Line, []Warning, error) {
	br := bufio.NewReader(r)
	var (
		lines    []Line
		warnings []Warning
	)
	for n := 1; ; n++ {
		raw, readErr := br.ReadBytes('\n')
		if len(raw) > 0 {
			complete := raw[len(raw)-1] == '\n'
			text := bytes.TrimSuffix(bytes.TrimSuffix(raw, []byte("\n")), []byte("\r"))
			if len(bytes.TrimSpace(text)) > 0 {
				line, warning := classify(n, text)
				lines = append(lines, line)
				if warning != nil {
					warnings = append(warnings, *warning)
				}
				if !complete {
					warnings = append(warnings, Warning{Kind: WarnNoTrailingNewline, Line: n})
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return lines, warnings, nil
			}
			return nil, nil, readErr
		}
	}
}

// classify reads one non-blank line, without its line ending.
func classify(number int, raw []byte) (Line, *Warning) {
	line := Line{Raw: raw, Number: number}
	if isConflictMarker(raw) {
		return line, &Warning{Kind: WarnConflictMarker, Line: number}
	}
	ev, err := parseEvent(raw)
	switch {
	case err == nil:
		line.Event = &ev
		return line, nil
	case errors.Is(err, errInvalidUTF8):
		return line, &Warning{Kind: WarnInvalidUTF8, Line: number}
	case errors.Is(err, errMissingField):
		return line, &Warning{Kind: WarnMissingField, Line: number}
	default:
		return line, &Warning{Kind: WarnInvalidJSON, Line: number}
	}
}

// isConflictMarker reports whether a line is one of the markers git leaves in a
// file it could not merge: <<<<<<<, =======, >>>>>>> (and ||||||| in the diff3
// style). An event line starts with "{", so it can never be taken for one.
func isConflictMarker(raw []byte) bool {
	for _, marker := range []string{"<<<<<<<", ">>>>>>>", "|||||||"} {
		if bytes.HasPrefix(raw, []byte(marker)) && (len(raw) == len(marker) || raw[len(marker)] == ' ') {
			return true
		}
	}
	return string(bytes.TrimRight(raw, " \t")) == "======="
}

// newResult collects the events of lines. Lines whose content is exactly the
// same are one event, so that a line that came back through a merge is not
// counted twice.
func newResult(lines []Line, warnings []Warning) Result {
	result := Result{Warnings: warnings}
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		if line.Event == nil {
			continue
		}
		if _, dup := seen[string(line.Raw)]; dup {
			continue
		}
		seen[string(line.Raw)] = struct{}{}
		result.Events = append(result.Events, *line.Event)
	}
	return result
}
