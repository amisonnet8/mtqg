# mtqg コマンドリファレンス

*[English](cli.md) | **日本語***

> 英語版 `cli.md` の日本語訳。内容は英語版と同じに保つ。食い違う場合は英語版が正。

```
mtqg <種類> <動詞> [引数]
mtqg <コマンド> [引数]
```

mtqg自身が出す文言は英語。記録の中身は書いたとおりに表示する。保存の形式は[schema_ja.md](schema_ja.md)にある。

## 種類と動詞

| | memo（`m`） | todo（`t`） | qa（`q`） | glossary（`g`） |
|---|---|---|---|---|
| `add` | `add <本文>` | `add <本文>` | `add <質問>`<br>`add <質問id> <回答>` | `add <用語> <定義>` |
| `done` | — | `done <id>` | `done <質問id>` | — |
| `reopen` | — | `reopen <id>` | `reopen <質問id>` | — |
| `list` | 全件 | 未完了（`--all`で全件） | 未クローズ（`--all`で全件） | 全件 |

- 種類は1文字に略せる。`mtqg t add ...`は`mtqg todo add ...`と同じ。動詞は常に必要
- 種類の違う記録に動詞を使う（たとえばtodoのIDに`mtqg qa done`）とエラーになり、正しいコマンドを示す

## その他のコマンド

| コマンド | 内容 |
|---|---|
| `mtqg edit <id> <本文>` | 記録の本文を置き換える。glossaryは定義を置き換え、用語は変えられない |
| `mtqg delete <id>` | 記録を隠す。質問を消すと、その回答も隠れる |
| `mtqg undo` | この端末から書いた最後の行を消す |
| `mtqg status` | 未完了の項目と未コミットの記録の概況 |
| `mtqg log [--limit N] [--kind K]` | 全種類を時系列で |
| `mtqg show <id>` | 1件の記録と、その全履歴 |
| `mtqg search <語>` | 本文の部分一致で探す |
| `mtqg review` | 並行した状態変更と、用語の重複定義 |
| `mtqg context [--max-tokens N]` | AIエージェント向けの要約 |
| `mtqg format [ファイル]` | 任意のテキストに含まれるイベント行を整形して表示する |
| `mtqg archive <開始>..<終了> [-n]` | 期間内の終わった項目を視界から外す |
| `mtqg init` | `.mtqg/`を作る |
| `mtqg version` | mtqgのバージョンと、リポジトリの形式のバージョンを表示する |

## 共通のオプション

| オプション | 内容 |
|---|---|
| `-C <パス>` | カレントディレクトリの代わりに`<パス>`から`.mtqg/`を探す |
| `--json` | 機械可読の出力（すべてのコマンド） |
| `--all` | 一覧で、終わった項目も含める |
| `--full-id` | IDを先頭10桁ではなく、完全な32桁で表示する |
| `--no-color` | 色を付けない。`NO_COLOR`にも従う |

## .mtqg/の探し方

mtqgはカレントディレクトリ（または`-C <パス>`）から上にたどる。各ディレクトリで、次の順に確認する。

1. `.mtqg/`がある：それを使う
2. `.git`がある（ディレクトリでもファイルでもよい）：ここがリポジトリのルートで、`.mtqg/`はない。mtqgは止まり、`mtqg init`の実行を案内する

mtqgはリポジトリのルートより上は見ない。gitリポジトリの外ではエラーで止まる。`.mtqg/`は1つのgitリポジトリに1つ。サブモジュールは独自の`.mtqg/`を持つ。

```
$ mtqg t add コメント処理のテストを足す
No .mtqg/ found. Run `mtqg init` (will be created at /home/me/sample-parser)
```

`.mtqg/`を作るのは`mtqg init`だけ。

## init

```
$ mtqg init
```

- 同じように上にたどる。リポジトリのルート（`.git`）で、どこで実行したかに関係なく、`.git`の隣に`.mtqg/`を作る
- 途中で既存の`.mtqg/`に当たったらエラー（`.mtqg/ already exists: <パス>`）
- gitリポジトリの外ではエラー
- `.mtqg/journal.jsonl`、`.mtqg/.gitattributes`、`.mtqg/.gitignore`、`.mtqg/version`、`.mtqg/SCHEMA.md`を作る
- gitの設定を変えず、コミットもしない。`.mtqg/`をコミットするよう案内する

## 記録を足す

```
mtqg m add エラーメッセージは英語で統一する
mtqg t add ブロックコメントの読み飛ばし
mtqg q add ブロックコメントの入れ子に対応する？
mtqg g add トークン 字句解析で切り出す最小単位
```

- 残りの引数は空白でつないで1つの本文にする。シェルの特殊文字（`#` `*` `(` `)` `&` `|` `<` `>`）を含む場合を除き、引用符は要らない
- glossaryは、最初の引数が用語、残りが定義
- 本文の代わりに`-`を渡すと、標準入力から読む：`git log -1 --format=%s | mtqg m add -`
- 本文がなければ`$EDITOR`が開く
- 出力は、作られた記録のIDだけ

```
$ mtqg t add ブロックコメントの読み飛ばし
6cad4a268d
```

## 質問と回答

- `mtqg q add <本文>`で質問を足す。`mtqg q add <質問id> <本文>`で、その質問に回答を足す。回答は独自のIDを持つ
- 回答することと閉じることは別。`mtqg q done <質問id>`で質問を閉じる。回答はいくつでも足せて、どれも他を置き換えない
- 回答を持てるのは質問だけ。回答に回答は付けられない

質問は4つの状態のどれかにある。

| 状態 | 回答 | 閉じている | `qa list`での表示 |
|---|---|---|---|
| 未回答 | 0 | いいえ | 表示する |
| 確定待ち | 1以上 | いいえ | 表示する（回答の件数付き） |
| 回答済み | 1以上 | はい | `--all`で表示 |
| 回答なしで閉じた | 0 | はい | `--all`で表示 |

| 渡したID | できること |
|---|---|
| 質問 | `q add`（回答）、`q done`、`q reopen`、`edit`、`delete`（回答も隠れる） |
| 回答 | `edit`、`delete`（その回答だけ。質問の状態は変わらない） |

## ID

- どの記録も完全なIDを持つ：小文字の16進32桁（ハイフンを除いたUUIDv4）。例：`81e74ef5e8e24d949ed904759531985d`
- mtqgは**先頭の10桁**を表示する（`81e74ef5e8`）。`--full-id`で完全なIDを表示する
- gitのコミットハッシュと同じく、一意に決まる前方一致を入力として受け付ける：`81e74ef5e8`、一意なら`81e7`でもよい
- 前方一致が複数の記録に当てはまるときは、止まって、候補を完全なIDで並べる

```
$ mtqg t done 70430f77ff
Ambiguous ID "70430f77ff" matches 2 records:
  70430f77ff4b475185d5cae12dff1a17  todo  ブロックコメントの読み飛ばし      claude-code  10:18
  70430f77ff91c2e04a8b33f1d7e6a025  memo  行末の // も読み飛ばすようにした  yamada       2027-03-02
```

- 記録どうしは完全なIDで参照する（回答の`re`）。短いIDが後から曖昧になっても、記録が指す先は変わらない

## edit、delete

```
$ mtqg delete 1012f037b6
Deleted: "ブロックコメントの入れ子に対応する？"
2 answers are also hidden (claude-code, yamada)
They remain in git history
```

どちらもイベントを追記する。ファイルからもgitの履歴からも何も消えない。

## undo

**今の端末から書いた最後の行**を消す。直前のミス（たとえば、質問IDを付け忘れて、回答を新しい質問として足してしまった）のためのもの。

```
$ mtqg q add 初版では非対応。需要が出たら再検討
301850c5a3
$ mtqg undo
Undone: qa add "初版では非対応。需要が出たら再検討" (301850c5a3)
```

- 対象：`journal.jsonl`の中で（`ts`の順で）、この記録者と、この端末の`tty`の値を持つ最後の行。端末を見分けられないときは、`tty`のない行を同じ端末とみなす
- 1段だけ。`undo`は繰り返せない
- コミット済みかどうかは見ない。すでに共有したものには`delete`を使う。他のブランチに届いた行を消すと、次のマージで戻ってくることがある

## 読む

### status

```
$ mtqg status
Open todos          5
Open questions      2  (1 awaiting confirmation)
Glossary            4  (1 with duplicate definitions)
Conflicts           1  -> mtqg review

Uncommitted records 3
```

### list

```
$ mtqg todo list
6cad4a268d  ブロックコメント /* */ の読み飛ばし     claude-code  10:18
6513270e26  文字列リテラル中の // を無視する        yamada       10:52
1e27a1c08a  エラー位置を行と列で表示する            claude-code  11:06
...
5 open (show done: --all)

$ mtqg qa list
2217beaddb  エラー位置は行と列の両方を出しますか？   claude-code  11:05  unanswered
1012f037b6  ブロックコメントの入れ子に対応する？     claude-code  09:10  2 answers, awaiting confirmation
           └ 初版では非対応。需要が出たら再検討     yamada       09:41
2 open (show all: --all)
```

`qa list`は、最新の回答と回答の件数を表示する。

### show

```
$ mtqg show 1012f037b6
qa  1012f037b6  open
  Q ブロックコメントの入れ子に対応する？        claude-code  09:10

  Answers (2)
  ae2eb1547f  一般的には対応するのが望ましい      claude-code  09:15  ai
  95e761d177  初版では非対応。需要が出たら再検討  yamada       09:41  human

  Events
    09:10  create  claude-code
    09:15  create  claude-code  → ae2eb1547f
    09:41  create  yamada       → 95e761d177
```

### review

```
$ mtqg review
Concurrent status changes (1)
  6b0d549b6f "行コメント // の読み飛ばし"
    10:15  claude-code  open -> done
    14:30  yamada       open -> done

Duplicate glossary definitions (1)
  ブロックコメント
    f29d0da995  yamada       /* と */ で囲むコメント
    0cb1e29c65  claude-code  複数行にわたって書けるコメント
```

並行した状態変更とは、同じ`from`の状態からの2つ以上の状態変更のこと。mtqgはどれが正しいかを決めない。

### context

AIエージェントがセッションの始めに読むもの。ここまでの過程（何が未完了か、何に答えが出たか、どの用語が合意されているか）を渡す。プロジェクトの説明、現在の仕様、ビルドの手順は含めない。それらはREADMEやエージェントの指示ファイルの役目。

```
$ mtqg context
# mtqg context — sample-parser (main)

This is the process record of this project. Read the following before you start working.
- Respect what has been decided (answered questions, memos stating a policy)
- Do not decide open questions on your own; confirm them
- Use terms as defined in the glossary
- Record questions, decisions, findings, and todos with mtqg as they come up

## Attention
- Glossary term "ブロックコメント" has conflicting definitions (mtqg review)
- 3 mtqg records are not committed

## Open todos (5)
- 6cad4a268d  ブロックコメント /* */ の読み飛ばし (claude-code, 10:18)
- 6513270e26 文字列リテラル中の // を無視する (yamada, 10:52)
- 1e27a1c08a エラー位置を行と列で表示する (claude-code, 11:06)
- 1818e81189 READMEに対応している構文を書く (yamada, 11:30)
- 2e44158bae コメント処理のテストケースを追加 (claude-code, 11:32)

## Open questions (2)
- 2217beaddb エラー位置は行と列の両方を出しますか？ (unanswered, claude-code, 11:05)
- 1012f037b6  ブロックコメントの入れ子に対応する？ (awaiting confirmation, 09:10)
    └ 初版では非対応。需要が出たら再検討 (yamada, human)

## Recent records (10, newest first)
- 11:24 claude-code glossary 字句解析：ソースを読み、トークンの並びに変換する処理
- 10:32 yamada      memo     エラーメッセージは英語で統一する方針
- 09:41 yamada      answer   初版では非対応。需要が出たら再検討 (to 1012f037b6)
- ... (7 more; see mtqg log)

## Glossary (4)
- トークン：字句解析で切り出す最小単位
- 字句解析：ソースを読み、トークンの並びに変換する処理
- ブロックコメント：/* と */ で囲むコメント (2 definitions)
- レキサ：字句解析を行う実装（Lexer構造体）

---
Read full entries with mtqg show <id>.
```

- 区画はこの順：Attention、Open todos、Open questions、Recent records、Glossary。空の区画は出さない
- どの項目もID、記録者、時刻を持つ。本文は1行に切り詰める
- 既定の分量：約2000トークン（`--max-tokens N`で変える）。超えたときは、次の順に削る：最近の記録（10→5→3→0件）、用語の定義（次に用語ごと）、回答の本文、古い質問。未完了のtodoは削らない。それでも超える場合は一覧を短くし、`(N more)`で終える
- 省略するときは、必ず何件省いたかと、どこで読めるかを書く
- `--json`では、区画ごとの配列、削ったかどうか、全体の件数を返す

## format

任意のテキストからイベント行を拾い、時刻順の1つの表にして、ローカル時間で表示する。

```
$ git show 3f9a1c0 | mtqg format
10:18  todo    6cad4a268d  ブロックコメント /* */ の読み飛ばし   claude-code
10:32  memo    81e74ef5e8  エラーメッセージは英語で統一する方針   yamada
11:05  qa      2217beaddb  エラー位置は行と列の両方を出しますか？  claude-code
11:32  todo    2e44158bae  コメント処理のテストケースを追加      claude-code
```

- 先頭の`+`・`-`（unified diff）は、読む前に取り除く
- JSONのイベントでない行は黙って飛ばす。`git show`や`git diff`の出力を丸ごと渡せる
- どこから来た行でもよい：`git diff`、`git diff main...feature`、`cat .mtqg/journal.jsonl`、`grep ... .mtqg/journal.jsonl`、引数で渡したファイル（`.mtqg/archive/`のファイルを含む）
- `+`・`-`の印は表示しない。ただし消えた行には常に印を付ける。`--mark`で全行に印を付ける

## archive

```
$ mtqg archive 2021-01-01..2024-09-18
$ mtqg archive 2021..2023
$ mtqg archive 202404..2024-09 -n
```

終わった項目のうち、最後のイベントが期間に入るものを、`journal.jsonl`から`.mtqg/archive/<開始>..<終了>.jsonl`に移す（何がアーカイブされるかは[schema_ja.md](schema_ja.md#アーカイブ)）。どのコマンドも`archive/`を自分から読むことはなく、アーカイブ済みのIDは単に見つからない。アーカイブのファイルを読むときは、`mtqg format`に渡す。

期間は1つの引数`<開始>..<終了>`で書く。

1. `.`が2つ以上続くところで分ける。`..`は必須
2. それぞれから`-`をすべて取り除く
3. 桁数で単位が決まる：8＝日、6＝月、4＝年
4. 左右は同じ単位でなければならない
5. 月・年は、その最初の日に始まり、最後の日に終わる。両端を含む。日付はローカルの日付

```
2021-01-01..2024-09-18    → 2021-01-01..2024-09-18
20210101....20240918      → 2021-01-01..2024-09-18
2021-0101..202409-18      → 2021-01-01..2024-09-18
2021..2023                → 2021-01-01..2023-12-31
202404..2024-09           → 2024-04-01..2024-09-30
2023..2023                → 2023-01-01..2023-12-31
```

エラーになるもの：`..`がない、8・6・4桁でない側がある、左右の単位が違う、存在しない日付、開始が終了より後、数字・`-`・`.`以外の文字がある。

```
$ mtqg archive 2021-0101..202409-18
Range: 2021-01-01..2024-09-18
Archived: 412 memos, 138 todos, 57 questions -> .mtqg/archive/2021-01-01..2024-09-18.jsonl
Skipped: 3 open todos, 1 open question, 24 glossary entries
```

- 1行目に、期間をどう読んだかを必ず表示する
- ファイル名は常に正規化した期間。同じ期間をもう一度アーカイブすると、同じファイルに追記する
- `-n`（`--dry-run`）は、何も移さずに同じ報告を表示する
- 相対的な日付（「2年前」など）は受け付けない

## version

`mtqg version`は、mtqgのバージョンと、`.mtqg/version`が宣言する形式のバージョンを表示する。mtqgは、自分の知るものより新しい形式のリポジトリは読み書きを断り、mtqgの更新を促す。

形式のバージョンは`0`（未確定）で、それを上げるコマンドはまだない。形式1以降ができたときに用意する（[schema_ja.md](schema_ja.md#バージョン)）。

## 記録者

| 項目 | 値 |
|---|---|
| 記録者名 | `git config user.name` |
| 記録者の種別 | CLIからは`human` |
| エディタ | `$EDITOR` |

設定ファイルはない。
