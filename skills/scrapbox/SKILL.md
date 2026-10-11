---
name: scrapbox
description: Read, search and edit Scrapbox (Cosense) pages with the `sb` CLI. Use when the user mentions Scrapbox or Cosense, pastes a scrapbox.io URL, asks to look something up in or gather context from their Scrapbox project, or asks to create, append to, fix or rename a Scrapbox page. Covers full-text, semantic and related-page search, exporting a topic as LLM context (Smart Context), and safe line-level edits.
---

# Scrapbox via `sb`

`sb` is a CLI for Scrapbox (Cosense). This skill tells you which command to use and how to edit pages without breaking them. For flags and output shapes, run `sb <command> --help`; every command's help lists its text and JSON output.

## Ground rules

- Pass `-o json` when you will parse the result. Pipe it to `jq`. `page code` and `page table` ignore `-o` and print raw text.
- stdout has results only. Hit counts, hints and errors go to stderr. Exit code is `0` on success and `1` on any error. In a pipe such as `sb ... | jq`, the exit code is jq's unless you `set -o pipefail`.
- A page argument is a title in the current project or a page URL (`https://scrapbox.io/<project>/<title>`). With a URL, the URL's project is used. Pass URLs the user gives you as they are. To follow a link to another project (`[/other-project/Page]`), pass `https://scrapbox.io/other-project/Page`.
- Always quote titles: `sb page get "Daily log"`.
- For writes, always pass the text with `--text` or `--file`. Without them `sb` reads stdin, which can block when stdin is an open pipe.
- Never put credentials in command arguments and never run `sb auth login` yourself, because it prompts for input. If authentication is missing, ask the user to run `! sb auth login`.

## Setup check

Run this once before the first command:

```bash
sb version && sb auth status
```

- `sb: command not found`: tell the user to install it (`go install github.com/hiroki1117/scrapbox-cli/cmd/sb@latest` or a release binary). Stop.
- When the user names a project, pass `-p <project>` on every call, including `sb context` and writes.
- No project named, none set, and no URL given: ask the user for the project name. Run `sb config set project <name>` only if the user asks for a default.
- Public projects need no auth. Private projects, `sb context` and all writes do. Writes need a Personal Access Token or Service Account; a `connect.sid` cookie is read-only.

## Choose a command

| Goal | Command |
|---|---|
| Read a page | `sb page get "<title>"` (`-n` for line numbers) |
| Find pages containing words | `sb search text <words>...` (AND; `--or` for OR) |
| Find topics when you do not know the exact words | `sb search vector "<phrase>"` |
| Narrow search to pages around a known page | `sb search related "<title>" <words>... [--hops 2]` |
| List pages linked with a page | `sb page related "<title>" [--hops 2]` |
| Get a topic and its neighborhood as one text | `sb context "<title>"` (requires auth) |
| List outgoing links of a page | `sb page links "<title>"` |
| Get all titles cheaply | `sb page titles` |
| Recent or popular pages | `sb page list --sort updated\|views --limit N` |
| Extract a code block or table | `sb page code "<title>" <filename>` / `sb page table "<title>" <name>` |

## Research workflow

1. Search. Start with `sb search text <keywords> --limit 10 -o json | jq -r '.pages[] | .title, ("  " + .lines[])'`. This prints each title followed by its matched lines.
   - Use specific terms that would appear in the page (`Export for AI`, `Smart Context`). Generic words such as `AI` or `ページ` match too many pages.
   - Several words are ANDed. If you get nothing, drop words or use `--or` before trying synonyms.
   - `sb search vector` matches only titles and links, not body text, and often returns only loosely related titles. Use it as a last resort to discover page names, then go back to `search text` or `page get`. Skip results marked `(empty)`; they are link-only pages.
2. Read the most relevant 1–3 pages with `sb page get`.
3. Expand only if the pages you read do not answer the question. If you need the surrounding topic, run `sb context "<title>"` (1 hop). Use `--hops 2` only when 1 hop is not enough, because it can exceed 1 MB. Without auth, use `sb page related` plus `sb page get` instead.
4. Cite. When you answer, give the page URL. `sb` does not output it for reads, so build it: `https://scrapbox.io/<project>/<title>`, with spaces replaced by `_` and the rest percent-encoded like `encodeURIComponent` (`Cosense Skill & CLI` → `Cosense_Skill_%26_CLI`). Non-ASCII characters may stay as they are.

Do not fetch a whole project with `page list --all` or `page titles` and then read pages one by one. Search first.

## Editing workflow

Writes change a shared wiki that other people read. Follow these steps every time.

1. Read the page right before you edit it: `sb page get "<title>" -n`. Line 1 is the title. Line numbers shift after every write, so never reuse numbers from an earlier read.
2. Pick the least fragile command:

   | Change | Command | Why |
   |---|---|---|
   | Add to the end | `sb page append` | No line numbers needed |
   | Fix words in a line | `sb page replace "<title>" "<old>" "<new>"` | Matches by content. Fails if `<old>` occurs 0 or 2+ times, so it never edits the wrong line |
   | Insert in the middle | `sb page insert --after N` | Needs a fresh `-n` read |
   | Remove lines | `sb page delete-lines "<title>" N-M` | Needs a fresh `-n` read |
   | New page | `sb page create` | Fails if the page exists; then use `append` |

   `replace` works within a single line, and `<new>` must be one line. To rewrite several lines, use `delete-lines` and then `insert`, re-reading with `-n` between the two writes.
3. Run with `--dry-run` first and check the preview. Added lines are marked `+`, changed lines `~` and deleted lines `-`.
4. Run the same command without `--dry-run`.
5. If it fails with `conflict`, someone else edited the page. Go back to step 1. `sb` never retries writes, so a failed write was not applied.

Write content in Scrapbox notation, not Markdown. See [references/notation.md](references/notation.md). Match the indentation and style of the lines around your change.

### Confirm with the user first

Ask before you run any of these. They affect more than the lines you were asked to change, or they cannot easily be undone.

- `sb page delete` (it only previews without `--yes`; show the user the preview, then add `--yes`)
- `sb page rename` (with `--update-links`, it also rewrites other pages)
- `sb page replace-links` (rewrites links in every page of the project; there is no dry run)
- `sb page replace --all`
- Any write to a page the user did not name

## Errors

| stderr contains | Meaning | Do |
|---|---|---|
| `not found` | Wrong project or title | Check the project, then find the title with `sb search text` or `sb page titles \| grep -i`. Case does not matter, but spelling and spaces do |
| `unauthorized` / `forbidden` | Auth missing or not enough for this action | Ask the user to run `! sb auth login`. For writes, a PAT is required |
| `conflict` | Page changed during the write | Re-read with `-n` and redo the edit |
| `occurs N times (lines ...)` | `replace` matched more than once | Use a longer `<old>` that is unique |
| `not found in the page body` | `replace` matched nothing | Re-read the page; the text may differ in spacing or brackets |

A query word that starts with `-` is an exclusion. Put `--` before the query so `sb` does not parse it as a flag: `sb search text -- golang -python`.
