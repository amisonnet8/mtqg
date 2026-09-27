# mtqg examples

*[日本語](examples_ja.md) | **English***

For the full command list, see the [command reference](../reference/cli.md); if this is your first look at mtqg, start with the [tour](../tour/tour.md). Here are two somewhat larger examples.

## A small project

This looks at the `.mtqg/` of the fictional parser project used throughout the [command reference](../reference/cli.md) (`e2e/testdata/examples/parser/`). The reference explains each command on its own; here the same repository is read straight through, as something that was actually used over a few days.

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

From stage 4 (external tool integration) on, mtqg has recorded its own development process with itself (see [`.claude/rules/mtqg-usage.md`](https://github.com/amisonnet8/mtqg/blob/main/.claude/rules/mtqg-usage.md)). It is not a fixture built for a document; it is records that actually piled up, which makes it the best example of what `qa`, `bug`, `rule` and `--at` look like in real use:

```
git clone https://github.com/amisonnet8/mtqg
cd mtqg
mtqg context
```

Stages 1 through 3 (building the core and the CLI) came before mtqg was used to record its own work, so that history is in `PLAN.md` and `docs/design/` instead.
