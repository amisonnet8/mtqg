package journal

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const (
	idA = "6b0d549b6f03475a8600a35a099950d8"
	idB = "1012f037b64c44228c38fb2918f135d2"
	idC = "95e761d177314f10b06bf2efc6f87718"
)

var yamada = Author{Kind: AuthorHuman, Name: "yamada"}

// The golden lines pin the bytes that are written. Any change here changes what
// is committed to people's repositories, so it must follow a change of
// docs/reference/schema.md.
func TestEncodeLineGolden(t *testing.T) {
	tests := []struct {
		name string
		ev   Event
		want string
	}{
		{
			name: "todo from the schema example",
			ev: Event{
				ID: idA, Op: OpCreate, Type: "todo", Status: "open", Text: "Support C syntax",
				V: 0, TS: "2026-09-17T00:00:00Z", Author: yamada,
			},
			want: `{"id":"6b0d549b6f03475a8600a35a099950d8","op":"create","type":"todo","status":"open","text":"Support C syntax","v":0,"ts":"2026-09-17T00:00:00Z","author":{"kind":"human","name":"yamada"}}` + "\n",
		},
		{
			name: "answer with re and tty",
			ev: Event{
				ID: idC, Op: OpCreate, Type: "qa", Re: idB, Text: "Not in the first version",
				V: 0, TS: "2026-09-17T00:41:00Z", Author: yamada, TTY: "3e9a0b12",
			},
			want: `{"id":"95e761d177314f10b06bf2efc6f87718","op":"create","type":"qa","re":"1012f037b64c44228c38fb2918f135d2","text":"Not in the first version","v":0,"ts":"2026-09-17T00:41:00Z","author":{"kind":"human","name":"yamada"},"tty":"3e9a0b12"}` + "\n",
		},
		{
			name: "reply to a bug",
			ev: Event{
				ID: idC, Op: OpCreate, Type: "bug", Re: idB, Text: "Reproduced on macOS too",
				V: 0, TS: "2026-09-17T01:20:00Z", Author: Author{Kind: AuthorAI, Name: "claude-code"},
			},
			want: `{"id":"95e761d177314f10b06bf2efc6f87718","op":"create","type":"bug","re":"1012f037b64c44228c38fb2918f135d2","text":"Reproduced on macOS too","v":0,"ts":"2026-09-17T01:20:00Z","author":{"kind":"ai","name":"claude-code"}}` + "\n",
		},
		{
			name: "status change",
			ev: Event{
				ID: idA, Op: OpStatus, From: "open", Status: "done",
				V: 0, TS: "2026-09-17T01:30:00Z", Author: Author{Kind: AuthorAI, Name: "claude-code"},
			},
			want: `{"id":"6b0d549b6f03475a8600a35a099950d8","op":"status","from":"open","status":"done","v":0,"ts":"2026-09-17T01:30:00Z","author":{"kind":"ai","name":"claude-code"}}` + "\n",
		},
		{
			name: "basis comes after status and is omitted when 0",
			ev: Event{
				ID: idA, Op: OpStatus, From: "open", Status: "done", Basis: 3,
				V: 0, TS: "2026-09-17T01:30:00Z", Author: Author{Kind: AuthorAI, Name: "claude-code"},
			},
			want: `{"id":"6b0d549b6f03475a8600a35a099950d8","op":"status","from":"open","status":"done","basis":3,"v":0,"ts":"2026-09-17T01:30:00Z","author":{"kind":"ai","name":"claude-code"}}` + "\n",
		},
		{
			name: "delete has no body",
			ev:   Event{ID: idA, Op: OpDelete, V: 0, TS: "2026-09-17T02:00:00Z", Author: yamada},
			want: `{"id":"6b0d549b6f03475a8600a35a099950d8","op":"delete","v":0,"ts":"2026-09-17T02:00:00Z","author":{"kind":"human","name":"yamada"}}` + "\n",
		},
		{
			name: "glossary in Japanese is not escaped",
			ev: Event{
				ID: idB, Op: OpCreate, Type: "glossary", Word: "字句解析", Text: "ソースを読んでトークン列にすること",
				V: 0, TS: "2026-09-17T01:00:00Z", Author: yamada,
			},
			want: `{"id":"1012f037b64c44228c38fb2918f135d2","op":"create","type":"glossary","word":"字句解析","text":"ソースを読んでトークン列にすること","v":0,"ts":"2026-09-17T01:00:00Z","author":{"kind":"human","name":"yamada"}}` + "\n",
		},
		{
			name: "angle brackets, ampersand and the JavaScript separators are not escaped",
			ev: Event{
				ID: idA, Op: OpEdit, Text: "a<b>&c\xe2\x80\xa8d\xe2\x80\xa9e",
				V: 0, TS: "2026-09-17T03:00:00Z", Author: yamada,
			},
			want: "{\"id\":\"6b0d549b6f03475a8600a35a099950d8\",\"op\":\"edit\",\"text\":\"a<b>&c\xe2\x80\xa8d\xe2\x80\xa9e\",\"v\":0,\"ts\":\"2026-09-17T03:00:00Z\",\"author\":{\"kind\":\"human\",\"name\":\"yamada\"}}\n",
		},
		{
			name: "newline, tab, quote, backslash and control characters are escaped",
			ev: Event{
				ID: idA, Op: OpEdit, Text: "line1\nline2\t\"q\" \\ \x01",
				V: 0, TS: "2026-09-17T03:00:00Z", Author: yamada,
			},
			want: `{"id":"6b0d549b6f03475a8600a35a099950d8","op":"edit","text":"line1\nline2\t\"q\" \\ \u0001","v":0,"ts":"2026-09-17T03:00:00Z","author":{"kind":"human","name":"yamada"}}` + "\n",
		},
		{
			name: "at comes after text and before v",
			ev: Event{
				ID: idA, Op: OpCreate, Type: "memo", Text: "See the spec",
				At: &At{Path: "docs/spec.md", Line: 42, Head: "3f9a1c0"},
				V:  0, TS: "2026-09-17T04:00:00Z", Author: yamada,
			},
			want: `{"id":"6b0d549b6f03475a8600a35a099950d8","op":"create","type":"memo","text":"See the spec","at":{"path":"docs/spec.md","line":42,"head":"3f9a1c0"},"v":0,"ts":"2026-09-17T04:00:00Z","author":{"kind":"human","name":"yamada"}}` + "\n",
		},
		{
			name: "v is written even when it is 0, and as itself otherwise",
			ev:   Event{ID: idA, Op: OpDelete, V: 1, TS: "2026-09-17T02:00:00Z", Author: yamada},
			want: `{"id":"6b0d549b6f03475a8600a35a099950d8","op":"delete","v":1,"ts":"2026-09-17T02:00:00Z","author":{"kind":"human","name":"yamada"}}` + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeLine(tt.ev)
			if err != nil {
				t.Fatalf("encodeLine: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("encodeLine mismatch\n got: %q\nwant: %q", got, tt.want)
			}
			if n := bytes.Count(got, []byte("\n")); n != 1 || got[len(got)-1] != '\n' {
				t.Errorf("a line must be exactly one line ending with LF, got %d line feeds in %q", n, got)
			}
		})
	}
}

func TestEncodeLineRefusesInvalidUTF8(t *testing.T) {
	ev := Event{
		ID: idA, Op: OpCreate, Type: "memo", Text: "caf\xe9",
		V: 0, TS: "2026-09-17T00:00:00Z", Author: yamada,
	}
	got, err := encodeLine(ev)
	if err == nil {
		t.Fatalf("encodeLine accepted invalid UTF-8 and wrote %q", got)
	}
	var invalid *InvalidEventError
	if !errors.As(err, &invalid) || invalid.Field != "text" || !errors.Is(err, ErrInvalidEvent) {
		t.Errorf("want an InvalidEventError for text, got %v", err)
	}
}

func TestParseEvent(t *testing.T) {
	t.Run("a line written by encodeLine reads back", func(t *testing.T) {
		want := Event{
			ID: idA, Op: OpCreate, Type: "memo", Text: "a<b>&c\xe2\x80\xa8 日本語\n2行目", Basis: 3,
			At: &At{Path: "a.go", Line: 3, Head: "abc"},
			V:  0, TS: "2026-09-17T04:00:00Z", Author: yamada, TTY: "3e9a0b12",
		}
		line, err := encodeLine(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := parseEvent(bytes.TrimSuffix(line, []byte("\n")))
		if err != nil {
			t.Fatalf("parseEvent: %v", err)
		}
		if got.At == nil || *got.At != *want.At {
			t.Fatalf("at: got %+v, want %+v", got.At, want.At)
		}
		got.At, want.At = nil, nil
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	tests := []struct {
		name    string
		line    string
		wantErr error
		check   func(t *testing.T, ev Event)
	}{
		{
			name: "key order, spaces and escapes do not matter",
			line: ` { "author" : {"name":"yamada","kind":"human"}, "ts":"2026-09-17T00:00:00Z", "v":0, "text":"\u65e5\u672c\u8a9e \u003c", "op":"create", "id":"` + idA + `" } `,
			check: func(t *testing.T, ev Event) {
				if ev.ID != idA || ev.Op != OpCreate || ev.Text != "日本語 <" || ev.Author != yamada {
					t.Errorf("unexpected event %+v", ev)
				}
			},
		},
		{
			name: "unknown fields are ignored",
			line: `{"id":"` + idA + `","op":"create","priority":"high","extra":{"a":[1,2]}}`,
			check: func(t *testing.T, ev Event) {
				if ev.ID != idA {
					t.Errorf("unexpected event %+v", ev)
				}
			},
		},
		{
			name: "a repeated key is accepted and the last value wins",
			line: `{"id":"` + idA + `","op":"create","text":"first","text":"second"}`,
			check: func(t *testing.T, ev Event) {
				if ev.Text != "second" {
					t.Errorf("text = %q, want the last value", ev.Text)
				}
			},
		},
		{
			name:    "key names are case-sensitive, so ID is not id",
			line:    `{"ID":"` + idA + `","op":"create"}`,
			wantErr: errMissingField,
		},
		{name: "no id", line: `{"op":"create"}`, wantErr: errMissingField},
		{name: "no op", line: `{"id":"` + idA + `"}`, wantErr: errMissingField},
		{name: "not JSON", line: `hello`, wantErr: errInvalidJSON},
		{name: "a JSON array", line: `[1,2]`, wantErr: errInvalidJSON},
		{name: "a JSON string", line: `"text"`, wantErr: errInvalidJSON},
		{name: "null", line: `null`, wantErr: errInvalidJSON},
		{name: "cut off", line: `{"id":"` + idA + `","op":"cre`, wantErr: errInvalidJSON},
		{name: "text after the object", line: `{"id":"` + idA + `","op":"create"} x`, wantErr: errInvalidJSON},
		{name: "a field of the wrong type", line: `{"id":5,"op":"create"}`, wantErr: errInvalidJSON},
		{name: "invalid UTF-8", line: "{\"id\":\"" + idA + "\",\"op\":\"create\",\"text\":\"caf\xe9\"}", wantErr: errInvalidUTF8},
		{name: "empty", line: ``, wantErr: errInvalidJSON},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, err := parseEvent([]byte(tt.line))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseEvent: %v", err)
			}
			tt.check(t, ev)
		})
	}
}

func TestParseEventLongLine(t *testing.T) {
	text := strings.Repeat("x", 1<<20)
	line, err := encodeLine(Event{ID: idA, Op: OpCreate, Type: "memo", Text: text, V: 0, TS: "2026-09-17T00:00:00Z", Author: yamada})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := parseEvent(bytes.TrimSuffix(line, []byte("\n")))
	if err != nil {
		t.Fatalf("parseEvent: %v", err)
	}
	if len(ev.Text) != len(text) {
		t.Errorf("text length = %d, want %d", len(ev.Text), len(text))
	}
}
