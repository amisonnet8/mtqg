# mtqg command reference

*[日本語](cli_ja.md) | **English***

```
mtqg <kind> <verb> [args]
mtqg <command> [args]
```

mtqg's own messages are in English. Record contents are shown as written, in
any language. The storage format is described in [schema.md](schema.md).

## Kinds and verbs

| | memo (`m`) | todo (`t`) | qa (`q`) | glossary (`g`) |
|---|---|---|---|---|
| `add` | `add <text>` | `add <text>` | `add <question>`<br>`add <question-id> <answer>` | `add <word> <definition>` |
| `done` | — | `done <id>` | `done <question-id>` | — |
| `reopen` | — | `reopen <id>` | `reopen <question-id>` | — |
| `list` | all | open (`--all`: all) | open (`--all`: all) | all |

- A kind can be abbreviated to one letter: `mtqg t add ...` is `mtqg todo add ...`.
  The verb is always required.
- Using a verb on a record of the wrong kind (for example `mtqg qa done` with a
  todo ID) is an error that names the right command.

## Other commands

| Command | Description |
|---|---|
| `mtqg edit <id> <text>` | Replace the text of a record. For glossary, the definition; the word cannot change |
| `mtqg delete <id>` | Hide a record. Deleting a question also hides its answers |
| `mtqg undo` | Remove the last line written from this terminal |
| `mtqg status` | Summary of open items and uncommitted records |
| `mtqg log [--limit N] [--kind K]` | All kinds in time order |
| `mtqg show <id>` | One record with its full history |
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
| `--json` | Machine-readable output (all commands) |
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
mtqg g add token The smallest unit produced by lexing
```

- The remaining arguments are joined with spaces. Quotes are not needed, except
  around shell special characters (`#` `*` `(` `)` `&` `|` `<` `>`).
- For glossary, the first argument is the word and the rest is the definition.
- `-` instead of the text reads it from standard input, to its end. Trailing
  line breaks are dropped: `git log -1 --format=%s | mtqg m add -`
- No text opens `$EDITOR` on an empty file, and what is saved is the text
  (trailing line breaks dropped). `$EDITOR` may hold arguments and quotes
  (`code --wait`); it is not run through a shell. If `$EDITOR` is not set, mtqg
  stops and says so.
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
- Only todos and questions have a state. For the ID of any other record, mtqg
  stops and says what it is. If another command is the right one, it names it:
  `81e74ef5e8 is a memo, not a todo`, and
  `2217beaddb is a question, not a todo; use `mtqg qa done 2217beaddb``.

## Questions and answers

- `mtqg q add <text>` adds a question. `mtqg q add <question-id> <text>` adds an
  answer to that question. An answer has its own ID.
- Answering and closing are separate: `mtqg q done <question-id>` closes the
  question. Any number of answers can be added; none replaces another.
- Only questions can have answers. Answers cannot be answered.

A question is in one of four states:

| State | Answers | Closed | Shown by `qa list` |
|---|---|---|---|
| unanswered | 0 | no | yes |
| awaiting confirmation | 1 or more | no | yes, with the number of answers |
| answered | 1 or more | yes | with `--all` |
| closed without answer | 0 | yes | with `--all` |

| ID given | Allowed |
|---|---|
| question | `q add` (answer), `q done`, `q reopen`, `edit`, `delete` (hides its answers too) |
| answer | `edit`, `delete` (that answer only; the question's state is unchanged) |

## IDs

- Every record has a full ID: 32 lowercase hex digits (a UUIDv4 without
  hyphens), e.g. `81e74ef5e8e24d949ed904759531985d`.
- mtqg shows the **first 10 digits** (`81e74ef5e8`). `--full-id` shows full IDs.
- Any unique prefix is accepted as input, like git commit hashes: `81e74ef5e8`,
  or `81e7` if unique.
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
Glossary            4  (1 with duplicate definitions)
Conflicts           1  -> mtqg review

Uncommitted records 3
```

- Each line is a count. `Uncommitted records` is the number of records that have
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
2217beaddb  Should error positions show both line and column?   claude-code  11:05  unanswered
1012f037b6  Should nested block comments be supported?          claude-code  09:10  2 answers, awaiting confirmation
           └ Not in the first version. Revisit if there is ...  yamada       09:41
2 open (show all: --all)
```

`qa list` shows the latest answer and the number of answers.

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
qa  1012f037b6  open
  Q Should nested block comments be supported?                     claude-code  09:10

  Answers (2)
  ae2eb1547f  Supporting them is generally preferable                claude-code  09:15  ai
  95e761d177  Not in the first version. Revisit if there is demand   yamada       09:41  human

  Events
    09:10  create  claude-code
    09:15  create  claude-code  → ae2eb1547f
    09:41  create  yamada       → 95e761d177
```

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
- Record questions, decisions, findings, and todos with mtqg as they come up

## Attention
- Glossary term "block comment" has conflicting definitions (mtqg review)
- 3 mtqg records are not committed

## Open todos (5)
- 6cad4a268d  Skip block comments /* */ (claude-code, 10:18)
- 6513270e26 Ignore // inside string literals (yamada, 10:52)
- 1e27a1c08a Show error positions as line and column (claude-code, 11:06)
- 1818e81189 List the supported syntax in the README (yamada, 11:30)
- 2e44158bae Add test cases for comment handling (claude-code, 11:32)

## Open questions (2)
- 2217beaddb Should error positions show both line and column? (unanswered, claude-code, 11:05)
- 1012f037b6  Should nested block comments be supported? (awaiting confirmation, 09:10)
    └ Not in the first version. Revisit if there is demand (yamada, human)

## Recent records (10, newest first)
- 11:24 claude-code glossary lexing: Reading source and turning it into a sequence of tokens
- 10:32 yamada      memo     Policy: use English for all error messages
- 09:41 yamada      answer   Not in the first version. Revisit if there is demand (to 1012f037b6)
- ... (7 more; see mtqg log)

## Glossary (4)
- token: The smallest unit produced by lexing
- lexing: Reading source and turning it into a sequence of tokens
- block comment: A comment enclosed in /* and */ (2 definitions)
- lexer: The implementation that does lexing (the Lexer struct)

---
Read full entries with mtqg show <id>.
```

- Sections, in order: Attention, Open todos, Open questions, Recent records,
  Glossary. Empty sections are omitted.
- Every item carries its ID, author and time. Text is cut to one line.
- Default budget: about 2000 tokens (`--max-tokens N` to change). Over budget,
  mtqg drops, in order: recent records (10 → 5 → 3 → 0), glossary definitions
  (then glossary words), answer texts, older questions. Open todos are not
  dropped; if still over budget, the list is shortened and ends with `(N more)`.
- Omissions always say how many were left out and where to read them.
- With `--json`: one array per section, whether it was cut, and the total counts.

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
Archived: 412 memos, 138 todos, 57 questions -> .mtqg/archive/2021-01-01..2024-09-18.jsonl
Skipped: 3 open todos, 1 open question, 24 glossary entries
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
