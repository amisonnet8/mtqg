# CLAUDE.md

このファイルは mtqg プロジェクトで作業する際、Claude Code が常に踏まえるべき前提を示す。

## mtqgとは

**(m)emo & rules・(t)odo・(q)a & bugs・(g)lossary** を、コードと同じgitリポジトリの中に追記だけで残し、人間とAIエージェントが共有するためのCLIツール。Go製。

作業の中で生まれる「気づき・やること・質問と答え・不具合とそのやり取り・用語の合意・この先も効き続ける決まり事」は、コードにもコミットにも残らず、チャットの履歴やメモアプリに散って失われる。mtqgはそれを`.mtqg/journal.jsonl`（1行1イベントのJSON Lines）に追記し、gitに乗せて共有する。

- **作る側から見ると**：バージョン管理はgitがやる。mtqgは、gitに残らない過程を追記していくだけ
- **使う側から見ると**：いつもの体験はほぼ同じまま、記録がもっと細かく、リポジトリに残る

判断に迷ったら、内部の機能は前者に、使う側の機能は後者に照らす（設計§2.6）。**書き込みの速さが最優先**（一番残したいのは、作業に取り掛かるときの数秒で消える気づき）。

## 参照すべきファイル

作業を始める前に、以下を確認すること。

- **`PLAN.md`** — 実装計画・進捗管理（開発の段階、段階1のステップ、現在地、未確認事項、保留事項）。**セッションをまたぐたびに「現在地」を確認・更新すること。**
- **`docs/reference/schema.md`** — データ形式の仕様（英語）。`.mtqg/SCHEMA.md`としてバイナリに埋め込まれ、mtqgを使う各プロジェクトに書き出される。日本語版は`schema_ja.md`
- **`docs/reference/cli.md`** — コマンドの仕様（英語）。日本語版は`cli_ja.md`
  - `docs/reference/`は**実装しながら育てる文書**であり、**実装と仕様がずれたらここを更新すること**
- **`docs/design/`** — 今の設計と、その判断の理由（日本語）。「なぜこうなっているか」はここにある。`docs/design/README.md`に章とファイルの対応がある。本文や`.claude/rules/`中の「設計§7.4」はここの節番号、「CLI設計§4.5」は`docs/design/cli.md`の節番号。**仕様と食い違う場合は`docs/reference/`が正**
- **`.claude/rules/bcuz-usage.md`** — bcuzの使い方。**Bashで実行するコマンドは、必ず`bcuz "理由" コマンド…`の形で打つ**（フックが、そうでないものを断る）。理由は「なぜ」を書く。**コマンドを打つ前に読む**
- **`.claude/rules/srwr-usage.md`** — srwrの使い方（**厳格モード**）。**`Edit`・`Write`は禁止されており、ファイルの変更は`look`／`edit`／`new`（理由`why`つき）で行う。作業のまとまりごとに`session`で題つきのテープを始め、コミットの前にテープを閉じる。ファイルを変える前に読む**
- **`.claude/rules/qsoku-usage.md`** — qsokuの使い方（`qsokufile`の名前の一覧、呼び方、書式、`//`、シェル連携）。**ビルド・テスト・lintは`go`などを直接打たず、ここの名前で実行する**
- **`.claude/rules/crube-usage.md`** — crubeの使い方（MCPの`crube`・`crube-ja`とCLI、場所の探し方、英語と日本語の説明の保ち方）。**場所を探す前、ファイルを足す・動かす・消す前に読む**
- **`.claude/rules/directory-structure.md`** — ディレクトリ構成（コアの2層とCLIの層、配置の判断基準、`schema.go`）。**新しいファイルを追加するときに読む**
- **`.claude/rules/journal-format.md`** — データ形式の扱い（追記だけの原則、書き込みの手順とロック、書き出し規則、形式のバージョン）。**データに触れる作業では必ず読む**
- **`.claude/rules/cli-output.md`** — CLI出力（英語、文言をCLIの層にまとめる、`--json`、正直に伝えること、`context`の書き方）
- **`.claude/rules/testing.md`** — テスト方針（開発環境の前提、動作確認の粒度、期待値の決め方、壊して確かめる（Skill`mutation-check`）、mtqg固有の検証項目、e2e、Trivy、ShellCheck、`-race`、Bashサンドボックス・CIの落とし穴）
- **`.claude/rules/datetime.md`** — 日時の扱い（データは UTC で持ち、表示だけ利用者のタイムゾーンに直す。瞬間で比べる、時計を差し替える）。**日時を保存・表示・比較するコードを書くときに読む**
- **`.claude/rules/working-with-human.md`** — 人間とのやり取り（頼み方の3種類、小さなつまずきの頼み方、確認の自動化、見え方を変えるときの了承）。**人間に何かを頼む前に読む**
- **`.claude/rules/naming.md`** — 命名規則（製品名、日本語と英語の用語の対応、コマンド、フィールド名）
- **`.claude/skills/`** — 繰り返す手順のSkill。**`commit-with-tape`**（テープを閉じてコミットする手順）と**`mutation-check`**（実装を1か所ずつ壊して、テストが効いているかを確かめる。`testing.md`「壊して確かめる」）。`mutation-check`は、仕様の約束を足したり変えたりした区切りで使う
- **`.claude/rules/git-integration.md`** — 製品としてのmtqgがgitに対してしてよいこと・いけないこと
- **`.claude/rules/distribution.md`** — 配布方法（`go install`、goreleaser、cgoなし、バージョンの取り方、リリースの流れ）
- **`.claude/rules/documentation.md`** — ドキュメントの書き方（例の値は道具で求める、`docs/design/`に書く経緯の線引き）。このファイルの「ドキュメントの書き方」「ドキュメントの言語」と合わせて読む
- **`.claude/rules/mtqg-usage.md`** — 段階4から、mtqg自身の開発過程を`.mtqg/`に記録する際の使い方（記録者、種類の使い分け、このPJと関係のないことは記録しない、こまめに記録する、使うバイナリ）

これらのルールファイルは、実装中に得た細かい気づき・教訓を Claude Code 自身が育てていくものである。新しく気づいたルール・踏んだ落とし穴があれば、該当するファイルに追記すること。どのファイルにも当てはまらない新しい種類の気づきであれば、新しいルールファイルを作ってよい（作ったらこの一覧にも足すこと）。

`testing.md`の一部には「**出所: 別プロジェクト**」と注記された節がある。これらは別プロジェクト（SanDBox・ExecDB）の開発中に実際に踏んだ事象であり、mtqgで発生したものではない。mtqgでも起こりうるため記録として引き継いでいる。**mtqgで新たに踏んだ事象を追記する際は、この注記を付けないこと**（出所が混ざると、どちらの記録か分からなくなる）。

## 開発の進め方

- 現在の段階とステップは`PLAN.md`の「現在地」を確認すること。進んだら、そちらを更新すること
- **Bashのコマンドは、すべて`bcuz "理由" コマンド…`で実行する**（`bcuz "lintの確認" qsoku lint`）。理由は「何をするか」でなく「なぜするか」。`bcuz`で始まらない・`&&`や`;`でつなぐコマンドは、`.claude/settings.json`の`PreToolUse`フック（`.claude/hooks/require-bcuz.sh`）が実行前に断る。**`git commit`だけは`bcuz`を通さない**（通すとコミットの後に記録が増えるため。理由はコミットメッセージに書く）。実行の記録は`.bcuz/history.jsonl`に残り（コミットする）、`bcuz -H`で見られる。**トークンなどの秘密を、コマンドの引数・`env`の指定・理由に書かない**（記録に残り、コミットされる）（`.claude/rules/bcuz-usage.md`）
- **ファイルの変更は、srwrの道具（`look`で見て`edit`、新規は`new`）で行い、すべてに理由（`why`）を添える。** 組み込みの`Edit`・`Write`は`.claude/settings.json`で禁止されている（厳格モード）。Bashでの書き換え（`sed -i`・リダイレクト）もしない
- **作業のまとまり（todo・bug・人間の依頼・作業の種類の変わり目）ごとに、最初に`session`で題つきのテープを始める。** 1つの作業＝1本のテープ、コミットも同じ区切り
- **コミットの前に、テープを閉じる**（`srwr tapes check`のあと`srwr tapes new`）。コミットまでの手順はSkill`commit-with-tape`。書き込み中の生のテープは`.gitignore`で対象外で、閉じて圧縮した`.srwr/tapes/*.tape.jsonl.gz`だけがgitに入る（`.claude/rules/srwr-usage.md`）
- **このリポジトリの中の場所は、まずcrubeで引く**（MCPの`find`・`tree`・`show`、または`crube find …`）。ファイル・ディレクトリを足す・動かす・消すときは、同じ変更の中で英語と日本語の説明を直し、コミットの前に両方の言語で`crube status --require all`を通す（`.claude/rules/crube-usage.md`）
- 実装後は`.claude/rules/testing.md`に従い、ビルド確認に加えて実際の動作確認（`bcuz "…" qsoku check`、必要に応じて`qsoku test`・`qsoku race`・`qsoku trivy`・`qsoku shellcheck`）を行うこと。判断に迷ったら実行する側に倒すこと
- **設計判断を変える場合は、まず`docs/reference/`を更新してから着手すること。** 実装が先行して仕様が後追いになると、どちらが正なのか分からなくなる。判断の理由が大きく変わる場合は、`docs/design/`の該当節を今の設計に合わせて直す。経緯をどこまで`docs/design/`に書き、どこからmtqgに記録するかは`.claude/rules/documentation.md`「`docs/design/`に書く経緯」に従う
- **依存ライブラリは、Goの標準ライブラリと`golang.org/x/`だけを使う。** それ以外は「どうしても必要な場合」の例外とし、①標準・`x/`に無い、または②自前で書くと量が多く、正しさや安全性の確認が難しい、のどちらかを説明して確認を取り、理由を`PLAN.md`に記録すること。追加したらTrivyでライセンス（MITと衝突しないか）と脆弱性を確認すること（`.claude/rules/testing.md`）。基準が線引きなのは、信頼の根をGoチームに絞り、そのたびに議論しなくて済むようにするため
- **看板としてのREADMEは最後に作る。** リポジトリは最初からpublicなので、未完成の間は「未完成で、まだ使わないでほしい」だけを書いた注意書きのREADME（`README.md`・`README_ja.md`）を置く。売り文句・機能の説明・使い方は足さない（`PLAN.md`「READMEとGitHubの看板」「公開の2段階」）

## 守ること（設計の芯）

実装の細部より優先する。これを崩す変更は、実装する前に必ず相談すること。

- **追記するだけ。** 既存の行を書き換えない。行を消す・移すのは`undo`と`archive`だけ（`.claude/rules/journal-format.md`）
- **gitの運用には何も要求しない。** mtqgはgitを読むだけで、コミット・push・設定の変更をしない（`.claude/rules/git-integration.md`）
- **重複や食い違いはエラーにせず、事実として見せる。** どちらが正しいかをmtqgが決めない
- **データの解釈はコアの1か所に集める。** 入口（CLI、将来のMCP・フック）ごとに状態の組み立てを重複させない（`.claude/rules/directory-structure.md`）
- **役割外のものを足さない。** 課題管理の機能（優先度、期限、担当者、カンバン、多段階のステータス、bugの重要度・再現手順の欄等）、版の前後関係を厳密に決める仕組み、記録者の名寄せ、成果物の位置の追跡などは採らない。判断の問いは「それは課題を管理するための機能か、過程を残すための情報か」（設計§2.7）

## ドキュメントの書き方

**提供しない機能については、事実だけを書き、代替手段を書かない。** 仕様書は「何を保証するか」を書く場所であり、使い方の指南を書くと、書いていない解法が選ばれにくくなって実装者の視野を狭める。

**例外:** 守らないと壊れるもの（`journal.jsonl`の衝突は両側の行を残して解決する、など）は、使い方の助言ではなく**契約**なので仕様に書く。両者を混同しないこと。

「あると期待されやすいのに無い機能」（設定ファイル、相対的な日付指定など）は、無いことを明記する価値がある。「無い機能を際限なく列挙する」のとは違うので、そこは判断すること。

**ドキュメントに載せるコマンドの実行例と出力例は、実際に動かして確かめたものにする**（`.claude/rules/testing.md`「e2eとdocsの例の確認」）。

## ドキュメントの言語

- **`CLAUDE.md`・`PLAN.md`・`.claude/rules/`は日本語のみ。** 英語版は作らない
- **`docs/design/`は日本語のみ。**
- **`docs/reference/`は英語版（`*.md`）と日本語版（`*_ja.md`）の2本立て。**
  - 英語版が正。`schema.md`は各プロジェクトの`.mtqg/SCHEMA.md`になり、初見の人やAIが読むため
  - **更新するときは、同じ変更の中で両方を直す。** 片方だけ直して食い違わせないこと
  - 見出しのすぐ下に相互リンクを置く（現在の言語を太字に）。英語版は`*[日本語](cli_ja.md) | **English***`、日本語版は`*[English](cli.md) | **日本語***`の形。ただし`schema.md`は`.mtqg/SCHEMA.md`として各プロジェクトに書き出され、そこに日本語版は無いのでリンクを置かない（本文中で日本語版の場所を文章で示す）
  - **英語版の例は、記録の中身も英語で書く**（`mtqg t add Add tests for comment handling`など）。日本語版の例は、記録の中身を日本語で書く。どちらもmtqgの出力の枠（見出し、メッセージ）は英語のまま（mtqgの出力は常に英語であるため）。1つの文書の中で、誰向けの資料か分からなくなる混ざり方をさせない
- **`docs/examples/`も英語版と日本語版の2本立て**（実装完了後に作った。入門から実例まで1つの文書にまとめている）。**完成として公開する時点**で、READMEを含む全文書が両方そろっている状態にする（`PLAN.md`「公開前にやること」）。未完成の間の注意書きのREADMEも、最初から英日の2本にする
- **コード・コメント・コミットメッセージ・CLI出力は英語。** 対話は日本語で行うが、これらは英語で書くこと（`.claude/rules/cli-output.md`）

## gitの扱い（Claude Codeの作業として）

- **pushは人間が行う。** Claude Codeはコミットまでで止めること。`git push`は`.claude/settings.json`で拒否（deny）されている
- 作業の区切りでこまめにコミットしてよい（消してしまったときに戻せるように）
- `git reset --hard`・`git clean`は、コミットしていない作業を消すので、使う前に必ず理由を説明すること
- **コミットの前に`git config user.email`／`user.name`を確かめる。** 正しい値は`65074207+amisonnet8@users.noreply.github.com`／`amisonnet8`（GitHubのプライバシー用noreplyアドレス）。違う値なら、`.git/config`の`[user]`（ローカルの上書き）を疑う。ローカルに設定が無いと、コンテナのglobalの設定（noreplyでないことがある）が使われる

## 権限・自動化について

`.claude/settings.json`（人間が管理する）により、次の設定になっている（2026-09-30、Bashサンドボックス導入に伴い改定）。

- 拒否（deny）：`git push`、`gh pr merge`、`gh release create`、`Edit`・`Write`・`MultiEdit`・`NotebookEdit`（srwrの厳格モード、下）
- **`bcuz`を通さないBashコマンドの拒否**：`PreToolUse`フック（`.claude/hooks/require-bcuz.sh`）が、`bcuz`で始まらないコマンド（`git commit`を除く。逆に`bcuz`を通した`git commit`を断る）と、後ろに別のコマンドをつなぐもの（`&&`・`||`・`;`・`&`・改行・単一引用符の外のコマンド置換）を、実行前に断る。直し方は断りのメッセージと`.claude/rules/bcuz-usage.md`に書いてある
- 実行前に確認（ask）：`git reset --hard`、`git clean`、`sudo`、`gh pr create`
- **Bashコマンドは既定でOSレベルのサンドボックス（Linux bubblewrap）内で実行される。** ファイルシステムの書き込み先（`~/.cache/go-build`・`/go`・`~/.cache/trivy`・`~/.cache/golangci-lint`。このコンテナのGOPATHは`/go`（`go env GOPATH`））とネットワーク接続先（`proxy.golang.org`・`sum.golang.org`・`mirror.gcr.io`・`github.com`・`api.github.com`）は許可リストで絞られている。この範囲内なら`curl`・`wget`・`go get`などは確認なしに実行できる（askの対象から外れた理由はこれ）。リストに無い宛先・書き込み先は`sandbox_violations`として拒否される。そのときは、`.claude/rules/working-with-human.md` 2章の形（原因と、`.claude/settings.json` のどこに何を足すか）で**短く頼む**（許可を広げるかは人間が判断する）。落とし穴は`.claude/rules/testing.md`「Bashサンドボックスの落とし穴」参照

確認を求められた場合、無理に実行しようとしない。`.claude/settings.json`を変更したいときは、変更案を差分で示すにとどめること（人間が貼るだけで済む形にする）。

**srwrの厳格モード**：`permissions.deny`に`Edit`・`Write`・`MultiEdit`・`NotebookEdit`があり、`.mcp.json`にsrwrのMCPサーバー、`PostToolUse`に`srwr hook`（読み取りと外部変更の記録）がある。テープ（`.srwr/tapes/`）はgitで共有する（鍵`.srwr/key`は入れない）。許可を広げる（緩いモードにする）のは人間の判断。詳細は`.claude/rules/srwr-usage.md`。

ビルドの自動フック（`PostToolUse`）が設定されている。srwrの`edit`・`new`で`.go`・`go.mod`・`go.sum`を変えると`.claude/hooks/build.sh`が`qsoku build`を実行し、失敗すると理由がClaude Codeに返る。複数ファイルにまたがる編集の途中は、まだ書いていないファイルを参照して一時的に失敗することがある（`PLAN.md`「保留事項」参照）。

**crubeの配線**：`.mcp.json`に、crubeのMCPサーバーが2つある（`crube`は英語、`crube-ja`は日本語。`crube mcp --allow-write`）。srwrの`new`のあとには`.claude/hooks/crube-untracked.sh`が、説明の無いパスを言語ごとに知らせる（止めはしない）。詳細は`.claude/rules/crube-usage.md`。

## mtqg自身の記録について

段階3まで、mtqg自身の経緯は`PLAN.md`と`docs/design/`に残した（mtqgはまだ使わなかった）。**段階4から`.mtqg/`に記録している**（`PLAN.md`「開発の段階」、`.claude/rules/mtqg-usage.md`）。`PLAN.md`は引き続き置き、区切りの記録・経緯の要約を担う。`docs/design/`には今の設計を書く（経緯をどこまで書くかは`.claude/rules/documentation.md`）。日々の細かい気づき・質問・不具合は`.mtqg/`へ。`.mtqg/`には決まり事（rule）を9件登録済み——`mtqg context`で確認できる。AIエージェントからは、フック（`.claude/settings.json`の`mtqg hook`）とMCPサーバー（`.mcp.json`の`mtqg mcp`）の両方で使う。どちらも呼ぶのは、作業中のソースから入れた`mtqg`（`.claude/rules/mtqg-usage.md`）。

## ルール・スキルの提案

作業を進める中で、以下に気づいたら、都度こちらから提案すること（提案するだけで、勝手に作成・適用はしない。判断はこちらが行う）。

- **新しいルールにした方がよさそうな知見**: 同じ種類の判断や落とし穴に複数回遭遇した、既存の`.claude/rules/`のどれにも当てはまらない、といった場合。どのファイルに追記すべきか、あるいは新規ファイルが必要かも合わせて提案する
- **Skill化した方が効率的そうな作業**: 「同じ手順を3回目繰り返している」など、パターン化できそうな一連の作業に気づいた場合。まだ実装が薄い段階では無理に提案する必要はない。パターンが実際に繰り返されてから提案すること

提案は気づいたタイミングで随時行ってよく、まとめて報告するために貯めておく必要はない。

This repository records its development with mtqg: run `mtqg context` at the start of a session, and see `.mtqg/SCHEMA.md` for the data format.
