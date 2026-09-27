# mtqgの実例

*[English](examples.md) | **日本語***

コマンドの完全な一覧は[コマンドリファレンス](../reference/cli_ja.md)、データの形式は[ジャーナルの形式](../reference/schema_ja.md)を参照。ここでは何もないところから、すべての種類にひととおり触れたあと、もう少し大きな実在のリポジトリを2つ見る。

## mtqgとは

作業をしていると、コードやコミットには残らない気づき・やること・質問と答え・不具合とそのやり取り・用語の合意が生まれる。今まではチャットの履歴やメモアプリに散らばって、後から探せなくなっていたはずのものだ。mtqgは、それを`.mtqg/journal.jsonl`という1つのファイルに追記していくだけのツールで、コードと同じgitリポジトリに置き、コードと一緒にコミット・共有する。

## 始める

インストールは`go install github.com/amisonnet8/mtqg/cmd/mtqg@latest`。あるプロジェクトのリポジトリで、一度だけ`mtqg init`する。

<!-- mtqg:example repo=none path=/home/me/sample-parser -->
```
$ mtqg init
Created .mtqg/ in /home/me/sample-parser
Commit it to share the records.
```

`.mtqg/`ができる。あとは案内どおり、これをコミットすればよい（`mtqg init`自体はコミットしない）。

## 起きたことを書き留める

作業中に気づいたこと・後で見返したいことは`memo`に書く。動詞は要らず`add`するだけで、出力は作られた記録のIDだけなので、書き込みを何も邪魔しない：

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg m add トークンは行番号と列番号を持つ
12676b86a4
```

`todo`も同じで、着手前に立てておき`done`で閉じる。開いているものは`t list`で見る（`--all`ですべて。いくつか溜まったプロジェクトでの見え方は[list](../reference/cli_ja.md#list)を参照）：

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg t add C言語の構文に対応する
db7f9223ba
```

判断に迷ったことは`qa`に、見つけた不具合とそのやり取りは`bug`に書く。どちらも同じ形（本体と、それへの返信）を持つ：本体のIDに続けて文章を書くと、それが答え・返信になる。ここから先は、しばらく作業が進んだ後の同じパーサープロジェクトを覗いてみる（このリポジトリに実際に付属しているフィクスチャ）：

<!-- mtqg:example repo=parser_ja ids=any -->
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

質問を閉じるのは`done`。答えたことと閉じたことは別の操作で、答えがあっても閉じるまでは「確定待ち」のまま表示される。`bug`も同じ形：

<!-- mtqg:example repo=parser_ja ids=any -->
```
$ mtqg b add 7f3a2b1c09 パニックせずエラーを返すようにして直した
23cb665e97
$ mtqg b done 7f3a2b1c09
Done: 7f3a2b1c09  空の入力でパーサーが落ちる
```

同じ言葉を違う意味で使ってしまうのを防ぐため、合意した用語は`glossary`に残す：

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg g add トークン 字句解析で切り出す最小単位
4dedba8ba4
```

読めばそのまま従える決まり事は`rule`に書く。形は`memo`と同じだが、`archive`で移されず、`context`でも予算に関わらず残り続ける：

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg r add エラーメッセージはすべて英語で書く
23da3c03be
```

## どこについて書いたかを示す

記録がコードのどこについてのものかを、`--at <path>[:<line>]`で書いた時点の事実として残せる。以後コードが変わっても更新されない（`line`は省略できる。`path`だけでも構わない）。以前書かれたこのメモにも付けてあり、`--json`で見ると`at`が付いているのが分かる（普通の`show`は表示しない）：

<!-- mtqg:example repo=at-demo_ja -->
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

## 見て回る

いくつか記録が溜まってきたところで、それを読み返す方法。`log`は、同じパーサープロジェクトで、新しい順にすべての種類を並べる：

<!-- mtqg:example repo=parser_ja ids=any -->
```
$ mtqg log --limit 5
11:32  todo      2e44158bae  コメント処理のテストケースを追加                      claude-code
11:30  todo      1818e81189  READMEに対応している構文を書く                        yamada
11:24  glossary  f28c105d1f  字句解析: ソースを読み、トークンの並びに変換する処理  claude-code
11:06  todo      1e27a1c08a  エラー位置を行と列で表示する                          claude-code
11:05  question  2217beaddb  エラー位置は行と列の両方を出しますか？                claude-code
5 of 22 records (--limit 0 for all)
```

`search <文字列>`は本文が一致するものを同じ並びで探し、`context`はこれから作業するAIエージェント（や人間）向けに、未完了のものと最近の記録をまとめる。この同じプロジェクトでの実行例は[search](../reference/cli_ja.md#search)・[context](../reference/cli_ja.md#context)を参照。

## ブランチ・マージ・AIエージェント

mtqgはgitの上に乗るだけで、git自身の運用には何も要求しない。`journal.jsonl`は追記だけなので、2つのブランチがそれぞれ記録を書いても、マージすれば両方の記録が残る（union merge。詳しくは[ジャーナルの形式のマージ](../reference/schema_ja.md#マージ)）。

Claude Codeなどのエージェントと使うときは`mtqg init --agent claude-code`で、フックとMCPサーバーの配線までまとめて済ませられる。エージェントは、セッションの始めに`mtqg context`を読み、作業の区切りごとに`memo`・`todo`・`qa`・`bug`を書きながら進める。詳しくは[エージェントのフック](../reference/cli_ja.md#エージェントのフック)・[MCPサーバー](../reference/cli_ja.md#mcpサーバー)を参照。

## 小さなプロジェクト

ここで見るのは、この文書と[コマンドリファレンス](../reference/cli_ja.md)のいたるところで例に使っている、同じ架空のパーサー開発プロジェクトの`.mtqg/`（`e2e/testdata/examples/parser_ja/`）。個々のコマンドの説明は上とリファレンス側に譲り、ここでは「実際に何日かにわたって使われたリポジトリを、後から通しで読む」という視点でたどる。

まず`context`で全体を見る：

<!-- mtqg:example repo=parser_ja -->
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

`Attention`が、並行した状態変更（同じtodoを2つのブランチが別々に閉じた）と、`glossary`の重複定義（`ブロックコメント`が2通り定義されている）を教えてくれる。どちらが正しいかはmtqgが決めないので、`mtqg review`で詳しく見て、人間が判断する（[review](../reference/cli_ja.md#review)に、この同じリポジトリでの実行例がある）。

区切りが付いた作業は`archive`でしまえる。ここでは2026-09-19に閉じた1件の質問と1件のバグを移してみる：

<!-- mtqg:example repo=parser_ja -->
```
$ mtqg archive 2026-09-19..2026-09-19
Range: 2026-09-19..2026-09-19
Archived: 1 question, 1 answer, 1 bug, 1 reply -> .mtqg/archive/2026-09-19..2026-09-19.jsonl
```

`journal.jsonl`から`archive/2026-09-19..2026-09-19.jsonl`へ、該当する行がそのまま移る。戻したくなったら、アーカイブのファイルを`journal.jsonl`の末尾に結合して消すだけでよい（`mtqg`のコマンドではなく、シェルでの操作）：

```
cat .mtqg/archive/2026-09-19..2026-09-19.jsonl >> .mtqg/journal.jsonl
rm .mtqg/archive/2026-09-19..2026-09-19.jsonl
```

## mtqg自身の開発

mtqgは、段階4（外部ツール連携）以降の自分自身の開発過程を、他ならぬmtqgで記録している（[`.claude/rules/mtqg-usage.md`](https://github.com/amisonnet8/mtqg/blob/main/.claude/rules/mtqg-usage.md)）。作られたフィクスチャではなく実際に積み重なった記録なので、`qa`・`bug`・`rule`・`--at`が実務でどう使われるかを見るには、これが一番の実例になる。

```
git clone https://github.com/amisonnet8/mtqg
cd mtqg
mtqg context
```

段階1〜3（コアとCLIができるまで）の経緯は、まだmtqgを自分自身の記録に使う前だったため、`PLAN.md`と`docs/design/`にある。
