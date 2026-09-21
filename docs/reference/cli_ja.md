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
| `mtqg log [--limit N] [--kind K]` | 全種類の記録を、新しいものから |
| `mtqg show <id>` | 1件の記録を、全文と履歴とともに |
| `mtqg search <語>` | 本文の部分一致で探す |
| `mtqg review` | 並行した状態変更と、用語の重複定義 |
| `mtqg context [--max-tokens N]` | AIエージェント向けの要約 |
| `mtqg format [ファイル]` | 任意のテキストに含まれるイベント行を整形して表示する |
| `mtqg archive <開始>..<終了> [-n]` | 期間内の終わった項目を視界から外す |
| `mtqg init` | `.mtqg/`を作る |
| `mtqg version` | mtqgのバージョンと、リポジトリの形式のバージョンを表示する |
| `mtqg help` | コマンドの一覧を表示する（`-h`、`--help`も同じ） |

## 共通のオプション

| オプション | 内容 |
|---|---|
| `-C <パス>` | カレントディレクトリの代わりに`<パス>`から`.mtqg/`を探す |
| `--json` | 機械可読の出力（すべてのコマンド） |
| `--all` | 一覧で、終わった項目も含める |
| `--full-id` | IDを先頭10桁ではなく、完全な32桁で表示する |
| `--no-color` | 色を付けない。`NO_COLOR`にも従う |

色は飾りにすぎず、出力先が端末のときだけ使う。

### オプションの位置

- オプションは記録の本文の**前**に置く。本文の最初の語からは、`-`で始まる語も含めてすべて本文になる：
  `mtqg t add fix the -x flag`は`fix the -x flag`を記録する
- `-`で始まる本文は、先に`--`を置く：`mtqg t add -- -1 is not allowed`。`-`だけの語は標準入力を意味する
  （[記録を足す](#記録を足す)）
- IDを取るコマンド（`done`、`reopen`など）と、本文のないコマンド（`list`、`status`など）は、
  オプションがどこにあってもよい：`mtqg t done 6cad --full-id`
- 特定のコマンドに結び付かないオプション（`-C`、`--no-color`）は、種類の前にも置ける：
  `mtqg -C ../other t list`
- 一覧にないオプションはエラー

## 終了コード

| コード | 意味 |
|---|---|
| 0 | 成功。すでに済んでいる変更も成功とする |
| 1 | 頼まれたことができなかった：`.mtqg/`がない、IDが見つからない、書き込みのロックが取れない、など |
| 2 | コマンドラインが間違っている：未知のコマンドやオプション、引数の不足 |

エラーは標準エラー出力に出す。警告（飛ばした行、衝突マーカー）も標準エラー出力に出し、終了コードは変えない。
標準出力を他の道具のためにきれいに保つため。

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
- gitの設定を変えず、コミットもしない。`.mtqg/`をコミットするよう案内する：

```
$ mtqg init
Created .mtqg/ in /home/me/sample-parser
Commit it to share the records.
```

## 記録を足す

```
mtqg m add エラーメッセージは英語で統一する
mtqg t add ブロックコメントの読み飛ばし
mtqg q add ブロックコメントの入れ子に対応する？
mtqg g add トークン 字句解析で切り出す最小単位
```

- 残りの引数は空白でつないで1つの本文にする。シェルの特殊文字（`#` `*` `(` `)` `&` `|` `<` `>`）を含む場合を除き、引用符は要らない
- glossaryは、最初の引数が用語、残りが定義。複数の語でできた用語は引用符が要る：
  `mtqg g add "block comment" /* と */ で囲むコメント`
- `mtqg q add <質問id> <本文>`は、質問ではなく回答を足す（[質問と回答](#質問と回答)）
- 本文の代わりに`-`を渡すと、標準入力を最後まで読む。末尾の改行は落とす：`git log -1 --format=%s | mtqg m add -`
- 引数がまったくなければ（`mtqg m add`、`mtqg t add`、`mtqg q add`）、空のファイルで`$EDITOR`が開き、保存した
  内容が本文になる（末尾の改行は落とす）。`$EDITOR`には引数や引用符を含められる（`code --wait`）。シェルは
  通さない。`$EDITOR`が設定されていなければ、止まってそう伝える
- エディタが開くのはこの場合だけ。足りない部分があるのはコマンドラインの誤りで、何も書かない：
  `mtqg q add <質問id>`（回答がない）、`mtqg g add`（用語がない）、`mtqg g add <用語>`（定義がない）。
  回答や定義の本文は`-`にでき、標準入力から読む
- 本文は複数行でもよい（標準入力かエディタから）。`list`は1行目だけを表示する
- 本文が空、または空白だけのときはエラー（`Aborting: the text is empty`）。何も書かない
- 出力は、作られた記録のIDだけ

```
$ mtqg t add ブロックコメントの読み飛ばし
6cad4a268d
```

## done、reopen

```
$ mtqg t done 6cad4a268d
Done: 6cad4a268d  ブロックコメント /* */ の読み飛ばし
$ mtqg t done 6cad4a268d
Already done: 6cad4a268d  ブロックコメント /* */ の読み飛ばし
$ mtqg t reopen 6cad4a268d
Reopened: 6cad4a268d  ブロックコメント /* */ の読み飛ばし
```

- 1行を出す：何をしたか、ID、本文の1行目。打ったIDが意図した記録だったと確かめられる
- すでにその状態のtodoには何も書かず、その旨を出す（`Already done: ...`、`Already open: ...`）。
  終了コードは0：同じ変更の繰り返しはエラーではない
- `mtqg q done`と`mtqg q reopen`は、質問に対して同じことをする
- 状態を持つのはtodoと質問だけ。それ以外の記録のIDには、止まって、それが何かを伝える。
  正しいコマンドが別にあるときは、それを示す：
  `81e74ef5e8 is a memo, not a todo`、`2217beaddb is a question, not a todo; use `mtqg qa done 2217beaddb``

## 質問と回答

- `mtqg q add <本文>`で質問を足す。`mtqg q add <質問id> <本文>`で、その質問に回答を足す。回答は独自のIDを持ち、
  その`re`には質問の完全なIDが入る
- 回答することと閉じることは別。`mtqg q done <質問id>`で質問を閉じる。回答はいくつでも足せて、どれも他を置き換えない。
  閉じた質問にも回答は足せる
- 回答を持てるのは質問だけ。回答に回答は付けられない

### 質問か、回答か

`q add`は、最初の引数だけで両者を見分ける。最初の引数が**16進数の4桁以上で、それだけでできている**
（`0-9`、`a-f`。大文字は小文字として読む）なら、それは回答先の質問のIDで、残りが回答の本文になる。
それ以外の最初の引数なら、全体が新しい質問の本文になる。

| 16進数4桁以上の最初の引数が当てはまるもの | 結果 |
|---|---|
| 質問1件 | 残りを、その質問への回答として足す |
| 質問でない記録1件 | エラー。それが何かを伝える |
| 複数の記録 | エラー。候補を完全なIDで並べる |
| 記録がない | エラー。何も書かない |

```
$ mtqg q add a8ec はどういう意味ですか？
No record matches "a8ec". A first word of 4 or more hex digits is read as the ID of the question to answer.
To ask a question that starts with it, put the whole text in quotes: mtqg qa add "a8ec ..."
```

- 引用符で括った本文は1つの引数になる。中に空白があれば16進数だけではないので、IDとは読まれない：
  `mtqg q add "a8ec はどういう意味ですか？"`は質問、`mtqg q add a8ec 見つからない場合のエラーコードです`は
  質問`a8ec`への回答
- 当てはまる記録がないときは、質問にせずエラーにする。打ち間違えたIDが、黙って新しい質問になることがないように
  するため。最初の語が16進数の文字だけでできた英語の質問（`Face detection is slow. Why?`、
  `Dead code: remove it?`）も同じように止まる。本文全体を引用符で括る。そのような語1つだけの質問は、
  標準入力から渡せる
- IDだけで、後ろに本文がないときはエラー。何も書かない

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
- gitのコミットハッシュと同じく、**4桁以上**で一意に決まる前方一致を入力として受け付ける：`81e74ef5e8`、
  一意なら`81e7`でもよい。それより短いものはエラー
  （`The ID "81e" is too short: give at least 4 digits`）。`add`や`bad`のような語がIDと間違われないように
  するため
- 前方一致が複数の記録に当てはまるときは、止まって、候補を完全なIDで並べる

```
$ mtqg t done 70430f77ff
Ambiguous ID "70430f77ff" matches 2 records:
  70430f77ff91c2e04a8b33f1d7e6a025  memo  行末の // も読み飛ばすようにした     yamada       2026-03-02
  70430f77ff4b475185d5cae12dff1a17  todo  ブロックコメント /* */ の読み飛ばし  claude-code  10:18
```

- 当てはまる記録がなければ止まる：`No record matches "6cad4"`。削除した記録は当てはまらない。
  大文字の16進は小文字として読む
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

Uncommitted records 3
```

- 各行は件数。`Open questions`は閉じていない質問の数で、`awaiting confirmation`はそのうち回答のあるもの。
  `Glossary`は用語の項目の数で、`with duplicate definitions`は2回以上定義された用語（文字まで同じもの）の数。
  `Uncommitted records`は、`journal.jsonl`の中に、最後のコミット（`HEAD`）にない行を
  1つ以上持つ記録の数：それ以降に作った・変えた記録。1つの記録は、行がいくつあっても1と数える。
  ステージしただけでコミットしていない行も、未コミットに数える。まだコミットがない、または
  `.mtqg/journal.jsonl`が追跡されていないときは、すべての記録を数える。gitを実行できないときは、
  この行は`unknown`になる

### list

```
$ mtqg todo list
6cad4a268d  ブロックコメント /* */ の読み飛ばし  claude-code  10:18
6513270e26  文字列リテラル中の // を無視する     yamada       10:52
1e27a1c08a  エラー位置を行と列で表示する         claude-code  11:06
3 open (show done: --all)

$ mtqg todo list --all
6cad4a268d  ブロックコメント /* */ の読み飛ばし  claude-code  10:18  done
6513270e26  文字列リテラル中の // を無視する     yamada       10:52
1e27a1c08a  エラー位置を行と列で表示する         claude-code  11:06
2 open, 1 done

$ mtqg qa list
1012f037b6  ブロックコメントの入れ子に対応する？    claude-code  09:10  2 answers, awaiting confirmation
          └ 初版では非対応。需要が出たら再検討      yamada       09:41
2217beaddb  エラー位置は行と列の両方を出しますか？  claude-code  11:05  unanswered
2 open (show done: --all)

$ mtqg qa list --all
c3b1f0d2e4  ライセンスは何にしますか？              yamada       2026-09-19  1 answer, done
          └ MITが一番単純です                       claude-code  2026-09-19
1012f037b6  ブロックコメントの入れ子に対応する？    claude-code  09:10       2 answers, awaiting confirmation
          └ 初版では非対応。需要が出たら再検討      yamada       09:41
2217beaddb  エラー位置は行と列の両方を出しますか？  claude-code  11:05       unanswered
2 open, 1 done
```

`qa list`は、質問の末尾に状態を表示し、その下に最新の回答を、字下げして、記録者と時刻とともに表示する。

| 状態 | 意味 |
|---|---|
| `unanswered` | 未クローズで、回答なし |
| `N answers, awaiting confirmation` | 未クローズで、回答あり |
| `N answers, done` | 閉じていて、回答あり（`--all`） |
| `done without answers` | 閉じていて、回答なし（`--all`） |

質問がジャーナルにない回答は、表示しない。

```
$ mtqg glossary list
5b7e2c9a41  トークン          字句解析で切り出す最小単位                  yamada       09:00
f29d0da995  ブロックコメント  /* と */ で囲むコメント                     yamada       09:30
0cb1e29c65  ブロックコメント  複数行にわたって書けるコメント              claude-code  10:00
f28c105d1f  字句解析          ソースを読み、トークンの並びに変換する処理  claude-code  11:24
4 terms (1 with duplicate definitions)
```

`glossary list`は、すべての項目を表示する：ID、用語、定義、記録者、時刻。同じ用語の項目は、書かれた順に2行
並び、フッターが2回以上定義された用語の数を伝える（`4 terms (1 with duplicate definitions)`）。
mtqgはどちらかを選ばない。

一覧の表示のしかた：

- 列は、ID、本文、記録者、時刻。時刻は、今日なら`HH:MM`、それ以外の日は`YYYY-MM-DD`（ローカル時間）
- 記録は、作った順（古いものが先）
- 本文は、記録の1行目。端末に出すときは、ウィンドウの幅に合わせて`...`で切る。パイプやファイルに出すときは、
  切らない
- 列は表示の幅で揃える：全角文字（たとえば日本語）は2列を取る
- `--all`のとき、終わった項目の末尾に`done`が付く
- 記録に含まれる制御文字（エスケープ文字など）は、表示するときに U+FFFD に置き換える。記録が端末の
  動作を変えられないようにするため。`--json`の出力は影響を受けない
- `mtqg memo list`は、すべてのmemoを表示し、`N memos`で終わる

### show

```
$ mtqg show 1012f037b6
question  1012f037b6  open
by claude-code (ai), 2026-09-21 09:10

  ブロックコメントの入れ子に対応する？

Answers (2)
  ae2eb1547f  claude-code (ai)  2026-09-21 09:15
    一般的には対応するのが望ましい
  95e761d177  yamada (human)    2026-09-21 09:41
    初版では非対応。需要が出たら再検討

Events
  2026-09-21 09:10  create  claude-code (ai)
  2026-09-21 09:15  create  claude-code (ai)  answer ae2eb1547f
  2026-09-21 09:41  create  yamada (human)    answer 95e761d177
```

- 1行目は、種類（`memo`、`todo`、`question`、`answer`、`glossary`）、ID、todoと質問なら状態。2行目は、誰がいつ
  書いたか（記録者の種別つき）
- 本文は、どう読まれる出力でも、すべての行を全文で表示する。制御文字は一覧と同じように置き換える。glossaryの
  項目は、定義の前に`Word: <用語>`を表示する。回答は、属する質問を表示する
- 質問は、回答を古いものから、記録者と時刻とともに並べる
- `Events`は、その記録に起きたことを、古いものから、ローカル時間の日付と時刻つきで並べる：その記録自身の
  イベント（`create`、`status`は`open -> done`の形、`edit`、`delete`）と、質問なら各回答の作成

### log

```
$ mtqg log --limit 6
11:32  todo      2e44158bae  コメント処理のテストケースを追加                      claude-code
11:30  todo      1818e81189  READMEに対応している構文を書く                        yamada
11:24  glossary  f28c105d1f  字句解析: ソースを読み、トークンの並びに変換する処理  claude-code
11:06  todo      1e27a1c08a  エラー位置を行と列で表示する                          claude-code
11:05  question  2217beaddb  エラー位置は行と列の両方を出しますか？                claude-code
10:52  todo      6513270e26  文字列リテラル中の // を無視する                      yamada
6 of 16 records (--limit 0 for all)

$ mtqg log --kind qa
11:05       question  2217beaddb  エラー位置は行と列の両方を出しますか？              claude-code
09:41       answer    95e761d177  (to 1012f037b6) 初版では非対応。需要が出たら再検討  yamada
09:15       answer    ae2eb1547f  (to 1012f037b6) 一般的には対応するのが望ましい      claude-code
09:10       question  1012f037b6  ブロックコメントの入れ子に対応する？                claude-code
2026-09-19  answer    d4c2a1e3f5  (to c3b1f0d2e4) MITが一番単純です                   claude-code
2026-09-19  question  c3b1f0d2e4  ライセンスは何にしますか？                          yamada       done
6 records
```

- 隠れていないすべての記録を、種類を問わず1件1行で、**新しいものから**表示する：時刻、種類（`memo`、`todo`、
  `question`、`answer`、`glossary`）、ID、本文、記録者。glossaryの項目は、用語、コロン、定義の順に表示する。回答は、
  属する質問の`(to <id>)`で始まる。終わったtodoと質問は、末尾に`done`が付く
- 時刻は、今日なら`HH:MM`、それ以外の日は`YYYY-MM-DD`で、記録を作った時刻。本文は一覧の決まりに従う（1行目だけ、
  端末に出すときだけ切る、制御文字は置き換える）
- `--limit N`は新しい方から`N`件を表示する。既定は20で、`0`は全件。`--kind K`は1つの種類だけを表示する：
  `memo`、`todo`、`qa`（質問と回答）、`glossary`、またはその1文字。どちらも`--limit=N`の形でも書ける。0以上の
  整数でない値や、存在しない種類は、コマンドラインの誤り
- 最後の行が件数を伝える：`N records`。省いたものがあるときは`N of M records (--limit 0 for all)`

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

```
$ mtqg version
mtqg v0.1.0
Repository format version: 0 (this mtqg supports up to 0)
```

- mtqgのバージョンは、ビルド時に指定した値、なければビルドのモジュールのバージョン（`go install ...@v0.1.0`なら
  `v0.1.0`のようなタグ、`go build`ならコミットの時刻とハッシュでできたバージョン）、なければ`dev`
- リポジトリの外、または`.mtqg/`のない場所では、2行目は
  `Repository format version: unknown (no .mtqg/ found)`になる

形式のバージョンは`0`（未確定）で、それを上げるコマンドはまだない。形式1以降ができたときに用意する（[schema_ja.md](schema_ja.md#バージョン)）。

## 記録者

| 項目 | 値 |
|---|---|
| 記録者名 | `MTQG_AUTHOR_NAME`、なければ`git config user.name` |
| 記録者の種別 | `MTQG_AUTHOR_KIND`（`human`か`ai`）、なければ`human` |
| エディタ | `$EDITOR` |

- コマンドラインから記録するAIエージェントは、2つの変数で名乗る：
  `MTQG_AUTHOR_KIND=ai MTQG_AUTHOR_NAME=claude-code`。`ai`のときは名前が必須。
  AIが`git config`の名前で記録されることはない
- `git config user.name`は、そのリポジトリについて読む（`git -C <ルート> config user.name`）ので、
  そのリポジトリの設定が効く。どちらからも名前が得られなければ、止まって、設定の方法を伝える。
  `MTQG_AUTHOR_KIND`が`human`でも`ai`でもなければエラー

変数は設定ファイルではない。設定ファイルはない。
