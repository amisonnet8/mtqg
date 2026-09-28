# mtqg examples

*[日本語](examples_ja.md) | **English***

For the full command list, see the [command reference](../reference/cli.md); for the data format, see [journal format](../reference/schema.md). This starts from nothing and works through every kind of record, then looks at two larger, real repositories.

## What mtqg is for

Working on something produces things that never make it into the code or a commit: things noticed along the way, things to do, questions and their answers, bugs and the back-and-forth about them, terms the team agreed on. Until now these scattered across chat history and note apps, and got lost. mtqg appends them to one file, `.mtqg/journal.jsonl`, in the same git repository as the code, committed and shared the same way.

## Getting started

Install with `go install github.com/amisonnet8/mtqg/cmd/mtqg@latest`. In a project's repository, run `mtqg init` once.

<!-- mtqg:example repo=none path=/home/me/sample-parser -->
```
$ mtqg init
Created .mtqg/ in /home/me/sample-parser
Commit it to share the records.
```

`.mtqg/` is created. Commit it, as the message says (`mtqg init` itself does not commit).

## Recording what happens

Things noticed while working, or worth looking back on, go in a `memo`. No verb, just `add`; the output is only the ID of the record that was made, so nothing gets in the way of writing:

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg m add Tokens carry their line and column
b6868dd1af
```

A `todo` is the same, recorded before you start it and closed with `done`; `t list` shows the open ones (`--all` for every one, see [list](../reference/cli.md#list) for what that looks like on a project with a few already in it):

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg t add Support C syntax
f7443f148b
```

Something you are unsure about goes in a `qa`; a bug you found, and the exchange about it, goes in a `bug`. Both have the same shape (a body, and replies to it): following the body's ID with more text makes that text an answer or a reply. From here on, this looks in on the same parser project a while later (the fixture that ships with this repository):

<!-- mtqg:example repo=parser ids=any -->
```
$ mtqg show 2217beaddb
question  2217beaddb  open
by claude-code (ai), 2026-09-21 11:05

  Should error positions show both line and column?

Events
  2026-09-21 11:05  create  claude-code (ai)
$ mtqg q add 2217beaddb Yes, callers usually want both
dfb354fcde
$ mtqg q done 2217beaddb
Done: 2217beaddb  Should error positions show both line and column?
```

`done` is what closes a question. Answering and closing are separate: a question with an answer still shows as "awaiting confirmation" until it is closed. `bug` works the same way:

<!-- mtqg:example repo=parser ids=any -->
```
$ mtqg b add 7f3a2b1c09 Fixed by returning an error instead of panicking
4b57df5ebc
$ mtqg b done 7f3a2b1c09
Done: 7f3a2b1c09  Parser crashes on empty input
```

An agreed term goes in the `glossary`, so the same word does not get used with two different meanings:

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg g add token The smallest unit produced by lexing
f40d750976
```

A convention that can be followed just by reading it goes in a `rule`. It has the same shape as a memo, but it is never moved by `archive`, and `context` never drops or truncates it, whatever the budget:

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg r add Use English for all error messages
ad8c4f3cb3
```

## Reading it back

Once a few records have piled up, here is how to read them: `log` lists every kind, newest first, back in the same parser project:

<!-- mtqg:example repo=parser ids=any -->
```
$ mtqg log --limit 5
11:32  todo      2e44158bae  Add test cases for comment handling                              claude-code
11:30  todo      1818e81189  List the supported syntax in the README                          yamada
11:24  glossary  f28c105d1f  lexing: Reading source and turning it into a sequence of tokens  claude-code
11:06  todo      1e27a1c08a  Show error positions as line and column                          claude-code
11:05  question  2217beaddb  Should error positions show both line and column?                claude-code
5 of 22 records (--limit 0 for all)
```

`search <text>` finds the records whose text matches, in the same order, and `context` summarizes the open items and the most recent records for an AI agent (or a person) about to start work; see [search](../reference/cli.md#search) and [context](../reference/cli.md#context) for worked examples on this same project.

## Branches, merges, and AI agents

mtqg only sits on top of git, and asks nothing of how git itself is run. `journal.jsonl` is append-only, so when two branches each write records, merging keeps both (a union merge; see [merging](../reference/schema.md#merging) in the journal format).

To use mtqg with an agent such as Claude Code, `mtqg init --agent claude-code` wires up both its hooks and the MCP server in one step. The agent reads `mtqg context` at the start of a session, and writes `memo`, `todo`, `qa` and `bug` records at each turning point along the way. See [agent hooks](../reference/cli.md#agent-hooks) and the [MCP server](../reference/cli.md#mcp-server) for details.

## A small project

This looks at the `.mtqg/` of the same fictional parser project, used throughout this document and the [command reference](../reference/cli.md) (`e2e/testdata/examples/parser/`). Individual commands are covered above and in the reference; here the same repository is read straight through, as something that was actually used over a few days.

Start with `context` for the lay of the land:

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

`Attention` points out a concurrent status change (two branches closed the same todo independently) and a duplicate glossary definition (`block comment` is defined two different ways). mtqg does not decide which one is right; `mtqg review` shows both in full for a person to judge (see [review](../reference/cli.md#review) for this same repository run through it).

Work that is settled can be tucked away with `archive`. Here, the one question and one bug that were closed on 2026-09-19 move out of the way:

<!-- mtqg:example repo=parser -->
```
$ mtqg archive 2026-09-19..2026-09-19
Range: 2026-09-19..2026-09-19
Archived: 1 question, 1 answer, 1 bug, 1 reply -> .mtqg/archive/2026-09-19..2026-09-19.jsonl
```

The matching lines move, byte for byte, from `journal.jsonl` to `archive/2026-09-19..2026-09-19.jsonl`. Bringing them back is just appending the archive file to `journal.jsonl` and removing it (a shell operation, not an mtqg command):

```
cat .mtqg/archive/2026-09-19..2026-09-19.jsonl >> .mtqg/journal.jsonl
rm .mtqg/archive/2026-09-19..2026-09-19.jsonl
```

## mtqg's own development

From stage 4 (external tool integration) on, mtqg has recorded its own development process with itself (see [`.claude/rules/mtqg-usage.md`](https://github.com/amisonnet8/mtqg/blob/main/.claude/rules/mtqg-usage.md)). It is not a fixture built for a document; it is records that actually piled up, which makes it the best example of what `qa`, `bug` and `rule` look like in real use:

```
git clone https://github.com/amisonnet8/mtqg
cd mtqg
mtqg context
```

Stages 1 through 3 (building the core and the CLI) came before mtqg was used to record its own work, so that history is in `PLAN.md` and `docs/design/` instead.
