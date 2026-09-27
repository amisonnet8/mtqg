# mtqgを歩いて回る

*[English](tour.md) | **日本語***

コマンドの完全な一覧は[コマンドリファレンス](../reference/cli_ja.md)、データの形式は[ジャーナルの形式](../reference/schema_ja.md)、もう少し大きな実例は[実例集](../examples/examples_ja.md)を参照。ここでは、何もないところから`mtqg`を使い始め、ひととおりの機能に触れる。

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

## 気づいたことを書き留める

作業中に気づいたこと・後で見返したいことは`memo`に書く。動詞は要らず、`add`するだけ。

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg m add トークンは行番号と列番号を持つ
8ebaa31223
```

出力は作られた記録のIDだけ。書き込みを何も邪魔しない。

## やることを書き留める

`todo`は、着手前に立てておくもの。`done`で閉じる。

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg t add C言語の構文に対応する
46d10cebf7
```

開いているものは`t list`で見る（`--all`ですべて）。いくつか溜まったプロジェクトでの見え方は[list](../reference/cli_ja.md#list)を参照。

## 質問と回答、不具合とそのやり取り

判断に迷ったことは`qa`に、見つけた不具合とそのやり取りは`bug`に書く。どちらも同じ形（本体と、それへの返信）を持つ：本体のIDに続けて文章を書くと、それが答え・返信になる。

ここから先は、しばらく作業が進んだ後の同じプロジェクトを覗いてみる（このリポジトリに実際に付属しているフィクスチャの続き）。

<!-- mtqg:example repo=parser_ja ids=any -->
```
$ mtqg show 2217beaddb
question  2217beaddb  open
by claude-code (ai), 2026-09-21 11:05

  エラー位置は行と列の両方を出しますか？

Events
  2026-09-21 11:05  create  claude-code (ai)
$ mtqg q add 2217beaddb 両方欲しいことが多い
38292fa574
$ mtqg q done 2217beaddb
Done: 2217beaddb  エラー位置は行と列の両方を出しますか？
```

質問を閉じるのは`done`。答えたことと閉じたことは別の操作で、答えがあっても閉じるまでは「確定待ち」のまま表示される。`bug`も同じ形：

<!-- mtqg:example repo=parser_ja ids=any -->
```
$ mtqg b add 7f3a2b1c09 パニックせずエラーを返すようにして直した
34249685ed
$ mtqg b done 7f3a2b1c09
Done: 7f3a2b1c09  空の入力でパーサーが落ちる
```

## 用語を決める

同じ言葉を違う意味で使ってしまうのを防ぐため、合意した用語は`glossary`に残す。

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg g add トークン 字句解析で切り出す最小単位
b015effafd
```

## 決まり事を残す

読めばそのまま従える決まり事は`rule`に書く。形は`memo`と同じだが、`archive`で移されず、`context`でも予算に関わらず残り続ける。

<!-- mtqg:example repo=empty ids=any -->
```
$ mtqg r add エラーメッセージはすべて英語で書く
e3ca0dd808
```

## どこについて書いたかを示す

記録がコードのどこについてのものかを、`--at <path>[:<line>]`で書いた時点の事実として残せる。以後コードが変わっても更新されない（`line`は省略できる。`path`だけでも構わない）。以前書かれたこのメモにも付けてあり、`--json`で見ると`at`が付いているのが分かる（普通の`show`は表示しない）：

<!-- mtqg:example repo=tour-at_ja -->
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

いくつか記録が溜まってきたところで、それを読み返す方法。`log`は、先ほどのプロジェクトで、新しい順にすべての種類を並べる。

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

## ブランチとマージ

mtqgはgitの上に乗るだけで、git自身の運用には何も要求しない。`journal.jsonl`は追記だけなので、2つのブランチがそれぞれ記録を書いても、マージすれば両方の記録が残る（union merge。詳しくは[ジャーナルの形式のマージ](../reference/schema_ja.md#マージ)）。

## AIエージェントと使う

Claude Codeなどのエージェントと使うときは`mtqg init --agent claude-code`で、フックとMCPサーバーの配線までまとめて済ませられる。エージェントは、セッションの始めに`mtqg context`を読み、作業の区切りごとに`memo`・`todo`・`qa`・`bug`を書きながら進める。詳しくは[エージェントのフック](../reference/cli_ja.md#エージェントのフック)・[MCPサーバー](../reference/cli_ja.md#mcpサーバー)を参照。

## この先

これでmtqgのひととおりの機能に触れた。すべてのコマンド・オプション・`--json`の形は[コマンドリファレンス](../reference/cli_ja.md)に、データ形式の細部は[ジャーナルの形式](../reference/schema_ja.md)に、もう少し大きな実例は[実例集](../examples/examples_ja.md)にある。
