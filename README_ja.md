# mtqg

*[English](README.md) | **日本語***

<p align="center">
  <img src="docs/assets/logo.svg" alt="mtqg" width="420">
</p>

<p align="center"><strong>gitリポジトリの中に、人間とAIエージェントのためのプロジェクトの記録を残す</strong></p>

<p align="center">
  <a href="https://github.com/amisonnet8/mtqg/actions/workflows/ci.yml"><img src="https://github.com/amisonnet8/mtqg/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/amisonnet8/mtqg/releases"><img src="https://img.shields.io/github/v/release/amisonnet8/mtqg" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/amisonnet8/mtqg" alt="License"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/amisonnet8/mtqg" alt="Go version"></a>
  <a href="https://pkg.go.dev/github.com/amisonnet8/mtqg"><img src="https://pkg.go.dev/badge/github.com/amisonnet8/mtqg.svg" alt="Go Reference"></a>
</p>

<p align="center">
  <a href="#features">Features</a> ·
  <a href="#demo">Demo</a> ·
  <a href="#install">Install</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#learn-more">Learn more</a>
</p>

作業をしていると、コードやコミットには残らない気づき・やること・質問と答え・不具合とそのやり取り・用語の合意が生まれる。今まではチャットの履歴やメモアプリに散らばって、後から探せなくなっていたはずのものだ。mtqgは、それを`.mtqg/journal.jsonl`という1つのファイルに追記していくだけのツールで、コードと同じgitリポジトリに置き、コードと一緒にコミット・共有する。

## Features

- 📝 **追記するだけ** — 既存の行は書き換えない。JSON Linesの追記だけなので、ブランチをまたいでも union merge でそのまま両方の記録が残る
- 🗂️ **6つの種類** — `memo`・`todo`・`qa`・`bug`・`glossary`・`rule`。優先度・期限・担当者・カンバンのような課題管理の機能は持たず、過程を残すことだけに絞っている
- 🤖 **AIエージェント向け** — `mtqg context`がセッション開始時に状況をまとめて渡す。Claude Codeとは`mtqg init --agent claude-code`でフックとMCPサーバーまで配線できる
- 🔌 **`--json`ですべて機械可読** — 補助ツールやエディタ拡張は`--json`だけでつながる。形を変えるときはフィールドを足すだけ
- 📦 **単一バイナリ・cgoなし** — Linux・macOS・Windowsで同じように動く。依存はGoの標準ライブラリと`golang.org/x/`だけ
- 🔍 **重複や食い違いは事実として見せる** — 同じ質問への並行した回答、用語の重複定義。どちらが正しいかはmtqgが決めず、`mtqg review`で人間が判断する

## Demo

<p align="center">
  <img src="docs/assets/demo.gif" alt="mtqgのデモ: init、todoと質問の記録、回答、log、context" width="700">
</p>

## Install

```bash
go install github.com/amisonnet8/mtqg/cmd/mtqg@latest
```

各OS向けのビルド済みバイナリは[Releases](https://github.com/amisonnet8/mtqg/releases)から（linux・darwin・windows、amd64・arm64）。

シェル補完（bash・zsh・fish・PowerShell）は[シェル補完](docs/reference/cli_ja.md#シェル補完)を参照。

## Quick start

プロジェクトのリポジトリで、一度だけ`mtqg init`する。

```
$ mtqg init
Created .mtqg/ in /home/me/sample-parser
Commit it to share the records.
```

気づいたこと・後で見返したいことは`memo`に、着手前のやることは`todo`に書く。動詞は要らず`add`するだけで、出力は作られた記録のIDだけ：

```
$ mtqg m add トークンは行番号と列番号を持つ
12676b86a4
$ mtqg t add C言語の構文に対応する
db7f9223ba
```

判断に迷ったことは`qa`に書く。本体のIDに続けて文章を書くと、それが答えになる。`done`は答えとは別の操作で、答えがあっても閉じるまでは「確定待ち」のまま表示される（ここから先は、しばらく作業が進んだ後の同じプロジェクトを覗く）：

```
$ mtqg show 2217beaddb
question  2217beaddb  open
by claude-code (ai), 2026-09-21 11:05

  エラー位置は行と列の両方を出しますか？

Events
  2026-09-21 11:05  create  claude-code (ai)
$ mtqg q add 2217beaddb 両方欲しいことが多い
da5dd6247c
$ mtqg q done 2217beaddb
Done: 2217beaddb  エラー位置は行と列の両方を出しますか？
```

<details>
<summary><code>mtqg context</code>の例を見る（AIエージェントがセッション開始時に読むもの）</summary>

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
- Glossary term "ブロックコメント" has conflicting definitions (see mtqg glossary list)
- todo 6b0d549b6f "行コメント // の読み飛ばし" has concurrent changes (see mtqg review)
- 3 mtqg records are not committed

## Open todos (5)
- 6cad4a268d ブロックコメント /* */ の読み飛ばし (claude-code, 10:18)
- 6513270e26 文字列リテラル中の // を無視する (yamada, 10:52)
- 1e27a1c08a エラー位置を行と列で表示する (claude-code, 11:06)
- 1818e81189 READMEに対応している構文を書く (yamada, 11:30)
- 2e44158bae コメント処理のテストケースを追加 (claude-code, 11:32)

## Open questions (2)
- 1012f037b6 ブロックコメントの入れ子に対応する？ (awaiting confirmation, claude-code, 09:10)
    └ 初版では非対応。需要が出たら再検討 (yamada, human)
- 2217beaddb エラー位置は行と列の両方を出しますか？ (unanswered, claude-code, 11:05)

## Open bugs (1)
- 7f3a2b1c09 空の入力でパーサーが落ちる (awaiting confirmation, yamada, 10:41)
    └ macOSでも再現した (claude-code, ai)

## Recent records (newest first)
- 11:32  claude-code  todo      2e44158bae  コメント処理のテストケースを追加
- 11:30  yamada       todo      1818e81189  READMEに対応している構文を書く
- 11:24  claude-code  glossary  f28c105d1f  字句解析: ソースを読み、トークンの並びに変換する処理
- 11:06  claude-code  todo      1e27a1c08a  エラー位置を行と列で表示する
- 11:05  claude-code  question  2217beaddb  エラー位置は行と列の両方を出しますか？
- 10:52  yamada       todo      6513270e26  文字列リテラル中の // を無視する
- 10:46  claude-code  reply     9a8b7c6d5e  空のファイルでも落ちる (to 1012f037b6)
- 10:45  claude-code  reply     3d8e4a0b12  macOSでも再現した (to 7f3a2b1c09)
- 10:41  yamada       bug       7f3a2b1c09  空の入力でパーサーが落ちる
- 10:32  yamada       memo      81e74ef5e8  エラーメッセージは英語で統一する方針
- (12 more; see mtqg log)

## Glossary (4)
- 5b7e2c9a41 トークン: 字句解析で切り出す最小単位
- f29d0da995 ブロックコメント: /* と */ で囲むコメント
- 0cb1e29c65 ブロックコメント: 複数行にわたって書けるコメント
- f28c105d1f 字句解析: ソースを読み、トークンの並びに変換する処理

---
Read full entries with mtqg show <id>.
```

</details>

<details>
<summary><code>--json</code>の例を見る</summary>

```
$ mtqg show --json e5a1b2c3d4
{
  "command": "show",
  "record": {
    "id": "e5a1b2c3d4e54f6a8b9c0d1e2f3a4b5c",
    "kind": "memo",
    "text": "トークンは行番号と列番号を持つ",
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
      "text": "トークンは行番号と列番号を持つ",
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

</details>

## Learn more

- [mtqgの実例](docs/examples/examples_ja.md) — 入門から実例まで、通しで読める1つの文書
- [コマンドリファレンス](docs/reference/cli_ja.md) — 全コマンドの仕様
- [ジャーナルの形式](docs/reference/schema_ja.md) — データ形式の仕様（`.mtqg/SCHEMA.md`として各プロジェクトに書き出される）
- [設計文書](docs/design/README.md) — 設計判断とその理由の記録（日本語のみ）

---

MIT License. 詳しくは[LICENSE](LICENSE)を参照。
