# fnazeの使い方

このリポジトリは、[fnaze](https://github.com/amisonnet8/fnaze)で、**コミットごとに、変えたファイルそれぞれについて「何を、なぜ変えたか」を1行で書いた説明を作り、そのコミットに入れる**（`.fnaze/changes/`）。人間は、この一覧を読めば、ファイルごとに何を、なぜ変えたかが分かる。言語は**英語（`en`）と日本語（`ja`）の両方**。英語のファイルは接尾辞なし（`<日時>_<UUID>.txt`）、日本語は`_ja.txt`。仕様は fnaze の文書（https://github.com/amisonnet8/fnaze/blob/main/docs/reference/commands.md と`file-format.md`）。

## 入口

| 入口 | 使い方 |
|---|---|
| MCP（`.mcp.json`の`fnaze`） | 読む`show`・`status`・`log`と、書く`new_group`・`set_title`・`set_description`・`remove_description`。**サーバーは1つ**で、言語は道具の引数（`lang`、`new_group`は`langs`）で渡す |
| CLI（`fnaze <引数>`） | 下の手順のコマンド。日本語は`--lang ja` |

- `fnaze`は`.devcontainer/postCreate.sh`が`@latest`で入れる（ほかの道具と同じ）。`fnaze --version`で確かめる。`.fnaze/`も、無ければ`postCreate.sh`が`fnaze init`で作る
- MCPのサーバーは`fnaze mcp --allow-write`で動く（書き込むのは`.fnaze/`の中だけ）。`.claude/settings.json`の`enabledMcpjsonServers`に`fnaze`が無いと、MCPは現れない（人間が管理する）。そのときはCLIで進める
- **`FNAZE_LANG`は設定しない**（`.devcontainer/`にも`.claude/settings.json`にも置かない）。書くときは、いつも`--lang en`・`--lang ja`を明示する（手元の環境に`FNAZE_LANG`があっても、同じ結果になる）
- 1つの言語だけで足りるプロジェクトなら、下の手順の`ja`の分を消し、この文書と`.crube/`の`.fnaze/changes/`の説明を直す

## コミットのたびに

**手順3〜5の`title`・`set`・`rm`は、いつも`--`を付けて打つ。** オプション（`--lang`・`--detail`）を`--`の前に、題・パス・desc を`--`の後ろに置く。`-`で始まる題・desc がオプションとして読まれるのを、気づかないうちに踏まないため（下の「落とし穴」）

1. 入れるファイルを`git add`し、`git diff --cached --name-only`で変わったファイルを確かめる
2. `fnaze new en ja`で組を作る。言語ごとに、1つ前のその言語のファイルのパスの行が、骨組みとして写ってくる
3. `fnaze title --lang en -- <題>`と`fnaze title --lang ja -- <題>`で題を書く。題は、このまとまりで何をしたかの1行。英語と日本語で同じことを書く
4. 変わったファイル・ディレクトリごとに、`fnaze set --lang en [--detail …] -- <path> <desc>`と`fnaze set --lang ja [--detail …] -- <path> <desc>`で書く
   - desc は「何を・なぜ変えたか」の1行。経緯や補足は`--detail`
   - ツールを通して変わったファイル（mtqg の`.mtqg/journal.jsonl`、crube の`.crube/structure*.txt`、`go get`の`go.sum`など）も、そのパスをエントリにする。desc には、どのツールのどの操作で何が変わったかを、分かる範囲で書く（mtqg ならレコードの ID）。特定できなければ、親のディレクトリの desc にまとめる
   - 説明のファイル自身（`.fnaze/changes/…`）は、エントリに書かない
   - ルート（`.`）はエントリにできない。テンプレートを写した最初のコミットのように全体が変わったときは、ルート直下のファイル・ディレクトリごとに書く
   - **2つの言語で、同じパスを同じ順に`set`する**。並びが違うと`status`が言語ごとのずれ（`lang-mismatch`）を報告する
5. このコミットで変わっていない、写ってきたパスは`fnaze rm --lang en -- <path>`と`fnaze rm --lang ja -- <path>`で消す
6. `fnaze status --require`を通し、説明のファイルを`git add .fnaze`してからコミットする
7. 置いた説明のファイルのパス（2つ）を、人間に伝える

- **説明のファイルを`Edit`・`Write`で直さない。** 題もエントリも、fnaze のコマンドで書く
- **pre-commit が止める**：`.githooks/pre-commit`（fnaze の`docs/examples/pre-commit`と、先頭のコメントのほかは同じ中身。`core.hooksPath`は`.devcontainer/postCreate.sh`が設定する）が、説明のファイルを足さないコミットと、いちばん新しい組を`fnaze status --require`が報告するコミットを止める。止まったら、説明を書いて直す。`--no-verify`で飛ばさない
- `.fnaze/changes/`の説明のファイルは、`.crube/ignore`で crube の対象から外してある（1つずつに説明を付けない）

## 落とし穴

- **`-`で始まる desc・題は、オプションとして読まれる**（`unknown option …`）。オプションは引数の前にも後ろにも置けるため。`fnaze set --lang ja -- <path> <desc>`のように、`--`の後ろに置く。「`-`で始まるときだけ付ける」では気づかずに抜けるので、「コミットのたびに」の初めのとおり、いつも付ける
- **`--`の後ろは、すべて引数として読まれる。** `--detail`などのオプションを`--`の後ろに置くと、オプションにならない。オプションは`--`の前に置く
- **`status --require`は、書き忘れたファイルを見つけない。** fnaze は git を読まないので、何が変わったかを知らない。手順1の一覧と、`fnaze show --lang en`・`--lang ja`を見比べる
- **`rm`で子が無くなった骨組みの行も消えるので、あとから`set`した同じディレクトリは末尾に足される**（エントリの並びが前のファイルと変わる。仕様どおり）。並びを保ちたければ、`rm`の前に`set`する。2つの言語で、`set`・`rm`を同じ順に打つ
- **`fnaze new`を言語なしで打つと、いちばん新しい組と同じ言語で作る。** 前の組が`ja`だけなら`ja`だけになる。`en ja`を並べる
- **`.fnaze/`は入れ子にできない。** 上のディレクトリに`.fnaze/`があると、fnaze はそちらを使う（`init`も「もうある」で止まる）。別のリポジトリの中にこのリポジトリを置いて作業しない
