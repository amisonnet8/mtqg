# mtqgの実例

*[English](examples.md) | **日本語***

コマンドの完全な一覧は[コマンドリファレンス](../reference/cli_ja.md)、最初に触るなら[mtqgを歩いて回る](../tour/tour_ja.md)を参照。ここでは、もう少し大きな実例を2つ見る。

## 小さなプロジェクト

ここで見るのは、[コマンドリファレンス](../reference/cli_ja.md)のいたるところで例に使っている、架空のパーサー開発プロジェクトの`.mtqg/`（`e2e/testdata/examples/parser_ja/`）。個々のコマンドの説明はリファレンス側に譲り、ここでは「実際に何日かにわたって使われたリポジトリを、後から通しで読む」という視点でたどる。

まず`context`で全体を見る：

<!-- mtqg:example repo=parser_ja -->
```
$ mtqg context
# mtqg context — sample-parser (main)

This is the process record of this project. Read the following before you start working.
- Follow the rules listed under Rules
- Respect answered questions
- Do not decide open questions on your own; confirm them
- Use terms as defined in the glossary
- Record questions, decisions, findings, bugs, and todos with mtqg as they come up

## Attention
- Glossary term "ブロックコメント" has conflicting definitions (see mtqg glossary list)
- todo 6b0d549b6f "行コメント // の読み飛ばし" has concurrent changes (see mtqg review)
- 3 mtqg records are not committed

## Open todos (5)
- 6cad4a268d ブロックコメント /* */ の読み飛ばし (claude-code, 10:18)
- 6513270e26 文字列リテラル中の // を無視する (yamada, 10:52)
- 1e27a1c08a エラー位置を行と列で表示する (claude-code, 11:06)
- 1818e81189 READMEに対応している構文を書く (yamada, 11:30)
- 2e44158bae コメント処理のテストケースを追加 (claude-code, 11:32)

## Open questions (2)
- 1012f037b6 ブロックコメントの入れ子に対応する？ (awaiting confirmation, claude-code, 09:10)
    └ 初版では非対応。需要が出たら再検討 (yamada, human)
- 2217beaddb エラー位置は行と列の両方を出しますか？ (unanswered, claude-code, 11:05)

## Open bugs (1)
- 7f3a2b1c09 空の入力でパーサーが落ちる (awaiting confirmation, yamada, 10:41)
    └ macOSでも再現した (claude-code, ai)

## Recent records (newest first)
- 11:32  claude-code  todo      2e44158bae  コメント処理のテストケースを追加
- 11:30  yamada       todo      1818e81189  READMEに対応している構文を書く
- 11:24  claude-code  glossary  f28c105d1f  字句解析: ソースを読み、トークンの並びに変換する処理
- 11:06  claude-code  todo      1e27a1c08a  エラー位置を行と列で表示する
- 11:05  claude-code  question  2217beaddb  エラー位置は行と列の両方を出しますか？
- 10:52  yamada       todo      6513270e26  文字列リテラル中の // を無視する
- 10:46  claude-code  reply     9a8b7c6d5e  空のファイルでも落ちる (to 1012f037b6)
- 10:45  claude-code  reply     3d8e4a0b12  macOSでも再現した (to 7f3a2b1c09)
- 10:41  yamada       bug       7f3a2b1c09  空の入力でパーサーが落ちる
- 10:32  yamada       memo      81e74ef5e8  エラーメッセージは英語で統一する方針
- (12 more; see mtqg log)

## Glossary (4)
- 5b7e2c9a41 トークン: 字句解析で切り出す最小単位
- f29d0da995 ブロックコメント: /* と */ で囲むコメント
- 0cb1e29c65 ブロックコメント: 複数行にわたって書けるコメント
- f28c105d1f 字句解析: ソースを読み、トークンの並びに変換する処理

---
Read full entries with mtqg show <id>.
```

`Attention`が、並行した状態変更（同じtodoを2つのブランチが別々に閉じた）と、`glossary`の重複定義（`ブロックコメント`が2通り定義されている）を教えてくれる。どちらが正しいかはmtqgが決めないので、`mtqg review`で詳しく見て、人間が判断する（[review](../reference/cli_ja.md#review)に、この同じリポジトリでの実行例がある）。

区切りが付いた作業は`archive`でしまえる。ここでは2026-09-19に閉じた1件の質問と1件のバグを移してみる：

<!-- mtqg:example repo=parser_ja -->
```
$ mtqg archive 2026-09-19..2026-09-19
Range: 2026-09-19..2026-09-19
Archived: 1 question, 1 answer, 1 bug, 1 reply -> .mtqg/archive/2026-09-19..2026-09-19.jsonl
```

`journal.jsonl`から`archive/2026-09-19..2026-09-19.jsonl`へ、該当する行がそのまま移る。戻したくなったら、アーカイブのファイルを`journal.jsonl`の末尾に結合して消すだけでよい（`mtqg`のコマンドではなく、シェルでの操作）：

```
cat .mtqg/archive/2026-09-19..2026-09-19.jsonl >> .mtqg/journal.jsonl
rm .mtqg/archive/2026-09-19..2026-09-19.jsonl
```

## mtqg自身の開発

mtqgは、段階4（外部ツール連携）以降の自分自身の開発過程を、他ならぬmtqgで記録している（[`.claude/rules/mtqg-usage.md`](https://github.com/amisonnet8/mtqg/blob/main/.claude/rules/mtqg-usage.md)）。作られたフィクスチャではなく実際に積み重なった記録なので、`qa`・`bug`・`rule`・`--at`が実務でどう使われるかを見るには、これが一番の実例になる。

```
git clone https://github.com/amisonnet8/mtqg
cd mtqg
mtqg context
```

段階1〜3（コアとCLIができるまで）の経緯は、まだmtqgを自分自身の記録に使う前だったため、`PLAN.md`と`docs/design/`にある。
