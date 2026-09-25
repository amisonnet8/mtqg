# ディレクトリ構成

mtqgは「**コア（ジャーナル層・モデル層）が核、CLIはコアを使う入口の1つ**」という構造を取る（設計§10.7）。新しいファイルを追加する際は、どの層に属するかを必ず意識すること。

```
mtqg/
├── CLAUDE.md               ← プロジェクトルール（参照先の案内）
├── PLAN.md                 ← 実装計画・進捗管理（段階、現在地、保留事項）
├── qsokufile               ← ビルド・テストの近道（qsoku使用。旧Makefile。docs/design/history.md 2026-09-23）
├── LICENSE                 （MIT）
├── go.mod / go.sum
├── .gitignore
├── .gitattributes          （`* text=auto eol=lf`）
├── .golangci.yaml           ← lintの設定（層の依存の向き、文言をCLIの層に限ることも機械的に検査）
├── trivy.yaml              ← 脆弱性・ライセンス検査の設定（testing.md）
├── schema.go               ← docs/reference/schema.md を embed するだけのパッケージ
├── cmd/mtqg/main.go        ← 引数を internal/cli に渡すだけ（数行）
├── internal/
│   ├── journal/            ← 【ジャーナル層】journal.jsonl の読み書きだけ
│   ├── model/              ← 【モデル層】イベントの意味
│   └── cli/                ← 【入口】引数の解釈、英語の文言、表の整形、--json
├── e2e/                    ← ビルドした本物のバイナリと本物のgitで動かすテスト（ビルドタグ e2e）。`examples_test.go`は`docs/reference/`の例の確認で、フィクスチャは`testdata/examples/`
├── docs/
│   ├── reference/          ← 仕様。英語版 schema.md・cli.md と日本語版 *_ja.md
│   ├── design/             ← 設計判断と理由の記録（日本語）
│   ├── tour/               ← 歩いて回る入門（実装完了後に作成。英語＋_ja）
│   └── examples/           ← 実例（実装完了後に作成。英語＋_ja）
├── .devcontainer/
├── .claude/                ← rules/（ルール）、skills/（繰り返す手順。mutation-check）、hooks/、settings.json（人間が管理）
└── .github/workflows/
```

`internal/`配下の具体的なファイル構成は、実装を進めながら決めてよい。ここでは「どの層に属するか」という配置の判断基準を定める。

## 配置の判断基準

- **`.mtqg/`の中のファイル（`journal.jsonl`・`archive/`・`version`・`.local/`）に触れるコード** → `internal/journal/`
  - `.mtqg/`の探索（設計§4.4）と`init`での作成、`version`の確認、行の読み込み、追記、書き直し（`undo`用の汎用操作`Rewrite`）、アーカイブへの移動（`archive.go`。追記と置き換えを1回のロックで）、ロック、IDの生成
  - **gitを読むコード**（`git config user.name`、コミット済みの`journal.jsonl`）は`internal/journal/git.go`に置く。`exec`で本物の`git`を呼ぶだけで、gitへは書かない（git-integration.md）。`.mtqg/`とその置かれたリポジトリの状況を読む部分なので、この層に属する
  - イベントが「todoの完了」か「用語の定義」かは**知らない**。1行のイベントとして扱うだけ
- **イベントの意味を扱うコード** → `internal/model/`
  - 状態の組み立て、検証（memoは完了にできない等）、`undo`・`archive`の対象の選び方（`undo.go`）、並行した状態変更・用語の重複定義・親のない返信の検出（`review.go`）、`archive`の対象の選び方（`archive.go`。期間は時刻で受け取り、日付の解釈はCLI）、検索（`search.go`）、`context`の中身の組み立てと削る順序（`context.go`。測ること・文章にすることはCLI）
  - ファイルを直接開かない。必ずジャーナル層を通す
- **利用者とのやり取り** → `internal/cli/`
  - 種類・動詞・オプションの**表をデータとして持つ**（`args.go`。`help`と`candidates`（シェル補完の候補。`candidates.go`）が同じ表を読む）、英語の文言は`messages.go`に1か所、表示（`render.go`）、本文の入力（引数・標準入力・`$EDITOR`）、環境変数（記録者）
  - 標準入出力・環境変数・現在時刻・端末・ファイルの読み込み（`format`の引数）を`Env`で注入し、`Run(env, args)`をテストから直接呼べるようにする
  - **シェル補完**：`candidates.go`が候補を計算する（文法を持たず、`args.go`の表と、記録を読むモデル層を使う）。`completion.go`が`completions/`の4つのスクリプト（bash・zsh・fish・PowerShell）を埋め込んで出す。スクリプトは**固定のテキスト**で、コマンドの一覧を持たない（毎回`mtqg candidates`に聞く）。`.bash`はShellCheckにかかる（`qsoku shellcheck`）
  - OSで分かれる小さな部分（色の有効化`ansi_*.go`、端末の識別`tty_*.go`）は、ファイルを`_windows.go`と`!windows`で分ける
  - 引数の解釈（`archive`の期間の解釈を含む。`archive.go`）、英語の文言、表の整形、`--json`の出力（形は`json.go`に1か所）、`context`を文章にすること（`context.go`）
  - コアは**構造化された結果とエラーの種類**を返す。文言にするのはここだけ（cli-output.md）
- 依存の向きは `cli → model → journal` の一方向。逆向きのimportを作らない

## 入口を増やすとき

段階4で `internal/mcp/`・`internal/hook/` が加わる（設計§11）。どちらもCLIと同じくコアを呼ぶだけの入口であり、**データの解釈をコア以外に書かない**（設計§11.1「解釈はGoのバイナリの1か所に集約する」）。入口ごとに状態の組み立てを重複させないこと。

## 各ファイル・ディレクトリの補足

- **`schema.go`**: Goの`embed`は、そのパッケージのディレクトリより下のファイルしか埋め込めない。`internal/journal/`から`docs/reference/schema.md`を`..`で辿って埋め込むことはできないため、ルートに埋め込むだけのパッケージを置き、`internal/`からそれを使う。**`schema.md`のコピーを作らない**（仕様の元の文書は1つ・設計§10.8）
- **`internal/`**: Goの仕組みとして、リポジトリの外からimportできない。**コアはライブラリとして公開しない**（設計§10.7）。外との約束はデータ形式（`docs/reference/schema.md`）とCLIの`--json`だけ
- **`go.sum`**: コミットする（改ざん検知・再現可能なビルドのため。Goの標準的な慣習）
- **VSCodeの設定**: `.devcontainer/devcontainer.json`の`customizations.vscode`に置く（拡張と設定を1か所にまとめる。開発環境はdevcontainerだけのため）。`.vscode/`は作らない
- **`.gitattributes`**: `* text=auto eol=lf`。Windowsランナーでの改行コード変換による誤検知を防ぐ（testing.md）。mtqgが`init`で`.mtqg/`の中に置く`.gitattributes`（`*.jsonl text eol=lf merge=union`）とは別物で、両者はぶつからない
- **`docs/reference/`**: 仕様。**実装しながら育てる文書**であり、実装と仕様がずれたらここを更新する。設計判断を変えるときは、まずここを更新してから着手する。英語版（正）と日本語版（`*_ja.md`）を同じ変更の中で両方直す。埋め込むのは英語版の`schema.md`だけ
- **`docs/design/`**: 設計時点の判断と理由の記録。仕様と食い違う場合は`docs/reference/`が正。ここは経緯として残し、書き換えて過去の理由を消さない
- **`docs/tour/`・`docs/examples/`**: 実装完了後に作る。今は作らない。作るときは英語版（`*.md`）と日本語版（`*_ja.md`）を最初から両方作る（`PLAN.md`「公開前にやること」）
- **README**: **看板としてのREADMEは最後に作る**（`PLAN.md`「READMEとGitHubの看板」）。先に作ってはいけない。ただしリポジトリはpublicなので、未完成の間は**注意書きだけの`README.md`・`README_ja.md`**をルートに置く（Step 1）。注意書きに売り文句・機能の説明・使い方を足さない
- **配布物（ビルド済みバイナリ）**: リポジトリにコミットしない（distribution.md）。`.gitignore`には**ルート直下に限定して**`/mtqg`・`/mtqg.exe`・`/dist/`と書く。`mtqg`とだけ書くと、ソースの`cmd/mtqg/`まで無視されて`main.go`がコミットされない（コマンド名とディレクトリ名が同じため）

## `.mtqg/`が段階4から実際に使われている

段階4（2026-09-25〜）から、mtqg自身の開発過程を`.mtqg/`に記録している（`.claude/rules/mtqg-usage.md`）。`mtqg init`はmain製の安定版バイナリで一度だけ実行済み。

## 段階4a（`mtqg hook`・`mtqg init --agent`）で足したもの

- **`internal/hook/`は作らなかった。** `mtqg hook`は`internal/cli/hook.go`・`hook_claude.go`に置く。`session-start`が出す文章は`mtqg context`と同じもので、データを文章にするのはCLIの層の役目（cli-output.md）だから。エージェント間で共通の判断（`ShouldPrompt`）はモデル層（`internal/model/session.go`）に、セッションの状態の読み書き（`.mtqg/.local/sessions/`）はジャーナル層（`internal/journal/session.go`）に置く。段階4bでMCPを足すときに、`internal/hook/`相当の層が本当に要るかは改めて判断する
- **`internal/cli/agent_claude.go`**：`mtqg init --agent claude-code`が`.claude/settings.json`・`CLAUDE.md`を書き換える。`.mtqg/`の外のファイルなのでジャーナル層ではなく、CLIの層に置く（利用者とのやり取りに近い：エージェントの設定ファイルという「入口の外側」を触るため）

## 段階4b（`mtqg mcp`）で足したもの

- **`internal/mcp/`は作らなかった。** `mtqg mcp`は`internal/cli/mcp.go`（サーバー起動、SDKとの接続）・`internal/cli/mcp_tools.go`（ツール表とハンドラ）に置く。MCPのハンドラが要るもの（エラーの文言と`kind`、記録のJSONの形、`context`の文章、本文の正規化、ジャーナルの開き方）は、すべて既に`internal/cli/`に1か所ずつある。別の層に分けると、これらを`cli`から公開して入口どうしの依存を作るか、複製するかになり、「入口ごとに状態の組み立てを重複させない」に反する。JSON-RPCのプロトコル層は依存ライブラリ（後述）が丸ごと引き受けるので、mtqgに残るのは「引数の解釈・文言・出力の形」だけで、これはCLIの層の役目。段階4aで`internal/hook/`を作らなかった判断と同じ構図
- **依存ポリシーの例外**：MCPプロトコル（JSON-RPC 2.0、stdio）の実装に`github.com/modelcontextprotocol/go-sdk`（Anthropic公式、MIT）を使う。CLAUDE.mdの「標準ライブラリと`golang.org/x/`だけ」の例外（理由・承認の経緯は`PLAN.md`「段階4b」）。**機械的な歯止め**として、`.golangci.yaml`の`depguard`で`internal/journal/**`・`internal/model/**`からこのSDKへのimportを禁じ、SDKに触れるのが入口（`internal/cli/`）だけであることを固定する

## 後の段階で増えるもの（今は作らない）

- VSCode拡張（設計§11.4）は、**別リポジトリにする方向**（最終判断は段階4b以降・`PLAN.md`）。TypeScriptで、本体とは`--json`でつながるだけなので、本体のリポジトリをGoのツールチェーンだけで完結させる。このリポジトリにTypeScriptのコードやNode.jsの設定を持ち込まないこと
- `.goreleaser.yaml`：公開の段階（distribution.md）
