# A tour of mtqg

*[日本語](tour_ja.md) | **English***

For the full command list, see the [command reference](../reference/cli.md); for the data format, see [journal format](../reference/schema.md); for a larger example, see the [examples](../examples/examples.md). This tour starts from nothing and touches every kind of record.

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

## Writing down what you notice

Things noticed while working, or worth looking back on, go in a `memo`. No verb, just `add`.

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg m add Tokens carry their line and column
d9f15ee168
```

The output is only the ID of the record that was made: nothing gets in the way of writing.

## Writing down what to do

A `todo` is something to record before you start it, and close with `done`.

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg t add Support C syntax
4f175fd3e4
```

`t list` shows the open ones (`--all` for every one); see [list](../reference/cli.md#list) for what that looks like on a project with a few todos in it already.

## Questions and answers, bugs and their replies

Something you are unsure about goes in a `qa`; a bug you found, and the exchange about it, goes in a `bug`. Both have the same shape (a body, and replies to it): following the body's ID with more text makes that text an answer or a reply.

From here on, this looks in on the same project a while later (the fixture that ships with this repository).

<!-- mtqg:example repo=parser ids=any -->
```
$ mtqg show 2217beaddb
question  2217beaddb  open
by claude-code (ai), 2026-09-21 11:05

  Should error positions show both line and column?

Events
  2026-09-21 11:05  create  claude-code (ai)
$ mtqg q add 2217beaddb Yes, callers usually want both
38164296a0
$ mtqg q done 2217beaddb
Done: 2217beaddb  Should error positions show both line and column?
```

`done` is what closes a question. Answering and closing are separate: a question with an answer still shows as "awaiting confirmation" until it is closed. `bug` works the same way:

<!-- mtqg:example repo=parser ids=any -->
```
$ mtqg b add 7f3a2b1c09 Fixed by returning an error instead of panicking
442f41515e
$ mtqg b done 7f3a2b1c09
Done: 7f3a2b1c09  Parser crashes on empty input
```

## Agreeing on a term

To keep the same word from being used with two different meanings, an agreed term goes in the `glossary`.

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg g add token The smallest unit produced by lexing
2b25f34203
```

## Keeping a rule in force

A convention that can be followed just by reading it goes in a `rule`. It has the same shape as a memo, but it is never moved by `archive`, and `context` never drops or truncates it, whatever the budget.

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg r add Use English for all error messages
89b25bf464
```

## Saying what a record is about

`--at <path>[:<line>]` records where in the project a record was written about, as a fact at the time of writing: it is never updated when the code changes (`line` is optional; `path` alone is fine). It was recorded on a memo written earlier this way, and shows up under `at` in `--json` (plain `show` does not print it):

<!-- mtqg:example repo=tour-at -->
```
$ mtqg show --json e5a1b2c3d4
{
  "command": "show",
  "record": {
    "id": "e5a1b2c3d4e54f6a8b9c0d1e2f3a4b5c",
    "kind": "memo",
    "text": "Tokens carry their line and column",
    "author": {
      "kind": "human",
      "name": "yamada"
    },
    "created": "2026-09-21T09:05:00Z",
    "updated": "2026-09-21T09:05:00Z",
    "at": {
      "path": "internal/lexer/token.go",
      "line": 42,
      "head": "3f9a1c0"
    }
  },
  "events": [
    {
      "id": "e5a1b2c3d4e54f6a8b9c0d1e2f3a4b5c",
      "op": "create",
      "type": "memo",
      "text": "Tokens carry their line and column",
      "at": {
        "path": "internal/lexer/token.go",
        "line": 42,
        "head": "3f9a1c0"
      },
      "v": 0,
      "ts": "2026-09-21T09:05:00Z",
      "author": {
        "kind": "human",
        "name": "yamada"
      }
    }
  ]
}
```

## Looking things over

Once a few records have piled up, here is how to read them back: `log` lists every kind, newest first, back in the same project seen earlier.

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

## Branches and merges

mtqg only sits on top of git, and asks nothing of how git itself is run. `journal.jsonl` is append-only, so when two branches each write records, merging keeps both (a union merge; see [merging](../reference/schema.md#merging) in the journal format).

## Working with an AI agent

To use mtqg with an agent such as Claude Code, `mtqg init --agent claude-code` wires up both its hooks and the MCP server in one step. The agent reads `mtqg context` at the start of a session, and writes `memo`, `todo`, `qa` and `bug` records at each turning point along the way. See [agent hooks](../reference/cli.md#agent-hooks) and the [MCP server](../reference/cli.md#mcp-server) for details.

## From here

That covers every kind of record mtqg keeps. Every command, option and the shape of `--json` are in the [command reference](../reference/cli.md); the data format's details are in [journal format](../reference/schema.md); a somewhat larger example is in [examples](../examples/examples.md).
