package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/output"
	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

func newSearchCmd(app *App) *cobra.Command {
	cmd := newGroupCmd("search", "Search pages (full-text, semantic, within related pages)", `Search pages in a project.

  text     keyword match over page bodies
  vector   semantic match over page titles and links; finds similar topics
  related  keyword match limited to the related pages of a page`)
	cmd.AddCommand(newSearchTextCmd(app), newSearchVectorCmd(app), newSearchRelatedCmd(app))
	return cmd
}

func newSearchTextCmd(app *App) *cobra.Command {
	var opts scrapbox.SearchOptions
	cmd := &cobra.Command{
		Use:     "text <query>...",
		Aliases: []string{"fulltext"},
		Short:   "Full-text search pages",
		Long: `Full-text search pages in a project.

Arguments are joined with spaces and matched as AND (use --or for OR).
Prefix a word with "-" to exclude it; put "--" before the query so it is not parsed as a flag.

Text output: page title followed by matched lines. The number of hits is printed to stderr.
JSON output: {count, pages: [{title, lines, words, ...}], ...}`,
		Example: `  sb search text golang cli
  sb search text --or golang rust
  sb search text -- golang -python
  sb search text golang -o json | jq -r '.pages[].title'`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := app.Project()
			if err != nil {
				return err
			}
			client, err := app.Client()
			if err != nil {
				return err
			}
			res, err := client.Pages.Search(cmd.Context(), project, strings.Join(args, " "), &opts)
			if err != nil {
				return err
			}
			return app.Printer().Print(res, func(w io.Writer) error {
				for i, p := range res.Pages {
					if i > 0 {
						fmt.Fprintln(w)
					}
					fmt.Fprintln(w, p.Title)
					// 先頭行はタイトルと重複するため除外する。
					for _, l := range p.Lines {
						if l == p.Title {
							continue
						}
						fmt.Fprintf(w, "    %s\n", strings.TrimSpace(l))
					}
				}
				fmt.Fprintf(app.IO.ErrOut, "%d pages found\n", res.Count)
				return nil
			})
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "maximum number of pages (default: all returned by the server)")
	cmd.Flags().BoolVar(&opts.Or, "or", false, "match pages containing any of the words")
	return cmd
}

func newSearchVectorCmd(app *App) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:     "vector <query>...",
		Aliases: []string{"semantic"},
		Short:   "Semantic search pages by meaning",
		Long: `Search pages whose topics are semantically close to the query, ordered by similarity.

Only page titles and links ([title] notation) are searched, not plain body text.
Results may include empty pages that exist only as links (marked "(empty)").
Use this to find related topics when you do not know the exact keywords.

Text output: SCORE, LINKED (backlink count), TITLE.
JSON output: {pages: [{title, score, exists, linked, id?, updated?, ...}]}`,
		Example: `  sb search vector "AIとの連携"
  sb search vector 設計 レビュー --limit 10 -o json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := app.Project()
			if err != nil {
				return err
			}
			client, err := app.Client()
			if err != nil {
				return err
			}
			res, err := client.Pages.SearchVector(cmd.Context(), project, strings.Join(args, " "))
			if err != nil {
				return err
			}
			if limit > 0 && len(res.Pages) > limit {
				res.Pages = res.Pages[:limit]
			}
			return app.Printer().Print(res, func(w io.Writer) error {
				tw := output.Table(w)
				fmt.Fprintln(tw, "SCORE\tLINKED\tTITLE")
				for _, p := range res.Pages {
					title := p.Title
					if !p.Exists {
						title += " (empty)"
					}
					fmt.Fprintf(tw, "%.3f\t%d\t%s\n", p.Score, p.Linked, title)
				}
				return tw.Flush()
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of pages (default: all returned by the server)")
	return cmd
}

func newSearchRelatedCmd(app *App) *cobra.Command {
	var (
		hops int
		opts scrapbox.RelatedOptions
	)
	cmd := &cobra.Command{
		Use:   "related <title|url> <query>...",
		Short: "Full-text search within the related pages of a page",
		Long: `Full-text search limited to the related pages (1-hop or 2-hop links) of a page.
This is the same as "2 hop search" in the Scrapbox related page list.

1-hop: pages linking to or linked from the page.
2-hop: pages sharing a link with the 1-hop pages.

` + pageArgHelp + `

Text output: UPDATED, LINKED (backlink count), TITLE.
JSON output: {hops, pages: [{title, descriptions, linksLc, ...}], pagination}`,
		Example: `  sb search related "Smart Context" API
  sb search related "Smart Context" API token --or --hops 2
  sb search related https://scrapbox.io/help-jp/Smart_Context API -o json`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Search = strings.Join(args[1:], " ")
			return runRelated(cmd, app, args[0], hops, &opts)
		},
	}
	cmd.Flags().IntVar(&hops, "hops", 1, "range of related pages: 1 or 2")
	cmd.Flags().BoolVar(&opts.Or, "or", false, "match pages containing any of the words")
	return cmd
}

// runRelated は関連ページを取得して出力する。page related と search related で共有する。
func runRelated(cmd *cobra.Command, app *App, arg string, hops int, opts *scrapbox.RelatedOptions) error {
	h := scrapbox.Hops(hops)
	if err := h.Validate(); err != nil {
		return err
	}
	ref, err := app.PageRef(arg)
	if err != nil {
		return err
	}
	client, err := app.Client()
	if err != nil {
		return err
	}
	res, err := client.Pages.Related(cmd.Context(), ref.Project, ref.Title, h, opts)
	if err != nil {
		return err
	}
	return app.Printer().Print(res, func(w io.Writer) error {
		tw := output.Table(w)
		fmt.Fprintln(tw, "UPDATED\tLINKED\tTITLE")
		for _, p := range res.Pages {
			fmt.Fprintf(tw, "%s\t%d\t%s\n", output.Time(p.Updated.Time()), p.Linked, p.Title)
		}
		if res.Pagination.HasNext {
			fmt.Fprintf(app.IO.ErrOut, "showing %d of %d pages (truncated by the server)\n", len(res.Pages), res.Pagination.Total)
		}
		return tw.Flush()
	})
}
