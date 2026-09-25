# mtqg command reference

*[日本語](cli_ja.md) | **English***

```
mtqg <kind> <verb> [args]
mtqg <command> [args]
```

mtqg's own messages are in English. Record contents are shown as written, in
any language. The storage format is described in [schema.md](schema.md).

## Kinds and verbs

| Kind | `add` | `done` | `reopen` | `list` |
|---|---|---|---|---|
| memo (`m`) | `add <text>` | — | — | all |
| todo (`t`) | `add <text>` | `done <id>` | `reopen <id>` | open (`--all`: all) |
| qa (`q`) | `add <question>`<br>`add <question-id> <answer>` | `done <question-id>` | `reopen <question-id>` | open (`--all`: all) |
| bug (`b`) | `add <bug>`<br>`add <bug-id> <reply>` | `done <bug-id>` | `reopen <bug-id>` | open (`--all`: all) |
| glossary (`g`) | `add <word> <definition>` | — | — | all |
| rule (`r`) | `add <text>` | — | — | all |

- A kind can be abbreviated to one letter: `mtqg t add ...` is `mtqg todo add ...`.
  The verb is always required.
- Using a verb on a record of the wrong kind (for example `mtqg qa done` with a
  todo ID) is an error that names the right command.
- `qa` and `bug` work the same way. See
  [Questions, answers, bugs and replies](#questions-answers-bugs-and-replies).

## Other commands

| Command | Description |
|---|---|
| `mtqg edit <id> [<text>]` | Replace the text of a record. For glossary, the definition; the word cannot change. Without the text, `$EDITOR` opens |
| `mtqg delete <id>` | Hide a record. Deleting a question or a bug also hides its answers or replies |
| `mtqg undo` | Remove the last line this author wrote from this terminal |
| `mtqg status` | Summary of open items and uncommitted records |
| `mtqg log [--limit N] [--kind K]` | Records of all kinds, newest first |
| `mtqg show <id>` | One record with its full text and history |
| `mtqg search <text>` | Records whose text contains `<text>`, newest first |
| `mtqg review` | Concurrent status changes, duplicate glossary definitions, and answers or replies with no parent |
| `mtqg context [--max-tokens N]` | Summary for AI agents |
| `mtqg format [--mark] [file]` | Pretty-print event lines found in any text |
| `mtqg archive <start>..<end> [-n]` | Move the items of a date range out of view (`-n`: only report) |
| `mtqg init` | Create `.mtqg/` |
| `mtqg version` | Show the mtqg version and the repository's format version |
| `mtqg completion <shell>` | Print the completion script of a shell: `bash`, `zsh`, `fish` or `powershell` |
| `mtqg candidates [--word=<partial>] -- <word>...` | List what can come next on a command line. The completion scripts call it. See [Shell completion](#shell-completion) |
| `mtqg help` | List the commands (`-h` and `--help` do the same) |

`-h` and `--help` show less the more of the command line is already known:
`mtqg -h` (or `mtqg help`) lists every command, `mtqg <kind> -h` (a kind with
no verb yet, for example `mtqg todo -h`) lists that kind's verbs, and
`mtqg <kind> <verb> -h` shows just that command's usage and summary.

## Global options

| Option | Description |
|---|---|
| `-C <path>` | Start looking for `.mtqg/` from `<path>` instead of the current directory |
| `--json` | Machine-readable output (all commands). See [JSON output](#json-output) |
| `--all` | In lists, include finished items |
| `--full-id` | Show full 32-digit IDs instead of the first 10 digits |
| `--no-color` | Disable color. `NO_COLOR` is also honored |

Color is only decoration and is used only when the output is a terminal.

### Where options go

- Options come **before** the text of a record. From the first word of the text
  on, every word is text, even one that starts with `-`:
  `mtqg t add fix the -x flag` records `fix the -x flag`.
- A text that itself starts with `-` needs `--` in front of it:
  `mtqg t add -- -1 is not allowed`. A single `-` means standard input (see
  [Adding records](#adding-records)).
- Commands that take an ID (`done`, `reopen`, ...) and commands without text
  (`list`, `status`, ...) accept options anywhere: `mtqg t done 6cad --full-id`.
- Options that are not tied to one command (`-C`, `--no-color`) may also come
  before the kind: `mtqg -C ../other t list`.
- An option that is not listed here is an error.

## JSON output

`--json` is for programs: an editor extension, a hook, a script. What it prints
is a promise: it changes only by adding fields, so a reader ignores the fields it
does not know. (The format is `0` until v1; see [schema.md](schema.md#versioning).
Until then this promise is not yet frozen either.)

- The output is **one JSON object**, indented with two spaces and ending with a
  line feed. Its first field is `command`, the command as it was typed without
  the abbreviation (`todo list`, `qa add`, `log`).
- Keys are `snake_case`. A field that has no value is left out, as in
  `journal.jsonl`; a count is never left out.
- **IDs are always the full 32 digits**, whatever `--full-id` says. **Times are
  UTC**, as in `journal.jsonl` (`2026-09-21T10:18:00Z`); showing them in local
  time is up to the reader.
- **Text is exactly what was written**: control characters are not replaced, a
  text is never cut to the width of the window, and there is no color. `--no-color`
  and `--full-id` change nothing. (The one exception is the descriptions of
  `candidates`, which are made for the menu of a shell; see
  [Shell completion](#shell-completion).)
- The output goes to standard output and, on success, nothing goes to standard
  error except warnings (below). Records that are hidden do not appear.

A record is an object:

| Field | Meaning |
|---|---|
| `id` | The full ID |
| `kind` | `memo`, `todo`, `question`, `answer`, `bug`, `reply`, `glossary` or `rule` |
| `word` | The word (glossary only) |
| `text` | The full text (the definition, for glossary) |
| `re` | The ID of the question or bug this answers or replies to (answer and reply only) |
| `status` | `open` or `done` (todo, question and bug only) |
| `author` | `{"kind": "human" or "ai", "name": "..."}` |
| `created` | The time of the first event |
| `updated` | The time of the last event |

<!-- mtqg:example repo=parser -->
```
$ mtqg memo list --json
{
  "command": "memo list",
  "records": [
    {
      "id": "81e74ef5e8e24d949ed904759531985d",
      "kind": "memo",
      "text": "Policy: use English for all error messages",
      "author": {
        "kind": "human",
        "name": "yamada"
      },
      "created": "2026-09-21T10:32:00Z",
      "updated": "2026-09-21T10:32:00Z"
    }
  ],
  "count": 1
}
```

A question or a bug listed by `qa list`, `bug list` or `show` also has
`replies`: its answers or replies, oldest first, as records. (The human form of
`qa list` shows only the latest one; `--json` gives them all.)

What each command prints, after `command`:

| Command | Fields |
|---|---|
| `memo add`, `todo add`, `qa add`, `bug add`, `glossary add`, `rule add` | `record`: the record that was written |
| `todo done`, `todo reopen`, `qa done`, ... | `record`, and `changed`: `false` if the record was in that state already and nothing was written |
| `edit` | `record` (with the new text), and `changed`: `false` if the text was the same and nothing was written |
| `delete` | `record`: what was hidden, and `hidden_replies`: the answers or replies that were hidden with it, as records (`[]` if none) |
| `undo` | `event`: the line that was removed, in the form of [schema.md](schema.md), and `record`: the record it belongs to as it was before, if it is in the journal |
| `search` | `query`, `records` (newest first), `count` |
| `review` | `concurrent_status_changes`: `{"record", "changes"}` for each record, with `changes` as lines of `journal.jsonl`; `duplicate_words`: `{"word", "records"}`; `unattached_replies`: `{"record", "re_record"}`, with `re_record` left out if the `re` names nothing in the journal. Each is `[]` if there is nothing |
| `archive` | `range`: `{"start", "end"}` (as read, `YYYY-MM-DD`); `file`: the archive file, relative to the repository; `dry_run`; `archived`: counts `memos`, `todos`, `questions`, `answers`, `bugs`, `replies`, `glossary_entries` (deleted ones), `rules` (deleted ones) and `records` (their total); `skipped`: counts `open_todos`, `open_questions`, `open_bugs`, `glossary_entries`, `rules`, `records`. When `archived.records` is 0 no file is made |
| `format` | `events` (in time order, as lines of `journal.jsonl`; a line that has a mark in the input has `mark`, `+` or `-`), `count` |
| `memo list`, `rule list` | `records`, `count` |
| `todo list` | `records` (`--all`: including done), `open`, `done` (counts of all in view, whatever `--all` says) |
| `qa list`, `bug list` | as `todo list`; each record has `replies` |
| `glossary list` | `records`, `entries` (their number), `duplicate_words` |
| `log` | `records` (newest first), `shown`, `total` |
| `show` | `record`, and `events`: what happened to it, oldest first, as lines of `journal.jsonl` in the form of [schema.md](schema.md) (for a question or a bug this includes the `create` of each reply) |
| `status` | `open_todos`, `open_questions`, `questions_awaiting_confirmation`, `open_bugs`, `bugs_awaiting_confirmation`, `glossary_entries`, `duplicate_words`, `concurrent_status_changes` (the number of records), `uncommitted_records` (`null` if git cannot be run) |
| `init` | `root`: where `.mtqg/` was created |
| `version` | `mtqg`: the version; `format`: `{"repository": N or null, "supported": N}` (`null` where there is no `.mtqg/`) |
| `completion` | `shell`: the shell that was asked for; `script`: the script |
| `candidates` | `candidates`: `{"value", "description"}` for each, in the order of the text form (`description` is left out if there is none), `count` |
| `help`, or `-h` on a command or a kind | `kinds`: `{"name", "short"}`; `commands`: `{"command", "usage", "summary", "available"}` (only that kind's, for `-h` on a kind). `available` is `false` for a command that is known and not yet built |
| `context` | See [context](#context) |

**Errors** are one line of JSON on **standard error**, and standard output stays
empty. The exit code is the same as without `--json`.

<!-- mtqg:example repo=parser -->
```
$ mtqg show zzzz --json
{"error":{"kind":"not_found","message":"No record matches \"zzzz\"","prefix":"zzzz"}}
```

`kind` says what went wrong, and `message` is the text mtqg would have printed
(lines joined with a line feed). The other fields depend on the kind:

| `kind` | Exit code | Other fields |
|---|---|---|
| `usage` (a mistake in the command line) | 2 | |
| `not_available` (a command that is not built yet) | 1 | |
| `not_in_repository`, `not_initialized`, `already_initialized`, `format_too_new`, `conflict_markers`, `lock_timeout` | 1 | |
| `no_author`, `bad_author_kind`, `empty_text`, `empty_word`, `invalid_text`, `input`, `editor`, `git_unavailable` | 1 | |
| `not_found` | 1 | `prefix` |
| `id_too_short` | 1 | `prefix` |
| `ambiguous` | 1 | `prefix`, `candidates`: the records the ID could mean |
| `wrong_kind` | 1 | `record`: what the ID is, `wanted`: the kind the command is for |
| `no_state`, `no_replies` | 1 | `record` |
| `nothing_to_undo` | 1 | |
| `has_later_events` (`undo` would leave events without their record) | 1 | `record` |
| `unknown` (anything else) | 1 | |

**Warnings** (a line that was skipped, a conflict marker, git that cannot be run)
are also one line each on standard error, and do not change the exit code:

<!-- mtqg:example repo=broken -->
```
$ mtqg todo list --json >/dev/null
{"warning":{"kind":"invalid_json","line":12,"message":"warning: .mtqg/journal.jsonl line 12 is not a valid JSON object; skipped it"}}
```

`kind` is `invalid_json`, `invalid_utf8`, `missing_field`, `conflict_marker`,
`no_trailing_newline`, `unreadable` or `git_unavailable`, and `line` is the line of
`journal.jsonl` where there is one. After the first five, one line says how many
more there were (`kind` `more`, with `count`).

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success. A change that was already made counts as success |
| 1 | The command could not do what was asked: no `.mtqg/`, an unknown ID, the write lock could not be taken, and so on |
| 2 | The command line is wrong: an unknown command or option, a missing argument |

Errors go to standard error. So do warnings (a line that was skipped, conflict
markers), which never change the exit code, so that standard output stays clean
for other tools.

## Finding .mtqg/

mtqg walks up from the current directory (or `-C <path>`). In each directory it
checks, in this order:

1. `.mtqg/` exists: use it.
2. `.git` exists (directory or file): this is the repository root and there is
   no `.mtqg/`. mtqg stops and asks you to run `mtqg init`.

mtqg never looks above the repository root. Outside a git repository it stops
with an error. There is one `.mtqg/` per git repository; a submodule has its
own.

<!-- mtqg:example repo=none path=/home/me/sample-parser -->
```
$ mtqg t add Add tests for comment handling
No .mtqg/ found. Run `mtqg init` (will be created at /home/me/sample-parser)
```

No command creates `.mtqg/` except `mtqg init`.

## init

```
mtqg init
```

- Walks up the same way. At the repository root (`.git`), it creates `.mtqg/`
  next to `.git`, regardless of where it was run.
- If it finds an existing `.mtqg/` on the way: error
  (`.mtqg/ already exists: <path>`).
- Outside a git repository: error.
- Creates `.mtqg/journal.jsonl`, `.mtqg/.gitattributes`, `.mtqg/.gitignore`,
  `.mtqg/version` and `.mtqg/SCHEMA.md`.
- Does not change git configuration and does not commit. It tells you to commit
  `.mtqg/`:

<!-- mtqg:example repo=none path=/home/me/sample-parser -->
```
$ mtqg init
Created .mtqg/ in /home/me/sample-parser
Commit it to share the records.
```

## Adding records

```
mtqg m add Use English for all error messages
mtqg t add Skip block comments
mtqg q add Should nested block comments be supported?
mtqg b add Parser crashes on empty input
mtqg g add token The smallest unit produced by lexing
mtqg r add Write mtqg records in English
```

- The remaining arguments are joined with spaces. Quotes are not needed, except
  around shell special characters (`#` `*` `(` `)` `&` `|` `<` `>`).
- For glossary, the first argument is the word and the rest is the definition.
  A word of several words needs quotes:
  `mtqg g add "block comment" A comment enclosed in /* and */`
- `mtqg q add <question-id> <text>` adds an answer instead of a question, and
  `mtqg b add <bug-id> <text>` adds a reply instead of a bug (see
  [Questions, answers, bugs and replies](#questions-answers-bugs-and-replies)).
- `-` instead of the text reads it from standard input, to its end. Trailing
  line breaks are dropped: `git log -1 --format=%s | mtqg m add -`
- **When there is no text to write, `$EDITOR` opens** on an empty file, and what is
  saved is the text (trailing line breaks dropped): with no arguments at all
  (`mtqg m add`, `mtqg t add`, `mtqg q add`, `mtqg b add`, `mtqg r add`), for the answer or
  the reply that follows an ID (`mtqg q add <question-id>`,
  `mtqg b add <bug-id>`) and for the definition that follows a word
  (`mtqg g add <word>`). `$EDITOR` may hold arguments and quotes (`code --wait`);
  it is not run through a shell. If `$EDITOR` is not set, mtqg stops and says so.
  An ID that matches no record stops before the editor opens.
- A word that is missing is a mistake in the command line, and nothing is
  written: `mtqg g add` (no word). The text of an answer, a reply or a
  definition can be `-`, to read it from standard input.
- A text can have several lines (from standard input or the editor). `list`
  shows the first line only.
- A text that is empty, or only white space, is an error
  (`Aborting: the text is empty`). Nothing is written.
- Output is only the new record's ID:

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg t add Skip block comments
6cad4a268d
```

## done, reopen

<!-- mtqg:example repo=parser -->
```
$ mtqg t done 6cad4a268d
Done: 6cad4a268d  Skip block comments /* */
$ mtqg t done 6cad4a268d
Already done: 6cad4a268d  Skip block comments /* */
$ mtqg t reopen 6cad4a268d
Reopened: 6cad4a268d  Skip block comments /* */
```

- One line is printed: what was done, the ID and the first line of the text, so
  that you can see the ID you typed was the record you meant.
- If the todo is already in that state, nothing is written and the line says so
  (`Already done: ...`, `Already open: ...`). The exit code is 0: repeating a
  change is not an error.
- `mtqg q done` and `mtqg q reopen` do the same for a question, and
  `mtqg b done` and `mtqg b reopen` for a bug.
- Only todos, questions and bugs have a state. For the ID of any other record,
  mtqg stops and says what it is. If another command is the right one, it names
  it: `81e74ef5e8 is a memo, not a todo`,
  `2217beaddb is a question, not a todo; use `mtqg qa done 2217beaddb``, and
  `7f3a2b1c09 is a bug, not a question; use `mtqg bug done 7f3a2b1c09``.

## Questions, answers, bugs and replies

A question and a bug have the same shape: a record that can be answered, which is
open until it is closed. They differ in what they are about. A question asks
something. A bug reports something that does not work, and the exchange about it;
it is closed when it is fixed or no longer pursued. Everything below holds for
both, with the words *bug* and *reply* where a question has *question* and
*answer*.

- `mtqg q add <text>` adds a question. `mtqg q add <question-id> <text>` adds an
  answer to that question. `mtqg b add <text>` adds a bug, and
  `mtqg b add <bug-id> <text>` adds a reply to that bug. An answer or a reply has
  its own ID, and its `re` holds the full ID of the question or the bug.
- Answering and closing are separate: `mtqg q done <question-id>` closes the
  question, and `mtqg b done <bug-id>` closes the bug. Any number of answers or
  replies can be added; none replaces another. A closed question or bug can
  still be answered or replied to.
- Only questions and bugs can have answers or replies. Answers and replies cannot
  be answered. An answer is always to a question, and a reply always to a bug:
  `mtqg b add <question-id> <text>` is an error that says the ID is a question.

### Question or answer? Bug or reply?

`q add` and `b add` tell them apart by their first argument alone. If the first
argument is **4 or more hex digits and nothing else** (`0-9`, `a-f`; upper case is
read as lower case), it is the ID of the question to answer (or the bug to reply
to), and the rest is the text of the answer (or the reply). Any other first
argument makes the whole text a new question (or bug).

| The first argument, made of 4 or more hex digits, matches | Result |
|---|---|
| one question (`q add`) or one bug (`b add`) | the rest is added as an answer or a reply to it |
| one record that is not a question (`q add`) or not a bug (`b add`) | error that says what it is |
| more than one record | error that lists the candidates with their full IDs |
| no record | error, and nothing is written |

<!-- mtqg:example repo=parser -->
```
$ mtqg q add a8ec What does this mean?
No record matches "a8ec". A first word of 4 or more hex digits is read as the ID of the question to answer.
To ask a question that starts with it, put the whole text in quotes: mtqg qa add "a8ec ..."
```

For `b add` the same error reads `... the ID of the bug to reply to.` and
`To report a bug that starts with it, ...: mtqg bug add "a8ec ..."`.

- A text in quotes is one argument. With a space in it, it is not hex digits
  only, so it is never an ID: `mtqg q add "a8ec What does this mean?"` asks a
  question, and `mtqg q add a8ec Not found error code` answers the question
  `a8ec`.
- No record matching is an error, not a new record, so that a mistyped ID never
  turns into a new question or bug without a word. A text in English whose first
  word is made of hex digits (`Face detection is slow. Why?`, `Dead code: remove
  it?`) stops the same way; quote the whole text. A text of one such word can be
  given on standard input.
- The ID with no text after it opens `$EDITOR` for the text (see
  [Adding records](#adding-records)).

A question or a bug is in one of four states:

| State | Answers or replies | Closed | Shown by `qa list` and `bug list` |
|---|---|---|---|
| unanswered | 0 | no | yes |
| awaiting confirmation | 1 or more | no | yes, with the number of answers or replies |
| answered | 1 or more | yes | with `--all` |
| closed without answer | 0 | yes | with `--all` |

| ID given | Allowed |
|---|---|
| question or bug | `add` (answer or reply), `done`, `reopen`, `edit`, `delete` (hides its answers or replies too) |
| answer or reply | `edit`, `delete` (that answer or reply only; the question's or bug's state is unchanged) |

## IDs

- Every record has a full ID: 32 lowercase hex digits (a UUIDv4 without
  hyphens), e.g. `81e74ef5e8e24d949ed904759531985d`.
- mtqg shows the **first 10 digits** (`81e74ef5e8`). `--full-id` shows full IDs.
- Any unique prefix of **4 or more digits** is accepted as input, like git
  commit hashes: `81e74ef5e8`, or `81e7` if unique. A shorter one is an error
  (`The ID "81e" is too short: give at least 4 digits`), so that a word such as
  `add` or `bad` is never taken for an ID.
- If a prefix matches more than one record, mtqg stops and lists the candidates
  with their full IDs:

<!-- mtqg:example repo=ambiguous -->
```
$ mtqg t done 70430f77ff
Ambiguous ID "70430f77ff" matches 2 records:
  70430f77ff91c2e04a8b33f1d7e6a025  memo  Parser now skips // at line end  yamada       2026-03-02
  70430f77ff4b475185d5cae12dff1a17  todo  Skip block comments /* */        claude-code  10:18
```

- If no record matches, mtqg stops: `No record matches "6cad4"`. A deleted
  record is never matched. Uppercase hex digits are read as lowercase.
- Records refer to each other by full ID (an answer's `re`), so a short ID
  becoming ambiguous later never changes what a record points to.

## edit, delete

<!-- mtqg:example repo=parser -->
```
$ mtqg edit 6cad4a268d Skip block comments and line comments
Edited: 6cad4a268d  Skip block comments and line comments
$ mtqg edit 6cad4a268d Skip block comments and line comments
Unchanged: 6cad4a268d  Skip block comments and line comments

$ mtqg delete 1012f037b6
Deleted: 1012f037b6  Should nested block comments be supported?
2 answers are also hidden (claude-code, yamada)
The lines remain in the journal and in git history
```

Both append an event. Nothing is removed from the file or from git history.

- `edit` replaces the text of any record: a memo, a todo, a question, an answer, a
  bug, a reply, or the definition of a glossary entry. Only the text changes. The
  word of a glossary entry cannot, and neither can the state of a todo, a
  question or a bug. The line printed is the ID and the first line of the new text.
- The text is given as when adding a record: the words after the ID, joined with
  spaces, or `-` for standard input. **Without a text, `$EDITOR` opens on the text
  the record has now**, and what is saved is the new text (see
  [Adding records](#adding-records)).
- If the new text is the same as the text now, nothing is written and the line
  says `Unchanged: ...`. The exit code is 0. A text that is empty, or only white
  space, is an error, and nothing is written.
- `delete` hides any record: no list, `show` or `search` shows it, and its ID
  matches nothing any more. Deleting a question or a bug hides its answers or
  replies too, and the output says how many and who wrote them. Deleting an answer
  or a reply hides that one only; the state of the question or the bug does not
  change. The last line always says that the lines remain in the journal and in
  git history.

## undo

Removes the **last line the current author wrote from the current terminal**, for a
mistake just made (for example an answer added as a new question because the
question ID was left out).

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg q add Not in the first version. Revisit if there is demand
64ce08e71a
$ mtqg undo
Undone: qa add "Not in the first version. Revisit if there is demand" (64ce08e71a)
```

- Target: of the lines in `journal.jsonl` with this author (kind and name, as for
  writing a record) and this terminal's `tty` value, the one with the latest
  `ts`; among lines of the same second, the one written last in the file. Any
  line counts: what `add`, `done`, `reopen`, `edit` and `delete` wrote. The
  words after `Undone:` are the type of the record, what the line did (`add`,
  `done`, `reopen`, `edit`, `delete`), the text and the ID. The time is the one the
  line was written with: a line written by a clock that runs ahead is the latest
  until the time catches up with it.
- **The terminal.** `tty` is 8 hex digits of a hash, so that the terminal can be
  told apart and nothing else is learned from it. It comes from `MTQG_TTY` if that
  is set (any text: give two sessions different values to keep them apart, or one
  value to make them the same). Otherwise, on Linux and macOS, it comes from the
  terminal that standard input, output or error is connected to. On Windows, and
  where none of them is a terminal (an agent that runs commands through pipes),
  the line has no `tty`, and lines without one count as one terminal. So the
  lines an AI agent wrote through pipes and the lines a person wrote at a terminal
  are never each other's target, even under the same name.
- One step only; `undo` does not repeat. What it removed is always printed.
- **It refuses when it would leave events without their record.** If the line is
  the `create` of a record that has other events (a status change, an edit, a
  delete, or answers or replies to it), nothing is removed, and the output says
  how many there are and points to `mtqg delete`. A line of `status`, `edit` or
  `delete` is removed whatever follows it.

<!-- mtqg:example repo=refuse -->
```
$ mtqg undo
Cannot undo: todo 4ffb865902 has 2 other events, and undoing its creation would leave them without a record
To hide it instead: mtqg delete 4ffb865902
```

- If there is no such line: `Nothing to undo: ...`. The exit code is 1.
- It does not check whether the line was committed. For something already
  shared, use `delete`: a removed line that already reached another branch can
  come back with the next merge. Every other line of the file stays as it is,
  byte for byte.

## Reading

### status

<!-- mtqg:example repo=parser -->
```
$ mtqg status
Open todos          5
Open questions      2  (1 awaiting confirmation)
Open bugs           1  (1 awaiting confirmation)
Glossary            4  (1 with duplicate definitions)
Conflicts           1  (concurrent changes; see mtqg review)

Uncommitted records 3
```

- Each line is a count. `Open questions` counts the questions that are not
  closed, and `awaiting confirmation` those of them that have an answer. `Open
  bugs` counts the bugs that are not closed the same way (`awaiting
  confirmation` are those that have a reply). `Glossary` counts the entries, and `with duplicate definitions` the words that
  are defined more than once (the same word, character for character).
  `Conflicts` counts the records that have concurrent status changes (see
  [review](#review)); the line is left out when there are none.
  `Uncommitted records` is the number of records that have
  at least one line in `journal.jsonl` that is not in the last commit (`HEAD`):
  records created or changed since then. A record counts once however many lines
  it has. Lines that are staged but not committed count as uncommitted. With no
  commit yet, or when `.mtqg/journal.jsonl` is not tracked, every record counts.
  If git cannot be run, the line shows `unknown`.

### list

<!-- mtqg:example repo=parser -->
```
$ mtqg todo list
6cad4a268d  Skip block comments /* */                claude-code  10:18
6513270e26  Ignore // inside string literals         yamada       10:52
1e27a1c08a  Show error positions as line and column  claude-code  11:06
1818e81189  List the supported syntax in the README  yamada       11:30
2e44158bae  Add test cases for comment handling      claude-code  11:32
5 open (show done: --all)

$ mtqg todo list --all
6b0d549b6f  Skip line comments //                    yamada       09:50  done
6cad4a268d  Skip block comments /* */                claude-code  10:18
6513270e26  Ignore // inside string literals         yamada       10:52
1e27a1c08a  Show error positions as line and column  claude-code  11:06
1818e81189  List the supported syntax in the README  yamada       11:30
2e44158bae  Add test cases for comment handling      claude-code  11:32
5 open, 1 done

$ mtqg qa list
1012f037b6  Should nested block comments be supported?            claude-code  09:10  2 answers, awaiting confirmation
          └ Not in the first version. Revisit if there is demand  yamada       09:41
2217beaddb  Should error positions show both line and column?     claude-code  11:05  unanswered
2 open (show done: --all)

$ mtqg qa list --all
c3b1f0d2e4  Which license should the parser use?                  yamada       2026-09-19  1 answer, done
          └ MIT is the simplest choice                            claude-code  2026-09-19
1012f037b6  Should nested block comments be supported?            claude-code  09:10       2 answers, awaiting confirmation
          └ Not in the first version. Revisit if there is demand  yamada       09:41
2217beaddb  Should error positions show both line and column?     claude-code  11:05       unanswered
2 open, 1 done

$ mtqg bug list
7f3a2b1c09  Parser crashes on empty input  yamada       10:41  1 reply, awaiting confirmation
          └ Reproduced on macOS too        claude-code  10:45
1 open (show done: --all)

$ mtqg bug list --all
b2c3d4e5f6  Linter crashes on tab characters       yamada       2026-09-19  1 reply, done
          └ Fixed by treating a tab as one column  claude-code  2026-09-19
7f3a2b1c09  Parser crashes on empty input          yamada       10:41       1 reply, awaiting confirmation
          └ Reproduced on macOS too                claude-code  10:45
1 open, 1 done
```

`qa list` and `bug list` are laid out the same way. Each question or bug ends
with its state, and the latest answer or reply is shown under it, indented, with
its author and time:

| State of a question | State of a bug | Meaning |
|---|---|---|
| `unanswered` | `no replies` | open, nothing added |
| `N answers, awaiting confirmation` | `N replies, awaiting confirmation` | open, with answers or replies |
| `N answers, done` | `N replies, done` | closed, with answers or replies (`--all`) |
| `done without answers` | `done without replies` | closed, nothing added (`--all`) |

An answer whose question is not in the journal, or a reply whose bug is not, is
not listed.

<!-- mtqg:example repo=parser -->
```
$ mtqg glossary list
5b7e2c9a41  token          The smallest unit produced by lexing                     yamada       09:00
f29d0da995  block comment  A comment enclosed in /* and */                          yamada       09:30
0cb1e29c65  block comment  A comment that can span multiple lines                   claude-code  10:00
f28c105d1f  lexing         Reading source and turning it into a sequence of tokens  claude-code  11:24
4 terms (1 with duplicate definitions)
```

`glossary list` shows every entry: the ID, the word, the definition, the author
and the time. Two entries for the same word are two lines, in the order they were
written, and the footer says how many words are defined more than once
(`4 terms (1 with duplicate definitions)`). mtqg does not choose between them.

How a list is shown:

- The columns are the ID, the text, the author and the time. The time is `HH:MM`
  for today and `YYYY-MM-DD` for any other day, in local time.
- Records are in the order they were created, oldest first.
- The text is the first line of the record. On a terminal it is cut with `...`
  to fit the width of the window; when the output goes to a pipe or a file it is
  never cut.
- Columns are aligned by display width: a full-width character (Japanese, for
  example) takes two columns.
- With `--all`, a finished item ends with `done`.
- Control characters in a record, such as the escape character, are replaced
  with U+FFFD when shown, so that a record cannot change what the terminal
  does. Output with `--json` is not affected.
- `mtqg memo list` shows every memo and ends with `N memos`.
- `mtqg rule list` shows every rule and ends with `N rules`.

### show

<!-- mtqg:example repo=parser -->
```
$ mtqg show 1012f037b6
question  1012f037b6  open
by claude-code (ai), 2026-09-21 09:10

  Should nested block comments be supported?

Answers (2)
  ae2eb1547f  claude-code (ai)  2026-09-21 09:15
    Supporting them is generally preferable
  95e761d177  yamada (human)    2026-09-21 09:41
    Not in the first version. Revisit if there is demand

Events
  2026-09-21 09:10  create  claude-code (ai)
  2026-09-21 09:15  create  claude-code (ai)  answer ae2eb1547f
  2026-09-21 09:41  create  yamada (human)    answer 95e761d177

$ mtqg show 7f3a2b1c09
bug  7f3a2b1c09  open
by yamada (human), 2026-09-21 10:41

  Parser crashes on empty input

Replies (1)
  3d8e4a0b12  claude-code (ai)  2026-09-21 10:45
    Reproduced on macOS too

Events
  2026-09-21 10:41  create  yamada (human)
  2026-09-21 10:45  create  claude-code (ai)  reply 3d8e4a0b12
```

- The first line names the kind (`memo`, `todo`, `question`, `answer`, `bug`,
  `reply`, `glossary` or `rule`), the ID and, for a todo, a question or a bug, its
  state. The second line
  says who wrote it and when, with the kind of author.
- The text is shown in full, with every line of it, however the output is
  read. Control characters are replaced as in a list. A glossary entry shows
  `Word: <word>` before its definition. An answer shows the question it belongs
  to, and a reply the bug. If the record its `re` names is not there
  (`to bug 7f3a2b1c09  (no such record)`) or is of another kind
  (`to 1012f037b6  (a question, not a bug)`), the line says so instead of naming
  a parent: such a record is not a reply to it (see [review](#review)).
- A question lists its answers, and a bug its replies (`Replies (2)`), oldest
  first, each with its author and time.
- `Events` lists what happened to the record, oldest first, with the full local
  date and time: its own events (`create`, `status` as `open -> done`, `edit`,
  `delete`) and, for a question or a bug, the creation of each answer or reply.

### log

<!-- mtqg:example repo=parser -->
```
$ mtqg log --limit 6
11:32  todo      2e44158bae  Add test cases for comment handling                              claude-code
11:30  todo      1818e81189  List the supported syntax in the README                          yamada
11:24  glossary  f28c105d1f  lexing: Reading source and turning it into a sequence of tokens  claude-code
11:06  todo      1e27a1c08a  Show error positions as line and column                          claude-code
11:05  question  2217beaddb  Should error positions show both line and column?                claude-code
10:52  todo      6513270e26  Ignore // inside string literals                                 yamada
6 of 22 records (--limit 0 for all)

$ mtqg log --kind qa
11:05       question  2217beaddb  Should error positions show both line and column?                     claude-code
09:41       answer    95e761d177  (to 1012f037b6) Not in the first version. Revisit if there is demand  yamada
09:15       answer    ae2eb1547f  (to 1012f037b6) Supporting them is generally preferable               claude-code
09:10       question  1012f037b6  Should nested block comments be supported?                            claude-code
2026-09-19  answer    d4c2a1e3f5  (to c3b1f0d2e4) MIT is the simplest choice                            claude-code
2026-09-19  question  c3b1f0d2e4  Which license should the parser use?                                  yamada       done
6 records

$ mtqg log --kind bug
10:46       reply  9a8b7c6d5e  (to 1012f037b6) Also fails with an empty file          claude-code
10:45       reply  3d8e4a0b12  (to 7f3a2b1c09) Reproduced on macOS too                claude-code
10:41       bug    7f3a2b1c09  Parser crashes on empty input                          yamada
2026-09-19  reply  d4e5f6a7b8  (to b2c3d4e5f6) Fixed by treating a tab as one column  claude-code
2026-09-19  bug    b2c3d4e5f6  Linter crashes on tab characters                       yamada       done
5 records
```

- Every record that is not hidden, of every kind, one line each, **newest
  first**: the time, the kind (`memo`, `todo`, `question`, `answer`, `bug`,
  `reply`, `glossary`, `rule`), the ID, the text and the author. A glossary entry shows
  its word, a colon and its definition. An answer or a reply starts with
  `(to <id>)`, the question or bug it belongs to. A finished todo, question or
  bug ends with `done`.
- The time is `HH:MM` for today and `YYYY-MM-DD` for any other day, the time the
  record was created. The text follows the rules of a list (first line only, cut
  only on a terminal, control characters replaced).
- `--limit N` shows the newest `N` records; the default is 20, and `0` shows all.
  `--kind K` shows one kind only: `memo`, `todo`, `qa` (questions and answers),
  `bug` (bugs and replies), `glossary` or `rule`, or the letter. Both may be written `--limit=N`. A value that is not
  a whole number of 0 or more, or a kind that does not exist, is a mistake in the
  command line.
- The last line counts the records: `N records`, or `N of M records (--limit 0
  for all)` when some are left out.

### search

<!-- mtqg:example repo=parser -->
```
$ mtqg search comment
11:32  todo      2e44158bae  Add test cases for comment handling                    claude-code
10:18  todo      6cad4a268d  Skip block comments /* */                              claude-code
10:00  glossary  0cb1e29c65  block comment: A comment that can span multiple lines  claude-code
09:50  todo      6b0d549b6f  Skip line comments //                                  yamada       done
09:30  glossary  f29d0da995  block comment: A comment enclosed in /* and */         yamada
09:10  question  1012f037b6  Should nested block comments be supported?             claude-code
6 records contain "comment"
```

- `mtqg search <text>` lists the records whose text contains `<text>`: the text of
  every record that is in view, of every kind, and for a glossary entry its word
  as well. The words after `search` are joined with spaces, as for a record.
- Case is ignored. Nothing else is: there are no patterns, no word boundaries and
  no filters (by kind, author or date).
- The lines are those of [log](#log), **newest first**, and all of the matches are
  shown. The text on the line is the first line of the record, so a match in a
  later line of a text is found without being visible: use `show`. The last line
  counts them: `N records contain "<text>"`. With no match it says
  `No records contain "<text>"`, and the exit code is still 0.
- Deleted records are not searched, and neither is `archive/`.

### review

What needs a person's eye: places where the journal holds facts that do not agree.
mtqg shows them and does not decide which one is right. A section with nothing in
it is left out; with nothing at all the output is `Nothing to review`. The exit code
is 0 either way.

<!-- mtqg:example repo=parser -->
```
$ mtqg review
Concurrent changes (1)
  todo 6b0d549b6f "Skip line comments //"
    2026-09-21 10:15  claude-code  open -> done
    2026-09-21 14:30  yamada       open -> done

Duplicate glossary definitions (1)
  block comment
    f29d0da995  yamada       A comment enclosed in /* and */
    0cb1e29c65  claude-code  A comment that can span multiple lines

Answers and replies with no parent (1)
  9a8b7c6d5e  reply  claude-code  Also fails with an empty file
    re 1012f037b6: a question, not a bug
```

- **Concurrent status changes.** Two independent signals catch a record whose
  changes were made without knowing of each other, either one enough to list it.
  ① A `status` event whose `from` is not the state the record is in at that
  point in the format's order (`ts`, then `id`), followed from the state the
  record was created in (two branches that each closed the same todo, merged
  later). ② A `status` event or an `edit` whose `basis` does not match how many
  events (including the `create`) the record actually had at that point: this
  also catches a status change and an edit made at once, since it does not
  matter which field either one touched. The record is listed with **all** of
  its status changes and edits, oldest first, each with its time, author and
  change (an edit shows as `edited`). An event with no `from` and no `basis` is
  not judged. The same line repeated is one event and is not a conflict. The
  state the lists show is the one the last change leaves.
- **Duplicate glossary definitions.** Each word that is defined more than once
  (character for character), with all of its entries in the order they were written.
- **Answers and replies with no parent.** An answer or a reply is bound to its
  question or bug only if its `re` names a record of the same `type` that can be
  replied to. One that names a record that is not in the journal, or a record of
  another kind, is not shown under any question or bug, and this is where it is
  found (`log` shows it, and so does `show` with its ID). Each line says what
  the `re` names: `not in the journal`, or the kind it is, and the kind it should
  be.

### context

What an AI agent reads when a session starts. It passes the process so far:
what is open, what was answered, which terms are agreed. It does not contain the
project description, the current specification or build instructions; those
belong to the README and the agent's instruction files.

<!-- mtqg:example repo=parser -->
```
$ mtqg context
# mtqg context — sample-parser (main)

This is the process record of this project. Read the following before you start working.
- Follow the rules listed under Rules
- Respect answered questions
- Do not decide open questions on your own; confirm them
- Use terms as defined in the glossary
- Record questions, decisions, findings, bugs, and todos with mtqg as they come up

## Attention
- Glossary term "block comment" has conflicting definitions (see mtqg glossary list)
- todo 6b0d549b6f "Skip line comments //" has concurrent changes (see mtqg review)
- 3 mtqg records are not committed

## Open todos (5)
- 6cad4a268d Skip block comments /* */ (claude-code, 10:18)
- 6513270e26 Ignore // inside string literals (yamada, 10:52)
- 1e27a1c08a Show error positions as line and column (claude-code, 11:06)
- 1818e81189 List the supported syntax in the README (yamada, 11:30)
- 2e44158bae Add test cases for comment handling (claude-code, 11:32)

## Open questions (2)
- 1012f037b6 Should nested block comments be supported? (awaiting confirmation, claude-code, 09:10)
    └ Not in the first version. Revisit if there is demand (yamada, human)
- 2217beaddb Should error positions show both line and column? (unanswered, claude-code, 11:05)

## Open bugs (1)
- 7f3a2b1c09 Parser crashes on empty input (awaiting confirmation, yamada, 10:41)
    └ Reproduced on macOS too (claude-code, ai)

## Recent records (newest first)
- 11:32  claude-code  todo      2e44158bae  Add test cases for comment handling
- 11:30  yamada       todo      1818e81189  List the supported syntax in the README
- 11:24  claude-code  glossary  f28c105d1f  lexing: Reading source and turning it into a sequence of tokens
- 11:06  claude-code  todo      1e27a1c08a  Show error positions as line and column
- 11:05  claude-code  question  2217beaddb  Should error positions show both line and column?
- 10:52  yamada       todo      6513270e26  Ignore // inside string literals
- 10:46  claude-code  reply     9a8b7c6d5e  Also fails with an empty file (to 1012f037b6)
- 10:45  claude-code  reply     3d8e4a0b12  Reproduced on macOS too (to 7f3a2b1c09)
- 10:41  yamada       bug       7f3a2b1c09  Parser crashes on empty input
- 10:32  yamada       memo      81e74ef5e8  Policy: use English for all error messages
- (12 more; see mtqg log)

## Glossary (4)
- 5b7e2c9a41 token: The smallest unit produced by lexing
- f29d0da995 block comment: A comment enclosed in /* and */
- 0cb1e29c65 block comment: A comment that can span multiple lines
- f28c105d1f lexing: Reading source and turning it into a sequence of tokens

---
Read full entries with mtqg show <id>.
```

With a smaller budget, what is left out is said in its section:

<!-- mtqg:example repo=parser -->
```
$ mtqg context --max-tokens 380
# mtqg context — sample-parser (main)

This is the process record of this project. Read the following before you start working.
- Follow the rules listed under Rules
- Respect answered questions
- Do not decide open questions on your own; confirm them
- Use terms as defined in the glossary
- Record questions, decisions, findings, bugs, and todos with mtqg as they come up

## Attention
- Glossary term "block comment" has conflicting definitions (see mtqg glossary list)
- todo 6b0d549b6f "Skip line comments //" has concurrent changes (see mtqg review)
- 3 mtqg records are not committed

## Open todos (5)
- (2 older; see mtqg todo list)
- 1e27a1c08a Show error positions as line and column (claude-code, 11:06)
- 1818e81189 List the supported syntax in the README (yamada, 11:30)
- 2e44158bae Add test cases for comment handling (claude-code, 11:32)

## Open questions (2)
- 1012f037b6 Should nested block comments be supported? (awaiting confirmation, claude-code, 09:10)
- 2217beaddb Should error positions show both line and column? (unanswered, claude-code, 11:05)
- (latest answers left out; see mtqg show <id>)

## Open bugs (1)
- 7f3a2b1c09 Parser crashes on empty input (awaiting confirmation, yamada, 10:41)
- (latest replies left out; see mtqg show <id>)

## Recent records (newest first)
- (22 more; see mtqg log)

## Glossary (4)
- token
- block comment (2 definitions)
- lexing
- (definitions left out; see mtqg glossary list)

---
Read full entries with mtqg show <id>.
```

- Sections, in order: Attention, Rules, Open todos, Open questions, Open bugs,
  Recent records, Glossary. A section with nothing in it is omitted.
- The first line is the repository (the name of the directory that holds
  `.mtqg/`) and the branch (`git branch --show-current`; left out when there is
  none, as on a detached HEAD). Then come the instructions for the reader, then
  the sections, then a line on how to read more.
- **Attention** names what needs action: each glossary word with more than one
  definition, and each record with concurrent status changes (see
  [review](#review); for each, the first five, then how many more), and how many
  records are not committed (`git` cannot be run: this line is left out).
- **Rules** lists every rule, oldest first, each with its ID, its author and the
  time. The text is shown in full, however long: it is never cut and never left
  out, whatever the budget (see below).
- Open todos, questions and bugs are oldest first, each with its ID, its author
  and the time (`HH:MM` for today, a date for another day, in local time). A
  question or a bug says `unanswered` (`no replies`) or `awaiting confirmation`
  (it has answers or replies and is not closed) and, under it, the latest answer
  or reply with its author and the kind of author.
- Recent records are the newest records of every kind, newest first, as `log`
  shows them, with the kind and the ID; an answer or a reply ends with the
  question or bug it belongs to. Glossary lists every entry, with its ID.
- A text is its first line, cut to 100 characters with `...`, and control
  characters are replaced as in a list. **A rule's text is the one exception: it
  is shown in full, every line of it**, so that a convention is never read half
  cut.
- **Budget.** `--max-tokens N` sets it (default 2000; `0` means no limit). It is
  **an estimate from the number of characters, not a token count**: 4 ASCII
  characters count as 1 token, and every other character counts as 1. The
  header, the instructions, Attention, **Rules**, the headings, the last line and
  **the newest 3 questions and the newest 3 bugs** are never left out: what is
  open, undecided, or a standing convention must stay in view. When the text is
  over budget, mtqg leaves out, in
  this order and only as much as it takes: recent records (10, then 5, then 3,
  then none), the definitions of the glossary (leaving each word), the latest
  answers and replies, the oldest questions and bugs beyond those 3 of each (of
  both sections together, one at a time), the oldest todos (one at a time, down
  to none). If that is not enough, the text is printed as it is.
- What is left out is always said, in its section, with where to read it:
  `- (7 more; see mtqg log)`, `- (3 older; see mtqg todo list)`,
  `- (definitions left out; see mtqg glossary list)`,
  `- (latest answers left out; see mtqg show <id>)`. The heading of a section
  counts everything in it, not what was shown.
- `--json` gives the same content, structured, after the same reduction. Besides
  `command`, the fields are `repository`, `branch`, `attention`, `truncated` (true
  if anything was left out), `max_tokens` (`null` with `0`) and
  `estimated_tokens` (of the text form), and one object for each section:
  `rules`, `open_todos`, `open_questions`, `open_bugs`, `recent`, `glossary`,
  each with `total` and `records` (`rules` is never reduced, so its `total`
  and `records` always agree). In `open_questions` and `open_bugs` a record has
  `reply_count` and, if it was not left out, `latest_reply`. A glossary record has
  `definitions` (how many the word has) and, if the definitions were left out,
  no `id` and no `text`. `attention` holds `{"kind": "duplicate_word", "word": ...}`,
  `{"kind": "concurrent_status_change", "id": ..., "text": ...}` (the full ID and the
  full text) and `{"kind": "uncommitted", "count": N}`.

## format

Picks event lines out of any text and prints them as one table in time order,
with local times. It reads a file, or standard input (no argument, or `-`), and
does not need `.mtqg/`: it can run anywhere.

<!-- mtqg:example repo=parser -->
```
$ git show HEAD | mtqg format
2026-09-21 10:18  todo      6cad4a268d  Skip block comments /* */                          claude-code
2026-09-21 10:32  memo      81e74ef5e8  Policy: use English for all error messages         yamada
2026-09-21 11:05  question  2217beaddb  Should error positions show both line and column?  claude-code
2026-09-21 11:06  todo      1e27a1c08a  Show error positions as line and column            claude-code
```

- The columns are the local date and time, what the line did, the ID, the text
  and the author. For a line that creates a record, what it did is the kind:
  `memo`, `todo`, `question`, `answer`, `bug`, `reply`, `glossary` or `rule` (an answer or
  a reply starts its text with `(to <id>)`, and a glossary entry with its word and
  a colon). For the other lines it is `done` or `reopen` (a change of state),
  `edit` or `delete`. An `edit` shows the new text. A change of state or a delete
  shows the text of the record if the line that created it is in the same input,
  and nothing otherwise.
- Lines are in time order (`ts`, then `id`, a creation before the changes of the
  same record), whatever order they were in.
- A leading `+` or `-` (unified diff) is removed before parsing. A line that a
  diff shows unchanged (it starts with a space) is not part of the change, so it
  is not shown; it only lends its text to a change of state or a delete of the same
  record in the same input.
- A line that is not a JSON event is skipped silently, so whole `git show` /
  `git diff` output can be passed. There are no warnings.
- Lines from any source work: `git diff`, `git diff main...feature`,
  `cat .mtqg/journal.jsonl`, `grep ... .mtqg/journal.jsonl`, or a file given as
  an argument (including files in `.mtqg/archive/`).
- `+`/`-` marks are not shown, except that removed lines (`-`) are always marked.
  `--mark` shows the mark of every line. The marks are a column of their own in
  front of the others, and only when there is a mark to show.
- Text follows the rules of a list (first line only, cut only on a terminal,
  control characters replaced).

## archive

```
mtqg archive 2021-01-01..2024-09-18
mtqg archive 2021..2023
mtqg archive 202404..2024-09 -n
```

Moves the items whose last event falls in the range from `journal.jsonl` to
`.mtqg/archive/<start>..<end>.jsonl` (see [schema.md](schema.md#archive) for
which items move: finished todos, questions and bugs, memos, and anything that
was deleted; rules and glossary entries move only when deleted). No command
reads `archive/` by itself; archived IDs are simply not found. To read an
archive file, pass it to `mtqg format`.

The range is one argument, `<start>..<end>`:

1. Split at a run of two or more `.`. The `..` is required.
2. Remove every `-` from each side.
3. The number of digits gives the unit: 8 = day, 6 = month, 4 = year.
4. Both sides must have the same unit.
5. A month or year starts on its first day and ends on its last day. Both ends
   are inclusive. Dates are local dates.

```
2021-01-01..2024-09-18    → 2021-01-01..2024-09-18
20210101....20240918      → 2021-01-01..2024-09-18
2021-0101..202409-18      → 2021-01-01..2024-09-18
2021..2023                → 2021-01-01..2023-12-31
202404..2024-09           → 2024-04-01..2024-09-30
2023..2023                → 2023-01-01..2023-12-31
```

Errors (exit code 2, like any mistake in the command line): no `..`; a side that
is not 8, 6 or 4 digits; different units on the two sides; a date that does not
exist; start after end; any character other than digits, `-` and `.`.

<!-- mtqg:example repo=years binary -->
```
$ mtqg archive 2024-0101..202412-31
Range: 2024-01-01..2024-12-31
Archived: 2 memos, 1 todo, 1 question, 2 answers, 1 bug, 1 reply -> .mtqg/archive/2024-01-01..2024-12-31.jsonl
Skipped: 1 open todo, 1 open bug, 2 glossary entries
```

- The first line always shows how the range was read.
- `Archived:` counts what moved, by kind (a kind that has none is left out); a
  deleted record counts under its kind. `-> ` names the file. When nothing moves
  it says `Archived: nothing`, and no file is made.
- `Skipped:` counts what has its last event in the range but stays: todos,
  questions and bugs that are open, glossary entries, and rules. It is left out
  when there are none.
- The file name is always the normalized range. Archiving the same range again
  appends to the same file.
- `-n` (`--dry-run`) prints the same report without moving anything, and does not
  make `archive/`. The first line ends with `(dry run)`, so that a report of what
  would move is not taken for one of what moved.
- Relative dates ("2 years ago") are not accepted.
- `archive` writes no event, so it needs no author (`git config user.name` may be
  unset).
- With `--json` the report is one object (see [JSON output](#json-output)); it
  counts and does not list the records. To see what an archive holds, pass its
  file to `mtqg format`.

With `-n`, the report of the example above is printed and nothing moves
(`Archived:` says what would):

<!-- mtqg:example repo=years binary -->
```
$ mtqg archive 2024..2024 -n
Range: 2024-01-01..2024-12-31 (dry run)
Archived: 2 memos, 1 todo, 1 question, 2 answers, 1 bug, 1 reply -> .mtqg/archive/2024-01-01..2024-12-31.jsonl
Skipped: 1 open todo, 1 open bug, 2 glossary entries
```

When nothing moves, no file is made or changed. Here the range above is
archived again, and only what stays is left:

<!-- mtqg:example repo=years binary setup="mtqg archive 2024..2024" -->
```
$ mtqg archive 2024..2024
Range: 2024-01-01..2024-12-31
Archived: nothing
Skipped: 1 open todo, 1 open bug, 2 glossary entries
```

A thread is dated by its last event, so a question that was closed in 2024 and
answered in 2025 is left by `2024..2024`, and moved, with its answer, by
`2025..2025` (here a dry run with `--json`, which counts and does not list):

<!-- mtqg:example repo=years binary -->
```
$ mtqg archive --json 2025..2025 -n
{
  "command": "archive",
  "range": {
    "start": "2025-01-01",
    "end": "2025-12-31"
  },
  "file": ".mtqg/archive/2025-01-01..2025-12-31.jsonl",
  "dry_run": true,
  "archived": {
    "memos": 1,
    "todos": 0,
    "questions": 1,
    "answers": 1,
    "bugs": 0,
    "replies": 0,
    "glossary_entries": 0,
    "rules": 0,
    "records": 3
  },
  "skipped": {
    "open_todos": 1,
    "open_questions": 0,
    "open_bugs": 0,
    "glossary_entries": 0,
    "rules": 0,
    "records": 1
  }
}
```

An archive file is read with `mtqg format`:

<!-- mtqg:example repo=years binary setup="mtqg archive 2024..2024" -->
```
$ mtqg format .mtqg/archive/2024-01-01..2024-12-31.jsonl
2024-01-15 09:10  memo      cafa0631b6  Policy: use English for all error messages                             yamada
2024-02-05 11:00  todo      546c4b4711  Add tests for comment handling                                         yamada
2024-03-11 15:30  done      546c4b4711  Add tests for comment handling                                         yamada
2024-04-02 09:00  question  1a1cc7a6ab  Should nested block comments be supported?                             yamada
2024-04-02 09:20  answer    64f59967e3  (to 1a1cc7a6ab) Supporting them is preferable, since C code uses them  claude-code
2024-04-03 10:15  answer    869916a85b  (to 1a1cc7a6ab) Not in the first version. Revisit if there is demand   yamada
2024-04-05 16:00  done      1a1cc7a6ab  Should nested block comments be supported?                             yamada
2024-06-02 14:00  memo      2e8a4b4d62  Tokens carry their line and column                                     yamada
2024-07-10 09:30  bug       74ede9841a  Parser crashes on empty input                                          yamada
2024-07-11 10:00  reply     9383a1dad2  (to 74ede9841a) Fixed by checking for the end of input first           claude-code
2024-07-12 17:45  done      74ede9841a  Parser crashes on empty input                                          yamada
```

To bring a range back, append its file to `journal.jsonl` and delete it (see
[schema.md](schema.md#archive)). There is no `unarchive` command.

## version

`mtqg version` prints the mtqg version and the format version declared in
`.mtqg/version`, kept apart:

<!-- mtqg:example skip="the version depends on how mtqg was built" -->
```
$ mtqg version
mtqg v0.1.0
Repository format version: 0 (this mtqg supports up to 0)
```

- The mtqg version is the one set when it was built, else the module version of
  the build (a tag such as `v0.1.0` for `go install ...@v0.1.0`; a version made
  of the commit time and hash for `go build`), else `dev`.
- Outside a repository, or where there is no `.mtqg/`, the second line reads
  `Repository format version: unknown (no .mtqg/ found)`.

mtqg refuses to read or write a repository whose format
version is newer than it knows, and asks you to update mtqg.

The format version is `0` (unstable) and there is no command to raise it yet.
One will be added when a format `1` or later exists (see
[schema.md](schema.md#versioning)).

## Shell completion

`mtqg completion <shell>` prints the completion script of a shell: `bash`,
`zsh`, `fish` or `powershell`. Any other word, or none, is a usage error that
names the four. Put the script where the shell looks for completions:

```
mtqg completion bash > ~/.local/share/bash-completion/completions/mtqg
mtqg completion zsh > "${fpath[1]}/_mtqg"
mtqg completion fish > ~/.config/fish/completions/mtqg.fish
mtqg completion powershell >> $PROFILE
```

The script is a short piece of fixed text that does not list the commands. Each
time TAB is pressed it asks `mtqg candidates` what can come next, so what is
offered never disagrees with the mtqg that is installed, and it includes the IDs
of the records of the repository the line is typed in.

What is completed:

| Where | Candidates |
|---|---|
| The first word | The kinds (`memo`, `todo`, `qa`, `bug`, `glossary`, `rule`; not their one-letter forms) and the commands that have no kind |
| After a kind | Its verbs |
| A word that starts with `-` | The options the command takes that are not on the line yet, and the global options |
| After `--kind` | `memo`, `todo`, `qa`, `bug`, `glossary`, `rule` |
| The ID of `todo done`, `qa done`, `bug done` | The open ones of that kind |
| The ID of `todo reopen`, `qa reopen`, `bug reopen` | The done ones of that kind |
| The ID of `show`, `edit`, `delete` | Every record in view, newest first |
| The first word of `qa add`, `bug add` | The questions (the bugs), open or done: it may be the ID of the one to answer (reply to) |
| After `completion` | The four shells |

Nothing else is completed: the text of a record, the words of `search`, the
range of `archive`, the value of `--limit`. Where a path is wanted (`-C`,
`format`), the completion of files that the shell has is left to work. The form
with an equals sign (`--kind=todo`) is not completed; write `--kind <TAB>`.

### candidates

`mtqg candidates [--word=<partial>] -- <word>...` is what the scripts call. It
is not meant to be typed, but it is a command like the others (it is in
`mtqg help`), and a program that wants to know what can come next on a line can
use it too.

- The words after `--` are the command line typed so far, without `mtqg`
  itself, as the shell split it. `--word` is the word being typed, empty when
  the cursor is at the start of a new word. It is not one of the words because
  shells drop an empty argument, and `--word=` is never empty.
- The output is one candidate per line: the value, and when there is something
  to say about it, a tab and a description (a summary for a command or a verb; the
  text of a record). Only the candidates that start with `--word` are listed, so
  the four shells offer the same, whatever they do with the list. There is
  no limit on the number.
- An ID is given as its first 10 digits, or as the full 32 digits when what was
  typed is longer than 10 digits or when the 10 digits fit more than one record in
  view, so that what is inserted always names one record.
- A description is the text on one line, with control characters replaced as in
  a list, cut to 60 columns of the display. It is cut with `--json` too, because
  it is for the menu of a shell.
- **Whatever the words say or the repository holds, the exit code is 0 and
  nothing goes to standard error**, not even a warning: no `.mtqg/`, a format
  from the future, lines that cannot be read, conflict markers — what can be
  offered is offered (the commands and options need no repository), because a
  warning printed at every TAB would break the line that is being typed. A
  mistake in the options of `candidates` itself is a usage error, as anywhere.
- A `-C` among the words is read: `mtqg -C ../other todo done <TAB>` offers the
  IDs of the repository in `../other`. The words are data: `candidates` does not
  act on them the way the other commands do.

What the shell offers when TAB is pressed after `mtqg t`, `mtqg todo re`,
`mtqg todo done ` and `mtqg log --kind ` (the value and the description are
separated by a tab):

<!-- mtqg:example repo=parser -->
```
$ mtqg candidates --word=t --
todo
$ mtqg candidates --word=re -- todo
reopen	Mark a todo as open again
$ mtqg candidates --word= -- todo done
2e44158bae	Add test cases for comment handling
1818e81189	List the supported syntax in the README
1e27a1c08a	Show error positions as line and column
6513270e26	Ignore // inside string literals
6cad4a268d	Skip block comments /* */
$ mtqg candidates --word= -- log --kind
memo
todo
qa
bug
glossary
```

Two records whose first 10 digits are the same are given in full:

<!-- mtqg:example repo=ambiguous -->
```
$ mtqg candidates --word=7043 -- show
70430f77ff4b475185d5cae12dff1a17	Skip block comments /* */
70430f77ff91c2e04a8b33f1d7e6a025	Parser now skips // at line end
```

## Authors

| Item | Value |
|---|---|
| author name | `MTQG_AUTHOR_NAME`, else `git config user.name` |
| author kind | `MTQG_AUTHOR_KIND` (`human` or `ai`), else `human` |
| terminal | `MTQG_TTY`, else the terminal that standard input, output or error is connected to; none if there is none |
| editor | `$EDITOR` |

- An AI agent that records through the command line says so with the two
  variables: `MTQG_AUTHOR_KIND=ai MTQG_AUTHOR_NAME=claude-code`. With `ai` the
  name is required, so an AI is never recorded under the name from `git config`.
- `git config user.name` is read for the repository (`git -C <root> config
  user.name`), so a setting of that repository applies. If no name comes from
  either place, mtqg stops and says how to set one. A `MTQG_AUTHOR_KIND` other
  than `human` or `ai` is an error.
- The terminal is written to every line as `tty`, 8 hex digits of a hash (see
  [undo](#undo)). It is left out where there is no terminal to identify.

The variables are not a configuration file. There is no configuration file.
