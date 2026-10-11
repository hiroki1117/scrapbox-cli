# scrapbox-cli

Scrapbox (Cosense) を操作する Go 製の CLI `sb`。現在は参照系の操作に対応している。

## インストール

```bash
go install github.com/hiroki1117/scrapbox-cli/cmd/sb@latest
# またはリポジトリ内で
make build   # bin/sb を生成
```

Go 1.23 以上が必要である。

## 設定

設定は環境変数とグローバルフラグで行う。フラグは環境変数より優先する。
認証情報はシェル履歴やプロセス一覧に残らないよう、環境変数からのみ読み込む。

| 環境変数 | フラグ | 説明 |
|---|---|---|
| `COSENSE_PROJECT_NAME` | `-p, --project` | デフォルトのプロジェクト名 |
| `COSENSE_PAT` | - | [Personal Access Token](https://scrapbox.io/settings/personal-access-tokens)（推奨） |
| `COSENSE_SERVICE_ACCOUNT_KEY` | - | Service Account のアクセスキー（Business プラン） |
| `COSENSE_SID` | - | `connect.sid` Cookie の値 |
| `SCRAPBOX_API_URL` | `--base-url` | API のベース URL（既定: `https://scrapbox.io/api`） |
| `SCRAPBOX_TIMEOUT` | `--timeout` | リクエストのタイムアウト（既定: `30s`） |

認証情報が複数ある場合は `COSENSE_PAT` > `COSENSE_SERVICE_ACCOUNT_KEY` > `COSENSE_SID` の順に使う。
`COSENSE_PROJECT_NAME` / `COSENSE_SID` / `SCRAPBOX_API_URL` は [scrapbox_mcp](https://github.com/hiroki1117/scrapbox_mcp) と共通である。

Public Project は認証なしで参照できる。Smart Context はログインが必要である。

## 使い方

ページを受け取るコマンドは、タイトルの代わりにページ URL（`https://scrapbox.io/<project>/<title>`）も受け付ける。
URL を渡した場合は URL のプロジェクトを使う。

```bash
export COSENSE_PROJECT_NAME=help-jp

# ページ
sb page get "Smart Context"                 # 本文を出力
sb page get "Smart Context" -n              # 行番号付き
sb page get https://scrapbox.io/help-jp/Smart_Context
sb page list --limit 20 --sort views        # 一覧
sb page list --all -o json                  # 全件をページングして取得
sb page titles                              # 全タイトル (軽量)
sb page links "Smart Context"               # ページ内のリンク
sb page related "Smart Context" --hops 2    # 関連ページリスト (1hop / 2hop)
sb page code "Page" main.go > main.go       # コードブロック (code:main.go)
sb page table "Page" members > members.csv  # テーブル (table:members) を CSV で

# 検索
sb search golang cli
sb search -- golang -python                 # 除外語は -- の後に書く

# Smart Context (Export for AI)
sb context "Smart Context" | pbcopy
sb context "Smart Context" --hops 2 > context.txt

# プロジェクト・ユーザー
sb project get [name]
sb project list                             # 参加プロジェクト (要認証)
sb whoami
```

### 出力形式

`-o json` を付けると API のレスポンス構造を保った JSON を出力する。`jq` との併用を想定している。

```bash
sb search golang -o json | jq -r '.pages[].title'
```

`page code` / `page table` は、リダイレクトでファイル保存する用途を想定し、`-o` に関わらず生テキストを出力する。

## 設計

```
cmd/sb/              エントリポイント (シグナル処理、終了コード、エラーヒント)
internal/
  cli/               cobra のコマンド定義。App が設定・Client・Printer を各コマンドへ渡す
  config/            環境変数から設定を読み込む
  output/            text / json の出力切り替え、テーブル・時刻の整形
  scrapbox/          REST API クライアント (CLI に依存しない)
    client.go        HTTP 処理の共通部 (リクエスト生成、エラー変換、JSON / テキストの取得)
    auth.go          Authenticator インターフェースと各認証方式
    errors.go        APIError と番兵エラー (ErrNotFound など、errors.Is で判定)
    pages.go         ページ API (一覧・取得・検索・タイトル・コード・テーブル)
    projects.go      プロジェクト API
    users.go         ユーザー API
    smartcontext.go  Smart Context API
```

依存の向きは `cmd → cli → (config, output, scrapbox)` で、`scrapbox` パッケージは他のどのパッケージにも依存しない。

### 拡張方法

- **API を追加する**: `scrapbox` の該当サービス（`PagesService` など）にメソッドを追加する。新しいリソースなら `type XxxService service` を定義し、`NewClient` で登録する。共通の HTTP 処理は `getJSON` / `getText` を使う。
- **コマンドを追加する**: `internal/cli` に `newXxxCmd(app *App) *cobra.Command` を定義し、親コマンドの `AddCommand` に加える。出力は `app.Printer().Print(v, textFunc)` を使えば `-o json` に自動で対応する。
- **認証方式を追加する**: `scrapbox.Authenticator` を実装し、`config.Credentials.Authenticator` で選択する。
- **書き込み系**: scrapbox_mcp と同様に WebSocket (Socket.IO) で実装する想定である。`scrapbox` パッケージに追加し、`cli` に `page create` などのサブコマンドを追加する。

## 開発

```bash
make test
make vet
make build
```
