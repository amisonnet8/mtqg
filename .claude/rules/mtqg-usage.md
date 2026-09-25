# mtqg自身の使い方（段階4から）

段階4（`PLAN.md`「開発の段階」）から、mtqg自身の開発過程を`.mtqg/`に記録する。ここでの`mtqg`は**mainから入れた安定版のバイナリ**（`go install ./cmd/mtqg`。ルート直下の開発中のビルド`./mtqg`とは別）。壊れた実装で自分自身の記録を壊さないため（distribution.md）。mainがマージで進むたびに、`go install ./cmd/mtqg`で入れ直す。

## 記録者

`.claude/settings.json`の`env`に`MTQG_AUTHOR_KIND=ai`・`MTQG_AUTHOR_NAME=claude-code`を設定済み（段階4a、`mtqg init --agent claude-code`で配線・2026-09-25）。コマンドの前に明示しなくても、このリポジトリでのmtqgコマンドは記録者`ai`/`claude-code`になる。人間が記録するときだけ、`.claude/settings.json`の値を上書きするよう明示して呼ぶ（例：`MTQG_AUTHOR_KIND=human MTQG_AUTHOR_NAME=amisonnet8 mtqg r add "..."`）。

## 何をどの種類で記録するか

qsoku側の運用（qsokuリポジトリ`.claude/rules/mtqg.md`）と同じ考え方。

| 種類 | 例 |
|---|---|
| `m add`（memo） | 実装しながらの気づき、後で見返したい観察 |
| `t add`（todo） | やること。終わったら`t done` |
| `q add`（question） | 判断に迷って人間に確認したいこと。`AskUserQuestion`で人間に聞いたときは、その場で質問と回答を`q add <id> <回答>`まで記録する（qsokuの報告「対話中のQ&Aが自動で残らない」への対応。段階4ではフックが自動記録しないため、AIが自分で書く） |
| `b add`（bug） | 見つけた不具合とそのやり取り。**その場で直したものも含めて記録する**（qsokuの報告「もう直したから記録は軽くていい、という判断バイアス」への対応） |
| `g add`（glossary） | 用語の合意 |
| `r add`（rule） | 読めばそのまま従える決まり事（`docs/design/history.md`2026-09-25の線引きに従う） |

## 作業を始めるとき

- `mtqg context`を読む。段階4以降、このリポジトリの現在地の一次情報は`.mtqg/`の記録であり、`PLAN.md`はその要約・区切りの記録に役割を絞る（`PLAN.md`は引き続き置く。qsokuと違い、mtqg自身は段階1〜3の経緯を`PLAN.md`と`docs/design/`にすでに大量に持つため）

## その他

- `.mtqg/.local/`はコミットされない
- `.mtqg/`の中のファイルを直接編集しない（すべてmtqgのコマンド経由）
- mtqgはgitに対して読むだけ
