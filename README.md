# scrapbox-cli

`sb` is a command line client for reading and editing [Scrapbox (Cosense)](https://scrapbox.io), written in Go.

## Features

- Read page content, page lists, titles, links, related pages, code blocks and tables
- Full-text search, semantic (vector) search, and search within related pages
- Export a page and its related pages as LLM-ready text with Smart Context ("Export for AI")
- Create, append to, edit, rename and delete pages
- Read private projects with a Personal Access Token, Service Account or cookie
- Print results as JSON with `-o json` for `jq` and AI agents

## Install

| Method | Steps |
|---|---|
| Binary | Download the archive for your OS from [Releases](https://github.com/hiroki1117/scrapbox-cli/releases) and put `sb` on your PATH |
| go install | `go install github.com/hiroki1117/scrapbox-cli/cmd/sb@latest` (Go 1.27+) |
| From source | `git clone`, then `make build` (creates `bin/sb`) |

Binaries are available for Linux, macOS and Windows on amd64 and arm64. Run `sb version` to check the version.
To verify that an archive was built by this repository's release workflow, run `gh attestation verify <archive> -R hiroki1117/scrapbox-cli`.

## Quick start

```bash
sb config set project help-jp      # 1. Set the default project
sb page get "Smart Context"        # 2. Read a page
sb search text API                 # 3. Search
sb auth login                      # Authenticate to read private projects
```

Run `sb --help` for the list of commands and `sb <command> --help` for details.
The examples use [help-jp](https://scrapbox.io/help-jp), a public project.

## Configuration

Settings come from flags, environment variables and the config file. The precedence is **flags > environment variables > config file**.

### Project

```bash
sb page list --project my-project     # Flag (short: -p)
export COSENSE_PROJECT_NAME=my-project # Environment variable
sb config set project my-project      # Save as the default in the config file
```

### Authentication (private projects)

Public projects need no authentication. Private projects, Smart Context and writes do.
Three methods are supported. Writes need a Personal Access Token or Service Account.

| Method | `--type` | Environment variable | Notes |
|---|---|---|---|
| Personal Access Token | `pat` (default) | `COSENSE_PAT` | Recommended. Issue one in your [settings](https://scrapbox.io/settings/personal-access-tokens) |
| Service Account | `service-account` | `COSENSE_SERVICE_ACCOUNT_KEY` | Business plan. Accesses a single project |
| connect.sid cookie | `sid` | `COSENSE_SID` | Cookie value of a logged-in browser |

Use `sb auth login` to save a credential in the config file. The input is hidden.

```bash
sb auth login                          # Enter a PAT at the prompt
sb auth login --type sid               # Use connect.sid
pbpaste | sb auth login --with-token   # Read from stdin
sb auth status                         # Show the current auth state (credential masked)
sb auth logout                         # Remove from the config file
```

- `login` verifies the credential against the API before saving. Use `--no-verify` to skip this.
- A Service Account is verified against the default project.
- Credentials are never accepted as command line arguments, to keep them out of shell history.
- If any credential is set in environment variables, credentials in the config file are ignored.
- If several methods are set, PAT > Service Account > connect.sid is used.

### Config file

The default path is `~/.config/sb/config.json` (change it with `$XDG_CONFIG_HOME/sb/config.json` or `SB_CONFIG`).
It contains credentials, so it is created with permission 0600.

```bash
sb config path               # Print the path
sb config get project
sb config set project my-project
sb config unset project
```

### Environment variables

| Variable | Flag | Description |
|---|---|---|
| `COSENSE_PROJECT_NAME` | `-p, --project` | Default project name |
| `COSENSE_PAT` | - | Personal Access Token |
| `COSENSE_SERVICE_ACCOUNT_KEY` | - | Service Account access key |
| `COSENSE_SID` | - | `connect.sid` cookie value |
| `SCRAPBOX_API_URL` | `--base-url` | API base URL (default: `https://scrapbox.io/api`) |
| `SCRAPBOX_TIMEOUT` | `--timeout` | Request timeout (default: `30s`) |
| `SCRAPBOX_MAX_RETRIES` | `--retries` | Retries on transient errors (default: `3`) |
| `SB_CONFIG` | - | Config file path |

`COSENSE_PROJECT_NAME`, `COSENSE_SID` and `SCRAPBOX_API_URL` are shared with [scrapbox_mcp](https://github.com/hiroki1117/scrapbox_mcp).

## Usage

Commands that take a page also accept a page URL (`https://scrapbox.io/<project>/<title>`) instead of a title.
With a URL, the project in the URL is used.

```bash
export COSENSE_PROJECT_NAME=help-jp

# Pages
sb page get "Smart Context"                 # Print the content
sb page get "Smart Context" -n              # With line numbers
sb page get https://scrapbox.io/help-jp/Smart_Context
sb page list --limit 20 --sort views        # List pages (--skip to offset)
sb page list --all -o json                  # Fetch all pages, paging automatically
sb page titles                              # All titles (lightweight)
sb page links "Smart Context"               # Links in the page
sb page related "Smart Context"             # Related pages (1-hop)
sb page related "Smart Context" --hops 2    # 2-hop pages
sb page code "Page" main.go > main.go       # Code block (code:main.go)
sb page table "Page" members > members.csv  # Table (table:members) as CSV

# Search
sb search text golang cli                   # Full-text search (AND)
sb search text --or golang rust             # OR search
sb search text golang --limit 10            # Limit the results
sb search text -- golang -python            # Put excluded words after --
sb search vector "AI integration"           # Semantic search (titles and links)
sb search related "Smart Context" API       # Full-text search within related pages (2 hop search)
sb search related "Smart Context" AI API --or --hops 2

# Write (requires a PAT or Service Account; --dry-run shows the result without writing)
# Writes use the same API as the official Cosense CLI, which is not publicly documented and may change.
sb page create "New page" -t "first line"   # Create a page (body from --text, --file or stdin)
sb page append "Daily log" -t "- done"      # Append lines to the end
date | sb page append "Daily log"           # Text from stdin
sb page insert "Page" --after 3 -t "text"   # Insert after line 3 (line numbers from page get -n)
sb page replace "Page" Scrapobx Scrapbox    # Replace text (must match once; --all for every match)
sb page delete-lines "Page" 5-7             # Delete lines 5 to 7
sb page rename "Old" "New" --update-links   # Rename and rewrite links to it
sb page delete "Old page"                   # Show the page to delete; add --yes to delete
sb page replace-links "Old" "New"           # Rewrite links across the project

# Smart Context (Export for AI)
sb context "Smart Context" | pbcopy
sb context "Smart Context" --hops 2 > context.txt

# Projects and users
sb project get [name]
sb project list                             # Projects you belong to (requires auth)
sb whoami

# Other
sb version
sb help environment                         # Environment variables and precedence
sb completion zsh > "${fpath[1]}/_sb"       # Shell completion (bash / zsh / fish / powershell)
```

### Output format

`-o json` prints JSON that keeps the API response structure. It is meant to be used with `jq`.

```bash
sb search text golang -o json | jq -r '.pages[].title'
```

`page code` and `page table` always print raw text regardless of `-o`, so they can be redirected to files.

## Design

```
cmd/sb/            Entry point (signals, exit code, error hints)
internal/
  cli/             cobra commands. App passes the config, Client and Printer to each command
  config/          Loads the config file and environment variables; saves the config file
  output/          Text / JSON output, table and time formatting
  scrapbox/        REST API client (auth, retries, error conversion, APIs)
```

Dependencies flow `cmd → cli → (config, output, scrapbox)`. The `scrapbox` package depends on no other package in this module.

### Extending

- **Add an API**: Add a method to the matching service in `scrapbox` (such as `PagesService`). For a new resource, define `type XxxService service` and register it in `NewClient`. Use `getJSON` / `getText` for the common HTTP handling.
- **Add a command**: Define `newXxxCmd(app *App) *cobra.Command` in `internal/cli` and add it to its parent with `AddCommand`. Print with `app.Printer().Print(v, textFunc)` to support `-o json` automatically. Write `Long` (including the output format) and `Example` (`help_test.go` checks this for read, search and write commands).
- **Add an auth method**: Implement `scrapbox.Authenticator` and add it to `config.AuthType` and `config.Credentials`.
- **Add a setting**: Add a field to `config.File` and register it in `configKeys` in `internal/cli/configcmd.go`.
- **Add a write operation**: Build `scrapbox.Change`s (`InsertBefore`, `Replace`, `DeleteLine`, `DeletePage`) and pass them to `applyEdit` in `internal/cli/edit.go`, which previews and submits them through `PagesService.PreviewEdit` / `SubmitEdit`.

### Help

`sb --help` covers how to get started and the common rules (argument format, how the project is chosen, output streams, exit codes). Each command's `--help` covers its output format and examples.
For AI agents, start by having them read `sb --help`.

### Retries

429 / 502 / 503 / 504 and network errors are retried up to 3 times with exponential backoff (with jitter).
A `Retry-After` header is honored. Change the count with `--retries` or `SCRAPBOX_MAX_RETRIES`; `0` disables retries.
Writes are never retried, so a change is never applied twice.

## Development

```bash
make test
make vet
make build
```

### CI

GitHub Actions (`.github/workflows/ci.yml`) runs the following on every push and pull request, and weekly.

| Job | What it does |
|---|---|
| lint | `gofmt` diff, `go mod tidy` diff, `go vet`, golangci-lint |
| test | `go test -race` on ubuntu / macos / windows × Go 1.27 (minimum) and stable |
| vulncheck | govulncheck checks dependencies and the standard library for known vulnerabilities |
| release-check | GoReleaser builds all targets to verify the release config (nothing is published) |

### Maintenance

- Dependabot opens up to two grouped pull requests a week: Go modules and GitHub Actions (pinned by SHA). Security updates are opened as soon as an advisory is published.
- Not covered by Dependabot, so update by hand:
  - Go: when a new Go version is released (every February and August), raise the `go` directive in `go.mod` and the minimum version in the CI matrix to the oldest supported Go.
  - The golangci-lint and GoReleaser versions in `.github/workflows/`.
- Report vulnerabilities as described in [SECURITY.md](SECURITY.md).

### Release

Pushing a tag starting with `v` runs `.github/workflows/release.yml`, which uses GoReleaser (`.goreleaser.yaml`) to build binaries and checksums for each OS, attest their build provenance, and publish them to GitHub Releases.
Only users with write access to this repository can push tags. All actions are pinned by commit SHA.

```bash
git tag v0.1.0
git push origin v0.1.0
make snapshot   # Build the same artifacts into dist/ without publishing (requires goreleaser)
```

## License

[MIT](LICENSE)
