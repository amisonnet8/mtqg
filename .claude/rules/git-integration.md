# gitとの関わり（製品としてのmtqg）

mtqgはgitの上に乗るツールだが、**gitの運用には何も要求しない**（設計§1.5、§4.3、§10.2）。この線を越える実装をしないこと。

## mtqgがgitに対してしてよいこと

- **読むだけ。** 必要な情報は本物の`git`コマンドに聞く
  - `git rev-parse --show-toplevel`、`.git`の有無（リポジトリの境界）
  - `git -C <ルート> config user.name`（記録者の名前。そのリポジトリの設定が効く）
  - `git branch --show-current`（今のブランチ名。`context`の最初の行。detached HEADでは空）
  - `git -C <ルート> show HEAD:.mtqg/journal.jsonl`（未コミットの記録の件数を出すため。`HEAD`の`journal.jsonl`の行と、今の行を比べる。`HEAD`やそのパスが無ければ、全行が未コミット）
  - `git -C <ルート> rev-parse HEAD`（`GitHead`。エージェントのフック（段階4a）が、セッション開始時と終了時で比べるため。コミットが無ければ空）
  - `git -C <ルート> status --porcelain -z`（`GitStatusDigest`。`.mtqg/`を含む行を除いてハッシュ化したもの。同じくフックが「作業ツリーが変わったか」を見るためだけに使い、`--json`にも表示にも中身は出さない）
- gitのライブラリ（go-gitなど）でgitの動きを再実装しない。`exec`で`git`を呼ぶ。gitの設定・フック・worktree・サブモジュールの扱いが本物と食い違うため
- gitの私的な領域（`.git/`の中）には何も置かない。mtqgの状態はすべて`.mtqg/`の中に収まる（このマシンだけの一時的なものは`.mtqg/.local/`・journal-format.md）

## mtqgがしてはいけないこと

- コミット、push、pull、fetch、ブランチ操作
- gitの設定（`git config`への書き込み）、フックの設置
- リポジトリ全体の`.gitattributes`・`.gitignore`の変更（`.mtqg/`の中の`.gitattributes`と`.gitignore`だけは`init`が置く）
- ブランチごとの専用ref、git notesなど、gitの仕組みに踏み込む保存方式（設計§4.2、§13）
- 版の前後関係を厳密に決める仕組み（論理時計など・設計§2.7）

## `mtqg init --agent`が触るのは`.mtqg/`の外だが、gitの設定ではない

`mtqg init --agent claude-code`（段階4a）は、明示的に`--agent`を付けたときだけ、`.claude/settings.json`・`CLAUDE.md`という**リポジトリの普通の追跡ファイル**を書き換える。これは「gitの設定」（`git config`、フック、`.git/`の中）ではなく、エージェント自身の設定ファイルなので、上の「してはいけないこと」には当たらない。書き込みは`os.OpenRoot`でリポジトリのルートを開いて行い（`.mtqg/`の外に出さない`os.Root`の使い方は`journal-format.md`と同じ考え方）、既存の内容を壊さない・コミットはしない（案内するだけ）。

## `.mtqg/`とgitの関係

- `.mtqg/`は普通のファイルとしてコミットされる。いつ・どうまとめてコミットするかは利用者の自由
- `init`は「`.mtqg/`をコミットしてください」と案内するだけ
- 記録はブランチに乗る（設計§4.2）。ブランチを切り替えれば、そのブランチの記録が見える
