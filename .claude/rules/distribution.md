# 配布方法

理由は設計§10.11。

## v0.xの間は`go install`だけ

```
go install github.com/amisonnet8/mtqg/cmd/mtqg@latest
```

モジュールパスは`github.com/amisonnet8/mtqg`。

- 配布のための仕組みを作らない。タグを打てば、それだけでバージョンを指定して入れられる
- **v0.1〜v0.2の間はタグを打たなかった（2026-09-21の人間の判断）。** リポジトリは最初からpublicで、タグを打つとGoのモジュールプロキシ（`proxy.golang.org`）にバージョンが記録され、後から消せない。未完成の版が`@latest`で入ってしまう懸念があった（`PLAN.md`「公開の2段階」）
- **`v0.2.0`から実際にタグを打つことにした（人間の判断、2026-09-25）。** 段階1（v0.1の条件）・段階3（v0.2の条件、qsokuでの検証）に加え、`rule`の追加・段階4a（エージェントのフック）・4b（MCP）まで進んだ現状を最初のタグとする。v0.1は別途打たない。**「公開中・未完成」の段階自体は変わらない**：README（`README.md`・`README_ja.md`）は看板化せず、注意書きのまま。タグは`go install ...@v0.2.0`のような**バージョン指定**を可能にするだけで、「完成として公開する」（`PLAN.md`「公開の2段階」）とは別の判断
- **`.goreleaser.yaml`とGitHub Releasesのバイナリ配布は、v1.0.0正式公開の準備として2026-09-28に作った。** 今の形は、下の「配布物の作り方（goreleaser）」と「リリースの流れ」（タグのpushだけで、検査・配布物・公開まで自動。2026-10-22、全PJで統一した人間の決定）。goreleaser自体は`go install github.com/goreleaser/goreleaser/v2@latest`で入れる開発ツールで、`go.mod`には入れない
- Homebrewなどのパッケージマネージャーは、要望が出てから考える

## 配布物の作り方（goreleaser）

`cmd/mtqg`の配布物は、`.goreleaser.yaml`・`.github/workflows/release.yml`・`tools/licenses`で作る（テンプレートと同じ形。見本は crube（https://github.com/amisonnet8/crube ））。

- **cgo を使わない**（`CGO_ENABLED=0`。下の「純粋なGoにする」）
- 配布物は linux・darwin・windows の amd64・arm64。linux・darwin は`.tar.gz`、windows は`.zip`。名前は`<名前>_<版>_<OS>_<アーキテクチャ>`（版は`v`なし）。`checksums.txt`を添える。アーカイブには、実行ファイル・`LICENSE`・`README.md`・`README_ja.md`・`licenses/`（依存のライセンス文）を入れる
- **版**：goreleaser が`-X main.version=<タグ>`（`v0.1.0`の形）を埋める。`main`パッケージに`var version string`を置き、空のとき（`go install`で入れたもの）は`runtime/debug.ReadBuildInfo`のモジュールの版を出すと、どちらでも同じ`v0.1.0`になる（crube の`cli.BuildVersion`。下の「バージョンの取り方」）
- **依存のライセンス文**を配布物に同梱する。`tools/licenses`が、3つの OS で実行ファイルに組み込まれるモジュールを`go list -deps`で求め、そのライセンスのファイルと Go の標準ライブラリの`LICENSE`を`bin/licenses/<モジュールのパス>/`に写す（goreleaser の`before`の手順）。ライセンスのファイルが無いモジュールがあれば失敗し、リリースが止まる
  - 依存を足す・外すと、`tools/licenses`の`TestRun`の一覧（`./cmd/mtqg`のもの）が落ちる。新しいモジュールのライセンスを確かめてから一覧を直す（Trivy の`qsoku trivy`も通す）

## リリースの流れ

ビルド済みのバイナリなどを GitHub Releases で配るときは、次の流れにする。どのプロジェクトも同じ。

1. **人間が**タグ（`v0.1.0`の形）を打って push する。人間がするのはこれだけ
2. タグで動く workflow（`.github/workflows/release.yml`）が、検査を通してから配布物を作り、Release に添付して**公開する**（本文は goreleaser が前のタグからのコミットで作る）

- **Claude Codeは、タグも Release も作らない。** `gh release create`は`.claude/settings.json`で拒否（deny）されている。Release を作るのは workflow の中だけ
- `release.yml`は、まず`ci.yml`をそのまま呼んで（`workflow_call`）普段の CI の検査をすべて通し、そのあと goreleaser（`.goreleaser.yaml`。`goreleaser-action`の`version: "~> v2"`で、v2 の最新）で配布物を作る。書き込みの権限（`contents: write`）は、Release を作るジョブだけに与える
- goreleaser はいったん**下書き**に添付し、すべて添付し終えてから、最後の手順（`gh release edit --draft=false`）で公開する。途中で落ちると下書きだけが残り、公開された Release に配布物が欠けることはない。同じタグでもう一度動かすと、その下書きを置き換える（`replace_existing_draft`）。落ちたときは、原因を直したコミットに新しいタグを打つか、同じコミットのままなら workflow を再実行する
- **公開済みの Release があるタグでは、何も変えずに失敗する**（最初の手順）。出し直すときは、新しいタグを打つ
- **タグを打つ前に、手元で`qsoku snapshot`を動かして中身を見る**。タグなしで`dist/`に同じ配布物を作る（版は直前のタグのパッチを1つ上げた`-snapshot`、バイナリの版は直前のタグ。このリポジトリでは`1.1.2-snapshot`と`v1.1.1`で、`dist/mtqg_linux_amd64_v1/mtqg version`で確かめられる。タグが1つも無ければ`0.0.1-snapshot`と`v0.0.0`）。goreleaser は`.devcontainer/postCreate.sh`が入れる。コンテナを作り直す前に要るときは、`go install github.com/goreleaser/goreleaser/v2@latest`（Bashサンドボックスの中では、`proxy.golang.org`と`storage.googleapis.com`への接続が要る）。`.goreleaser.yaml`を変えたら`qsoku snapshot`で確かめる
- ビルド済みのバイナリは、リポジトリにコミットしない（`.claude/rules/directory-structure.md`「配布物」）

## 純粋なGoにする（cgoを使わない）

- devcontainerは`CGO_ENABLED=0`。各OS向けのバイナリを簡単に作れるようにするため
- ロック（`flock`・`LockFileEx`）は`golang.org/x/sys`経由で、cgoなしで呼ぶ
- cgoを要する依存を足さない（`-race`だけは例外。testing.md）

## バージョンの取り方

- `mtqg version`のバージョンは **`runtime/debug.ReadBuildInfo`で取る**
- ビルド時の埋め込み（`-ldflags "-X ..."`）は`go install`では効かないため、それだけに頼らない。`go install ...@v0.2.0`で入れたときもバージョンが出るようにする
- 優先順位の目安：`-ldflags`で埋め込まれた値（goreleaser の`main.version`。タグの形`v0.1.0`） → ビルド情報のモジュールのバージョン → `dev`
- `go build`でビルドした場合、ビルド情報にはVCSから作られた疑似バージョンが入ることがある。実際の値を確かめてから表示の仕方を決めること
- `mtqg version`は、mtqg本体のバージョンと、リポジトリの`.mtqg/version`（形式のバージョン）を**区別して**両方表示する

## リポジトリにバイナリをコミットしない

- ビルドした`mtqg`、`dist/`は`.gitignore`に入れる
