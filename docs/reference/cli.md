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

- A kind can be abbreviated to one letter: `mtqg t add ...` is `mtqg todo add ...`.
  The verb is always required.
- Using a verb on a record of the wrong kind (for example `mtqg qa done` with a
  todo ID) is an error that names the right command.
- `qa` and `bug` work the same way. See
  [Questions, answers, bugs and replies](#questions-answers-bugs-and-replies).

## Other commands

| Command | Description |
|---|---|
| `mtqg edit <id> <text>` | Replace the text of a record. For glossary, the definition; the word cannot change |
| `mtqg delete <id>` | Hide a record. Deleting a question or a bug also hides its answers or replies |
| `mtqg undo` | Remove the last line written from this terminal |
| `mtqg status` | Summary of open items and uncommitted records |
| `mtqg log [--limit N] [--kind K]` | Records of all kinds, newest first |
| `mtqg show <id>` | One record with its full text and history |
| `mtqg search <text>` | Substring search in record text |
| `mtqg review` | Concurrent status changes and duplicate glossary definitions |
| `mtqg context [--max-tokens N]` | Summary for AI agents |
| `mtqg format [file]` | Pretty-print event lines found in any text |
| `mtqg archive <start>..<end> [-n]` | Move finished items of a date range out of view |
| `mtqg init` | Create `.mtqg/` |
| `mtqg version` | Show the mtqg version and the repository's format version |
| `mtqg help` | List the commands (`-h` and `--help` do the same) |

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
  and `--full-id` change nothing.
- The output goes to standard output and, on success, nothing goes to standard
  error except warnings (below). Records that are hidden do not appear.

A record is an object:

| Field | Meaning |
|---|---|
| `id` | The full ID |
| `kind` | `memo`, `todo`, `question`, `answer`, `bug`, `reply` or `glossary` |
| `word` | The word (glossary only) |
| `text` | The full text (the definition, for glossary) |
| `re` | The ID of the question or bug this answers or replies to (answer and reply only) |
| `status` | `open` or `done` (todo, question and bug only) |
| `author` | `{"kind": "human" or "ai", "name": "..."}` |
| `created` | The time of the first event |
| `updated` | The time of the last event |

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
| `memo add`, `todo add`, `qa add`, `bug add`, `glossary add` | `record`: the record that was written |
| `todo done`, `todo reopen`, `qa done`, ... | `record`, and `changed`: `false` if the record was in that state already and nothing was written |
| `memo list` | `records`, `count` |
| `todo list` | `records` (`--all`: including done), `open`, `done` (counts of all in view, whatever `--all` says) |
| `qa list`, `bug list` | as `todo list`; each record has `replies` |
| `glossary list` | `records`, `entries` (their number), `duplicate_words` |
| `log` | `records` (newest first), `shown`, `total` |
| `show` | `record`, and `events`: what happened to it, oldest first, as lines of `journal.jsonl` in the form of [schema.md](schema.md) (for a question or a bug this includes the `create` of each reply) |
| `status` | `open_todos`, `open_questions`, `questions_awaiting_confirmation`, `open_bugs`, `bugs_awaiting_confirmation`, `glossary_entries`, `duplicate_words`, `uncommitted_records` (`null` if git cannot be run) |
| `init` | `root`: where `.mtqg/` was created |
| `version` | `mtqg`: the version; `format`: `{"repository": N or null, "supported": N}` (`null` where there is no `.mtqg/`) |
| `help`, or `-h` on a command | `kinds`: `{"name", "short"}`; `commands`: `{"command", "usage", "summary", "available"}`. `available` is `false` for a command that is known and not yet built |
| `context` | See [context](#context) |

**Errors** are one line of JSON on **standard error**, and standard output stays
empty. The exit code is the same as without `--json`.

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
| `unknown` (anything else) | 1 | |

**Warnings** (a line that was skipped, a conflict marker, git that cannot be run)
are also one line each on standard error, and do not change the exit code:

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

```
$ mtqg t add Add tests for comment handling
No .mtqg/ found. Run `mtqg init` (will be created at /home/me/sample-parser)
```

No command creates `.mtqg/` except `mtqg init`.

## init

```
$ mtqg init
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
- No arguments at all (`mtqg m add`, `mtqg t add`, `mtqg q add`, `mtqg b add`)
  opens `$EDITOR` on an empty file, and what is saved is the text (trailing line
  breaks dropped). `$EDITOR` may hold arguments and quotes (`code --wait`); it
  is not run through a shell. If `$EDITOR` is not set, mtqg stops and says so.
- Only that case opens the editor. A part that is missing is a mistake in the
  command line, and nothing is written: `mtqg q add <question-id>` (no answer),
  `mtqg b add <bug-id>` (no reply), `mtqg g add` (no word) and
  `mtqg g add <word>` (no definition). The text of an answer, a reply or a
  definition can be `-`, to read it from standard input.
- A text can have several lines (from standard input or the editor). `list`
  shows the first line only.
- A text that is empty, or only white space, is an error
  (`Aborting: the text is empty`). Nothing is written.
- Output is only the new record's ID:

```
$ mtqg t add Skip block comments
6cad4a268d
```

## done, reopen

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
- The ID with no text after it is an error, and nothing is written.

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

```
$ mtqg delete 1012f037b6
Deleted: "Should nested block comments be supported?"
2 answers are also hidden (claude-code, yamada)
They remain in git history
```

Both append an event. Nothing is removed from the file or from git history.

## undo

Removes the **last line written from the current terminal**, for a mistake just
made (for example an answer added as a new question because the question ID was
left out).

```
$ mtqg q add Not in the first version. Revisit if there is demand
301850c5a3
$ mtqg undo
Undone: qa add "Not in the first version. Revisit if there is demand" (301850c5a3)
```

- Target: the last line in `journal.jsonl` (by `ts`) with this author and this
  terminal's `tty` value. When no terminal can be identified, lines without
  `tty` count as the same terminal.
- One step only; `undo` does not repeat.
- It does not check whether the line was committed. For something already
  shared, use `delete`: a removed line that already reached another branch can
  come back with the next merge.

## Reading

### status

```
$ mtqg status
Open todos          5
Open questions      2  (1 awaiting confirmation)
Open bugs           1  (1 awaiting confirmation)
Glossary            4  (1 with duplicate definitions)

Uncommitted records 3
```

- Each line is a count. `Open questions` counts the questions that are not
  closed, and `awaiting confirmation` those of them that have an answer. `Open
  bugs` counts the bugs that are not closed the same way (`awaiting
  confirmation` are those that have a reply). `Glossary` counts the entries, and `with duplicate definitions` the words that
  are defined more than once (the same word, character for character).
  `Uncommitted records` is the number of records that have
  at least one line in `journal.jsonl` that is not in the last commit (`HEAD`):
  records created or changed since then. A record counts once however many lines
  it has. Lines that are staged but not committed count as uncommitted. With no
  commit yet, or when `.mtqg/journal.jsonl` is not tracked, every record counts.
  If git cannot be run, the line shows `unknown`.

### list

```
$ mtqg todo list
6cad4a268d  Skip block comments /* */                claude-code  10:18
6513270e26  Ignore // inside string literals         yamada       10:52
1e27a1c08a  Show error positions as line and column  claude-code  11:06
3 open (show done: --all)

$ mtqg todo list --all
6cad4a268d  Skip block comments /* */                claude-code  10:18  done
6513270e26  Ignore // inside string literals         yamada       10:52
1e27a1c08a  Show error positions as line and column  claude-code  11:06
2 open, 1 done

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

### show

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
  `reply` or `glossary`), the ID and, for a todo, a question or a bug, its
  state. The second line
  says who wrote it and when, with the kind of author.
- The text is shown in full, with every line of it, however the output is
  read. Control characters are replaced as in a list. A glossary entry shows
  `Word: <word>` before its definition. An answer shows the question it belongs
  to, and a reply the bug.
- A question lists its answers, and a bug its replies (`Replies (2)`), oldest
  first, each with its author and time.
- `Events` lists what happened to the record, oldest first, with the full local
  date and time: its own events (`create`, `status` as `open -> done`, `edit`,
  `delete`) and, for a question or a bug, the creation of each answer or reply.

### log

```
$ mtqg log --limit 6
11:32  todo      2e44158bae  Add test cases for comment handling                              claude-code
11:30  todo      1818e81189  List the supported syntax in the README                          yamada
11:24  glossary  f28c105d1f  lexing: Reading source and turning it into a sequence of tokens  claude-code
11:06  todo      1e27a1c08a  Show error positions as line and column                          claude-code
11:05  question  2217beaddb  Should error positions show both line and column?                claude-code
10:52  todo      6513270e26  Ignore // inside string literals                                 yamada
6 of 20 records (--limit 0 for all)

$ mtqg log --kind qa
11:05       question  2217beaddb  Should error positions show both line and column?                     claude-code
09:41       answer    95e761d177  (to 1012f037b6) Not in the first version. Revisit if there is demand  yamada
09:15       answer    ae2eb1547f  (to 1012f037b6) Supporting them is generally preferable               claude-code
09:10       question  1012f037b6  Should nested block comments be supported?                            claude-code
2026-09-19  answer    d4c2a1e3f5  (to c3b1f0d2e4) MIT is the simplest choice                            claude-code
2026-09-19  question  c3b1f0d2e4  Which license should the parser use?                                  yamada       done
6 records

$ mtqg log --kind bug
10:45       reply  3d8e4a0b12  (to 7f3a2b1c09) Reproduced on macOS too                claude-code
10:41       bug    7f3a2b1c09  Parser crashes on empty input                          yamada
2026-09-19  reply  d4e5f6a7b8  (to b2c3d4e5f6) Fixed by treating a tab as one column  claude-code
2026-09-19  bug    b2c3d4e5f6  Linter crashes on tab characters                       yamada       done
4 records
```

- Every record that is not hidden, of every kind, one line each, **newest
  first**: the time, the kind (`memo`, `todo`, `question`, `answer`, `bug`,
  `reply`, `glossary`), the ID, the text and the author. A glossary entry shows
  its word, a colon and its definition. An answer or a reply starts with
  `(to <id>)`, the question or bug it belongs to. A finished todo, question or
  bug ends with `done`.
- The time is `HH:MM` for today and `YYYY-MM-DD` for any other day, the time the
  record was created. The text follows the rules of a list (first line only, cut
  only on a terminal, control characters replaced).
- `--limit N` shows the newest `N` records; the default is 20, and `0` shows all.
  `--kind K` shows one kind only: `memo`, `todo`, `qa` (questions and answers),
  `bug` (bugs and replies) or `glossary`, or the letter. Both may be written `--limit=N`. A value that is not
  a whole number of 0 or more, or a kind that does not exist, is a mistake in the
  command line.
- The last line counts the records: `N records`, or `N of M records (--limit 0
  for all)` when some are left out.

### review

```
$ mtqg review
Concurrent status changes (1)
  6b0d549b6f "Skip line comments //"
    10:15  claude-code  open -> done
    14:30  yamada       open -> done

Duplicate glossary definitions (1)
  block comment
    f29d0da995  yamada       A comment enclosed in /* and */
    0cb1e29c65  claude-code  A comment that can span multiple lines
```

A concurrent status change is two or more status changes from the same `from`
state. mtqg does not decide which one is right.

### context

What an AI agent reads when a session starts. It passes the process so far:
what is open, what was answered, which terms are agreed. It does not contain the
project description, the current specification or build instructions; those
belong to the README and the agent's instruction files.

```
$ mtqg context
# mtqg context — sample-parser (main)

This is the process record of this project. Read the following before you start working.
- Respect what has been decided (answered questions, memos stating a policy)
- Do not decide open questions on your own; confirm them
- Use terms as defined in the glossary
- Record questions, decisions, findings, bugs, and todos with mtqg as they come up

## Attention
- Glossary term "block comment" has conflicting definitions (see mtqg glossary list)
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
- 10:45  claude-code  reply     3d8e4a0b12  Reproduced on macOS too (to 7f3a2b1c09)
- 10:41  yamada       bug       7f3a2b1c09  Parser crashes on empty input
- 10:32  yamada       memo      81e74ef5e8  Policy: use English for all error messages
- 10:18  claude-code  todo      6cad4a268d  Skip block comments /* */
- (10 more; see mtqg log)

## Glossary (4)
- 5b7e2c9a41 token: The smallest unit produced by lexing
- f29d0da995 block comment: A comment enclosed in /* and */
- 0cb1e29c65 block comment: A comment that can span multiple lines
- f28c105d1f lexing: Reading source and turning it into a sequence of tokens

---
Read full entries with mtqg show <id>.
```

With a smaller budget, what is left out is said in its section:

```
$ mtqg context --max-tokens 380
# mtqg context — sample-parser (main)

This is the process record of this project. Read the following before you start working.
- Respect what has been decided (answered questions, memos stating a policy)
- Do not decide open questions on your own; confirm them
- Use terms as defined in the glossary
- Record questions, decisions, findings, bugs, and todos with mtqg as they come up

## Attention
- Glossary term "block comment" has conflicting definitions (see mtqg glossary list)
- 3 mtqg records are not committed

## Open todos (5)
- 6cad4a268d Skip block comments /* */ (claude-code, 10:18)
- 6513270e26 Ignore // inside string literals (yamada, 10:52)
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
- (20 more; see mtqg log)

## Glossary (4)
- token
- block comment (2 definitions)
- lexing
- (definitions left out; see mtqg glossary list)

---
Read full entries with mtqg show <id>.
```

- Sections, in order: Attention, Open todos, Open questions, Open bugs, Recent
  records, Glossary. A section with nothing in it is omitted.
- The first line is the repository (the name of the directory that holds
  `.mtqg/`) and the branch (`git branch --show-current`; left out when there is
  none, as on a detached HEAD). Then come the instructions for the reader, then
  the sections, then a line on how to read more.
- **Attention** names what needs action: each glossary word with more than one
  definition (the first five; then how many more) and how many records are not
  committed (`git` cannot be run: this line is left out).
- Open todos, questions and bugs are oldest first, each with its ID, its author
  and the time (`HH:MM` for today, a date for another day, in local time). A
  question or a bug says `unanswered` (`no replies`) or `awaiting confirmation`
  (it has answers or replies and is not closed) and, under it, the latest answer
  or reply with its author and the kind of author.
- Recent records are the newest records of every kind, newest first, as `log`
  shows them, with the kind and the ID; an answer or a reply ends with the
  question or bug it belongs to. Glossary lists every entry, with its ID.
- A text is its first line, cut to 100 characters with `...`, and control
  characters are replaced as in a list.
- **Budget.** `--max-tokens N` sets it (default 2000; `0` means no limit). It is
  **an estimate from the number of characters, not a token count**: 4 ASCII
  characters count as 1 token, and every other character counts as 1. The
  header, the instructions, Attention, the headings, the last line and **the
  newest 3 questions and the newest 3 bugs** are never left out: what is open and
  undecided must stay in view. When the text is over budget, mtqg leaves out, in
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
  `open_todos`, `open_questions`, `open_bugs`, `recent`, `glossary`, each with
  `total` and `records`. In `open_questions` and `open_bugs` a record has
  `reply_count` and, if it was not left out, `latest_reply`. A glossary record has
  `definitions` (how many the word has) and, if the definitions were left out,
  no `id` and no `text`. `attention` holds `{"kind": "duplicate_word", "word": ...}`
  and `{"kind": "uncommitted", "count": N}`.
- Concurrent status changes (`mtqg review`) are not in Attention yet.

## format

Picks event lines out of any text and prints them as one table in time order,
with local times.

```
$ git show 3f9a1c0 | mtqg format
10:18  todo    6cad4a268d  Skip block comments /* */                            claude-code
10:32  memo    81e74ef5e8  Policy: use English for all error messages           yamada
11:05  qa      2217beaddb  Should error positions show both line and column?    claude-code
11:32  todo    2e44158bae  Add test cases for comment handling                  claude-code
```

- A leading `+` or `-` (unified diff) is removed before parsing.
- Lines that are not JSON events are skipped silently, so whole `git show` /
  `git diff` output can be passed.
- Lines from any source work: `git diff`, `git diff main...feature`,
  `cat .mtqg/journal.jsonl`, `grep ... .mtqg/journal.jsonl`, or a file given as
  an argument (including files in `.mtqg/archive/`).
- `+`/`-` marks are not shown, except that removed lines are always marked.
  `--mark` shows marks on every line.

## archive

```
$ mtqg archive 2021-01-01..2024-09-18
$ mtqg archive 2021..2023
$ mtqg archive 202404..2024-09 -n
```

Moves finished items whose last event falls in the range from `journal.jsonl`
to `.mtqg/archive/<start>..<end>.jsonl` (see [schema.md](schema.md#archive)
for what is archived). No command reads `archive/` by itself; archived IDs are
simply not found. To read an archive file, pass it to `mtqg format`.

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

Errors: no `..`; a side that is not 8, 6 or 4 digits; different units on the
two sides; a date that does not exist; start after end; any character other
than digits, `-` and `.`.

```
$ mtqg archive 2021-0101..202409-18
Range: 2021-01-01..2024-09-18
Archived: 412 memos, 138 todos, 57 questions, 12 bugs -> .mtqg/archive/2021-01-01..2024-09-18.jsonl
Skipped: 3 open todos, 1 open question, 2 open bugs, 24 glossary entries
```

- The first line always shows how the range was read.
- The file name is always the normalized range. Archiving the same range again
  appends to the same file.
- `-n` (`--dry-run`) prints the same report without moving anything.
- Relative dates ("2 years ago") are not accepted.

## version

`mtqg version` prints the mtqg version and the format version declared in
`.mtqg/version`, kept apart:

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

## Authors

| Item | Value |
|---|---|
| author name | `MTQG_AUTHOR_NAME`, else `git config user.name` |
| author kind | `MTQG_AUTHOR_KIND` (`human` or `ai`), else `human` |
| editor | `$EDITOR` |

- An AI agent that records through the command line says so with the two
  variables: `MTQG_AUTHOR_KIND=ai MTQG_AUTHOR_NAME=claude-code`. With `ai` the
  name is required, so an AI is never recorded under the name from `git config`.
- `git config user.name` is read for the repository (`git -C <root> config
  user.name`), so a setting of that repository applies. If no name comes from
  either place, mtqg stops and says how to set one. A `MTQG_AUTHOR_KIND` other
  than `human` or `ai` is an error.

The variables are not a configuration file. There is no configuration file.
