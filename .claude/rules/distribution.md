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
- **`.goreleaser.yaml`とGitHub Releasesのバイナリ配布は、v1.0.0正式公開の準備として2026-09-28に作った。** `v*`タグのpushで`.github/workflows/release.yml`が`goreleaser/goreleaser-action`を呼び、`cmd/mtqg`をlinux・darwin・windows × amd64・arm64向けにビルドしてGitHub Releasesに置く（アーカイブは`tar.gz`、Windowsだけ`zip`）。バージョンは`-ldflags "-X .../internal/cli.version={{.Version}}"`で埋め込む（下記「バージョンの取り方」の優先順位の1番目）。`CGO_ENABLED=0`はビルド設定に明示（devcontainerの既定と合わせる）。手元での確認は`goreleaser check`と`goreleaser release --snapshot --clean`（実際にタグをpushせず、ビルドとアーカイブだけ試す）。goreleaser自体は`go install github.com/goreleaser/goreleaser/v2@latest`で入れる開発ツールで、`go.mod`には入れない
- Homebrewなどのパッケージマネージャーは、要望が出てから考える

## 純粋なGoにする（cgoを使わない）

- devcontainerは`CGO_ENABLED=0`。各OS向けのバイナリを簡単に作れるようにするため
- ロック（`flock`・`LockFileEx`）は`golang.org/x/sys`経由で、cgoなしで呼ぶ
- cgoを要する依存を足さない（`-race`だけは例外。testing.md）

## バージョンの取り方

- `mtqg version`のバージョンは **`runtime/debug.ReadBuildInfo`で取る**
- ビルド時の埋め込み（`-ldflags "-X ..."`）は`go install`では効かないため、それだけに頼らない。`go install ...@v0.2.0`で入れたときもバージョンが出るようにする
- 優先順位の目安：`-ldflags`で埋め込まれた値 → ビルド情報のモジュールのバージョン → `dev`
- `go build`でビルドした場合、ビルド情報にはVCSから作られた疑似バージョンが入ることがある。実際の値を確かめてから表示の仕方を決めること
- `mtqg version`は、mtqg本体のバージョンと、リポジトリの`.mtqg/version`（形式のバージョン）を**区別して**両方表示する

## リポジトリにバイナリをコミットしない

- ビルドした`mtqg`、`dist/`は`.gitignore`に入れる
