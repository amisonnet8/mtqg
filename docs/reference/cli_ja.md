# mtqg コマンドリファレンス

*[English](cli.md) | **日本語***

> 英語版 `cli.md` の日本語訳。内容は英語版と同じに保つ。食い違う場合は英語版が正。

```
mtqg <種類> <動詞> [引数]
mtqg <コマンド> [引数]
```

mtqg自身が出す文言は英語。記録の中身は書いたとおりに表示する。保存の形式は[schema_ja.md](schema_ja.md)にある。

## 種類と動詞

| 種類 | `add` | `done` | `reopen` | `list` |
|---|---|---|---|---|
| memo（`m`） | `add <本文>` | — | — | 全件 |
| todo（`t`） | `add <本文>` | `done <id>` | `reopen <id>` | 未完了（`--all`で全件） |
| qa（`q`） | `add <質問>`<br>`add <質問id> <回答>` | `done <質問id>` | `reopen <質問id>` | 未クローズ（`--all`で全件） |
| bug（`b`） | `add <バグ>`<br>`add <バグid> <返信>` | `done <バグid>` | `reopen <バグid>` | 未クローズ（`--all`で全件） |
| glossary（`g`） | `add <用語> <定義>` | — | — | 全件 |

- 種類は1文字に略せる。`mtqg t add ...`は`mtqg todo add ...`と同じ。動詞は常に必要
- 種類の違う記録に動詞を使う（たとえばtodoのIDに`mtqg qa done`）とエラーになり、正しいコマンドを示す
- `qa`と`bug`は同じように動く。[質問、回答、バグ、返信](#質問回答バグ返信)を参照

## その他のコマンド

| コマンド | 内容 |
|---|---|
| `mtqg edit <id> [<本文>]` | 記録の本文を置き換える。glossaryは定義を置き換え、用語は変えられない。本文がなければ`$EDITOR`が開く |
| `mtqg delete <id>` | 記録を隠す。質問やバグを消すと、その回答や返信も隠れる |
| `mtqg undo` | この記録者がこの端末から書いた最後の行を消す |
| `mtqg status` | 未完了の項目と未コミットの記録の概況 |
| `mtqg log [--limit N] [--kind K]` | 全種類の記録を、新しいものから |
| `mtqg show <id>` | 1件の記録を、全文と履歴とともに |
| `mtqg search <語>` | 本文に`<語>`を含む記録を、新しいものから |
| `mtqg review` | 並行した状態変更、用語の重複定義、親のない回答・返信 |
| `mtqg context [--max-tokens N]` | AIエージェント向けの要約 |
| `mtqg format [--mark] [ファイル]` | 任意のテキストに含まれるイベント行を整形して表示する |
| `mtqg archive <開始>..<終了> [-n]` | 期間内の項目を視界から外す（`-n`：報告だけ） |
| `mtqg init` | `.mtqg/`を作る |
| `mtqg version` | mtqgのバージョンと、リポジトリの形式のバージョンを表示する |
| `mtqg help` | コマンドの一覧を表示する（`-h`、`--help`も同じ） |

## 共通のオプション

| オプション | 内容 |
|---|---|
| `-C <パス>` | カレントディレクトリの代わりに`<パス>`から`.mtqg/`を探す |
| `--json` | 機械可読の出力（すべてのコマンド）。[JSON出力](#json出力)を参照 |
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

## JSON出力

`--json`はプログラム（エディタの拡張、フック、スクリプト）のためのもの。出力は約束であり、変えるときはフィールドを足すだけにする。読む側は、知らないフィールドを無視する。（形式はv1まで`0`で、[schema_ja.md](schema_ja.md#バージョン)にある。それまではこの約束もまだ固定ではない。）

- 出力は**JSONオブジェクト1つ**。2スペースで字下げし、改行で終える。最初のフィールドは`command`で、打ったコマンドを略さずに書く（`todo list`、`qa add`、`log`）
- キーは`snake_case`。値のないフィールドは`journal.jsonl`と同じく省く。件数は省かない
- **IDは、`--full-id`の有無にかかわらず、常に完全な32桁。** **時刻はUTC**で、`journal.jsonl`と同じ形（`2026-09-21T10:18:00Z`）。ローカル時間で見せるのは読む側の仕事
- **本文は書かれたとおり。** 制御文字を置き換えず、窓の幅で切らず、色も付けない。`--no-color`と`--full-id`は何も変えない
- 出力は標準出力へ。成功したとき、標準エラー出力へは警告（後述）のほか何も出さない。隠された記録は出ない

記録はオブジェクトで、次のフィールドを持つ。

| フィールド | 意味 |
|---|---|
| `id` | 完全なID |
| `kind` | `memo`、`todo`、`question`、`answer`、`bug`、`reply`、`glossary`のいずれか |
| `word` | 用語（glossaryのみ） |
| `text` | 本文の全文（glossaryでは定義） |
| `re` | 回答・返信が向かう質問・バグのID（回答と返信のみ） |
| `status` | `open`または`done`（todo、質問、バグのみ） |
| `author` | `{"kind": "human"または"ai", "name": "..."}` |
| `created` | 最初のイベントの時刻 |
| `updated` | 最後のイベントの時刻 |

```
$ mtqg memo list --json
{
  "command": "memo list",
  "records": [
    {
      "id": "81e74ef5e8e24d949ed904759531985d",
      "kind": "memo",
      "text": "エラーメッセージは英語で統一する方針",
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

`qa list`・`bug list`・`show`に出る質問とバグは、`replies`（回答または返信を、古い順に、記録として）も持つ。（人間向けの`qa list`は最新の1件だけを出すが、`--json`は全件を返す。）

`command`のあとに、コマンドごとに次のものが出る。

| コマンド | フィールド |
|---|---|
| `memo add`、`todo add`、`qa add`、`bug add`、`glossary add` | `record`：書いた記録 |
| `todo done`、`todo reopen`、`qa done`、... | `record`と`changed`（すでにその状態で、何も書かなかったときは`false`） |
| `edit` | `record`（新しい本文で）と`changed`（本文が同じで、何も書かなかったときは`false`） |
| `delete` | `record`：隠した記録、`hidden_replies`：一緒に隠れた回答・返信を、記録として（なければ`[]`） |
| `undo` | `event`：消した行を[schema_ja.md](schema_ja.md)の形で、`record`：その行が属する記録の、消す前の姿（ジャーナルにあれば） |
| `search` | `query`、`records`（新しい順）、`count` |
| `review` | `concurrent_status_changes`：記録ごとの`{"record", "changes"}`（`changes`は`journal.jsonl`の行）、`duplicate_words`：`{"word", "records"}`、`unattached_replies`：`{"record", "re_record"}`（`re`がジャーナルの何も指さないときは`re_record`を出さない）。どれも、なければ`[]` |
| `archive` | `range`：`{"start", "end"}`（読んだとおりの`YYYY-MM-DD`）、`file`：アーカイブのファイル（リポジトリからの相対）、`dry_run`、`archived`：件数`memos`・`todos`・`questions`・`answers`・`bugs`・`replies`・`glossary_entries`（削除したもの）・`records`・`lines`（移した行数）、`skipped`：件数`open_todos`・`open_questions`・`open_bugs`・`glossary_entries`・`records`。`archived.records`が0のときファイルは作らない |
| `format` | `events`（時刻順。`journal.jsonl`の行として。入力で印があった行は`mark`（`+`か`-`）を持つ）、`count` |
| `memo list` | `records`、`count` |
| `todo list` | `records`（`--all`で終わったものも含む）、`open`、`done`（`--all`にかかわらず、見える記録すべての件数） |
| `qa list`、`bug list` | `todo list`と同じ。各記録が`replies`を持つ |
| `glossary list` | `records`、`entries`（その数）、`duplicate_words` |
| `log` | `records`（新しい順）、`shown`、`total` |
| `show` | `record`と`events`：その記録に起きたことを、古い順に、[schema_ja.md](schema_ja.md)の形の`journal.jsonl`の行として（質問・バグでは、各返信の`create`も含む） |
| `status` | `open_todos`、`open_questions`、`questions_awaiting_confirmation`、`open_bugs`、`bugs_awaiting_confirmation`、`glossary_entries`、`duplicate_words`、`concurrent_status_changes`（記録の数）、`uncommitted_records`（gitを実行できなければ`null`） |
| `init` | `root`：`.mtqg/`を作った場所 |
| `version` | `mtqg`：バージョン、`format`：`{"repository": Nまたはnull, "supported": N}`（`.mtqg/`がなければ`null`） |
| `help`、またはコマンドへの`-h` | `kinds`：`{"name", "short"}`、`commands`：`{"command", "usage", "summary", "available"}`。`available`は、名前は知っているがまだ作っていないコマンドでは`false` |
| `context` | [context](#context)を参照 |

**エラー**は、**標準エラー出力**に1行のJSONで出し、標準出力は空のまま。終了コードは`--json`なしと同じ。

```
$ mtqg show zzzz --json
{"error":{"kind":"not_found","message":"No record matches \"zzzz\"","prefix":"zzzz"}}
```

`kind`は何が起きたかを、`message`はmtqgが出したはずの文章（複数行は改行でつないだもの）を表す。ほかのフィールドは`kind`による。

| `kind` | 終了コード | ほかのフィールド |
|---|---|---|
| `usage`（コマンドラインの誤り） | 2 | |
| `not_available`（まだ作っていないコマンド） | 1 | |
| `not_in_repository`、`not_initialized`、`already_initialized`、`format_too_new`、`conflict_markers`、`lock_timeout` | 1 | |
| `no_author`、`bad_author_kind`、`empty_text`、`empty_word`、`invalid_text`、`input`、`editor`、`git_unavailable` | 1 | |
| `not_found` | 1 | `prefix` |
| `id_too_short` | 1 | `prefix` |
| `ambiguous` | 1 | `prefix`、`candidates`：IDが指しうる記録 |
| `wrong_kind` | 1 | `record`：IDが実際に指すもの、`wanted`：コマンドが対象とする種類 |
| `no_state`、`no_replies` | 1 | `record` |
| `nothing_to_undo` | 1 | |
| `has_later_events`（`undo`すると、記録のないイベントが残る） | 1 | `record` |
| `unknown`（それ以外） | 1 | |

**警告**（飛ばした行、衝突マーカー、実行できないgit）も、1件ずつ1行で標準エラー出力に出し、終了コードは変えない。

```
$ mtqg todo list --json >/dev/null
{"warning":{"kind":"invalid_json","line":12,"message":"warning: .mtqg/journal.jsonl line 12 is not a valid JSON object; skipped it"}}
```

`kind`は`invalid_json`、`invalid_utf8`、`missing_field`、`conflict_marker`、`no_trailing_newline`、`unreadable`、`git_unavailable`のいずれかで、`line`は`journal.jsonl`の行がある場合にその行。最初の5件のあとは、あと何件あったかを1行で言う（`kind`は`more`で、`count`を持つ）。

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
mtqg b add 空の入力でパーサーが落ちる
mtqg g add トークン 字句解析で切り出す最小単位
```

- 残りの引数は空白でつないで1つの本文にする。シェルの特殊文字（`#` `*` `(` `)` `&` `|` `<` `>`）を含む場合を除き、引用符は要らない
- glossaryは、最初の引数が用語、残りが定義。複数の語でできた用語は引用符が要る：
  `mtqg g add "block comment" /* と */ で囲むコメント`
- `mtqg q add <質問id> <本文>`は、質問ではなく回答を足し、`mtqg b add <バグid> <本文>`は、バグではなく返信を足す
  （[質問、回答、バグ、返信](#質問回答バグ返信)）
- 本文の代わりに`-`を渡すと、標準入力を最後まで読む。末尾の改行は落とす：`git log -1 --format=%s | mtqg m add -`
- **書く本文がないときは、`$EDITOR`が開く**。空のファイルが開き、保存した内容が本文になる（末尾の改行は落とす）：
  引数がまったくないとき（`mtqg m add`、`mtqg t add`、`mtqg q add`、`mtqg b add`）、IDのあとの回答・返信がないとき
  （`mtqg q add <質問id>`、`mtqg b add <バグid>`）、用語のあとの定義がないとき（`mtqg g add <用語>`）。
  `$EDITOR`には引数や引用符を含められる（`code --wait`）。シェルは通さない。`$EDITOR`が設定されていなければ、止まって
  そう伝える。どの記録にも当てはまらないIDは、エディタが開く前に止まる
- 足りない語があるのはコマンドラインの誤りで、何も書かない：`mtqg g add`（用語がない）。回答・返信・定義の本文は`-`にでき、
  標準入力から読む
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
- `mtqg q done`と`mtqg q reopen`は質問に対して、`mtqg b done`と`mtqg b reopen`はバグに対して、同じことをする
- 状態を持つのはtodo・質問・バグだけ。それ以外の記録のIDには、止まって、それが何かを伝える。
  正しいコマンドが別にあるときは、それを示す：
  `81e74ef5e8 is a memo, not a todo`、`2217beaddb is a question, not a todo; use `mtqg qa done 2217beaddb``、
  `7f3a2b1c09 is a bug, not a question; use `mtqg bug done 7f3a2b1c09``

## 質問、回答、バグ、返信

質問とバグは同じ形をしている：回答を付けられる記録で、閉じるまで未クローズのまま。違いは、何について書くか。質問は
何かを尋ねる。バグは、動かないものの報告と、それについてのやり取りで、直ったとき、またはもう追わないときに閉じる。
以下はどちらにも当てはまる。質問の「質問」「回答」の代わりに、バグでは「バグ」「返信」と読む。

- `mtqg q add <本文>`で質問を足す。`mtqg q add <質問id> <本文>`で、その質問に回答を足す。`mtqg b add <本文>`で
  バグを足し、`mtqg b add <バグid> <本文>`で、そのバグに返信を足す。回答と返信は独自のIDを持ち、その`re`には
  質問またはバグの完全なIDが入る
- 回答することと閉じることは別。`mtqg q done <質問id>`で質問を閉じ、`mtqg b done <バグid>`でバグを閉じる。
  回答や返信はいくつでも足せて、どれも他を置き換えない。閉じた質問やバグにも、回答や返信は足せる
- 回答や返信を持てるのは質問とバグだけ。回答や返信に回答は付けられない。回答は常に質問へ、返信は常にバグへ付く：
  `mtqg b add <質問id> <本文>`は、そのIDが質問であることを伝えるエラーになる

### 質問か、回答か。バグか、返信か

`q add`と`b add`は、最初の引数だけで両者を見分ける。最初の引数が**16進数の4桁以上で、それだけでできている**
（`0-9`、`a-f`。大文字は小文字として読む）なら、それは回答先の質問（または返信先のバグ）のIDで、残りが回答（または返信）の
本文になる。それ以外の最初の引数なら、全体が新しい質問（またはバグ）の本文になる。

| 16進数4桁以上の最初の引数が当てはまるもの | 結果 |
|---|---|
| 質問1件（`q add`）またはバグ1件（`b add`） | 残りを、それへの回答または返信として足す |
| 質問でない（`q add`）、またはバグでない（`b add`）記録1件 | エラー。それが何かを伝える |
| 複数の記録 | エラー。候補を完全なIDで並べる |
| 記録がない | エラー。何も書かない |

```
$ mtqg q add a8ec はどういう意味ですか？
No record matches "a8ec". A first word of 4 or more hex digits is read as the ID of the question to answer.
To ask a question that starts with it, put the whole text in quotes: mtqg qa add "a8ec ..."
```

`b add`では、同じエラーが`... the ID of the bug to reply to.`と
`To report a bug that starts with it, ...: mtqg bug add "a8ec ..."`になる。

- 引用符で括った本文は1つの引数になる。中に空白があれば16進数だけではないので、IDとは読まれない：
  `mtqg q add "a8ec はどういう意味ですか？"`は質問、`mtqg q add a8ec 見つからない場合のエラーコードです`は
  質問`a8ec`への回答
- 当てはまる記録がないときは、新しい記録にせずエラーにする。打ち間違えたIDが、黙って新しい質問やバグになることが
  ないようにするため。最初の語が16進数の文字だけでできた英語の本文（`Face detection is slow. Why?`、
  `Dead code: remove it?`）も同じように止まる。本文全体を引用符で括る。そのような語1つだけの本文は、
  標準入力から渡せる
- IDだけで、後ろに本文がないときは、`$EDITOR`が開いて本文を書く（[記録を足す](#記録を足す)）

質問とバグは、4つの状態のどれかにある。

| 状態 | 回答または返信 | 閉じている | `qa list`・`bug list`での表示 |
|---|---|---|---|
| 未回答 | 0 | いいえ | 表示する |
| 確定待ち | 1以上 | いいえ | 表示する（回答または返信の件数付き） |
| 回答済み | 1以上 | はい | `--all`で表示 |
| 回答なしで閉じた | 0 | はい | `--all`で表示 |

| 渡したID | できること |
|---|---|
| 質問またはバグ | `add`（回答または返信）、`done`、`reopen`、`edit`、`delete`（回答や返信も隠れる） |
| 回答または返信 | `edit`、`delete`（その回答や返信だけ。質問やバグの状態は変わらない） |

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
$ mtqg edit 6cad4a268d ブロックコメントと行コメントの読み飛ばし
Edited: 6cad4a268d  ブロックコメントと行コメントの読み飛ばし
$ mtqg edit 6cad4a268d ブロックコメントと行コメントの読み飛ばし
Unchanged: 6cad4a268d  ブロックコメントと行コメントの読み飛ばし

$ mtqg delete 1012f037b6
Deleted: 1012f037b6  ブロックコメントの入れ子に対応する？
2 answers are also hidden (claude-code, yamada)
The lines remain in the journal and in git history
```

どちらもイベントを追記する。ファイルからもgitの履歴からも何も消えない。

- `edit`はどの記録の本文でも置き換える：memo、todo、質問、回答、バグ、返信、glossaryの項目の定義。変わるのは本文だけ。
  glossaryの用語は変えられず、todo・質問・バグの状態も変わらない。出す行は、IDと新しい本文の1行目
- 本文は記録を足すときと同じ渡し方：IDのあとの語を空白でつなぐか、標準入力なら`-`。**本文がなければ、その記録の今の本文を
  入れた`$EDITOR`が開き**、保存した内容が新しい本文になる（[記録を足す](#記録を足す)）
- 新しい本文が今の本文と同じなら、何も書かず、`Unchanged: ...`と言う。終了コードは0。本文が空、または空白だけのときは
  エラーで、何も書かない
- `delete`はどの記録でも隠す：どの一覧、`show`、`search`にも出ず、そのIDは何にも当てはまらなくなる。質問やバグを消すと、
  その回答や返信も隠れ、出力は何件が誰のものかを言う。回答や返信を消すと、その1件だけが隠れ、質問やバグの状態は変わらない。
  最後の行は、行がジャーナルにもgitの履歴にも残ることを、必ず言う

## undo

**今の記録者が今の端末から書いた最後の行**を消す。直前のミス（たとえば、質問IDを付け忘れて、回答を新しい質問として足してしまった）のためのもの。

```
$ mtqg q add 初版では非対応。需要が出たら再検討
acbf90978f
$ mtqg undo
Undone: qa add "初版では非対応。需要が出たら再検討" (acbf90978f)
```

- 対象：`journal.jsonl`の行のうち、この記録者（種別と名前。記録を書くときと同じ決め方）と、この端末の`tty`の値を持つものの
  中で`ts`が最後のもの。同じ秒のものは、ファイルの中で最後に書かれたもの。どの行でもよい：`add`、`done`、`reopen`、
  `edit`、`delete`が書いた行。`Undone:`の後には、記録の`type`、その行がしたこと（`add`、`done`、`reopen`、`edit`、
  `delete`）、本文、IDを出す。時刻は、その行が書かれたときの時刻：進んだ時計で書かれた行は、時刻が追いつくまで最新になる
- **端末。** `tty`は、ハッシュの16進8桁で、端末を見分けるだけで、ほかのことは分からない。`MTQG_TTY`があればそれから作る
  （どんな文字列でもよい：2つのセッションを分けたければ違う値を、同じにしたければ同じ値を渡す）。なければ、LinuxとmacOSでは、
  標準入力・標準出力・標準エラー出力のうち端末につながっているものから作る。Windowsと、どれも端末でないとき（パイプで
  コマンドを走らせるエージェント）は、その行に`tty`がなく、`tty`のない行は同じ端末とみなす。つまり、AIエージェントが
  パイプ越しに書いた行と、人が端末で書いた行は、同じ名前でも互いの対象にならない
- 1段だけ。`undo`は繰り返せない。何を消したかは必ず出す
- **記録がなくなってイベントだけが残るときは、断る。** 消す行が、ほかのイベント（状態変更、`edit`、`delete`、または
  その記録への回答・返信）を持つ記録の`create`なら、何も消さず、いくつあるかを言って、`mtqg delete`を案内する。`status`・`edit`・
  `delete`の行は、その後に何があっても消す

```
$ mtqg undo
Cannot undo: todo c4a225e916 has 2 other events, and undoing its creation would leave them without a record
To hide it instead: mtqg delete c4a225e916
```

- 対象の行がなければ、`Nothing to undo: ...`。終了コードは1
- コミット済みかどうかは見ない。すでに共有したものには`delete`を使う。他のブランチに届いた行を消すと、次のマージで戻ってくることがある。
  ほかの行は、1バイトも変えずに残る

## 読む

### status

```
$ mtqg status
Open todos          5
Open questions      2  (1 awaiting confirmation)
Open bugs           1  (1 awaiting confirmation)
Glossary            4  (1 with duplicate definitions)
Conflicts           1  (concurrent status changes; see mtqg review)

Uncommitted records 3
```

- 各行は件数。`Open questions`は閉じていない質問の数で、`awaiting confirmation`はそのうち回答のあるもの。
  `Open bugs`は閉じていないバグの数（`awaiting confirmation`はそのうち返信のあるもの）。`Glossary`は用語の項目の数で、`with duplicate definitions`は2回以上定義された用語（文字まで同じもの）の数。
  `Conflicts`は並行した状態変更のある記録の数（[review](#review)）で、なければこの行は出さない。
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

$ mtqg bug list
7f3a2b1c09  空の入力でパーサーが落ちる  yamada       10:41  1 reply, awaiting confirmation
          └ macOSでも再現した           claude-code  10:45
1 open (show done: --all)

$ mtqg bug list --all
b2c3d4e5f6  タブ文字でリンターが落ちる       yamada       2026-09-19  1 reply, done
          └ タブを1桁として数えるよう直した  claude-code  2026-09-19
7f3a2b1c09  空の入力でパーサーが落ちる       yamada       10:41       1 reply, awaiting confirmation
          └ macOSでも再現した                claude-code  10:45
1 open, 1 done
```

`qa list`と`bug list`は同じ配置で表示する。質問やバグの末尾に状態を表示し、その下に最新の回答または返信を、
字下げして、記録者と時刻とともに表示する。

| 質問の状態 | バグの状態 | 意味 |
|---|---|---|
| `unanswered` | `no replies` | 未クローズで、何も足されていない |
| `N answers, awaiting confirmation` | `N replies, awaiting confirmation` | 未クローズで、回答または返信あり |
| `N answers, done` | `N replies, done` | 閉じていて、回答または返信あり（`--all`） |
| `done without answers` | `done without replies` | 閉じていて、何も足されていない（`--all`） |

質問がジャーナルにない回答、バグがジャーナルにない返信は、表示しない。

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

$ mtqg show 7f3a2b1c09
bug  7f3a2b1c09  open
by yamada (human), 2026-09-21 10:41

  空の入力でパーサーが落ちる

Replies (1)
  3d8e4a0b12  claude-code (ai)  2026-09-21 10:45
    macOSでも再現した

Events
  2026-09-21 10:41  create  yamada (human)
  2026-09-21 10:45  create  claude-code (ai)  reply 3d8e4a0b12
```

- 1行目は、種類（`memo`、`todo`、`question`、`answer`、`bug`、`reply`、`glossary`）、ID、todo・質問・バグなら状態。2行目は、誰がいつ
  書いたか（記録者の種別つき）
- 本文は、どう読まれる出力でも、すべての行を全文で表示する。制御文字は一覧と同じように置き換える。glossaryの
  項目は、定義の前に`Word: <用語>`を表示する。回答は属する質問を、返信は属するバグを表示する。`re`が指す記録がなかったり
  （`to bug 7f3a2b1c09  (no such record)`）、別の種類だったり（`to 1012f037b6  (a question, not a bug)`）するときは、
  親の名前は出さず、そのことを言う：そのような記録は、それへの返信ではない（[review](#review)）
- 質問は回答を、バグは返信を（`Replies (2)`）、古いものから、記録者と時刻とともに並べる
- `Events`は、その記録に起きたことを、古いものから、ローカル時間の日付と時刻つきで並べる：その記録自身の
  イベント（`create`、`status`は`open -> done`の形、`edit`、`delete`）と、質問やバグなら各回答・各返信の作成

### log

```
$ mtqg log --limit 6
11:32  todo      2e44158bae  コメント処理のテストケースを追加                      claude-code
11:30  todo      1818e81189  READMEに対応している構文を書く                        yamada
11:24  glossary  f28c105d1f  字句解析: ソースを読み、トークンの並びに変換する処理  claude-code
11:06  todo      1e27a1c08a  エラー位置を行と列で表示する                          claude-code
11:05  question  2217beaddb  エラー位置は行と列の両方を出しますか？                claude-code
10:52  todo      6513270e26  文字列リテラル中の // を無視する                      yamada
6 of 22 records (--limit 0 for all)

$ mtqg log --kind qa
11:05       question  2217beaddb  エラー位置は行と列の両方を出しますか？              claude-code
09:41       answer    95e761d177  (to 1012f037b6) 初版では非対応。需要が出たら再検討  yamada
09:15       answer    ae2eb1547f  (to 1012f037b6) 一般的には対応するのが望ましい      claude-code
09:10       question  1012f037b6  ブロックコメントの入れ子に対応する？                claude-code
2026-09-19  answer    d4c2a1e3f5  (to c3b1f0d2e4) MITが一番単純です                   claude-code
2026-09-19  question  c3b1f0d2e4  ライセンスは何にしますか？                          yamada       done
6 records

$ mtqg log --kind bug
10:46       reply  9a8b7c6d5e  (to 1012f037b6) 空のファイルでも落ちる           claude-code
10:45       reply  3d8e4a0b12  (to 7f3a2b1c09) macOSでも再現した                claude-code
10:41       bug    7f3a2b1c09  空の入力でパーサーが落ちる                       yamada
2026-09-19  reply  d4e5f6a7b8  (to b2c3d4e5f6) タブを1桁として数えるよう直した  claude-code
2026-09-19  bug    b2c3d4e5f6  タブ文字でリンターが落ちる                       yamada       done
5 records
```

- 隠れていないすべての記録を、種類を問わず1件1行で、**新しいものから**表示する：時刻、種類（`memo`、`todo`、
  `question`、`answer`、`bug`、`reply`、`glossary`）、ID、本文、記録者。glossaryの項目は、用語、コロン、定義の順に
  表示する。回答と返信は、属する質問やバグの`(to <id>)`で始まる。終わったtodo・質問・バグは、末尾に`done`が付く
- 時刻は、今日なら`HH:MM`、それ以外の日は`YYYY-MM-DD`で、記録を作った時刻。本文は一覧の決まりに従う（1行目だけ、
  端末に出すときだけ切る、制御文字は置き換える）
- `--limit N`は新しい方から`N`件を表示する。既定は20で、`0`は全件。`--kind K`は1つの種類だけを表示する：
  `memo`、`todo`、`qa`（質問と回答）、`bug`（バグと返信）、`glossary`、またはその1文字。どちらも`--limit=N`の形でも書ける。0以上の
  整数でない値や、存在しない種類は、コマンドラインの誤り
- 最後の行が件数を伝える：`N records`。省いたものがあるときは`N of M records (--limit 0 for all)`

### search

```
$ mtqg search コメント
11:32  todo      2e44158bae  コメント処理のテストケースを追加                  claude-code
10:18  todo      6cad4a268d  ブロックコメント /* */ の読み飛ばし               claude-code
10:00  glossary  0cb1e29c65  ブロックコメント: 複数行にわたって書けるコメント  claude-code
09:50  todo      6b0d549b6f  行コメント // の読み飛ばし                        yamada       done
09:30  glossary  f29d0da995  ブロックコメント: /* と */ で囲むコメント         yamada
09:10  question  1012f037b6  ブロックコメントの入れ子に対応する？              claude-code
6 records contain "コメント"
```

- `mtqg search <語>`は、本文に`<語>`を含む記録を出す：見える記録すべての本文（種類を問わない）と、glossaryの項目ではその
  用語も。`search`のあとの語は、記録のときと同じく空白でつなぐ
- 大文字と小文字は区別しない。ほかには何もない：パターンも、語の境界も、絞り込み（種類、記録者、期間）もない
- 行は[log](#log)と同じで、**新しいものから**、当てはまるものをすべて出す。行に出す本文は記録の1行目なので、本文の2行目以降で
  当たったものは、見つかっても見えない：`show`を使う。最後の行が件数を言う：`N records contain "<語>"`。当たるものがなければ
  `No records contain "<語>"`で、終了コードは0
- 削除した記録は探さない。`archive/`も探さない

### review

人の目が要るところ：ジャーナルに、食い違う事実があるところ。mtqgはそれを見せるだけで、どれが正しいかは決めない。
中身のない区画は出さず、何もなければ`Nothing to review`。終了コードはどちらでも0。

```
$ mtqg review
Concurrent status changes (1)
  todo 6b0d549b6f "行コメント // の読み飛ばし"
    2026-09-21 10:15  claude-code  open -> done
    2026-09-21 14:30  yamada       open -> done

Duplicate glossary definitions (1)
  ブロックコメント
    f29d0da995  yamada       /* と */ で囲むコメント
    0cb1e29c65  claude-code  複数行にわたって書けるコメント

Answers and replies with no parent (1)
  9a8b7c6d5e  reply  claude-code  空のファイルでも落ちる
    re 1012f037b6: a question, not a bug
```

- **並行した状態変更。** 1つの記録の状態変更を、形式の順（`ts`、次に`id`）に取り、記録が作られたときの状態から追う。`from`が、その時点の
  記録の状態と違う変更は、その前の変更を見ていない人が書いたもの（同じtodoをそれぞれ閉じた2つのブランチを、あとでマージした、など）。
  その記録は、状態変更を**すべて**、古いものから、時刻・記録者・変更とともに並べる。`from`のない変更は判定しない。同じ行の繰り返しは
  1つのイベントで、食い違いではない。一覧が出す状態は、最後の変更が残した状態
- **用語の重複定義。** 2回以上定義された用語（文字まで同じもの）ごとに、その項目すべてを、書かれた順に
- **親のない回答・返信。** 回答・返信が質問・バグに結び付くのは、`re`が、同じ`type`の、返信を付けられる記録を指すときだけ。
  ジャーナルにない記録や、別の種類の記録を指す`re`は、どの質問・バグの下にも出ず、ここで見つかる（`log`と、IDを渡した`show`にも出る）。
  各行は、`re`が指すものを言う：`not in the journal`、または、それが何で、何であるべきか

### context

AIエージェントがセッションの始めに読むもの。ここまでの過程（何が未完了か、何に答えが出たか、どの用語が合意されているか）を渡す。プロジェクトの説明、現在の仕様、ビルドの手順は含めない。それらはREADMEやエージェントの指示ファイルの役目。

```
$ mtqg context
# mtqg context — sample-parser (main)

This is the process record of this project. Read the following before you start working.
- Respect what has been decided (answered questions, memos stating a policy)
- Do not decide open questions on your own; confirm them
- Use terms as defined in the glossary
- Record questions, decisions, findings, bugs, and todos with mtqg as they come up

## Attention
- Glossary term "ブロックコメント" has conflicting definitions (see mtqg glossary list)
- todo 6b0d549b6f "行コメント // の読み飛ばし" has concurrent status changes (see mtqg review)
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

分量を小さくすると、削ったものはその区画の中で言う：

```
$ mtqg context --max-tokens 380
# mtqg context — sample-parser (main)

This is the process record of this project. Read the following before you start working.
- Respect what has been decided (answered questions, memos stating a policy)
- Do not decide open questions on your own; confirm them
- Use terms as defined in the glossary
- Record questions, decisions, findings, bugs, and todos with mtqg as they come up

## Attention
- Glossary term "ブロックコメント" has conflicting definitions (see mtqg glossary list)
- todo 6b0d549b6f "行コメント // の読み飛ばし" has concurrent status changes (see mtqg review)
- 3 mtqg records are not committed

## Open todos (5)
- (5 older; see mtqg todo list)

## Open questions (2)
- 1012f037b6 ブロックコメントの入れ子に対応する？ (awaiting confirmation, claude-code, 09:10)
- 2217beaddb エラー位置は行と列の両方を出しますか？ (unanswered, claude-code, 11:05)
- (latest answers left out; see mtqg show <id>)

## Open bugs (1)
- 7f3a2b1c09 空の入力でパーサーが落ちる (awaiting confirmation, yamada, 10:41)
- (latest replies left out; see mtqg show <id>)

## Recent records (newest first)
- (22 more; see mtqg log)

## Glossary (4)
- トークン
- ブロックコメント (2 definitions)
- 字句解析
- (definitions left out; see mtqg glossary list)

---
Read full entries with mtqg show <id>.
```

- 区画はこの順：Attention、Open todos、Open questions、Open bugs、Recent records、Glossary。空の区画は出さない
- 最初の行はリポジトリ（`.mtqg/`のあるディレクトリの名前）とブランチ（`git branch --show-current`。detached HEADなどで無いときは出さない）。次に読み手への指示、区画、続きの読み方の行が並ぶ
- **Attention**は、行動が要るものを名指しする：定義が2つ以上ある用語と、並行した状態変更のある記録（[review](#review)。それぞれ先頭の5つ。残りは件数）と、コミットされていない記録の数（gitを実行できないときは、この行は出さない）
- 未完了のtodo・質問・バグは古い順で、ID、記録者、時刻（今日は`HH:MM`、別の日は日付。ローカル時間）を持つ。質問とバグは`unanswered`（バグは`no replies`）または`awaiting confirmation`（回答・返信があり、閉じていない）と書き、その下に最新の回答・返信を、記録者と記録者の種別とともに出す
- 最近の記録は、`log`と同じく、全種類の新しい記録を新しい順に、種類とIDとともに出す。回答・返信の終わりに、向かう質問・バグを書く。Glossaryは全項目を、IDとともに出す
- 本文は1行目を100文字で`...`で切ったもの。制御文字は一覧と同じく置き換える
- **分量。** `--max-tokens N`で決める（既定2000。`0`は上限なし）。**文字数からの見積もりで、トークン数そのものではない**：ASCIIの4文字を1トークン、それ以外の1文字を1トークンとして数える。冒頭の行、読み手への指示、Attention、見出し、最後の行、**最新の3件の質問と最新の3件のbug**は削らない：未決のことは見えていなければならない。上限を超えるときは、次の順に、必要な分だけ削る：最近の記録（10件、5件、3件、なし）、用語の定義（用語は残す）、最新の回答・返信、それぞれ最新の3件より古い質問とbug（2つの区画をあわせて、1件ずつ）、古いtodo（1件ずつ、なくなるまで）。それでも収まらないときは、そのまま出す
- 削ったものは、必ずその区画の中で、どこで読めるかとともに言う：`- (7 more; see mtqg log)`、`- (3 older; see mtqg todo list)`、`- (definitions left out; see mtqg glossary list)`、`- (latest answers left out; see mtqg show <id>)`。区画の見出しの件数は、見せた数ではなく、その区画の全部の数
- `--json`は、同じ削り方をしたあとの同じ内容を、構造にして返す。`command`のほかに、`repository`、`branch`、`attention`、`truncated`（何か削ったら`true`）、`max_tokens`（`0`のときは`null`）、`estimated_tokens`（文章の形の見積もり）と、区画ごとのオブジェクト`open_todos`、`open_questions`、`open_bugs`、`recent`、`glossary`（それぞれ`total`と`records`を持つ）。`open_questions`と`open_bugs`の記録は`reply_count`と、削っていなければ`latest_reply`を持つ。glossaryの記録は`definitions`（その用語の定義の数）を持ち、定義を削ったときは`id`と`text`を持たない。`attention`は`{"kind": "duplicate_word", "word": ...}`、`{"kind": "concurrent_status_change", "id": ..., "text": ...}`（完全なIDと全文）、`{"kind": "uncommitted", "count": N}`

## format

任意のテキストからイベント行を拾い、時刻順の1つの表にして、ローカル時間で表示する。ファイルか標準入力を読み、
`.mtqg/`は要らない：どこでも動く（引数がないか`-`なら標準入力）。

```
$ git show HEAD | mtqg format
2026-09-21 10:18  todo      6cad4a268d  ブロックコメント /* */ の読み飛ばし     claude-code
2026-09-21 10:32  memo      81e74ef5e8  エラーメッセージは英語で統一する方針    yamada
2026-09-21 11:05  question  2217beaddb  エラー位置は行と列の両方を出しますか？  claude-code
2026-09-21 11:06  todo      1e27a1c08a  エラー位置を行と列で表示する            claude-code
```

- 列は、ローカルの日付と時刻、その行がしたこと、ID、本文、記録者。記録を作る行なら、したことは種類：`memo`、`todo`、
  `question`、`answer`、`bug`、`reply`、`glossary`（回答・返信の本文は`(to <id>)`で、glossaryの項目は用語とコロンで
  始まる）。それ以外の行は、`done`か`reopen`（状態の変更）、`edit`、`delete`。`edit`は新しい本文を出す。状態の変更と
  `delete`は、その記録を作った行が同じ入力にあれば、その記録の本文を出し、なければ何も出さない
- 行は時刻順（`ts`、次に`id`。同じ記録では作成が変更より先）で、入力での順番によらない
- 先頭の`+`・`-`（unified diff）は、読む前に取り除く。diffが変更なしとして出す行（先頭が空白）は、その変更の一部ではないので
  表には出さない：同じ入力にある、同じ記録の状態変更や`delete`に、本文を貸すだけ
- JSONのイベントでない行は黙って飛ばす。`git show`や`git diff`の出力を丸ごと渡せる。警告は出さない
- どこから来た行でもよい：`git diff`、`git diff main...feature`、`cat .mtqg/journal.jsonl`、`grep ... .mtqg/journal.jsonl`、引数で渡したファイル（`.mtqg/archive/`のファイルを含む）
- `+`・`-`の印は表示しない。ただし消えた行（`-`）には常に印を付ける。`--mark`で全行の印を出す。印は、ほかの列の前の、独立した列で、
  出す印があるときだけ付く
- 本文は一覧の決まりに従う（1行目だけ、端末に出すときだけ切る、制御文字は置き換える）

## archive

```
$ mtqg archive 2021-01-01..2024-09-18
$ mtqg archive 2021..2023
$ mtqg archive 202404..2024-09 -n
```

最後のイベントが期間に入る項目を、`journal.jsonl`から`.mtqg/archive/<開始>..<終了>.jsonl`に移す（どの項目が移るかは[schema_ja.md](schema_ja.md#アーカイブ)：終わったtodo・質問・bug、memo、削除した記録）。どのコマンドも`archive/`を自分から読むことはなく、アーカイブ済みのIDは単に見つからない。アーカイブのファイルを読むときは、`mtqg format`に渡す。

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

エラーになるもの（終了コード2。コマンドラインの誤りと同じ）：`..`がない、8・6・4桁でない側がある、左右の単位が違う、存在しない日付、開始が終了より後、数字・`-`・`.`以外の文字がある。

```
$ mtqg archive 2021-0101..202409-18
Range: 2021-01-01..2024-09-18
Archived: 412 memos, 138 todos, 57 questions, 81 answers, 12 bugs, 20 replies -> .mtqg/archive/2021-01-01..2024-09-18.jsonl
Skipped: 3 open todos, 1 open question, 2 open bugs, 24 glossary entries
```

- 1行目に、期間をどう読んだかを必ず表示する
- `Archived:`は移したものを種類ごとに数える（1つもない種類は出さない）。削除した記録は、その種類として数える。`-> `はファイルの名前。何も移さないときは`Archived: nothing`と言い、ファイルは作らない
- `Skipped:`は、最後のイベントが期間に入るのに残るものを数える：未完了のtodo・質問・bugと、glossaryの項目。なければ行ごと出さない
- ファイル名は常に正規化した期間。同じ期間をもう一度アーカイブすると、同じファイルに追記する
- `-n`（`--dry-run`）は、何も移さずに同じ報告を表示し、`archive/`も作らない。1行目の終わりに`(dry run)`が付き、「移す予定」の報告を「移した」報告と取り違えないようにする
- 相対的な日付（「2年前」など）は受け付けない
- `archive`はイベントを書かないので、記録者は要らない（`git config user.name`が未設定でもよい）
- `--json`のときは、報告は1つのオブジェクトになる（[JSON出力](#json出力)）。数えるだけで、記録の一覧は出さない。アーカイブの中身は、そのファイルを`mtqg format`に渡して見る

期間を戻すには、そのファイルを`journal.jsonl`の末尾に結合して消す（[schema_ja.md](schema_ja.md#アーカイブ)）。`unarchive`コマンドはない。

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
| 端末 | `MTQG_TTY`、なければ標準入力・標準出力・標準エラー出力がつながっている端末。なければなし |
| エディタ | `$EDITOR` |

- コマンドラインから記録するAIエージェントは、2つの変数で名乗る：
  `MTQG_AUTHOR_KIND=ai MTQG_AUTHOR_NAME=claude-code`。`ai`のときは名前が必須。
  AIが`git config`の名前で記録されることはない
- `git config user.name`は、そのリポジトリについて読む（`git -C <ルート> config user.name`）ので、
  そのリポジトリの設定が効く。どちらからも名前が得られなければ、止まって、設定の方法を伝える。
  `MTQG_AUTHOR_KIND`が`human`でも`ai`でもなければエラー
- 端末は、すべての行に`tty`として、ハッシュの16進8桁で書く（[undo](#undo)）。見分ける端末がないときは書かない

変数は設定ファイルではない。設定ファイルはない。
