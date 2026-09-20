# 配布方法

理由は設計§10.11。

## v0.xの間は`go install`だけ

```
go install github.com/amisonnet8/mtqg/cmd/mtqg@latest
```

モジュールパスは`github.com/amisonnet8/mtqg`。

- 配布のための仕組みを作らない。タグを打てば、それだけでバージョンを指定して入れられる
- **v0.1まではタグを打たない。** リポジトリは最初からpublicで、タグを打つとGoのモジュールプロキシ（`proxy.golang.org`）にバージョンが記録され、後から消せない。未完成の版が`@latest`で入ってしまう（`PLAN.md`「公開の2段階」）
- 公開の段階で、GoReleaser（`.goreleaser.yaml`）で各OS向けのバイナリを作り、GitHub Releasesに置く。**今は作らない**
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
