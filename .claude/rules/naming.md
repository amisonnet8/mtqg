# 命名規則

## 製品名

- **`mtqg`は常に小文字**で書く。文頭でも`Mtqg`・`MTQG`にしない（コマンド名とそろえる）
- 由来は **(m)emo・(t)odo・(q)a・(g)lossary**。順序にも意味がある（設計§3）
- **`bug`・`rule`を足したが、名前は変えない**（設計§3。GitHubのDescriptionは`(q)a & bugs`と書く）。今後、種類が増えても改名しない
- 旧称（tdmqa、tmqg）は使わない

## 用語の対応

設計文書は日本語、コード・CLI出力・`docs/reference/`は英語で書く。同じものを指す言葉がずれないよう、次の対応を使う。

| 日本語（設計文書・対話） | 英語（コード・出力・reference） | 補足 |
|---|---|---|
| 記録 | record | memo・todo・qa・bug・glossary・ruleの1件 |
| 種類 | kind | memo / todo / qa / bug / glossary / rule。JSONのフィールド名は`type` |
| イベント | event | `journal.jsonl`の1行 |
| ジャーナル | journal | `journal.jsonl` |
| ID、短縮ID、完全なID | ID, short ID, full ID | 完全なIDは16進32桁、短縮IDは表示用の先頭10桁 |
| 記録者 | author | `author.kind`（`human`/`ai`）と`author.name` |
| 端末識別子 | terminal ID | フィールド名は`tty` |
| 質問 / 回答 | question / answer | どちらも`type:"qa"`。回答は`re`を持つ |
| バグ / 返信 | bug / reply | どちらも`type:"bug"`。返信は`re`を持つ。バグは不具合の報告とそのやり取り（課題管理ではない・設計§2.7） |
| 親の記録 | parent | 回答・返信の`re`が指す、質問・バグ。返信は親と同じ`type`を持つ |
| 用語 / 定義 | word / definition | glossaryの`word`と`text` |
| 決まり事 | rule | 読めば従える、この先も効き続ける決まり事。形はmemoと同じ |
| 未完了・未クローズ / 完了 | open / done | 状態の値 |
| 確定待ち | awaiting confirmation | 回答はあるが閉じていない質問 |
| 並行した状態変更 | concurrent status changes | 同じ`from`からの別々の変更 |
| アーカイブ | archive | |
| 形式のバージョン | format version | `.mtqg/version`と各行の`v`。mtqg本体のバージョンと混同しない |
| 経緯、過程 | process, history | 「成果物に残らない過程」 |

## 表示する種類の名前

`show`・`log`・エラー文言に出す種類の名前は、`memo`・`todo`・`question`・`answer`・`bug`・`reply`・`glossary`・`rule`（モデル層の`Record.Kind()`。文の中では`glossary entry`）。JSONの`type`は`qa`と`bug`のままで、質問と回答、バグと返信は`re`の有無で見分ける。`log --kind`の値は`type`と同じ`memo`・`todo`・`qa`・`bug`・`glossary`・`rule`（1文字も可）で、`qa`は質問と回答の両方、`bug`はバグと返信の両方。

## コマンド

- 体系は`mtqg <種類> <動詞>`。種類は1文字に略せる（`m`・`t`・`q`・`b`・`g`・`r`）。**動詞は省略しない。例外規則を作らない**
- 動詞は`add`・`done`・`reopen`・`list`・`edit`・`delete`の**6語だけ**。種類ごとに別の単語（`ask`・`answer`・`close`など）を足さない
- `edit`・`delete`は種類なしのコマンド（IDがあれば種類は要らない）
- オプションはgitの慣習に寄せる（`-C <パス>`、`-n`／`--dry-run`など）

## JSONのフィールド名

- `docs/reference/schema.md`の表にあるものだけ。**新しいフィールドを足すときは、先に`schema.md`（と`schema_ja.md`）に書く**
- `term`は使わない（用語はglossaryの`word`、端末は`tty`。以前`term`が両方の意味で使われかけた経緯がある）

## Goコードの命名

- Goの標準的な慣習に従う（`gofmt`・`golangci-lint`）
- パッケージ名は層の名前にそろえる：`journal`・`model`・`cli`
- 上の用語の対応表の英語をそのまま型名・関数名に使う（`Event`、`Record`、`Origin`、`Author`など）。同じものに別の英語を当てない
