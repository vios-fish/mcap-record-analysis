# mcap-record-order

[MCAP](https://mcap.dev/spec) ファイルに含まれる Record の **物理的な出現順序** をそのまま一覧表示する CLI ツールです。

`github.com/foxglove/mcap/go/mcap` の `Lexer` を直接利用し、再順序化や統合を行わずにファイル先頭から末尾までトークン単位で走査します。Chunk は標準ではそのまま1行で表示し、`--expand-chunks` 指定時のみ解凍して内部の Schema / Channel / Message などをインライン展開します。

## 特徴

- ファイル中の **すべての Record 種別** を順序付きで表示  
  Header / Schema / Channel / Message / Chunk / MessageIndex / ChunkIndex / Attachment / AttachmentIndex / Statistics / Metadata / MetadataIndex / SummaryOffset / DataEnd / Footer
- Record ごとに識別に役立つ属性を併記（topic 名、channel ID、log time、圧縮方式、オフセット 等）
- Chunk の中を見たい場合は `--expand-chunks` で解凍展開
- Attachment は `AttachmentCallback` 経由で正しい順序のまま出力

## 必要条件

- Go 1.22 以上（本リポジトリは Go 1.26 で動作確認）

## ビルド

```sh
go build -o mcap-record-order ./
```

## 使い方

```sh
# ファイルの物理レイアウト順に表示（Chunk は展開せず1行）
./mcap-record-order path/to/file.mcap

# Chunk を解凍し中身の Schema / Channel / Message も含めて表示
./mcap-record-order --expand-chunks path/to/file.mcap
```

引数は MCAP ファイルへのパス1つだけです。

### オプション

| オプション | 既定値 | 説明 |
| --- | --- | --- |
| `--expand-chunks` | `false` | Chunk を解凍してその中のレコードもインラインで出力する |

## 出力例

サンプルファイル（Chunk 圧縮 zstd、Message 5本、Attachment / Metadata 各1）に対する出力：

### 既定（Chunk は1行）

```
# mcap record order: sample.mcap (expandChunks=false)
INDEX  RECORD           DETAIL
0      Header           profile="demo" library="mcap go v1.7.3; mcap-record-order/gen"
1      Attachment       name="note.txt" mediaType="text/plain" logTime=1 createTime=0 dataSize=5
2      Metadata         name="info" entries=1
3      Chunk            messageStart=0 messageEnd=4000000 compression="zstd" uncompressedSize=294 compressedRecordsLen=164
4      MessageIndex     channelID=1 entries=3
5      MessageIndex     channelID=2 entries=2
6      DataEnd          dataSectionCRC=0x00000000
7      Schema           id=1 name="Foo" encoding="jsonschema" dataLen=2
8      Channel          id=1 schemaID=1 topic="/foo" encoding="json"
9      Channel          id=2 schemaID=1 topic="/bar" encoding="json"
10     Statistics       messages=5 schemas=1 channels=2 chunks=1 attachments=1 metadata=1
11     ChunkIndex       chunkStart=165 chunkLength=217 messageStart=0 messageEnd=4000000 compression="zstd"
12     AttachmentIndex  offset=66 length=68 name="note.txt"
13     MetadataIndex    offset=134 length=31 name="info"
14     SummaryOffset    groupOpcode=0x03 groupStart=505 groupLength=38
...
20     Footer           summaryStart=505 summaryOffsetStart=889 summaryCRC=0x00000000
# total records: 21
```

### `--expand-chunks`（Chunk 内も展開）

```
6      Message          channelID=1 seq=0 logTime=0 publishTime=0 dataLen=7
7      Message          channelID=2 seq=1 logTime=1000000 publishTime=1000000 dataLen=7
8      Message          channelID=1 seq=2 logTime=2000000 publishTime=2000000 dataLen=7
...
```

INDEX はファイル先頭からの通し番号で、Attachment（callback 経由）も同一の連番に組み込まれます。

## 出力カラム

| 列 | 内容 |
| --- | --- |
| `INDEX` | 出現順の通し番号（0 始まり） |
| `RECORD` | Record 種別名（MCAP 仕様の Op 名に準拠） |
| `DETAIL` | その Record を識別するのに有用な属性 |

`DETAIL` で表示する代表的な属性：

- **Header**: `profile`, `library`
- **Schema**: `id`, `name`, `encoding`, `dataLen`
- **Channel**: `id`, `schemaID`, `topic`, `encoding`
- **Message**: `channelID`, `seq`, `logTime`, `publishTime`, `dataLen`
- **Chunk**: `messageStart`, `messageEnd`, `compression`, `uncompressedSize`, `compressedRecordsLen`
- **MessageIndex / ChunkIndex / AttachmentIndex / MetadataIndex**: オフセットや対象 ID
- **Attachment**: `name`, `mediaType`, `logTime`, `createTime`, `dataSize`
- **Statistics**: 各種カウンタ
- **SummaryOffset**: `groupOpcode`, `groupStart`, `groupLength`
- **Footer**: `summaryStart`, `summaryOffsetStart`, `summaryCRC`

## 設計メモ

- **`Lexer` を採用した理由**: `Reader.Messages()` のような高水準 API は Message のみを返し、しかも内部で並び替えやインデックス参照を行うため「ファイル上の出現順を見る」用途には適しません。`Lexer` はストリーミングでバイト列をトークン化するので、物理レイアウトをそのまま観察できます。
- **Chunk の取り扱い**: 既定では `LexerOptions.EmitChunks = true` とし、Chunk 自体を1つの Record として出力します。`--expand-chunks` を渡すと `EmitChunks = false` に切り替わり、Chunk が透過的に解凍され、その中の Schema / Channel / Message が逐次トークンとして流れてきます。
- **Attachment**: `Lexer` は Attachment を通常のトークンとしては emit せず `AttachmentCallback` 経由で渡します。本ツールではコールバック内でも同じ通し番号カウンタを使い、ファイル上の正しい位置に Attachment 行を差し込みます。
- **無効 Chunk**: `EmitInvalidChunks: true` を有効化しているため、CRC が壊れた Chunk があれば `InvalidChunk` として表示されます。

## ディレクトリ構成

```
.
├── main.go              # CLI 本体
├── internal/gen/        # 動作確認用のサンプル mcap 生成ヘルパー
│   └── main.go
├── go.mod
└── go.sum
```

サンプル mcap を作る場合：

```sh
go run ./internal/gen sample.mcap
./mcap-record-order sample.mcap
```

## 参考

- MCAP 仕様: <https://mcap.dev/spec>
- Foxglove 製 Go ライブラリ: <https://github.com/foxglove/mcap/tree/main/go/mcap>
