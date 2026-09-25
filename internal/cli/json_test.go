package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// jsonFixture is a journal with most kinds in it (rule is covered by its own,
// smaller tests instead, to keep this fixture's counts from having to change),
// some of it awkward to write as JSON: text with < and &, a control character,
// several lines, Japanese.
func jsonFixture(h *harness) {
	h.setJournal(
		record(idA, "todo", "Skip <block> comments & more", "yamada", "2026-09-17T10:18:00Z"),
		change(idA, "open", "done", "yamada", "2026-09-17T10:40:00Z"),
		record(idC, "todo", "Show error positions\nas line and column", "yamada", "2026-09-17T11:06:00Z"),
		record(idM, "memo", "Policy: \x1b[31mred\x1b[0m and 日本語", "yamada", "2026-09-17T10:32:00Z"),
		question(idQ2, "Should nested block comments be supported?", nameC, "2026-09-17T09:10:00Z"),
		answerLine(idA1, idQ2, "Supporting them is preferable", nameC, "ai", "2026-09-17T09:15:00Z"),
		answerLine(idA2, idQ2, "Not in the first version", "yamada", "human", "2026-09-17T09:41:00Z"),
		bugLine(idBug1, "Parser crashes on empty input", "yamada", "2026-09-17T10:41:00Z"),
		replyLine(idRep1, idBug1, "Reproduced on macOS too", nameC, "ai", "2026-09-17T10:45:00Z"),
		entryLine(idW1, "lexing", "Reading source and turning it into tokens", "yamada", "2026-09-17T11:24:00Z"),
		entryLine(idW2, "lexing", "Splitting text into words", nameC, "2026-09-17T11:30:00Z"),
	)
}

// jsonObject reads what a command printed as one JSON object, and fails if it is
// anything else: more than one value, or something that is not an object.
func jsonObject(t *testing.T, out string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(out))
	var obj map[string]any
	if err := dec.Decode(&obj); err != nil {
		t.Fatalf("not a JSON object: %v\n%s", err, out)
	}
	if dec.More() {
		t.Fatalf("more than one JSON value:\n%s", out)
	}
	if !strings.HasSuffix(out, "}\n") {
		t.Fatalf("output does not end with a closing brace and a line feed: %q", out)
	}
	return obj
}

func field(t *testing.T, obj map[string]any, path ...string) any {
	t.Helper()
	var v any = obj
	for _, p := range path {
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("%v: %v is not an object", path, v)
		}
		v, ok = m[p]
		if !ok {
			t.Fatalf("no field %v in %v", path, obj)
		}
	}
	return v
}

func records(t *testing.T, obj map[string]any, key string) []map[string]any {
	t.Helper()
	list, ok := obj[key].([]any)
	if !ok {
		t.Fatalf("%s is not an array: %v", key, obj[key])
	}
	out := make([]map[string]any, len(list))
	for i, e := range list {
		out[i] = e.(map[string]any)
	}
	return out
}

func TestJSONTodoListIsPinned(t *testing.T) {
	h := initialized(t)
	h.setJournal(
		record(idA, "todo", "Skip <block> comments & more", "yamada", "2026-09-17T10:18:00Z"),
		change(idA, "open", "done", "yamada", "2026-09-17T10:40:00Z"),
		record(idC, "todo", "Show positions", nameC, "2026-09-17T11:06:00Z"),
	)
	code, out, errOut := h.run("todo", "list", "--all", "--json")
	wantExit(t, code, 0, out, errOut)
	want := `{
  "command": "todo list",
  "records": [
    {
      "id": "6cad4a268d0f4e2f8c1b7a3d5e9f0a11",
      "kind": "todo",
      "text": "Skip <block> comments & more",
      "status": "done",
      "author": {
        "kind": "human",
        "name": "yamada"
      },
      "created": "2026-09-17T10:18:00Z",
      "updated": "2026-09-17T10:40:00Z"
    },
    {
      "id": "1e27a1c08a3b4c5d8e9f0a1b2c3d4e33",
      "kind": "todo",
      "text": "Show positions",
      "status": "open",
      "author": {
        "kind": "human",
        "name": "claude-code"
      },
      "created": "2026-09-17T11:06:00Z",
      "updated": "2026-09-17T11:06:00Z"
    }
  ],
  "open": 1,
  "done": 1
}
`
	if out != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", out, want)
	}
}

func TestJSONEveryCommandPrintsOneObject(t *testing.T) {
	h := initialized(t)
	jsonFixture(h)
	for _, tt := range []struct {
		args    []string
		command string
	}{
		{[]string{"memo", "list"}, "memo list"},
		{[]string{"todo", "list"}, "todo list"},
		{[]string{"qa", "list"}, "qa list"},
		{[]string{"bug", "list"}, "bug list"},
		{[]string{"glossary", "list"}, "glossary list"},
		{[]string{"log"}, "log"},
		{[]string{"show", idQ2[:10]}, "show"},
		{[]string{"status"}, "status"},
		{[]string{"context"}, "context"},
		{[]string{"version"}, "version"},
		{[]string{"help"}, "help"},
		{[]string{"todo", "done", "--help"}, "help"},
		{[]string{"todo", "done", idC[:10]}, "todo done"},
		{[]string{"todo", "reopen", idC[:10]}, "todo reopen"},
		{[]string{"qa", "done", idQ2[:10]}, "qa done"},
		{[]string{"bug", "done", idBug1[:10]}, "bug done"},
		{[]string{"memo", "add", "a memo"}, "memo add"},
		{[]string{"todo", "add", "a todo"}, "todo add"},
		{[]string{"qa", "add", "a question?"}, "qa add"},
		{[]string{"qa", "add", idQ2[:10], "an answer"}, "qa add"},
		{[]string{"bug", "add", "a bug"}, "bug add"},
		{[]string{"glossary", "add", "word", "a definition"}, "glossary add"},
		{[]string{"edit", idC[:10], "a new text"}, "edit"},
		{[]string{"search", "block"}, "search"},
		{[]string{"review"}, "review"},
		{[]string{"format"}, "format"},
		{[]string{"delete", idM[:10]}, "delete"},
		{[]string{"undo"}, "undo"}, // removes the line of the glossary entry that was added above
	} {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			args := append([]string{"--json"}, tt.args...)
			code, out, errOut := h.run(args...)
			wantExit(t, code, 0, out, errOut)
			if errOut != "" {
				t.Errorf("stderr %q", errOut)
			}
			obj := jsonObject(t, out)
			if !strings.HasPrefix(out, "{\n  \"command\": "+`"`+tt.command+`"`) {
				t.Errorf("the first field is not the command %q:\n%s", tt.command, out)
			}
			if obj["command"] != tt.command {
				t.Errorf("command = %v, want %q", obj["command"], tt.command)
			}
		})
	}
}

func TestJSONIsTheSameWhateverHowItIsShown(t *testing.T) {
	h := initialized(t)
	jsonFixture(h)
	for _, args := range [][]string{
		{"log", "--limit", "0"},
		{"qa", "list", "--all"},
		{"show", idQ2[:10]},
		{"todo", "list", "--all"},
		{"glossary", "list"},
		{"status"},
		{"search", "block"},
		{"review"},
	} {
		plain := append([]string{"--json"}, args...)
		_, want, _ := h.run(plain...)

		// --full-id and --no-color change nothing, and neither does a narrow window.
		h.env.StdoutIsTerminal, h.env.StdoutWidth, h.env.ANSI = true, 24, true
		_, got, _ := h.run(append([]string{"--json", "--full-id", "--no-color"}, args...)...)
		_, gotColor, _ := h.run(plain...)
		h.env.StdoutIsTerminal, h.env.StdoutWidth, h.env.ANSI = false, 0, false

		if got != want || gotColor != want {
			t.Errorf("%v: the output depends on how it is shown:\n%s\nvs\n%s", args, want, got)
		}
		if strings.Contains(want, "\x1b") || strings.Contains(want, "...") {
			t.Errorf("%v: color or a cut text in the JSON:\n%s", args, want)
		}
	}
}

func TestJSONRecordsAreAsWritten(t *testing.T) {
	h := initialized(t)
	jsonFixture(h)

	_, out, _ := h.run("--json", "memo", "list")
	// The control character is JSON's to escape; it is not replaced with U+FFFD.
	if !strings.Contains(out, `\u001b[31mred\u001b[0m and 日本語`) || strings.Contains(out, "\uFFFD") {
		t.Errorf("stdout %q", out)
	}
	rec := records(t, jsonObject(t, out), "records")[0]
	if rec["text"] != "Policy: \x1b[31mred\x1b[0m and 日本語" {
		t.Errorf("text = %q", rec["text"])
	}

	// The whole text, every line, and no escaping of < and &.
	_, out, _ = h.run("--json", "todo", "list", "--all")
	list := records(t, jsonObject(t, out), "records")
	if list[0]["text"] != "Skip <block> comments & more" || list[1]["text"] != "Show error positions\nas line and column" {
		t.Errorf("texts %q, %q", list[0]["text"], list[1]["text"])
	}
	if !strings.Contains(out, "Skip <block> comments & more") {
		t.Errorf("< or & was escaped: %s", out)
	}

	// Full IDs and UTC times, and the fields that a kind does not have are left out.
	if id := list[0]["id"].(string); id != idA {
		t.Errorf("id = %q, want the full ID", id)
	}
	if list[0]["created"] != "2026-09-17T10:18:00Z" || list[0]["updated"] != "2026-09-17T10:40:00Z" {
		t.Errorf("times %v %v", list[0]["created"], list[0]["updated"])
	}
	for _, key := range []string{"word", "re", "replies"} {
		if _, ok := list[0][key]; ok {
			t.Errorf("a todo has %s", key)
		}
	}
	memo := records(t, jsonObject(t, mustRun(h, "--json", "memo", "list")), "records")[0]
	if _, ok := memo["status"]; ok {
		t.Error("a memo has a status")
	}
}

func mustRun(h *harness, args ...string) string {
	h.t.Helper()
	code, out, errOut := h.run(args...)
	if code != 0 {
		h.t.Fatalf("%v: exit %d: %s", args, code, errOut)
	}
	return out
}

func TestJSONListsAndCounts(t *testing.T) {
	h := initialized(t)
	jsonFixture(h)

	t.Run("todos: --all changes the records and not the counts", func(t *testing.T) {
		open := jsonObject(t, mustRun(h, "--json", "todo", "list"))
		all := jsonObject(t, mustRun(h, "--json", "todo", "list", "--all"))
		if len(records(t, open, "records")) != 1 || len(records(t, all, "records")) != 2 {
			t.Errorf("records: %v, %v", open["records"], all["records"])
		}
		for _, o := range []map[string]any{open, all} {
			if o["open"] != float64(1) || o["done"] != float64(1) {
				t.Errorf("counts %v %v", o["open"], o["done"])
			}
		}
	})

	t.Run("a question has all its answers, oldest first, and they point at it", func(t *testing.T) {
		obj := jsonObject(t, mustRun(h, "--json", "qa", "list"))
		q := records(t, obj, "records")[0]
		if q["kind"] != "question" || q["status"] != "open" {
			t.Errorf("question %v", q)
		}
		replies := records(t, q, "replies")
		if len(replies) != 2 || replies[0]["id"] != idA1 || replies[1]["id"] != idA2 {
			t.Fatalf("replies %v", replies)
		}
		for _, r := range replies {
			if r["kind"] != "answer" || r["re"] != idQ2 {
				t.Errorf("answer %v", r)
			}
		}
		if replies[0]["author"].(map[string]any)["kind"] != "ai" {
			t.Errorf("author %v", replies[0]["author"])
		}
	})

	t.Run("a bug has replies; one with none has an empty list", func(t *testing.T) {
		h.run("bug", "add", "Another bug")
		obj := jsonObject(t, mustRun(h, "--json", "bug", "list"))
		list := records(t, obj, "records")
		if len(list) != 2 || list[0]["kind"] != "bug" {
			t.Fatalf("records %v", list)
		}
		if r := records(t, list[0], "replies"); len(r) != 1 || r[0]["kind"] != "reply" {
			t.Errorf("replies %v", list[0]["replies"])
		}
		if r, ok := list[1]["replies"].([]any); !ok || len(r) != 0 {
			t.Errorf("a bug without replies: %v (want [])", list[1]["replies"])
		}
	})

	t.Run("an empty list is [] and not left out", func(t *testing.T) {
		e := initialized(t)
		for _, args := range [][]string{{"memo", "list"}, {"todo", "list"}, {"qa", "list"}, {"log"}, {"glossary", "list"}} {
			out := mustRun(e, append([]string{"--json"}, args...)...)
			if !strings.Contains(out, `"records": []`) {
				t.Errorf("%v: %s", args, out)
			}
		}
	})

	t.Run("glossary counts entries and the words that have more than one", func(t *testing.T) {
		obj := jsonObject(t, mustRun(h, "--json", "glossary", "list"))
		if obj["entries"] != float64(2) || obj["duplicate_words"] != float64(1) || len(records(t, obj, "records")) != 2 {
			t.Errorf("%v", obj)
		}
		if records(t, obj, "records")[0]["word"] != "lexing" {
			t.Errorf("word %v", records(t, obj, "records")[0]["word"])
		}
	})

	t.Run("log is newest first, with how many were left out", func(t *testing.T) {
		obj := jsonObject(t, mustRun(h, "--json", "log", "--limit", "3"))
		list := records(t, obj, "records")
		if len(list) != 3 || obj["shown"] != float64(3) || obj["total"].(float64) < 10 {
			t.Fatalf("%v", obj)
		}
		if list[0]["created"].(string) < list[1]["created"].(string) {
			t.Errorf("not newest first: %v", list)
		}
	})

	t.Run("memos are counted", func(t *testing.T) {
		obj := jsonObject(t, mustRun(h, "--json", "memo", "list"))
		if obj["count"] != float64(1) {
			t.Errorf("%v", obj)
		}
	})
}

func TestJSONShow(t *testing.T) {
	h := initialized(t)
	jsonFixture(h)
	obj := jsonObject(t, mustRun(h, "--json", "show", idQ2[:10]))
	rec := field(t, obj, "record").(map[string]any)
	if rec["id"] != idQ2 || len(records(t, rec, "replies")) != 2 {
		t.Errorf("record %v", rec)
	}
	// The events are the lines of the journal: its create and the creates of the
	// two answers, in the form of the format.
	events := records(t, obj, "events")
	if len(events) != 3 {
		t.Fatalf("events %v", events)
	}
	if events[0]["op"] != "create" || events[0]["type"] != "qa" || events[0]["v"] != float64(0) || events[0]["ts"] != "2026-09-17T09:10:00Z" {
		t.Errorf("first event %v", events[0])
	}
	if events[1]["id"] != idA1 || events[1]["re"] != idQ2 || events[2]["id"] != idA2 {
		t.Errorf("answers in the events: %v", events[1:])
	}

	// An answer is a record with re, and has no replies.
	obj = jsonObject(t, mustRun(h, "--json", "show", idA2[:10]))
	rec = field(t, obj, "record").(map[string]any)
	if rec["kind"] != "answer" || rec["re"] != idQ2 {
		t.Errorf("answer %v", rec)
	}
	if _, ok := rec["replies"]; ok {
		t.Error("an answer has replies")
	}

	// A todo that was closed has both of its events.
	obj = jsonObject(t, mustRun(h, "--json", "show", idA[:10]))
	events = records(t, obj, "events")
	if len(events) != 2 || events[1]["op"] != "status" || events[1]["from"] != "open" || events[1]["status"] != "done" {
		t.Errorf("events %v", events)
	}
}

func TestJSONAddAndChange(t *testing.T) {
	h := initialized(t)
	h.vars["MTQG_AUTHOR_KIND"], h.vars["MTQG_AUTHOR_NAME"] = "ai", "claude-code"

	obj := jsonObject(t, mustRun(h, "--json", "todo", "add", "Write the docs"))
	rec := field(t, obj, "record").(map[string]any)
	id := rec["id"].(string)
	if len(id) != 32 || rec["kind"] != "todo" || rec["text"] != "Write the docs" || rec["status"] != "open" ||
		field(t, rec, "author", "kind") != "ai" || field(t, rec, "author", "name") != "claude-code" {
		t.Errorf("record %v", rec)
	}
	if !strings.Contains(h.readJournal(), id) {
		t.Error("the record was not written")
	}

	// A reply says what it is a reply to.
	q := jsonObject(t, mustRun(h, "--json", "qa", "add", "Which license?"))
	qid := field(t, q, "record", "id").(string)
	a := jsonObject(t, mustRun(h, "--json", "qa", "add", qid[:10], "MIT"))
	if field(t, a, "record", "kind") != "answer" || field(t, a, "record", "re") != qid {
		t.Errorf("answer %v", a)
	}

	// done says whether it changed anything, and does not write when it did not.
	obj = jsonObject(t, mustRun(h, "--json", "todo", "done", id[:10]))
	if obj["changed"] != true || field(t, obj, "record", "status") != "done" {
		t.Errorf("done: %v", obj)
	}
	before := h.readJournal()
	obj = jsonObject(t, mustRun(h, "--json", "todo", "done", id[:10]))
	if obj["changed"] != false || field(t, obj, "record", "status") != "done" || h.readJournal() != before {
		t.Errorf("done again: %v", obj)
	}
	obj = jsonObject(t, mustRun(h, "--json", "todo", "reopen", id[:10]))
	if obj["changed"] != true || field(t, obj, "record", "status") != "open" {
		t.Errorf("reopen: %v", obj)
	}
}

func TestJSONStatusInitVersionAndHelp(t *testing.T) {
	h := newHarness(t)

	t.Run("init", func(t *testing.T) {
		obj := jsonObject(t, mustRun(h, "--json", "init"))
		if obj["command"] != "init" || obj["root"] != h.root {
			t.Errorf("%v", obj)
		}
	})

	t.Run("status has every count, and the number that are not committed", func(t *testing.T) {
		jsonFixture(h)
		obj := jsonObject(t, mustRun(h, "--json", "status"))
		want := map[string]float64{
			"open_todos": 1, "open_questions": 1, "questions_awaiting_confirmation": 1,
			"open_bugs": 1, "bugs_awaiting_confirmation": 1, "glossary_entries": 2,
			"duplicate_words": 1, "uncommitted_records": 10,
		}
		for k, v := range want {
			if obj[k] != v {
				t.Errorf("%s = %v, want %v", k, obj[k], v)
			}
		}
	})

	t.Run("status without git says null and warns", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		code, out, errOut := h.run("--json", "status")
		wantExit(t, code, 0, "", errOut)
		obj := jsonObject(t, out)
		if v, ok := obj["uncommitted_records"]; !ok || v != nil {
			t.Errorf("uncommitted_records = %v (present %v), want null", v, ok)
		}
		w := jsonObject(t, errOut)["warning"].(map[string]any)
		if w["kind"] != "git_unavailable" {
			t.Errorf("warning %v", w)
		}
	})

	t.Run("version", func(t *testing.T) {
		obj := jsonObject(t, mustRun(h, "--json", "version"))
		if field(t, obj, "format", "repository") != float64(0) || field(t, obj, "format", "supported") != float64(0) || obj["mtqg"] == "" {
			t.Errorf("%v", obj)
		}
		// Where there is no .mtqg/, the format of the repository is null.
		bare := newHarness(t)
		obj = jsonObject(t, mustRun(bare, "--json", "version"))
		if v, ok := field(t, obj, "format").(map[string]any)["repository"]; !ok || v != nil {
			t.Errorf("repository = %v (present %v), want null", v, ok)
		}
	})

	t.Run("help lists the kinds and every command, and says which are built", func(t *testing.T) {
		withUnbuiltCommand(t)
		obj := jsonObject(t, mustRun(h, "--json", "help"))
		kinds := records(t, obj, "kinds")
		if len(kinds) != 6 || kinds[3]["name"] != "bug" || kinds[3]["short"] != "b" || kinds[5]["name"] != "rule" || kinds[5]["short"] != "r" {
			t.Errorf("kinds %v", kinds)
		}
		available := map[string]any{}
		for _, c := range records(t, obj, "commands") {
			available[c["command"].(string)] = c["available"]
		}
		if available["bug done"] != true || available["context"] != true || available["edit"] != true || available["undo"] != true || available["archive"] != true || available["unbuilt"] != false {
			t.Errorf("available %v", available)
		}
	})
}

// oneLineOfJSON reads standard error that must be exactly one line of JSON.
func oneLineOfJSON(t *testing.T, errOut string) map[string]any {
	t.Helper()
	if strings.Count(errOut, "\n") != 1 || !strings.HasSuffix(errOut, "\n") {
		t.Fatalf("stderr is not one line: %q", errOut)
	}
	var obj map[string]any
	if err := json.NewDecoder(bytes.NewReader([]byte(errOut))).Decode(&obj); err != nil {
		t.Fatalf("stderr is not JSON: %v\n%s", err, errOut)
	}
	return obj
}

func TestJSONErrors(t *testing.T) {
	h := initialized(t)
	withUnbuiltCommand(t)
	jsonFixture(h)
	before := h.readJournal()

	for _, tt := range []struct {
		name  string
		args  []string
		code  int
		kind  string
		extra func(t *testing.T, e map[string]any)
	}{
		{"no such ID", []string{"show", "ffff0000"}, 1, "not_found", func(t *testing.T, e map[string]any) {
			if e["prefix"] != "ffff0000" || e["message"] != `No record matches "ffff0000"` {
				t.Errorf("%v", e)
			}
		}},
		{"an ID that is too short", []string{"show", "6ca"}, 1, "id_too_short", func(t *testing.T, e map[string]any) {
			if e["prefix"] != "6ca" {
				t.Errorf("%v", e)
			}
		}},
		{"the wrong kind", []string{"todo", "done", idQ2[:10]}, 1, "wrong_kind", func(t *testing.T, e map[string]any) {
			if e["wanted"] != "todo" || field(t, e, "record", "id") != idQ2 || field(t, e, "record", "kind") != "question" ||
				!strings.Contains(e["message"].(string), "mtqg qa done") {
				t.Errorf("%v", e)
			}
		}},
		{"a memo has no state", []string{"qa", "done", idM[:10]}, 1, "wrong_kind", nil},
		{"an answer cannot be replied to", []string{"qa", "add", idA1[:10], "text"}, 1, "wrong_kind", func(t *testing.T, e map[string]any) {
			if e["wanted"] != "question" || field(t, e, "record", "kind") != "answer" {
				t.Errorf("%v", e)
			}
		}},
		{"an unknown command", []string{"frobnicate"}, 2, "usage", nil},
		{"an unknown option", []string{"todo", "list", "--frobnicate"}, 2, "usage", nil},
		{"a missing argument", []string{"todo", "done"}, 2, "usage", nil},
		{"a wrong limit", []string{"log", "--limit", "many"}, 2, "usage", nil},
		{"a command that is not built", []string{"unbuilt"}, 1, "not_available", nil},
		{"an empty text", []string{"todo", "add", " "}, 1, "empty_text", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"--json"}, tt.args...)
			code, out, errOut := h.run(args...)
			if code != tt.code || out != "" {
				t.Fatalf("exit %d (want %d), stdout %q, stderr %q", code, tt.code, out, errOut)
			}
			e := field(t, oneLineOfJSON(t, errOut), "error").(map[string]any)
			if e["kind"] != tt.kind || e["message"] == "" {
				t.Errorf("error %v, want kind %q", e, tt.kind)
			}
			if tt.extra != nil {
				tt.extra(t, e)
			}
		})
	}
	if h.readJournal() != before {
		t.Error("an error wrote to the journal")
	}

	t.Run("a mistake in the command line is one where --json comes last too", func(t *testing.T) {
		code, out, errOut := h.run("frobnicate", "--json")
		if code != 2 || out != "" || field(t, oneLineOfJSON(t, errOut), "error", "kind") != "usage" {
			t.Errorf("exit %d, stdout %q, stderr %q", code, out, errOut)
		}
	})

	t.Run("an ID that means two records names both, with their full IDs", func(t *testing.T) {
		amb := initialized(t)
		amb.setJournal(record(idA, "todo", "first", "yamada", "2026-09-17T10:18:00Z"), record(idB, "todo", "second", "yamada", "2026-09-17T10:19:00Z"))
		code, out, errOut := amb.run("--json", "todo", "done", "6cad4a26")
		if code != 1 || out != "" {
			t.Fatalf("exit %d, stdout %q", code, out)
		}
		e := field(t, oneLineOfJSON(t, errOut), "error").(map[string]any)
		c := records(t, e, "candidates")
		if e["kind"] != "ambiguous" || e["prefix"] != "6cad4a26" || len(c) != 2 || c[0]["id"] != idA || c[1]["id"] != idB {
			t.Errorf("%v", e)
		}
	})

	t.Run("without .mtqg/", func(t *testing.T) {
		bare := newHarness(t)
		code, out, errOut := bare.run("--json", "status")
		e := field(t, oneLineOfJSON(t, errOut), "error").(map[string]any)
		if code != 1 || out != "" || e["kind"] != "not_initialized" {
			t.Errorf("exit %d, stdout %q, error %v", code, out, e)
		}
	})

	t.Run("without a name for the author", func(t *testing.T) {
		n := initialized(t)
		git(t, n.root, "config", "--unset", "user.name")
		code, out, errOut := n.run("--json", "todo", "add", "x")
		if code != 1 || out != "" || field(t, oneLineOfJSON(t, errOut), "error", "kind") != "no_author" {
			t.Errorf("exit %d, stdout %q, stderr %q", code, out, errOut)
		}
	})
}

func TestJSONWarnings(t *testing.T) {
	h := initialized(t)
	lines := []string{record(idA, "todo", "ok", "yamada", "2026-09-17T10:18:00Z")}
	for i := 0; i < 7; i++ {
		lines = append(lines, "garbage")
	}
	h.setJournal(lines...)

	code, out, errOut := h.run("--json", "todo", "list")
	wantExit(t, code, 0, "", errOut)
	// The list is still one object, and the warnings are lines of JSON on standard
	// error: the first five, and how many more.
	if len(records(t, jsonObject(t, out), "records")) != 1 {
		t.Errorf("stdout %s", out)
	}
	got := strings.Split(strings.TrimSuffix(errOut, "\n"), "\n")
	if len(got) != 6 {
		t.Fatalf("stderr has %d lines:\n%s", len(got), errOut)
	}
	first := oneLineOfJSON(t, got[0]+"\n")["warning"].(map[string]any)
	if first["kind"] != "invalid_json" || first["line"] != float64(2) || !strings.Contains(first["message"].(string), "line 2") {
		t.Errorf("first warning %v", first)
	}
	more := oneLineOfJSON(t, got[5]+"\n")["warning"].(map[string]any)
	if more["kind"] != "more" || more["count"] != float64(2) {
		t.Errorf("last warning %v", more)
	}

	// Without --json, the words are as they were.
	_, out, errOut = h.run("todo", "list")
	if !strings.Contains(errOut, "warning: .mtqg/journal.jsonl line 2 is not a valid JSON object; skipped it") || strings.Contains(errOut, `"warning"`) || strings.Contains(out, "{") {
		t.Errorf("stdout %q, stderr %q", out, errOut)
	}
}
