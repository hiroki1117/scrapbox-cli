package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/output"
	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

func newPageCmd(app *App) *cobra.Command {
	cmd := newGroupCmd("page", "Read and edit pages", "")
	cmd.AddGroup(
		&cobra.Group{ID: groupRead, Title: "Read commands:"},
		&cobra.Group{ID: groupWrite, Title: "Write commands (require a Personal Access Token or Service Account):"},
	)
	addToGroup(cmd, groupRead,
		newPageGetCmd(app),
		newPageListCmd(app),
		newPageTitlesCmd(app),
		newPageLinksCmd(app),
		newPageRelatedCmd(app),
		newPageCodeCmd(app),
		newPageTableCmd(app),
	)
	addToGroup(cmd, groupWrite,
		newPageCreateCmd(app),
		newPageAppendCmd(app),
		newPageInsertCmd(app),
		newPageReplaceCmd(app),
		newPageDeleteLinesCmd(app),
		newPageRenameCmd(app),
		newPageDeleteCmd(app),
		newPageReplaceLinksCmd(app),
	)
	return cmd
}

func newPageGetCmd(app *App) *cobra.Command {
	var number bool
	cmd := &cobra.Command{
		Use:   "get <title|url>",
		Short: "Print the content of a page",
		Long: `Print the content of a page in Scrapbox notation. The first line is the title.

` + pageArgHelp + `

Text output: page lines as-is (with -n, prefixed by line numbers).
JSON output: {title, lines: [{text, created, updated, ...}], links, descriptions,
             relatedPages, views, linked, created, updated, ...}`,
		Example: `  sb page get "Smart Context"
  sb page get https://scrapbox.io/help-jp/Smart_Context
  sb page get "Smart Context" -o json | jq -r '.lines[].text'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := fetchPage(cmd, app, args[0])
			if err != nil {
				return err
			}
			page := t.page
			return app.Printer().Print(page, func(w io.Writer) error {
				width := len(strconv.Itoa(len(page.Lines)))
				for i, l := range page.Lines {
					if number {
						fmt.Fprintf(w, "%*d  ", width, i+1)
					}
					fmt.Fprintln(w, l.Text)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVarP(&number, "number", "n", false, "prefix each line with its line number")
	return cmd
}

func newPageListCmd(app *App) *cobra.Command {
	var (
		opts scrapbox.ListOptions
		all  bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pages in a project",
		Long: `List pages in a project. Pinned pages come first and are marked "[pin]".

Text output: UPDATED, VIEWS, LINKED (backlink count), TITLE.
JSON output: {projectName, count, skip, limit, pages: [{title, descriptions,
             views, linked, pin, created, updated, ...}]}`,
		Example: `  sb page list --limit 20 --sort views
  sb page list --skip 100 --limit 100
  sb page list --all -o json | jq -r '.pages[].title'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, client, err := app.projectClient()
			if err != nil {
				return err
			}

			var list *scrapbox.PageList
			if all {
				// Wrap in a PageList so the JSON shape is the same with or without --all.
				list = &scrapbox.PageList{ProjectName: project, Skip: opts.Skip, Pages: []scrapbox.PageSummary{}}
				for p, err := range client.Pages.ListAll(cmd.Context(), project, &opts) {
					if err != nil {
						return err
					}
					list.Pages = append(list.Pages, p)
				}
				list.Count, list.Limit = opts.Skip+len(list.Pages), len(list.Pages)
			} else {
				list, err = client.Pages.List(cmd.Context(), project, &opts)
				if err != nil {
					return err
				}
			}

			return app.Printer().Print(list, func(w io.Writer) error {
				tw := output.Table(w)
				fmt.Fprintln(tw, "UPDATED\tVIEWS\tLINKED\tTITLE")
				for _, p := range list.Pages {
					title := p.Title
					if p.Pinned() {
						title = "[pin] " + title
					}
					fmt.Fprintf(tw, "%s\t%d\t%d\t%s\n", output.Time(p.Updated.Time()), p.Views, p.Linked, title)
				}
				return tw.Flush()
			})
		},
	}
	f := cmd.Flags()
	f.IntVar(&opts.Limit, "limit", 100, fmt.Sprintf("number of pages to fetch (max %d; with --all, per request)", scrapbox.MaxListLimit))
	f.IntVar(&opts.Skip, "skip", 0, "number of pages to skip")
	f.StringVar(&opts.Sort, "sort", "", "sort key: updated|created|accessed|linked|views|title (default: updated)")
	f.BoolVar(&all, "all", false, "fetch all pages, paging automatically")
	return cmd
}

func newPageTitlesCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "titles",
		Short: "Print all page titles in a project",
		Long: `Print all page titles in a project, including the links of each page.
Lighter than 'page list --all'; useful for grep, fzf or checking whether a page exists.

Text output: one title per line.
JSON output: [{id, title, links, updated}]`,
		Example: `  sb page titles | grep -i golang
  sb page titles -o json | jq -r '.[] | select(.links | index("Go")) | .title'`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, client, err := app.projectClient()
			if err != nil {
				return err
			}
			var entries []scrapbox.TitleEntry
			for e, err := range client.Pages.Titles(cmd.Context(), project) {
				if err != nil {
					return err
				}
				entries = append(entries, e)
			}
			return app.Printer().Print(entries, func(w io.Writer) error {
				for _, e := range entries {
					fmt.Fprintln(w, e.Title)
				}
				return nil
			})
		},
	}
}

func newPageLinksCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "links <title|url>",
		Short: "Print links written in a page",
		Long: `Print the links ([title] notation) written in a page, including links to empty pages.
Backlinks are not included; use 'sb page related' for pages linking to this page.

` + pageArgHelp + `

Text output: one link title per line.
JSON output: ["title", ...]`,
		Example: `  sb page links "Smart Context"
  sb page links "Smart Context" | xargs -I{} sb page get {}`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := fetchPage(cmd, app, args[0])
			if err != nil {
				return err
			}
			page := t.page
			return app.Printer().Print(page.Links, func(w io.Writer) error {
				for _, l := range page.Links {
					fmt.Fprintln(w, l)
				}
				return nil
			})
		},
	}
}

func newPageRelatedCmd(app *App) *cobra.Command {
	var hops int
	cmd := &cobra.Command{
		Use:   "related <title|url>",
		Short: "List related pages (1-hop or 2-hop links)",
		Long: `List pages in the related page list of a page.

1-hop: pages linking to or linked from the page.
2-hop: pages sharing a link with the 1-hop pages (1-hop pages are not included).
To filter related pages by keywords, use 'sb search related'.

` + pageArgHelp + `

Text output: UPDATED, LINKED (backlink count), TITLE.
JSON output: {hops, pages: [{title, descriptions, linksLc, ...}], pagination}`,
		Example: `  sb page related "Smart Context"
  sb page related "Smart Context" --hops 2 -o json | jq -r '.pages[].title'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRelated(cmd, app, args[0], hops, nil)
		},
	}
	cmd.Flags().IntVar(&hops, "hops", 1, "range of related pages: 1 or 2")
	return cmd
}

func newPageCodeCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "code <title|url> <filename>",
		Short: "Print a code block (code:<filename>) in a page",
		Long: `Print the content of a code block written as "code:<filename>" in a page.

` + pageArgHelp + `

Output is always the raw code (-o is ignored) so it can be redirected to a file.`,
		Example: `  sb page code "Page Title" main.go > main.go
  sb page code "Page Title" script.sh | sh`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return printRaw(cmd, app, args[0], args[1], (*scrapbox.PagesService).Code)
		},
	}
}

func newPageTableCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "table <title|url> <filename>",
		Short: "Print a table (table:<filename>) in a page as CSV",
		Long: `Print a table written as "table:<filename>" in a page, converted to CSV.

` + pageArgHelp + `

Output is always CSV (-o is ignored) so it can be redirected to a file.`,
		Example: `  sb page table "Page Title" members > members.csv`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return printRaw(cmd, app, args[0], args[1], (*scrapbox.PagesService).Table)
		},
	}
}

// pageTarget is a fetched page with the client used to fetch it.
type pageTarget struct {
	project string
	page    *scrapbox.Page
	client  *scrapbox.Client
}

// fetchPage fetches the page given as an argument.
// With authentication the API returns a missing page as an empty page, so it is reported as not found.
func fetchPage(cmd *cobra.Command, app *App, arg string) (*pageTarget, error) {
	ref, client, err := app.pageClient(arg)
	if err != nil {
		return nil, err
	}
	page, err := client.Pages.Get(cmd.Context(), ref.Project, ref.Title)
	if err == nil && !page.Persistent {
		err = scrapbox.ErrNotFound
	}
	if errors.Is(err, scrapbox.ErrNotFound) {
		return nil, fmt.Errorf("page %q does not exist in %s: %w", ref.Title, ref.Project, scrapbox.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}
	return &pageTarget{ref.Project, page, client}, nil
}

// rawFetcher fetches a file in a page (code block or table) as raw text.
type rawFetcher func(s *scrapbox.PagesService, ctx context.Context, project, title, filename string) (string, error)

// printRaw fetches a file in a page and prints it as-is, regardless of the output format.
// Code and CSV are meant to be redirected to files, so they are not wrapped in JSON.
func printRaw(cmd *cobra.Command, app *App, arg, filename string, fetch rawFetcher) error {
	ref, client, err := app.pageClient(arg)
	if err != nil {
		return err
	}
	body, err := fetch(client.Pages, cmd.Context(), ref.Project, ref.Title, filename)
	if err != nil {
		return err
	}
	_, err = io.WriteString(app.IO.Out, body)
	return err
}
