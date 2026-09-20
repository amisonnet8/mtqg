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
5. **Step 5: CLI順3** — `context`、全コマンドの`--json`。AIに渡せる
6. **Step 6: CLI順4** — `edit`、`delete`、`undo`、`search`、`review`、`format`
7. **Step 7: CLI順5** — `archive`（`-n`を含む）
8. **Step 8: e2e・docsの例の確認** — 複数クローン・ブランチをまたぐ検証（`.claude/rules/testing.md`「mtqg固有の検証項目」）、`docs/reference/`の例の実測確認の仕組み。Step 3以降、できるところから並行して足してよい
9. **Step 9: シェル補完**

Step 3が動いた時点でサンプルPJ（段階2）を始められる。

**段階1の意図的なスコープ外（段階4以降）:** `mtqg mcp`、`mtqg hook`、`mtqg init --agent`、VSCode拡張、`mtqg upgrade`（形式2が出るまで作らない）、`docs/tour/`・`docs/examples/`、看板としてのREADME（注意書きだけのREADMEはStep 1で置く）、GitHub Releasesのバイナリ。

**段階1完了（v0.1）の判定:** `make check`・`make test`・`make race`が通る／GitHub Actionsの3OSマトリクスがgreen（Windowsでのロックと一時ファイルの置き換えを含む）／`go install`で入れたバイナリで`mtqg version`が意味のあるバージョンを出す／`docs/reference/`の例が実際の出力と一致している。

## 現在地

**段階1 Step 1（足場固め）：ファイルの作成と手元の動作確認は完了（2026-09-20）。次の3つが済めばStep 1完了。**

1. **最初のコミット**：`git config user.email`をGitHubのnoreplyに直してから（グローバルの`~/.gitconfig`は個人のGmailのままなので、このリポジトリだけ上書きする）。コミットは、既存の文書一式と、Step 1の足場の2つに分ける
2. **`PostToolUse`フックの反映**：提案済み（下記「保留事項」）。`.claude/settings.json`は人間が書き換える
3. **最初のpushでCIの3OSがgreenになること**：`.github/workflows/ci.yml`の構文は、ローカルにYAMLの検証手段が無く未検証。Windowsでの`choco install make`、macOSの`-race`、Trivyの脆弱性DBの取得（ghcrの取得制限）は、初回のCIで初めて確かめられる

手元で確かめたこと：`make build`・`make check`（vet・lint・単体テスト）・`make race`・`make shellcheck`・`make trivy`がすべて通る。`make test`は、e2eがまだ無いので「無い」と表示して終わるだけ（Step 8で中身を入れる）。Trivyは依存が0件のため、ライセンスの検出はまだ確かめられていない（「未確認事項」2）。

リポジトリにあるコードは、`go.mod`、`schema.go`（`docs/reference/schema.md`をembed、`schema_test.go`で確認）、`cmd/mtqg/main.go`（空の入口）のみ。`internal/`はまだ無い（Step 2から）。

**公開の方針を決めた（2026-09-20）：リポジトリは最初からpublicにする。** 完成してから公開するのではなく、未完成のまま公開し、READMEの注意書きで「まだ使わないでほしい」と伝える（下記「公開の2段階」）。GitHubリポジトリの作成と`git init`は人間が行い、済んだらStep 1に入る。

**名前の同名チェック（2026-09-20、Claude Codeが実施。公開中・未完成の段階として十分）：** `amisonnet8/mtqg`は未使用（404）。pkg.go.devの検索は0件、モジュールプロキシにも記録なし。npm・PyPI・crates.io・Homebrew（core）に`mtqg`は無い。GitHubの名前検索では別のアカウントの`mtqg`（中国語のEC系の記述が2件）と、ベトナム語の略語`MTQG`（「国家目標」）を使うリポジトリがあるが、いずれも無関係で、Goのパスは`github.com/amisonnet8/mtqg`なので衝突しない。**製品・企業・ブランドの同名チェック（同日）：** Web検索（ソフトウェア・製品・企業・商標・略語の意味）で、`MTQG`という製品・企業・ブランドは見つからなかった。紛らわしいのは綴りが近いものだけ：`MTG`（株式会社MTG＝美容機器、Magic: The Gathering、「meeting」の略）と`MQTT`（IoTのプロトコルとそのツール群）。どちらも綴りが違い、分野も違う。`mtqg.com`はGoDaddyの売り出し用の駐車ページ（製品や企業のサイトではない）。`mtqg.dev`・`mtqg.io`は名前解決できなかったが、登録の有無は確認できていない（RDAPが403）。**未確認：** apt・Scoop・winget・Nixなどの他のパッケージマネージャー、正式な商標調査（USPTO・J-PlatPat・WIPOの検索は、Web検索では確かめられなかった）、ドメインの登録状況。完成として公開するとき（「公開前にやること」）に、範囲を広げて再確認する

`docs/design/`は、開発開始前に書かれた「構想メモ」と「CLI検討案」を分割したもの。未決事項はこのファイルに移した。

## 未確認事項（実装前に決める・確かめる）

1. ~~Goのモジュールパス。~~ **確定：`github.com/amisonnet8/mtqg`**（コマンドは`github.com/amisonnet8/mtqg/cmd/mtqg`）
2. **Trivyのライセンス検出がGoの依存で効くか。** `trivy.yaml`は配置済み（脆弱性とライセンス、HIGH・CRITICALで失敗、ライセンスの分類はTrivyの既定＝GPL系は失敗、MIT・BSD・Apache・MPLは通る）。最初の依存を足したとき（`go mod download`後）に、依存のライセンスが実際に検出・表示されることを確かめる
3. **JSONの書き出しに`encoding/json`と`encoding/json/v2`（Go 1.27）のどちらを使うか。** 書き出し規則（`.claude/rules/journal-format.md`）を満たせばどちらでもよい。v2の既定のエスケープの挙動を確かめて決める
4. **e2eの仕組み。** `testscript`を第一候補として、Step 3〜8の間に決める。決めたら`.claude/rules/testing.md`に追記する
5. **ロックの待ち時間**（何秒待ってエラーにするか）

## 保留事項

- **`PostToolUse`フックが未設定（2026-09-20に提案済み、反映待ち）。** `Makefile`ができたので、`.claude/hooks/`にスクリプトを置き、`.go`・`go.mod`・`go.sum`の編集後に`make build`を実行し、失敗時は理由をClaudeへ返す形を提案した。`.claude/settings.json`は人間が管理しているので、設定の変更は提案にとどめる。スクリプトはShellCheckの対象になる（`make shellcheck`）。複数ファイルにまたがる編集の途中は、まだ書いていないファイルを参照してビルドが一時的に失敗し、その通知が出る点に注意
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

材料：名前の由来を縦に並べて見せる。

```
(m)emo
(t)odo
(q)a
(g)lossary
```

**GitHubのDescription（仮決め。看板のREADMEを作るときに一緒に見直す）**

> mtqg - (m)emo, (t)odo, (q)a, (g)lossary: a project journal in your git repo, for humans and AI agents.

未完成の間は、末尾に`(work in progress)`を足す。READMEを開かない人にも伝わるようにするため。

- READMEは`README.md`（英語）と`README_ja.md`（日本語）を同時に作る
- 短さを優先した。候補から外した要素：`append-only`（設計の芯だが長くなる）、「チャットの履歴に埋もれない」という売り文句（READMEで打ち出す）

**GitHubのTopics**

- 今付ける：`cli`、`go`、`git`、`jsonl`、`append-only`、`developer-tools`、`ai-agents`、`llm`、`decision-log`、`todo`、`glossary`、`knowledge-management`
- 段階4でできてから足す：`mcp`、`claude-code`、`vscode-extension`（まだない機能のTopicsを先に付けない）
