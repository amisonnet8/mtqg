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
8. **Step 8: e2e・docsの例の確認** — 複数クローン・ブランチをまたぐ検証（`.claude/rules/testing.md`「mtqg固有の検証項目」）、`docs/reference/`の例の実測確認の仕組み。Step 3以降、できるところから並行して足してよい。**例の取得は、Step 3・4・4.5・5・6・7で同じ手作業（日時とIDを固定した記録を作る→本物のバイナリで動かす→文書の該当ブロックを差し替える）を6回繰り返した**（作業用のスクリプトはセッションの一時領域で、リポジトリには無い）。**完了（2026-09-21）：**フィクスチャを`e2e/testdata/examples/`に置き、文書の例をHTMLコメントの印で`Run`（時計を固定）と本物のバイナリに突き合わせる仕組みにした。`make docs-examples`が文書に書き戻す（`testing.md`「e2eとdocsの例の確認」）。一覧の未実施のe2eもすべて足した
9. **Step 9: シェル補完** — **実装と手元の検証が済んだ（2026-09-21。CIの確認待ち）。**動的：`mtqg completion <shell>`（bash・zsh・fish・PowerShell）が固定のスクリプトを出し、TABのたびに`mtqg candidates`が`args.go`の表から候補（コマンド・オプション・`--kind`の値・ID）を計算する。本物の4つのシェルで動かして確かめた（`e2e/completion_test.go`）

Step 3が動いた時点でサンプルPJ（段階2）を始められる。

**段階1の意図的なスコープ外（段階4以降）:** `mtqg mcp`、`mtqg hook`、`mtqg init --agent`、VSCode拡張、`mtqg upgrade`（形式2が出るまで作らない）、`docs/tour/`・`docs/examples/`、看板としてのREADME（注意書きだけのREADMEはStep 1で置く）、GitHub Releasesのバイナリ。

**段階1完了（v0.1）の判定:** `make check`・`make test`・`make race`が通る／GitHub Actionsの3OSマトリクスがgreen（Windowsでのロックと一時ファイルの置き換えを含む）／`go install`で入れたバイナリで`mtqg version`が意味のあるバージョンを出す／`docs/reference/`の例が実際の出力と一致している。

## 現在地

**段階1 Step 9（シェル補完）：完了（2026-09-21、CIの3OSがgreen。人間が確認）。これで段階1のステップはすべて終わった。次は「段階1完了（v0.1）の判定」（下）と、段階2（サンプルPJ）へ進むかの人間の判断。v0.1のタグは打たない（決定済み。下）。**

**できたもの：**
- **仕様**（`docs/reference/cli.md`・`cli_ja.md`の「Shell completion」。実装より先に書いた）。`mtqg completion <shell>`（`bash`・`zsh`・`fish`・`powershell`。ほかは終了コード2）と、`mtqg candidates [--word=<打ちかけの語>] -- <語>...`。候補は1行1件（`値`、または`値<TAB>説明`）。`help`にも`--json`のコマンド一覧にも出る（隠しコマンドにしない）。
- **`internal/cli/candidates.go`**：語を寛容に歩き（`parseArgs`は使い回さない。誤りで止まるため）、位置から候補を決める。**文法を持たず、`args.go`の表を読む**。表に`ids`（`idsOpen`・`idsDone`・`idsAll`・`idsParents`）と`choices`のフィールドを足した。IDはモデル層から新しい順に。**何があっても終了コード0で、標準エラー出力に何も出さない**（`.mtqg/`が無い、形式が新しすぎる、衝突マーカー、読めない行）。説明は1行・60桁で切る（`--json`でも）。
- **`internal/cli/completion.go`と`completions/`**：4つのスクリプト（固定のテキスト。コマンドの一覧を持たない）を`embed`して出す。打ちかけの語は`--word=<語>`で渡す（空の引数を渡さない）。
- **環境**：`.devcontainer/postCreate.sh`にzsh・fish・PowerShell（Linux版のpwsh、Microsoftのaptリポジトリ）を足した。今のコンテナには同じコマンドで入れた（fish 3.6.0、pwsh 7.6.6）。`make shellcheck`が`.bash`も対象にする。

**Step 9で決めたこと（この会話で確認済み。理由は`docs/design/history.md`）：** ①動的（スクリプトに文法を焼き込まない）。②候補を出すコマンドは公開（`candidates`）。③4つのシェル（最初はbash・zsh・fishの3つを推奨したが、人間の判断でPowerShellも入れ、テストは手元のLinux版pwshで動かす）。④コマンド・オプション・`--kind`の値に加えてIDまで補完する。実装前に決めたこと：1文字の短縮形は候補に出さない、`todo add`などの本文と用語は補完しない、`--kind=todo`の形は補完しない（`--kind <TAB>`）、10桁が重なるIDは完全なIDで出す、打ちかけの語は`--word=`で渡す。

**手元で確かめたこと：** `make check`・`make test`（e2e）・`make race`・`make trivy`・`make shellcheck`が通る。macOS・Windows向けに`go vet`（`-tags e2e`も）とテストのコンパイルが通る（**実行はCI**）。**本物のシェルで動かして確かめた**：bash（`COMP_WORDS`を組んで`_mtqg`）、fish（`complete -C`）、PowerShell（`TabExpansion2`。引用符つきの`-C`、行の途中でのTAB、`$LASTEXITCODE`が変わらないことも）、zsh（`_mtqg`を直接。補完のしくみ自体は端末が要るので動かしていない）。壊して確かめた（すべて検出）：候補の絞り込み・並び・10桁の扱い・オプションの表との一致・警告を出す・スクリプトの語の渡し方（4つとも）など（`testing.md`）。1件だけ生き残った：fishの`"--word=$cur"`の引用符を外しても、fish 3.6では通る（残した）。`make docs-examples`は冪等（例は30個ずつ、英日同数）。

**実際のシェルで動かして見つけた不具合（直した）：**fishのスクリプトで、語が0個のとき`printf '%s\n' $words`が空行を1つ出し、空の語が1つ渡って「未知のコマンド」になり、`mtqg t<TAB>`が何も出さなかった。

**CIで確かめられたこと：** 3OSのCIがgreen（人間が確認）。**分からないこと：**どのシェルのe2eが実際に動き、どれがskipされたか（CIは`go test -v`ではないので、skipは見えない。見たくなったら`-run 'TestCompletionIn' -v`を足したジョブで確かめる）。**確かめられないこと：**対話のシェルでの実際のTAB（bashが`=`で語を切ること、zshの補完のしくみ自体、fish 4系、PowerShell 5.1の引数の渡し方）。

**段階1完了（v0.1）の判定の進み具合：** `make check`・`make test`・`make race`は通る（手元）／**3OSのCIがgreen：確認済み（2026-09-21、人間）**／`go build`のバイナリの`mtqg version`は`v0.0.0-<コミット時刻>-<ハッシュ>+dirty`の疑似バージョンを出す。**`go install ...@タグ`で意味のあるバージョンが出るかは、タグを打つまで確かめられない**（`@main`で試すこともできるが、プロキシにコミットが記録されるので、人間の判断を待つ）／`docs/reference/`の例は実際の出力と一致している（30個、`make docs-examples`が冪等）。**判定の4項目のうち、`go install`の1つだけが未確認。**

**v0.1のタグは打たない（人間の判断、2026-09-21）。** 「公開の2段階」の決まり（v0.1まではタグを打たない。タグはプロキシに残り、後から消せない）のまま、リポジトリは「公開中・未完成」の段階に留まる。看板のREADMEと`docs/tour/`・`docs/examples/`は作らない。したがって、判定の残り1項目（`go install ...@タグ`で意味のあるバージョンが出るか）は、タグを打つ日まで確かめない。

**Step 9に含めなかったもの：** 用語（`g add <TAB>`）・`archive`の期間・`search`の語・`--limit`の値の補完、`docs/tour/`・`docs/examples/`、看板としてのREADME（実装完了後）。

**Skillを作った（人間の判断。Step 9の提案から）：** `.claude/skills/mutation-check/`。「壊して確かめる」（変異を入れて、テストが落ちるかを見て、戻す）を、Step 7・8・9で毎回その場のシェル関数で書いていたので固定した。`mutate.sh`が、ファイルを退避し、文字列を置き換え、テストを走らせ、killed／SURVIVED／コンパイルできない、を言って、**どう終わっても元に戻す**。`SKILL.md`に、変異の一覧の書き方と、生き残った変異の読み方（変異が何も変えていない／テストが足りない／本当に同じ意味）を書いた。`mutate.sh`は、killed・生き残り・コンパイルできない・文字列が無い・変異前にテストが落ちている・使い方の誤り、の6つの終わり方と、SIGTERMで中断しても元に戻り一時ファイルが残らないことを、実際に動かして確かめた（`make shellcheck`の対象）。

**（前の状態）** **段階1 Step 8（e2eとdocsの例の確認）：完了（2026-09-21、CIの3OSがgreen。人間が確認）。次はStep 9（シェル補完）。** これで段階1は、シェル補完を除いて終わった。

**できたもの：**
- **docsの例の確認**（`e2e/examples_test.go`）。`docs/reference/cli.md`・`cli_ja.md`の`$`で始まるコードブロックは、直前のHTMLコメント`<!-- mtqg:example repo=parser -->`の指定で、`e2e/testdata/examples/`のフィクスチャ（`parser`・`parser_ja`・`years`・`ambiguous`・`refuse`・`broken`。`none`・`empty`は特別）から作ったリポジトリのコピーで動かし、出力を文書と比べる。**`Run`を直接、`Env.Now`（2026-09-21 12:00 UTC）と`Env.Location`を固定して呼ぶ**（時計の裏口を製品に作らない）。`binary`の印のブロック（`archive`・`format`）は、本物のバイナリでも同じ出力になることを確かめる。`make docs-examples`が実際の出力を文書に書き戻す（**例は手で書かない**）。例が28個（うち1つは`skip=`）、英語版・日本語版とも。取りこぼしを止めるテスト（印なし、理由なしの`skip=`、フィクスチャの言語違い、2言語で例の数が違う）。
- **e2eの残り**（`.claude/rules/testing.md`の一覧はすべてe2eにある）：`boundaries_test.go`（衝突マーカー、知らない`version`、`.local/`を消した後、submoduleと既存`.mtqg/`への`init`）、`lock_unix_test.go`（worktreeとシンボリックリンクのロック。**LinuxとmacOSだけ**）、`records_test.go`（曖昧なID、種類の取り違え、`edit`・`delete`・`search`）。

**Step 8で決めたこと（この会話で確認済み。理由は`docs/design/history.md`）：** ①`Run`を時計を固定して直接呼ぶ（`MTQG_NOW`は作らない）。②印はフェンスの直前のHTMLコメント。③今ある例は、1つの整合したフィクスチャから作り直して差分を確認する。④e2eは一覧の未実施をすべて足す。**`testscript`は要らないと結論した**（未確認事項4）。

**手元で確かめたこと：** `make check`・`make test`（e2e）・`make race`・`make trivy`・`make shellcheck`が通る。macOS・Windows向けに`go vet`（`-tags e2e`）とテストのコンパイルが通る（**実行はCI**）。仕組みを壊して確かめた（すべてテストが検出）：例を1文字変える、印を外す、理由のない`skip=`、`ids=any`を外す、英語版が日本語のフィクスチャを使う、残った印、本物のバイナリの出力を変える。`make docs-examples`を続けて2回実行しても差分が出ない。新しいe2eは、製品側を壊して確かめた（すべてe2eが検出）：追記が衝突マーカーを無視する、新しい形式を受け入れる、`.local/`を作り直さない、ロックが待たない、曖昧な前方一致が最初の1件を選ぶ、質問を消しても回答が残る、3桁のIDを受け入れる。

**作り直しで見えたこと：** 例は28個（うち1つは`skip=`で、`mtqg version`）。食い違っていたのは、英語版が1つのブロック（`todo list`の2つの例。Step 3のフィクスチャのまま「3 open」だったが、同じ文書の`status`は5）、日本語版が2つのブロック（同じ例と、`undo`の拒否の例のID。英語版と別の値）だけで、残りは最初から実際の出力と一致していた。実装の不具合は見つからなかった。

**CIで見つかったこと：** 最初のpushで、例の確認のうち`path=`を使う4つ（`init`と`No .mtqg/ found`の例、英日）がmacOSとWindowsだけで落ちた。mtqgはリンクと短縮名を解決したパス（`/private/var`、Windowsの長い名前）を出すのに、例を動かす一時ディレクトリを解決していなかった。`EvalSymlinks`で解決して直り、CIの3OSがgreenになった（`testing.md`。手元では`TMPDIR`をリンクにすると再現する）。**CIで確かめられたこと：** 例の確認（`git show HEAD`の出力、`format`が相対パスのファイルを読むこと、`.git`の読み取り専用のファイルを含むコピー）と新しいe2eが、Windows・macOSで通る。**確かめられないこと：** `lock_unix_test.go`はWindowsでは走らない（ジャーナル層のテストが担当）。

**Step 8に含めなかったもの：** シェル補完（Step 9）、`docs/tour/`・`docs/examples/`（実装完了後）、`schema.md`の例（コマンドの出力ではないので対象外）。

**提案（作っていない）：** 「例のブロックを足すときは、印を付けて`make docs-examples`」という手順は`.claude/rules/testing.md`に書いた。Skillにするほどの繰り返しではない。

**（前の状態）** **段階1 Step 7（`archive`）：完了（2026-09-21、CIの3OSがgreen。人間が確認。Step 5・6の分も同時に確認済み）。次はStep 8（e2eとdocsの例の仕組み）。** これで**段階1のコマンドはすべてそろった**（残りはStep 8の検証の仕組みとStep 9のシェル補完）。

**できたもの：** ジャーナル層の`Archive`（アーカイブへ追記して`fsync`→`journal.jsonl`を置き換え。移す行＋残す行が読んだ行と合わなければ何も書かない。`Rewrite`と読み込み・置き換えを共有）。モデル層の`ArchiveTargets`（項目＝記録＋従う回答・返信。最後のイベントで期間を判定）。CLIの`archive`（範囲の解釈`parseRange`、`-n`／`--dry-run`、`--json`は件数だけ、記録者は要らない）。未実装のコマンドが無くなったので、`not_available`のテストは`withUnbuiltCommand`（テストの中で仮のコマンドを足す）に直した（e2eは該当のケースを外した）。

**Step 7で決めたこと（この会話で確認済み。理由は`docs/design/history.md`）：** ①削除した記録は種類も状態も問わず対象。②質問・バグの「最後のイベント」は回答・返信（削除したものを含む）まで含めたスレッド全体。③親のない回答・返信はmemoと同じ1項目。④`--json`は件数だけ。実装前に決めたこと：`Archived:`に回答・返信の件数を足す、`-n`は1行目に`(dry run)`、何も移さないときは`Archived: nothing`でファイルを作らない、順序は「アーカイブへ追記して`fsync`→`journal.jsonl`を置き換え」、`create`の無い`id`のイベントは移さず残す。実装中に決めたこと：JSONに`lines`（移した行数）は入れない（dry runでは生の行数が取れず、`records`で足りる）。

**手元で確かめたこと：** `make check`・`make test`（e2e）・`make race`・`make trivy`・`make shellcheck`が通る。macOS・Windows向けに`go vet`（`-tags e2e`も）とテストのコンパイルが通る（**実行はCI**）。壊して確かめた変異（すべてテストが検出。1つだけ最初は生き残り、`from`がゼロ時刻の場合のテストを足した）：順序を逆にする、ロックを外す、行数の検査を外す、何も移さなくても`archive/`を作る、改行を足さない、glossaryを移す、未完了を移す、回答を親と移さない・回答の日付を見ない、削除を移さない、終了日を含めない・開始日を含めない、UTCで判定する、`-n`が書く、読めなかった行も移す、記録者を要る、範囲の誤りを通す。`cli.md`・`cli_ja.md`の`archive`の例は、**日時とIDを固定した記録（2024・2025年）を本物のバイナリで動かした実際の出力**（英語版は中身も英語、日本語版は日本語）。

**実装して見えたことと、直したこと：** ①**別々のブランチが隣り合う行をそれぞれアーカイブしてマージすると、unionマージは両方の行を`journal.jsonl`に戻す**（アーカイブのファイルにも残る。もう一度アーカイブすれば片付く）。設計§9.1の「割り切っていること」と同じ仕組みで、e2eで実際に確かめた。行が離れていれば戻らない。仕様（`schema.md`）に1文足した。②並行テストの「証明にならない」ガードが、実行時間を縮めたら間欠的に発火した（追記を決めた回数で止めていたため）。追記を「アーカイブが必要な回数を終えるまで続ける」形にして直した（`testing.md`）。

**Step 7に含めなかったもの：** docsの例の確認の仕組み化（Step 8。**同じ手作業が今回で6回目**）、シェル補完（Step 9）、`unarchive`（作らない・設計§9.1）。

**CIで確かめられたこと（Step 5〜7）：** Windows・macOSでの`archive/`の作成と追記、`journal.jsonl`の置き換え、`context`・`edit`・`undo`・`review`など新しいコマンドの動き、`MTQG_TTY`からの`tty`（e2e）。**CIで確かめられないこと：** Windowsの色と端末の幅（変わらず）。デバイス番号からの`tty`（CIには端末が無い）。書き込み失敗で順序を確かめるテストは、Windowsとrootではskipされる。

**（前の状態）** **段階1 Step 6（`edit`・`delete`・`undo`・`search`・`review`・`format`、`tty`）：実装と手元の検証が済んだ（2026-09-21）。CIの3OSの確認待ち（人間がpushして確認。Step 5の分も未確認）。次はStep 7（`archive`）。** これで、記録を書く・読む・直す・消す・取り消す・探す・食い違いを見る、が一通りそろった。

**できたもの：** `tty`（端末のデバイス番号か`MTQG_TTY`のsha256の先頭8桁。`Env.TTY`と`tty_unix.go`／`tty_windows.go`）。`edit`（今の本文を入れて`$EDITOR`が開く。同じ本文なら`Unchanged:`）、`delete`（一緒に隠れた回答・返信の件数と記録者を言う）、`$EDITOR`は書く本文が無いときすべて（`q add <id>`・`b add <id>`・`g add <word>`）で開く。`undo`（モデル層の`UndoTarget`・`Orphaned`・`CanUndo`。CLIは`Rewrite`の中で行を返すか、断って何も書かない）。`search`（`Search`。`log`と同じ行）。`format`（`.mtqg/`を要らない。順序はモデル層の`EventOrder`と共有。diffの文脈行は表に出さない）。`review`（`ConcurrentStatusChanges`・`UnattachedReplies`）。`status`の`Conflicts`の行、`context`のAttentionの並行した状態変更、`show`の`re`の文言（`to bug <id>`と言い切らない）。

**Step 6で決めたこと（この会話で確認済み。理由は`docs/design/history.md`）：** ①種類をまたぐ`re`の行は**案C**。②`$EDITOR`は**書く本文が無いときすべて**で開く。③`undo`が断るのは**`create`を消すと他のイベントが記録なしで残るときだけ**。④`tty`は**端末のデバイス番号のハッシュ、`MTQG_TTY`を今入れる**。実装前に決めたこと：`undo`の「最後の行」は`ts`が最後、同じ秒ならファイルで最後（`id`はランダム）。並行した状態変更は、その時点の状態と`from`の比較（同じ`from`が2つ、では`open→done→open→done`を誤検出する）。

**手元で確かめたこと：** `make check`・`make test`（e2e）・`make race`・`make trivy`・`make shellcheck`が通る。macOS・Windows向けに`go vet`（`-tags e2e`も）とテストのコンパイルが通る（**実行はCI**）。壊して確かめた変異（すべてテストが検出）：`edit`が`word`・状態も書く、同じ本文でも書く、`delete`が回答の件数を言わない、`undo`が端末・記録者を見ない、宙に浮く判定をしない、同じ秒を`id`で決める、時刻でなくファイルの位置だけで選ぶ、同じ内容の行を1つしか消さない、`search`が大小を区別する・用語を見ない、`format`が時刻順に並べない・`-`の行に印を付けない・文脈行を出す、並行した状態変更が「同じ`from`」で判定する・判定しない、`status`・`context`が`Conflicts`を出さない、`UnattachedReplies`が全部の返信を返す。`cli.md`・`cli_ja.md`の`status`・`log`・`search`・`review`・`context`・`format`・`edit`／`delete`・`undo`の例は、**日時とIDを固定したフィクスチャ（並行した状態変更と、親のない返信を入れたもの）を本物のバイナリで動かした実際の出力**（英語版は中身も英語、日本語版は日本語。`undo`の例だけは、`init`しただけの小さなリポジトリで取った）。

**実装して見えたことと、直したこと：** ①`git show`の出力を`format`に渡すと、diffの文脈行（変更していない行）まで表に出て、そのコミットで書かれていない行が混ざった。**直した：**文脈行は表に出さず、状態変更・削除の本文を補うだけにした（仕様が先）。②`undo`の例をフィクスチャで取ると、フィクスチャにある今日の未来の時刻の行が「最新」になり、別の行が消えた。これは`ts`で選ぶ以上の事実でもあるので、**仕様に1文足した**（進んだ時計で書かれた行は、時刻が追いつくまで最新）。例は小さなリポジトリで取り直した。③`show`の文言が、`re`が質問を指す返信を`to bug <id>`と言っていた欠陥（Step 4.5から）は、案Cとして直した。④「まだ作っていないコマンド」のテストが`undo`を使っていて、作ったら3か所が落ちた。今は`archive`を使っている（Step 7で未実装のコマンドが無くなる。`testing.md`）。

**Step 6に含めなかったもの：** `archive`（Step 7）、docsの例の確認の仕組み化（Step 8。**今回で同じ手作業が5回目**）、シェル補完（Step 9）、並行した**回答**の検出と`search`の絞り込み（サンプルPJで判断）。

**CIで確かめられないこと：** Windowsの色と端末の幅（変わらず）。デバイス番号からの`tty`は、CIには端末が無いので確かめられない（手元のptyでは、`script`で実行して`tty`が付くことと、パイプ越しでは付かないことを確かめた）。`MTQG_TTY`からの`tty`は、e2eが3OSで確かめる。

**（前の状態）段階1 Step 5（`context`と全コマンドの`--json`）：実装と手元の検証が済んだ（2026-09-21）。CIの3OSの確認待ち（人間がpushして確認）。** これで**AIに渡せる**：`mtqg context`をセッションの始めに読ませ、`--json`で拡張やスクリプトがつながる。サンプルPJ（段階2）でAIにも使わせられる。

**できたもの：** `--json`（実装済みの全コマンド。`internal/cli/json.go`に形を1か所）と、エラー・警告の報告（`report.go`。人間向けの文言とJSONの`kind`を1か所で作る。`describe`を`reportOf`に一般化し、既存の文言は変えていない）。`context`：ジャーナル層に`GitBranch`（`git branch --show-current`）、モデル層に`context.go`（中身と削る順序：`Context`・`Steps`・`Reduced(n)`）、CLIに`context.go`（文章にする、文字数からの見積もり、収まる最小の削りを二分探索）。`--max-tokens`（既定2000、`0`で上限なし）。`help --json`は、まだ作っていないコマンドも`available:false`で返す（Step 9のシェル補完の土台）。

**Step 5で決めたこと（この会話で確認済み。理由は`docs/design/history.md`）：** `--json`は常にJSONオブジェクト1つ（最初のフィールドは`command`）。エラーと警告も、標準エラー出力に1行のJSON（`kind`で機械が見分ける）。`context`の分量は文字数からの近似（ASCII 4文字＝1トークン、ほかは1文字＝1トークン）で、近似であることを仕様に明記。削る順序は仕様で固定。ブランチ名は`git branch --show-current`。下書きの`(mtqg review)`は`(see mtqg glossary list)`にした（`review`はStep 6）。

**手元で確かめたこと：** `make check`・`make test`（e2e）・`make race`・`make trivy`・`make shellcheck`が通る。macOS・Windows向けに`go vet`とテストのコンパイルが通る（**実行はCI**）。`--json`の4つの変異（本文を`sanitize`する、IDを短縮する、エラーを標準出力へ、UTCでなくする）と、`context`の変異（todoを新しい順に削る、定義より先に質問を削る、予算を無視する、完了したtodoが混ざる、下限を外す・変える、bugだけ下限を外す）を、テストが検出した（完了したtodoと、bugだけ下限を外す変異は、最初はテストが検出できなかったので、フィクスチャとテストを足した）。`cli.md`・`cli_ja.md`の`--json`と`context`の例は、**日時とIDを固定した記録を本物のバイナリで動かした実際の出力**（英語版は中身も英語、日本語版は日本語）。

**実装して見えたことと、直したこと：** 仕様どおりの削る順序では、質問とbugをすべて削ってから初めてtodoを削るので、todoが多いと質問が1件も見えなくなった。**直した（人間の決定）：最新の3件の質問と3件のbugは、予算に収まらなくても最後まで削らない。** それより古い質問・bugを（2つの区画をあわせて）古い順に削り、そのあとでtodoをなくなるまで削る。元の順序は3つのAI（Copilot、Gemini、ChatGPT）に尋ねて決めたもので、今回の問題も同じ3つに尋ね、3つとも「下限を持たせる」を選んだ（交互に削る案は採らなかった。`docs/design/history.md`）。下限の3件が妥当かは、サンプルPJで判断する。

**Step 5に含めなかったもの：** `review`と、`status`・`context`の「並行した状態変更」（Step 6）、`edit`・`delete`・`undo`・`search`・`format`（Step 6）、`archive`（Step 7）、シェル補完（Step 9）。種類をまたぐ`re`の行の読み方（A・B・C）はStep 6の`review`で決める（今は今の振る舞いのまま`--json`に出る）。

**CIで確かめられないこと：** Windowsの色と端末の幅（変わらず）。`context`のブランチ名は、e2eが本物のgitで確かめる（3OSで初めて動く）。

**（前の状態）段階1 Step 4.5（5つ目の種類`bug`）：完了（2026-09-21、CIの3OSがgreen。人間が確認）。次はStep 5（`context`と全コマンドの`--json`）。** ユーザーの決定で`bug`を足した（サブコマンド`bug`、1文字`b`。qaと同じ形で、`type`に`bug`を足すだけ。返信の`re`はbugだけを指す）。`bug add`（不具合と返信）・`list`・`done`・`reopen`、`show`・`log --kind bug`・`status`の`Open bugs`の行。5種類（memo・todo・qa・bug・glossary）が揃った。

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
4. **e2eの仕組み。** **決定（2026-09-21）：`e2e/`に、ビルドタグ`e2e`のGoのテスト。ビルドした本物のバイナリと本物のgitを`exec`で動かす。** `testscript`は`golang.org/x/`ではないので使わない（依存の基準）。**Step 8で再評価し、`testscript`は要らないと確定した**（2026-09-21。文書の印と`-update`で足りた。`.claude/rules/testing.md`）
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

- `context`の出力内容と分量、上限を超えたときの優先順位（設計§9、§11.3）。**質問とbugの下限（最新の3件ずつ）が妥当か**（上の「実装して見えたことと、直したこと」）。既定の2000トークンが多すぎ・少なすぎないか。冒頭の指示文を外すオプション（`--no-guide`）が要るか。「最近の記録」が上の区画と重なっている（同じ記録を2回読む）のが無駄でないか
- qaの回答の「確定」をどう表現するか（設計§5.2）。`qa list`で確定待ちをどう見分けさせるか（記録者の種別で足りるか、確定の事実を別に持つか）
- 変更前の状態（`from`）による並行した状態変更の判定方式の妥当性（設計§7.6）
- 並行した回答を`review`で検出するか
- memoやtodoにも親（`re`）を持たせたくなるか
- **bugの使い分けと線引き：** bugとqa・todoで書き分けに迷うか。bugに重要度・再現手順などの欄が欲しくなるか（設計§2.7の線引き「bugはqaと同じ形に留める」が保てるか）
- ~~**種類をまたぐ`re`の行の読み方**~~ **決定（2026-09-21、Step 6）：案C。**読み方は今のまま（返信として結び付けず、エラーにもしない）、`show`の文言を直し、`review`に「親のない回答・返信」の区画を足す。サンプルPJで、この区画が実際に役に立つか（そもそもそのような行が生まれるか）を見る
- AIの記録者名の粒度（モデル名まで残すか）（設計§5.2）
- 成果物の位置（`at`）の形式（設計§5.5）
- `log`の既定の件数と並び順、`search`に必要な絞り込み（種類、記録者、期間。今は持たない）
- 引用符なし入力で実際に困る場面がどれくらいあるか
- `undo`の端末識別子（デバイス番号のハッシュ）で誤爆が起きるか（閉じた端末の番号は再利用される。起きたら、シェルのプロセスIDなどを混ぜる。設計§4.5）。~~後続イベントがある場合に断るか~~は決定済み（Step 6：`create`を消すと記録なしでイベントが残るときだけ断る）
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
