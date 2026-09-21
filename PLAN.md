# PLAN

mtqgの実装計画・進捗管理ドキュメント。実装が進むにつれて随時更新すること（特に「現在地」「保留事項」は、セッションをまたぐたびに参照・更新する）。

段階の理由は設計§12。段階3まではmtqg自身の経緯をこのファイルと`docs/design/`に残し、段階4からは`.mtqg/`に記録する。

## 開発の段階

| 段階 | すること | 区切り | mtqg自身の記録 |
|---|---|---|---|
| 1. コアとCLIを作る | ジャーナル層を先に作って固め、モデル層はCLIのコマンドと一緒に足していく。記録作成、表示、`context`、`status`、`--json`、シェル補完 | 完了で**v0.1** | 使わない（このファイル） |
| 2. サンプルPJで使う | 別のサンプルPJにmtqgを導入し、CLIだけで記録しながら開発する | | 使わない |
| 3. 問題を吸収する | 出た問題をCLIに反映する。2と3を繰り返す | 完了で**v0.2** | 使わない |
| 4. 外部ツール連携 | MCP・フック・VSCode拡張（設計§11）を、mtqgで記録しながら開発する | | **ここから使う**（形式0） |
| 5. 問題を吸収する | 4で出た問題を片付ける。4と5を繰り返す | 完了で**v1** | 使う |
| 6. v1を確定する | 4〜5の記録をAIが形式1に変換し、形式を1として確定する | | 使う（形式1） |
| 7. 移行 | このファイル等によるルール運用から完全に移行する | | 使う |

### 区切りの条件

| 区切り | 条件 |
|---|---|
| **v0.1** | コアとCLI（段階1の範囲）ができた |
| **v0.2** | サンプルPJを1つ最後までやり切り、その間に出た「CLIで直すもの」が片付いた |
| **v1** | 設計§11の機能をmtqgで記録しながら開発し、その間に出た問題が片付いた |

- v1確定前は、`.mtqg/version`と各行の`v`を`0`（形式が未確定）とする。形式を変えても`version`を上げず、過去の行の書き換え・破棄・変換を許す
- サンプルPJのデータは使い捨てにしてよい
- **v1確定時の変換**：段階4〜5でmtqg自身に記録したデータは捨てずに、`docs/reference/schema.md`（と`schema_ja.md`）をv1として確定させ、それを仕様としてAIが形式1へ変換する。`id`・`ts`・`author`は変えない。変換後、全行が警告なく読めること、件数と状態が一致することを確かめ、変換と`version`を`1`にする変更を1つのコミットにまとめる。これは一度きりの作業で、`mtqg upgrade`とは別

## 段階1のステップ

順番は **足場 → ジャーナル層 → モデル層とCLI** 。

- **ジャーナル層は先に作って固める。** 仕様（`docs/reference/schema.md`、`.claude/rules/journal-format.md`）がほぼ決まっていて使われ方に左右されず、ロックや並行書き込みのような難しいところを先に片付けられるため
- **モデル層は先に作らない。** 状態の組み立て・検証・`context`の中身などは、コマンドが何を必要とするかで形が決まる。CLIのコマンドを作るときに、そのコマンドに要る分だけ一緒に足していく。「コアを全部作ってからCLI」ではない（設計§10.7「使われ方が見えないまま作り込むと、要らない汎用性を作りがち」）


1. **Step 1: 足場固め**
   - `go.mod`（モジュールパス`github.com/amisonnet8/mtqg`）、`Makefile`、`.gitignore`、`schema.go`（`docs/reference/schema.md`をembed）、`cmd/mtqg/main.go`（空の入口）
   - `.golangci.yaml`と`trivy.yaml`は配置済み。`make check`・`make trivy`から使う（`golangci-lint run`、`trivy fs --config trivy.yaml .`）
   - `Makefile`のターゲット：`build`、`test`（e2e）、`check`（fmt・vet・lint・単体テスト）、`fmt`、`race`、`trivy`、`shellcheck`
   - `.gitignore`：ビルドした`mtqg`、`dist/`、`.claude/settings.local.json`。**`mtqg`はルート直下に限定して`/mtqg`と書く**（`mtqg`だけだと`cmd/mtqg/`まで無視され、`main.go`がコミットされない）。Windows向けの`/mtqg.exe`も入れる。public化で個人の設定が見えないよう、最初の`git add`より前に置く（2026-09-20、先に配置済み）
   - `README.md`・`README_ja.md`：**注意書きだけ**（未完成で、まだ使わないでほしい旨。「READMEとGitHubの看板」の「未完成の間の注意書き」）。看板としてのREADMEではない
   - `.github/workflows/`：3OSマトリクスで`make check`、`race`（ubuntu・macOS）、`shellcheck`（ubuntu）、`trivy`（ubuntu）。落とし穴は`.claude/rules/testing.md`。リポジトリはpublicなので、Actionsの分数は気にしなくてよい
   - **`Makefile`ができた時点で、`PostToolUse`フック（`.go`・`go.mod`・`go.sum`の編集後に`make build`）の設定を提案する**（「保留事項」参照）
2. **Step 2: ジャーナル層**（コアの下の層を先に固める。この時点ではモデル層・CLIのコマンドは作らない）
   - `.mtqg/`の探索（`-C`を含む）、`version`の確認、読み込み（壊れた行・衝突マーカーの警告、同一行の同一視）、追記（書き出し規則、`.mtqg/.local/`のロック、UUIDの生成）、書き直しの汎用操作（一時ファイルは`.mtqg/.local/tmp/`）
   - ロックの並行テスト（追記どうし、追記と書き直し）、`make race`
3. **Step 3: CLI順1** — `init`、`m add`、`t add`、`t done`、`t list`、`status`。モデル層はここから、各コマンドに要る分だけ足していく（Step 3〜9共通）。**これだけでGoogle Keepの代わりになる。ここから自分で使い始められる**
4. **Step 4: CLI順2** — `q add`（質問・回答）／`done`／`reopen`／`list`、`g add`／`list`、`show`、`log`。4種類が揃う
   - **Step 4.5: 5つ目の種類`bug`**（2026-09-21に挿入。Step 5〜9の番号は変えない）— `bug add`（不具合と返信）／`done`／`reopen`／`list`。qaと同じ形（親＋`re`を持つ返信）で、データ形式は`type`に`bug`を足すだけ。`--json`の形は外部との約束になるので、種類が5つ揃ってからStep 5に入る
5. **Step 5: CLI順3** — `context`、全コマンドの`--json`。AIに渡せる
6. **Step 6: CLI順4** — `edit`、`delete`、`undo`、`search`、`review`、`format`
7. **Step 7: CLI順5** — `archive`（`-n`を含む）
8. **Step 8: e2e・docsの例の確認** — 複数クローン・ブランチをまたぐ検証（`.claude/rules/testing.md`「mtqg固有の検証項目」）、`docs/reference/`の例の実測確認の仕組み。Step 3以降、できるところから並行して足してよい。**例の取得は、Step 3・4・4.5で同じ手作業（日時とIDを固定した記録を作る→本物のバイナリで動かす→文書の該当ブロックを差し替える）を3回繰り返した**（作業用のスクリプトはセッションの一時領域で、リポジトリには無い）。ここで、フィクスチャをGoのテストの側に置き、文書の例と突き合わせる仕組みにする（`testing.md`「e2eとdocsの例の確認」）
9. **Step 9: シェル補完**

Step 3が動いた時点でサンプルPJ（段階2）を始められる。

**段階1の意図的なスコープ外（段階4以降）:** `mtqg mcp`、`mtqg hook`、`mtqg init --agent`、VSCode拡張、`mtqg upgrade`（形式2が出るまで作らない）、`docs/tour/`・`docs/examples/`、看板としてのREADME（注意書きだけのREADMEはStep 1で置く）、GitHub Releasesのバイナリ。

**段階1完了（v0.1）の判定:** `make check`・`make test`・`make race`が通る／GitHub Actionsの3OSマトリクスがgreen（Windowsでのロックと一時ファイルの置き換えを含む）／`go install`で入れたバイナリで`mtqg version`が意味のあるバージョンを出す／`docs/reference/`の例が実際の出力と一致している。

## 現在地

**段階1 Step 4.5（5つ目の種類`bug`）：完了（2026-09-21、CIの3OSがgreen。人間が確認）。次はStep 5（`context`と全コマンドの`--json`）。** ユーザーの決定で`bug`を足した（サブコマンド`bug`、1文字`b`。qaと同じ形で、`type`に`bug`を足すだけ。返信の`re`はbugだけを指す）。`bug add`（不具合と返信）・`list`・`done`・`reopen`、`show`・`log --kind bug`・`status`の`Open bugs`の行。5種類（memo・todo・qa・bug・glossary）が揃った。

**決めたこと（この会話で確認済み。理由は`docs/design/history.md`）：** 返信の呼び名は`reply`（`show`・`log`の種類の列は memo / todo / question / answer / bug / reply / glossary）。Step 4.5として独立させ、CIを通してからStep 5に入る。仕様書でのbugは「不具合そのもの」（不具合の報告と、そのやり取り。`done`は、直った／もう追わない）。**名前は`mtqg`のまま**（設計§3の「要判断」を「改名しない」と決定）。課題管理への線引きは設計§2.7に足した：bugはqaと同じ形（親＋返信、open/doneだけ）に留め、重要度・担当者・再現手順・影響バージョンの欄は持たない。

**できたもの：** ジャーナル層に`TypeBug`（`append.go`の許可リストを含む）。モデル層は、qaとbugを1つの「親と返信」の形にまとめた：`Parents(typ)`・`Replies(parentID)`・`HasParent`・`ParentKind`・`ReplyKind`・`ParentCreate`・`ReplyCreate`・`CanHaveReplies`・`IsReply`・`NoRepliesError`。**返信は親と同じ`type`を持つ**：`re`が別の`type`の記録（や回答）を指していても、返信にならず、エラーにもならない。CLIの層は、`kindSpec`の表に`type`を持たせ、`qa.go`を`thread.go`にして両方の種類が同じコードを使う。種類違いの案内（`mtqg qa done`が要る記録に`mtqg t done`を打った、など）は、種類の表から作る（今まで2つの`if`で書いていた）。

**手元で確かめたこと：** `make check`・`make test`（e2e）・`make race`・`make trivy`・`make shellcheck`が通る。macOS・Windows向けに`go vet`とテストのコンパイルが通る（**実行はCI**）。モデル層の4つの変異（返信が親の`type`を見ない、`parentOf`が`type`を見ない、bugを消しても返信が残る、返信が常に`qa`で書かれる）とCLIの3つの変異（bugの一覧が質問を出す、案内が常に`qa`、打ち間違えたIDが新しいbugになる）を、テストが検出した。`cli.md`・`cli_ja.md`の新しい例（`status`・`bug list`・`show`・`log --kind bug`）と、文章中のエラー文言は、**日時とIDを固定した記録を本物のバイナリで動かした実際の出力**（英語版は中身も英語、日本語版は日本語）。

**Step 4.5に含めなかったもの：** `--json`と`context`のbugの区画（Step 5。`cli.md`の`context`の下書きには`## Open bugs`を足してある）、`review`と`status`の`Conflicts`（Step 6）、`edit`・`delete`・`undo`・`search`・`format`（Step 6）、`archive`（Step 7）。`format`の例の種類の列が`qa`のままなので、Step 6で`question`・`answer`・`bug`・`reply`に揃える。

**（前の状態）段階1 Step 4（CLI順2）：完了（2026-09-21、CIの3OSがgreen。人間が確認）。次はStep 5（`context`と全コマンドの`--json`）。** 4種類（memo・todo・qa・glossary）が揃い、書いた記録を`show`・`log`で読み返せる。

**できたもの：** モデル層に、質問と回答（`Questions`・`Answers`・`HasQuestion`）、`Glossary`、`DuplicateWords`、`All`、`History`、`QuestionCreate`・`AnswerCreate`・`GlossaryCreate`、`Summary`の拡張、IDの最短4桁（`MinIDDigits`・`IsIDLike`・`TooShortError`）。質問を消すと回答も隠れる判定（`visible`）。CLIの層に、`q add`（質問と回答）・`q list`・`q done`／`reopen`、`g add`・`g list`、`show`、`log`（`--limit`・`--kind`）、`status`の`Open questions`・`Glossary`の行。一覧の整形は表の関数1つ（`formatTable`）にまとめ、todo・質問・用語・`log`が使う。コマンドごとの「値を取るオプション」と「必須の語数」を文法の表に持たせた。

**手元で確かめたこと：** `make check`・`make test`（e2e）・`make race`・`make trivy`・`make shellcheck`が通る。macOS・Windows向けに`go vet`とテストのコンパイルが通る（**実行はCI**）。モデル層とCLIの新しい振る舞いは、壊して確かめた（打ち間違えたIDが黙って質問になる、回答が最古になる、`log`が古い順になる、質問を消しても回答が残る、など、13の変異をテストが検出）。`cli.md`・`cli_ja.md`の新しい例（`status`・`qa list`・`glossary list`・`show`・`log`・`q add`のエラー）は、**日時とIDを固定した記録を本物のバイナリで動かした実際の出力**。

**Step 4で決めたこと（この会話で確認済み。理由は`docs/design/history.md`）：** IDは**4桁以上**で受け付ける（今までは下限なし）。`q add`は、**最初の引数が16進数の4桁以上だけでできていれば回答先のID**（当てはまる記録がなければ質問にせずエラーにして、引用符で括るよう案内する）。IDだけで本文がなければエラー（結果、`$EDITOR`が開くのは引数がまったくないときだけで、回答と定義を`$EDITOR`で書く方法は無い。Step 6の`edit`で見直す）。`log`は**新しい順・20件**（暫定。サンプルPJで判断する）。glossaryの重複は`word`の完全一致。

**Step 4に含めなかったもの：** `--json`（Step 5）、`status`の`Conflicts`の行と`review`（Step 6）、`edit`・`delete`・`undo`・`search`・`format`（Step 6）、`archive`（Step 7）、`tty`（Step 6）。`format`の例の種類の列が`qa`のままなので、Step 6で`question`・`answer`に揃える。

**CIで確かめられたこと（Step 4.5）：** bugの追加を含め、3OSでgreen（人間が確認）。

**CIで確かめられたこと（Step 4）：** Windows・macOSでの新しいコマンドの出力（e2eの`TestQuestionsAnswersAndTheGlossary`の`└`の文字、日本語の桁揃えを含む）。**CIで確かめられないこと：** Windowsの色と端末の幅（CIに端末が無い。Step 3から変わらない）。

**（前の状態）段階1 Step 3（CLI順1）：完了（2026-09-21、CIの3OSがgreen。人間が確認）。** Windowsで見つかった`Init`の後片付けの不具合（開いたままの`os.Root`が`RemoveAll`を妨げる）は、閉じてから消すよう直した（`testing.md`）。これで、**Google Keepの代わりに自分で使い始められる**（`init`・`m add`・`m list`・`t add`・`t list`・`t done`・`t reopen`・`status`・`version`・`help`）。サンプルPJ（段階2）を始められる。

**できたもの：** ジャーナル層に`Init`と`git.go`（`user.name`、コミット済みの`journal.jsonl`）、モデル層（`Build`・`Resolve`・`ResolveKind`・`MemoCreate`・`TodoCreate`・`SetStatus`。qa・glossaryも表せる形）、CLIの層（`internal/cli/`。文法を表でデータとして持つ。`messages.go`に英語の文言をすべて置く）、`e2e/`（本物のバイナリと本物のgit。`merge=union`で2つのブランチの記録が両方残ることも確認）、`make test`、CIの`check`ジョブに`make test`。

**手元で確かめたこと：** `make check`（`vet`は`-tags e2e`も）・`make test`・`make race`・`make trivy`・`make shellcheck`が通る。macOS・Windows向けに`go vet`とテストのコンパイルが通る（`ansi_windows.go`を含む。**実行はCIで**）。`cli.md`・`cli_ja.md`の出力例（`init`・`todo list`・`done`／`reopen`・曖昧なID）は、**実際に動かした結果**（サンプルの日付・IDで固定した記録を、本物のバイナリで動かした）。

**実装しながら決めたこと・見つかったこと：**
- **不具合を1つ直した（`Build`）：** 時刻が記録の`create`より前の変更を、「作成のない記録への変更」として捨てていた。時計の進んだマシンが作ったtodoを、時計が正しいマシンが完了にすると、その完了が消える。作成を先に適用し、時刻が前の変更も、その後で順に適用するようにした（`schema.md`・`schema_ja.md`に1文、`history.md`に理由）。出力例を実際に動かして取ろうとして見つかった
- **Goのソースに見えない文字が入っていた：** `\uXXXX`のエスケープが、書き込みの途中で実際の文字に展開されていた（`render.go`の双方向制御文字、Step 2の`event_test.go`のU+2028）。gosecのG116が検出。数値・`\x`のバイト列に直し、`source_test.go`が全`.go`を検査する（`testing.md`）
- オプションの解釈は、`flag.FlagSet`ではなく**自前の小さな解釈**（約60行、標準ライブラリだけ）。`--`とインターリーブ（`t done <id> --full-id`）を、仕様どおりに扱うため
- **`//nolint:gosec`を1か所だけ、確認を取って足した**（`$EDITOR`の起動。G204）。除外の設定（`.golangci.yaml`の`exclusions`）は足していない。`.golangci.yaml`には`run.build-tags: e2e`を足した（e2eもlintの対象）
- `mtqg version`の表示は、`go build`が疑似バージョン（`v0.0.0-<時刻>-<ハッシュ>+dirty`）を作ることを確かめてから決めた（`-ldflags`の値 → ビルドのモジュールのバージョン → `dev`）
- `init`が作る`.mtqg/`は`0750`、中のファイルは`os.Root.Create`（`0666`からumaskを引いたもの。普通のファイル）

**CIで確かめられないこと：** Windowsでの色（`SetConsoleMode`）と端末の幅（`x/term`）は、CIでは端末が無いので**確かめられない**（手元にWindowsも無い）。`$EDITOR`の起動と引用符（`"C:\Program Files\..."`）はe2eが確かめる。macOSの`/var`→`/private/var`、`-C`とパスの表示。

**Step 3に含めなかったもの：** `--json`（Step 5）、qa・glossary（Step 4）、`status`のqa・glossary・Conflictsの行（Step 4・6）、端末識別子`tty`と`undo`（Step 6。今は`tty`を書かない）、シェル補完（Step 9）。**AIの記録者は環境変数で名乗る**（`MTQG_AUTHOR_KIND`・`MTQG_AUTHOR_NAME`）。指示ファイルに書く運用は、Step 5・段階4で整える。それまで、環境変数の無いAIの記録は人間の名前になる。

**依存の例外の記録：** なし。

**（前の状態）段階1 Step 2（ジャーナル層）：完了（2026-09-20、CIの3OSがgreen）。** Step 1（足場）は完了済み（コミット`9e3bc6f`・`4bc8efd`、CIの3OSがgreen、`PostToolUse`フックの反映）。

**ジャーナル層（`internal/journal/`）にあるもの**（計画は承認済み。JSONは`encoding/json/v2`、ロックの待ち時間は5秒、`golang.org/x/sys`を追加）：`errors.go`（エラーの種類）、`id.go`（UUIDv4）、`event.go`（書き出し・読み取り。JSONを触るのはここだけ）、`read.go`（`Scan`）、`find.go`（`.mtqg/`の探索）、`version.go`、`journal.go`（`Open`・`Read`）、`lock*.go`（`flock`／`LockFileEx`）、`append.go`、`rewrite.go`＋`replace_*.go`。テストは`*_test.go`（ゴールデン、別プロセスの並行、ロックを持つプロセスの`Kill`、シンボリックリンク、ベンチ）。

**手元で確かめたこと：** `make build`・`make check`・`make race`（10回×3の繰り返しでも失敗なし）・`make shellcheck`・`make trivy`が通る。macOS・Windows向けに`go vet`とテストバイナリのコンパイルが通る（実行はCIで確認）。ロックを無効にする変異と、`Rewrite`からロックを外す変異で、それぞれテストが失敗することを手で確かめた（`testing.md`）。Trivyは`golang.org/x/sys`のBSD-3-Clauseを検出し、脆弱性は0件。

**CIで見つかった問題（2026-09-20、macOS）：** `TestAppendConcurrentGoroutines`が`openat .local/lock: no such file or directory`で失敗した（`check`と`race`の2件。同じ原因）。`.local/`が無い状態（cloneした直後）で、複数のgoroutineが同時に`MkdirAll`と`OpenFile`をすると起きる。**原因は特定できていない**：Linuxでは、3000回×16並行のストレステストでも再現せず、標準ライブラリの`os.Root`にdarwin向けの特別な処理も見当たらない。対策として、①ロックのファイルを先に開き、`.local/`が無いときだけ作る（通常時は`MkdirAll`を呼ばない）、②`no such file`は上限つきで再試行する（`.local/`は「いつ消えても困らない」ものなので、開く直前に消えた場合にも正しい）、③macOSのCIで競合を毎回検出するテスト（`TestLockWhenManyStartTogetherWithoutTheLocalDirectory`）を足した。**この修正でmacOSのCIが通った（人間が確認）。** ただし原因は特定できないまま（macOSのカーネルの挙動か、Goの不具合かは不明）。再発したら、`TestLockWhenManyStartTogetherWithoutTheLocalDirectory`が検出する。

**CIで確かめたこと（人間が確認、3OSがgreen）：** WindowsでのLockFileEx（別プロセスの排他、`Kill`後の解放）、Windowsでの置き換え（開いているファイルへの再試行）、macOSでのロックとシンボリックリンク（`/var`→`/private/var`）、`make race`（ubuntu・macOS）。Windowsのシンボリックリンクのテストは、権限が無ければskipされる（skipされたかは未確認）。

**実装しながら決めたこと（計画に無かったもの）：**
- **`.mtqg/`の中はすべて`os.Root`経由**（`Journal.openRoot`）。外を指すシンボリックリンクに書かない。悪意のあるリポジトリが`journal.jsonl`や`.local`を外へのリンクとしてコミットしていても、cloneした直後に動かされたmtqgがリンク先へ書かない。gosecのG703も、除外を足さずに解消した（`journal-format.md`）
- `Append`が`journal.jsonl`を新しく作るときの権限は`0600`（gosecのG302を、除外なしで通すため）。**`init`（Step 3）が作る`journal.jsonl`の権限は、そのとき決める**（普通のファイルとして共有されるので`0644`にしたいが、gosecの除外が要るなら確認を取る）。書き直しは、既存の権限を引き継ぐ
- **ロックは公平ではない。** 休みなしで追記し続けると、待っている書き直しを締め出しうる。実際の使い方（短いコマンド、間に休み）では問題にならないと判断した。`lock`のコメントに書いた
- 追記は、ファイルの末尾が改行でなければ改行を先に足す（書きかけの行に次の行がくっつかない）
- `make build`は全パッケージをコンパイルする（`cmd/mtqg`がimportしていないパッケージのビルドエラーを、`PostToolUse`フックが見逃さないため）

**性能（AMD Ryzen 9・Linux、1行約250バイト）：** 追記は1万行で0.95ms、10万行で7.7ms（毎回、全体を読んで衝突マーカーを探す。「書き込みの速さが最優先」を満たす）。読み込みは1万行で19ms、10万行で180ms（1行あたり約1.8µs）。書き直しは1万行で25ms、10万行で159ms。読み込みが10万行で目立つ点は、サンプルPJで使ってから判断する（遅ければ、読み取りの割り当てを減らす、`archive`で小さく保つ）。JSONのv1との速度比較はしていない（v2は速さではなく、正しさで選んだ）。

**Step 2に含めなかったもの：** `init`と`.mtqg/`の作成、`git config user.name`・端末識別子の解決（どちらもStep 3）、アーカイブへの移動（Step 7）。

**公開の方針を決めた（2026-09-20）：リポジトリは最初からpublicにする。** 完成してから公開するのではなく、未完成のまま公開し、READMEの注意書きで「まだ使わないでほしい」と伝える（下記「公開の2段階」）。GitHubリポジトリの作成と`git init`は人間が行い、済んだらStep 1に入る。

**名前の同名チェック（2026-09-20、Claude Codeが実施。公開中・未完成の段階として十分）：** `amisonnet8/mtqg`は未使用（404）。pkg.go.devの検索は0件、モジュールプロキシにも記録なし。npm・PyPI・crates.io・Homebrew（core）に`mtqg`は無い。GitHubの名前検索では別のアカウントの`mtqg`（中国語のEC系の記述が2件）と、ベトナム語の略語`MTQG`（「国家目標」）を使うリポジトリがあるが、いずれも無関係で、Goのパスは`github.com/amisonnet8/mtqg`なので衝突しない。**製品・企業・ブランドの同名チェック（同日）：** Web検索（ソフトウェア・製品・企業・商標・略語の意味）で、`MTQG`という製品・企業・ブランドは見つからなかった。紛らわしいのは綴りが近いものだけ：`MTG`（株式会社MTG＝美容機器、Magic: The Gathering、「meeting」の略）と`MQTT`（IoTのプロトコルとそのツール群）。どちらも綴りが違い、分野も違う。`mtqg.com`はGoDaddyの売り出し用の駐車ページ（製品や企業のサイトではない）。`mtqg.dev`・`mtqg.io`は名前解決できなかったが、登録の有無は確認できていない（RDAPが403）。**未確認：** apt・Scoop・winget・Nixなどの他のパッケージマネージャー、正式な商標調査（USPTO・J-PlatPat・WIPOの検索は、Web検索では確かめられなかった）、ドメインの登録状況。完成として公開するとき（「公開前にやること」）に、範囲を広げて再確認する

`docs/design/`は、開発開始前に書かれた「構想メモ」と「CLI検討案」を分割したもの。未決事項はこのファイルに移した。

## 未確認事項（実装前に決める・確かめる）

1. ~~Goのモジュールパス。~~ **確定：`github.com/amisonnet8/mtqg`**（コマンドは`github.com/amisonnet8/mtqg/cmd/mtqg`）
2. ~~Trivyのライセンス検出がGoの依存で効くか。~~ **確認済み（2026-09-20）：効く。** `golang.org/x/sys`を足した後、閾値を一時的に下げて（`--severity UNKNOWN,LOW,...`、設定ファイルは変更しない）実行すると、`golang.org/x/sys`のBSD-3-Clause（分類はnotice、深刻度はLOW）が検出・表示された。既定の閾値（HIGH・CRITICAL）では通る。依存を足したら同じ手順で確かめること
3. ~~JSONの書き出しに`encoding/json`と`encoding/json/v2`のどちらを使うか。~~ **確定：`encoding/json/v2`**（2026-09-20）。v1は`SetEscapeHTML(false)`でもU+2028・U+2029を常にエスケープし、不正なUTF-8を黙って置き換える。理由は`docs/design/history.md`。JSONを扱うのは`internal/journal/event.go`だけにして、ゴールデンテストで固定する
4. **e2eの仕組み。** **決定（2026-09-21）：`e2e/`に、ビルドタグ`e2e`のGoのテスト。ビルドした本物のバイナリと本物のgitを`exec`で動かす。** `testscript`は`golang.org/x/`ではないので使わない（依存の基準）。Step 8で、docsの例の確認に「どうしても必要か」を再評価する（`.claude/rules/testing.md`に記録済み）
5. ~~ロックの待ち時間~~ **確定：5秒**（2026-09-20）。ロックを持つのは追記の一瞬か書き直しの間だけなので、5秒待って取れなければ、ロックを持ったまま固まったプロセスがいるとみなす。テストでは短い値に差し替えられるようにする

## 保留事項

- **`PostToolUse`フックは反映済み（2026-09-20）。** `.claude/hooks/build.sh`が、`.go`・`go.mod`・`go.sum`の編集後に`make build`を実行し、失敗時は終了コード2で理由をClaudeへ返す。スクリプトはShellCheckの対象（`make shellcheck`。追跡されているので自動で拾われる）。複数ファイルにまたがる編集の途中は、まだ書いていないファイルを参照してビルドが一時的に失敗し、その通知が出る点に注意。`.claude/settings.json`は人間が管理するので、今後の変更は原則として提案にとどめる（今回は人間が明示的に許可した）
- **`ubuntu-latest`がUbuntu 26.04に移行する（2026-10-19開始、11-19完了。段階的）。** 今は対応しない：警告だけでCIは通っており、移行期間中にCIが自動で新しいイメージで動く。`ubuntu-latest`を前提にしているのは`.github/workflows/ci.yml`と`.claude/rules/testing.md`。**10月中旬に見直すこと。** 移行期間中にCIが落ちたら、`ubuntu-24.04`に一時的に固定して原因を調べ、直してから戻す。影響を受けそうなのは、`shellcheck`ジョブが使うプリインストールの`shellcheck`と、`race`が使う`gcc`（移行の告知にこれらのバージョン変更の記載は無く、変わらないとも言えない）。`ubuntu-26.04`のラベルは今でも使える。出典：actions/runner-images#14748
- **devcontainer.json反映待ちリスト**：コンテナをリビルドせずに進める間、手動でインストール・設定したものはここに追記し、区切りでまとめて`.devcontainer/`へ反映する
  - （なし）
- **サンプルPJの題材は未定**（設計§12.3）。急がない（始めるのはStep 3以降）。選び方の基準：数日〜2週間程度、仕様に迷いどころがある（qaとglossaryが自然に生まれる）、AIエージェントと一緒に作る、少しはブランチ・worktree・別端末で並行して記録する
  - 選択肢：まったく別の題材にする／下記の着想のごく一部だけを切り出す（「実行ファイルがデータを抱え、`-c`で状態を進め、`undo`・`redo`できる」部分だけなら2週間に収まり、将来の大きなPJの最初の実験にもなる）
- **保留：v1以降の「本番の題材」候補（サンプルPJには大きすぎる）**
  - 着想：**サンドボックス化・SQLite・Lua・ファイルシステムを実行ファイルに組み込み、実行ファイルのコマンド実行だけでアプリにできるツール**。SanDBoxの延長のポータブルなデータベースとしてではなく、その上でアプリを作れる土台（フレームワーク）として見せる。理論上はSanDBoxの知見で実現できるはず
  - サンプルPJに向かない理由：4つの要素それぞれに設計の山があり、数日〜2週間の基準に収まらない。題材そのものの開発が主役になり、mtqgの評価から離れてしまう
  - 本番の題材に向く理由：規模が大きく長く続き、並列開発が自然に起き、判断の積み重ねがそのまま資産になる。サンプルPJで形を整えたmtqgを、ここで本格的に使う
  - これまでに出た考え（着想のメモ）：
    - 使う人は2段階：フレームワークで**アプリを作る人**と、できた**アプリを使う人**。アプリを使う人は中の仕組みを意識しない
    - **ファイル1つがアプリであり、そのアプリの1つの実体でもある。** `cp`するだけで実体が増え、人に渡せばデータごと動く。スナップショットは機能として持たず`cp`に任せる。「ツール＋既存のLinuxコマンド」で機能を増やす方針（mtqgの「版の管理はgitに任せる」と同じ考え方）
    - コマンドを打つたびに中のデータが書き換わり、`undo`・`redo`で行き来できる
    - フレームワークが引き受けるもの：状態の保存、`undo`・`redo`、いつコピーしても壊れていない書き換え（一時ファイル経由の置き換え）、`-c`とREPLの受け付け、`--json`と終了コード、排他。ファイルの外に状態を持たない
    - 早くから出そうな問い：アプリの定義の書き方、定義とデータの同居のさせ方、実行中の自分自身の書き換え（Windowsの上書き不可、macOSのコード署名、ウイルス対策ソフト）、履歴の持ち方、同時実行の排他、gitとの相性、「アプリ」「実体」「エンジン」などの用語の使い分け

### サンプルPJで判断すること

使ってみないと判断できないので、段階2〜3で決める。

- `context`の出力内容と分量、上限を超えたときの優先順位（設計§9、§11.3）。冒頭の指示文を外すオプション（`--no-guide`）が要るか
- qaの回答の「確定」をどう表現するか（設計§5.2）。`qa list`で確定待ちをどう見分けさせるか（記録者の種別で足りるか、確定の事実を別に持つか）
- 変更前の状態（`from`）による並行した状態変更の判定方式の妥当性（設計§7.6）
- 並行した回答を`review`で検出するか
- memoやtodoにも親（`re`）を持たせたくなるか
- **bugの使い分けと線引き：** bugとqa・todoで書き分けに迷うか。bugに重要度・再現手順などの欄が欲しくなるか（設計§2.7の線引き「bugはqaと同じ形に留める」が保てるか）
- **種類をまたぐ`re`の行の読み方**（他のツールが書いた行、または手で編集した行。mtqg自身は作れず、`b add <質問id>`はエラーで何も書かない）：今は、返信でも返信先のある記録でもない「返信先のない返信」として読み、エラーにしない。**その結果、`bug list`・`qa list`・親の`show`に出ず、見えるのは`log`とIDを指した`show`だけ**。`show`は、`re`が質問を指していても`to bug <id>`と書く（実際の種類と食い違う）。選択肢は、A：今のまま`show`の文言だけ直す、B：型を見ずに`re`の先へぶら下げる、C：Aに加えて`review`（Step 6）に「親がない・別の種類を指している返信」を載せる（勧めはC）。Step 6の`review`を作るときに決める
- AIの記録者名の粒度（モデル名まで残すか）（設計§5.2）
- 成果物の位置（`at`）の形式（設計§5.5）
- `log`の既定の件数と並び順、`search`に必要な絞り込み（種類、記録者、期間）
- 引用符なし入力で実際に困る場面がどれくらいあるか
- `undo`の端末識別子の作り方（ttyのパスだけで足りるか）と、後続イベントがある場合に断るか
- `journal.jsonl`の衝突を手作業で解決して困るか（困るなら自動解決のコマンドを足す）

### 段階4で決めること

- 終了時フックで記録を促す条件（「一定の作業が行われた」の判定方法）（設計§11.3）
- フックのセッションの記録：**まず何も保存しない方法を探す**（フックの入力のセッション情報と`journal.jsonl`の`ts`で判断できないか）。保存が必要なときだけ`.mtqg/.local/sessions/`を使う
- 最初に対応するエージェント以外のアダプタの対象と範囲（設計§11.3）
- 各エージェントの指示ファイルに書くルールの具体的な文面（設計§11.3）
- `mtqg init --agent`と「既存の`.mtqg/`があればエラー」のルールの関係
- VSCode拡張の初期機能の範囲、4画面の具体的なデザイン（特にQA表のレイアウト）、メンションを扱うか（設計§11.4）
- **VSCode拡張を別リポジトリにするかの最終判断**（別リポジトリにする方向）。別にする場合：`--json`の形を`docs/reference/cli.md`に約束として書く、拡張は`mtqg version`で対応バージョンを確かめる、mtqg自身の記録が2つのリポジトリの`.mtqg/`に分かれる（v1確定時の変換も両方で行う）。`mtqg mcp`・`mtqg hook`は同じバイナリに入るので本体に置く（決定済み）

### 公開の2段階

リポジトリはpublicで作る（2026-09-20決定。理由：privateのActionsは無料枠を消費し、3OSマトリクスではmacOSが10倍、Windowsが2倍で減るため）。そのため「公開」は2段階になる。

| 段階 | 状態 | READMEと文書 |
|---|---|---|
| **公開中・未完成**（Step 1〜v0.1の前まで） | リポジトリはpublic。使ってほしくない | 注意書きだけの`README.md`・`README_ja.md`。`docs/tour/`・`docs/examples/`は無くてよい |
| **完成として公開**（v0.1以降、看板を掲げるとき） | 使ってよい | 下記「公開前にやること」をすべて満たす。看板としてのREADMEを最後に作る |

公開中・未完成の間の決まり：

- **タグを打たない（v0.1まで）。** タグを打つとGoのモジュールプロキシ（`proxy.golang.org`）にバージョンが記録され、後から消せない。未完成の版が`go install ...@latest`で入ってしまう
- **コミットのメールアドレスは、GitHubのnoreply（`ID+ユーザー名@users.noreply.github.com`）にする。** 履歴に残り、public化で見えるようになる。コミット済みのアドレスを直すには履歴の書き換えが要る。最初のコミットの前に`git config user.email`を決めること
- **名前の同名チェック**（下記「公開前にやること」）は、リポジトリを作る前に済ませるのが望ましい。public化すると名前を変えにくくなる
- READMEの注意書きは、範囲を「未完成で使わないでほしい」だけにする。売り文句・機能の説明・使い方は書かない（看板は最後。「READMEとGitHubの看板」）
- 完成として公開するとき、READMEの注意書きを外し、看板に置き換える。外した後もGitHubのDescriptionの`(work in progress)`を忘れずに外す

### 公開前にやること（完成として公開するとき）

- **文書を英語版と日本語版でそろえる。** 完成として公開する時点で、次のすべてに英語版（`*.md`）と日本語版（`*_ja.md`）がある状態にする
  - 相互リンクを**見出しのすぐ下**（本文の一番上）に置く。現在の言語を太字にする

    ```
    英語版： *[日本語](README_ja.md) | **English***
    日本語版： *[English](README.md) | **日本語***
    ```

  - `README.md` / `README_ja.md`（看板としてのREADMEは最後に作る。下記。注意書きだけのREADMEは、Step 1で先に置く）
  - `docs/reference/`（`schema.md`・`cli.md`は英語版・日本語版とも作成済み。以後も同じ変更で両方を直す）
  - `docs/tour/`、`docs/examples/`（実装完了後に作る。作るときに最初から両方作る）
  - `docs/design/`は日本語のみで、英語版は作らない（設計の経緯の記録であり、利用者向けではないため）
  - 英語版の例は記録の中身も英語、日本語版は日本語（`CLAUDE.md`「ドキュメントの言語」）
- 名前の同名チェック（GitHub、Goのパッケージ等で`mtqg`が使われていないか）と、種別が変わった場合の名前の扱い（設計§3）

## READMEとGitHubの看板

**看板としてのREADMEは最後に作る（英語版と日本語版の両方）。** READMEは中身の説明ではなく、リポジトリの**看板**（外から見た顔）。作りながら見えてきたmtqgの一番の魅力を、完成に近い段階でまとめて打ち出す。初期に書くと、その時点の想像で作った看板に後の実装や文書が引っ張られるおそれがあるため、**段階1〜3の間に看板を作らないこと。**

ただし、リポジトリをpublicで作るため（「公開の2段階」）、**未完成の間は注意書きだけのREADMEを置く。** 注意書きは看板ではない。売り文句・機能の説明・使い方・名前の由来を書かず、「未完成で、まだ使わないでほしい」だけを伝える。看板を引っ張る想像が入り込まないので、上の理由には当たらない。

### 未完成の間の注意書き

`README.md`（英語）と`README_ja.md`（日本語）を同時に置く。相互リンクを見出しのすぐ下に置き、現在の言語を太字にする（「公開前にやること」）。

```markdown
# mtqg

*[日本語](README_ja.md) | **English***

> **Work in progress. Not ready to use yet.**
> mtqg is under active development. There is no released version, and the
> data format and commands may change without notice. Please do not use it in
> your projects for now.
```

日本語版は同じ内容を日本語で書く。**これ以上足さない。** 足したくなったら、それは看板の仕事なので最後に回す。

材料：名前の由来を縦に並べて見せる（`bug`はqaの隣に添える。見せ方は、看板のREADMEを作るときに決める）。

```
(m)emo
(t)odo
(q)a
(g)lossary
```

**GitHubのDescription（仮決め。看板のREADMEを作るときに一緒に見直す）**

> mtqg - (m)emo, (t)odo, (q)a & bugs, (g)lossary: a project journal in your git repo, for humans and AI agents.

`bug`を足した（2026-09-21）ので、`(q)a`のあとに`& bugs`を添えた。名前は変えない（設計§3）。**実際のDescriptionは、当面`work in progress`とだけ書いてある**（人間の判断、2026-09-21）。気にしなくてよい。看板のREADMEを作るときに、この仮決めと一緒に見直す。

未完成の間は、末尾に`(work in progress)`を足す。READMEを開かない人にも伝わるようにするため。

- READMEは`README.md`（英語）と`README_ja.md`（日本語）を同時に作る
- 短さを優先した。候補から外した要素：`append-only`（設計の芯だが長くなる）、「チャットの履歴に埋もれない」という売り文句（READMEで打ち出す）

**GitHubのTopics**

- 今付ける：`cli`、`go`、`git`、`jsonl`、`append-only`、`developer-tools`、`ai-agents`、`llm`、`decision-log`、`todo`、`glossary`、`knowledge-management`
- 段階4でできてから足す：`mcp`、`claude-code`、`vscode-extension`（まだない機能のTopicsを先に付けない）
