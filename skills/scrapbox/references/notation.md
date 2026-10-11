# Scrapbox notation (for writing pages)

Scrapbox is not Markdown. Markdown headings, `**bold**`, `- ` lists and fenced code blocks show up as literal text. Use this notation.

## Lines and structure

- One line is one paragraph. Line 1 of a page is its title.
- Indentation makes a bullet list. Each leading space (or tab) adds one level. Do not write `- ` or `* `.
- Leave an empty line to separate blocks.

```
Topic
 First point
  Detail under the first point
 Second point
```

## Inline notation

| Write | Result |
|---|---|
| `[Page title]` | Link to a page in the same project (shown as an empty page if it does not exist yet) |
| `#tag` | Link, shown as a hashtag. Use `[tag with spaces]` for titles with spaces |
| `[/other-project/Page]` | Link to a page in another project |
| `[https://example.com label]` or `[label https://example.com]` | External link with a label |
| `[* Heading]` | Bold. `[** ...]`, `[*** ...]` are larger, and work as headings |
| `[/ italic]` / `[- strike]` | Italic / strikethrough |
| `` `code` `` | Inline code |
| `[$ x^2]` | TeX formula |
| `> quoted` | Quote (at the start of a line) |
| `[https://gyazo.com/...]` | Image (an image URL in brackets) |
| `[username.icon]` | User or page icon |

Brackets always make a link or decoration. To write a literal `[`, put it in inline code.

## Code blocks

`code:<filename>` starts a block. Every following line indented one more level belongs to it. The block ends at the first line that is not indented. `sb page code "<title>" <filename>` prints it back.

```
code:hello.go
 package main

 func main() {}
```

## Tables

`table:<name>` starts a table. Each following indented line is a row, with cells separated by tabs. `sb page table "<title>" <name>` prints it as CSV.

```
table:members
 name	role
 Alice	owner
```

When you pass text with `--text`, write real tab characters, for example with `$'...'` in bash: `--text $'table:t\n a\tb'`. Using `--file` is easier.

## Style

- Link concepts that have, or should have, their own page. Links build the related-page graph that `sb page related`, `sb search related` and `sb context` use.
- Do not link every noun. Unneeded links add empty pages to the graph.
- Follow the conventions already in the project: read a few nearby pages before writing a new one.
