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

設定はフラグ・環境変数・設定ファイルの 3 か所で行える。優先順位は **フラグ > 環境変数 > 設定ファイル** である。

### プロジェクトの指定

```bash
sb page list --project my-project     # フラグで指定 (短縮形 -p)
export COSENSE_PROJECT_NAME=my-project # 環境変数で指定
sb config set project my-project      # 設定ファイルにデフォルトとして保存
```

ページ URL を引数に渡した場合は、URL のプロジェクトを使う。

### 認証 (Private Project)

Public Project は認証なしで参照できる。Private Project と Smart Context には認証が必要である。
次の 3 つの方式に対応している。

| 方式 | `--type` | 環境変数 | 説明 |
|---|---|---|---|
| Personal Access Token | `pat` (既定) | `COSENSE_PAT` | 推奨。[ユーザー設定](https://scrapbox.io/settings/personal-access-tokens)で発行する |
| Service Account | `service-account` | `COSENSE_SERVICE_ACCOUNT_KEY` | Business プラン。1 つのプロジェクトにだけアクセスできる |
| connect.sid Cookie | `sid` | `COSENSE_SID` | ブラウザでログイン中の Cookie の値 |

認証情報を設定ファイルに保存する場合は `sb auth login` を使う。入力は画面に表示しない。

```bash
sb auth login                          # PAT をプロンプトで入力
sb auth login --type sid               # connect.sid を使う
pbpaste | sb auth login --with-token   # 標準入力から読む
sb auth status                         # 現在の認証状態 (認証情報はマスク表示)
sb auth logout                         # 設定ファイルから削除
```

- `login` は保存前に API へアクセスして認証情報を検証する。検証を省く場合は `--no-verify` を付ける。
- Service Account はユーザーに紐づかないので、デフォルトプロジェクトへのアクセスで検証する。
- 認証情報はシェル履歴に残らないよう、コマンドライン引数では受け付けない。
- 環境変数に認証情報が 1 つでもあれば、設定ファイルの認証情報は使わない。
- 複数の方式が設定されている場合は PAT > Service Account > connect.sid の順に使う。

### 設定ファイル

既定のパスは `~/.config/sb/config.json` である（`$XDG_CONFIG_HOME/sb/config.json`、または `SB_CONFIG` で変更できる）。
認証情報を含むので、パーミッション 0600 で作成する。

```bash
sb config path               # パスを表示
sb config get project
sb config set project my-project
sb config unset project
```

### 環境変数一覧

| 環境変数 | フラグ | 説明 |
|---|---|---|
| `COSENSE_PROJECT_NAME` | `-p, --project` | デフォルトのプロジェクト名 |
| `COSENSE_PAT` | - | Personal Access Token |
| `COSENSE_SERVICE_ACCOUNT_KEY` | - | Service Account のアクセスキー |
| `COSENSE_SID` | - | `connect.sid` Cookie の値 |
| `SCRAPBOX_API_URL` | `--base-url` | API のベース URL（既定: `https://scrapbox.io/api`） |
| `SCRAPBOX_TIMEOUT` | `--timeout` | リクエストのタイムアウト（既定: `30s`） |
| `SB_CONFIG` | - | 設定ファイルのパス |

`COSENSE_PROJECT_NAME` / `COSENSE_SID` / `SCRAPBOX_API_URL` は [scrapbox_mcp](https://github.com/hiroki1117/scrapbox_mcp) と共通である。

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
  config/            設定ファイル・環境変数から設定を読み込む / 設定ファイルの保存
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
- **認証方式を追加する**: `scrapbox.Authenticator` を実装し、`config.AuthType` と `config.Credentials` に追加する。
- **設定項目を追加する**: `config.File` にフィールドを追加し、`internal/cli/configcmd.go` の `configKeys` に登録する。
- **書き込み系**: scrapbox_mcp と同様に WebSocket (Socket.IO) で実装する想定である。`scrapbox` パッケージに追加し、`cli` に `page create` などのサブコマンドを追加する。

## 開発

```bash
make test
make vet
make build
```
