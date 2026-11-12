# crubeの使い方

このリポジトリは、[crube](https://github.com/amisonnet8/crube)で、自分のファイルとディレクトリに説明を付けている（`.crube/`）。**Claude Codeは、場所を探すときも、ファイルを足す・動かす・消すときも、crubeを使う。** 説明は英語（`.crube/structure.txt`）と日本語（`.crube/structure_ja.txt`）の2つで持つ。

## 入口

| 入口 | 使い方 |
|---|---|
| MCP（`.mcp.json`の`crube`） | `find`・`tree`・`ls`・`show`・`status`と、書き込みの`set_description`・`move_description`・`remove_description`。**サーバーは1つ**で、言語は道具の引数`lang`で渡す（日本語は`lang: "ja"`で`.crube/structure_ja.txt`。省くと英語の`.crube/structure.txt`） |
| CLI（`crube <引数>`） | `crube find …`。日本語は`crube find … --lang ja` |

- `crube`は`.devcontainer/postCreate.sh`が`@latest`で入れる（ほかの道具と同じ）。`crube --version`で確かめる
- MCPのサーバーは`crube mcp --allow-write`で動く（書き込むのは`.crube/`の中だけ）。`.claude/settings.json`の`enabledMcpjsonServers`に`crube`が無いと、MCPは現れない（人間が管理する）。そのときはCLIで進める
- 1つの言語だけで足りるプロジェクトなら、使わない言語のファイルと、`.claude/hooks/crube-untracked.sh`の言語の一覧を消し、この文書を直す
- **言語は、MCP でも CLI でもいつも明示する**（`lang: "en"`・`"ja"`、`--lang en`・`--lang ja`）。省くと、サーバーや手元の環境の`CRUBE_LANG`しだいで、思っていない言語のファイルに書く。`--lang`・`lang`を付ければ`CRUBE_LANG`は使われない
- `lang`の引数と`--lang`は crube v0.2.0 から。`crube --version`が古ければ入れ直し、MCP は`/mcp`でつなぎ直す

## 場所を探すとき

- **まず crube で引く**：`find <キーワード>`（パス・desc・detail）、`tree <ディレクトリのエントリのパス>`、`show <パス>`。目的の場所の見当をつけてから、`Grep`・`Read`で中身を読む
- `Grep`は、コードの中身（関数名・文字列）を探すときに使う。「何がどこにあるか」は crube で引く
- 引けなかった・説明が分かりにくかったときは、その場で説明を直す（下の「説明を保つ」）

## 説明を保つ

- **ファイル・ディレクトリを足す・動かす・消すときは、同じ変更の中で説明を直す**：足したら`set`（`set_description`）、動かしたら`mv`（`move_description`）、消したら`rm`（`remove_description`）
- **英語と日本語の両方を直す**。言語ごとのファイルは独立で、片方だけ直すともう片方が古くなる
- desc は1行で「何がそこにあるか」。詳しいこと（いつ使うか、ほかとの関係）は detail に書く。長さの上限は crube の文書（https://github.com/amisonnet8/crube/blob/main/docs/reference/file-format.md ）
- `.crube/`の中は、`Edit`で直接直してもよい（タブのインデントを崩さない）
- コミットしないものを`.gitignore`に足したら、`.crube/ignore`にも足す（crube は`.gitignore`を読まない）
- テンプレートから作ったときの説明は、テンプレートのファイルのもの。プロジェクトの中身が決まったら（`CLAUDE.md`の「このプロジェクトについて」など）、ルートや`internal/`・`docs/`の説明も合わせて直す

## 気づくための仕組み

- **`Write`のあと**：フック`.claude/hooks/crube-untracked.sh`が、作ったファイルと、その上のディレクトリのうち説明の無いものを、言語ごとに知らせる（止めはしない）。知らされたら、続けて両方の言語で説明を付ける
- **コミットの前**：`crube status --require all --lang en`と`crube status --require all --lang ja`の両方を通す。Bashで作った・動かした・消したもの（`git rm`、ツールが書いたもの）は、フックが見ないので、ここで見つかる

## 落とし穴

- **`tree`・`set`・`rm`・`mv`に渡すのはエントリのパス**（CLI は現在地から、MCP はルートから読む）。ディレクトリの末尾の`/`は省ける（`tree .claude/hooks`）。ただし、`set`で新しいディレクトリのエントリを作るときは`/`を付ける（付けないと`is a directory (use …/)`で止まる）。`ls`・`show`に渡すのは実在するパス
- **`status`の出力のパスはルートから。** それを CLI の`set`・`mv`・`rm`に渡すときは、ルートで打つ（または`-C <ルート>`）。MCP の道具はルートから読むので、そのまま渡せる
- **CLI の`set`は、`-`で始まる desc をオプションとして読む**（`unknown option …`）。いつも`crube set --lang ja [--detail …] -- <path> <desc>`と`--`を付けて打つ。オプション（`--lang`・`--detail`）は`--`の前に置く（後ろに置くと引数として読まれ、`too many arguments`で止まる）。MCP の`set_description`は引数が分かれているので、この問題は無い
- **`.crube/`は入れ子にできない。** 上のディレクトリに`.crube/`があると、crube はそちらを使う（`init`も「もうある」で止まる）。別のリポジトリの中にこのリポジトリを置いて作業しない
