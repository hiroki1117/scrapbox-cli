package cli

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/output"
	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

func newPageCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "page",
		Short: "Read pages",
	}
	cmd.AddCommand(
		newPageGetCmd(app),
		newPageListCmd(app),
		newPageTitlesCmd(app),
		newPageLinksCmd(app),
		newPageRelatedCmd(app),
		newPageCodeCmd(app),
		newPageTableCmd(app),
	)
	return cmd
}

const pageArgHelp = "<title|url> accepts a page title in the current project or a page URL like https://scrapbox.io/<project>/<title>."

func newPageGetCmd(app *App) *cobra.Command {
	var number bool
	cmd := &cobra.Command{
		Use:   "get <title|url>",
		Short: "Print the content of a page",
		Long:  "Print the content of a page.\n\n" + pageArgHelp,
		Example: `  sb page get "Page Title"
  sb page get https://scrapbox.io/help-jp/Smart_Context
  sb page get "Page Title" -o json | jq '.lines[].text'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			page, err := fetchPage(cmd, app, args[0])
			if err != nil {
				return err
			}
			return app.Printer().Print(page, func(w io.Writer) error {
				width := len(fmt.Sprint(len(page.Lines)))
				for i, l := range page.Lines {
					if number {
						fmt.Fprintf(w, "%*d  %s\n", width, i+1, l.Text)
					} else {
						fmt.Fprintln(w, l.Text)
					}
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
		Example: `  sb page list --limit 20 --sort views
  sb page list --all -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := app.Project()
			if err != nil {
				return err
			}
			client, err := app.Client()
			if err != nil {
				return err
			}

			var pages []scrapbox.PageSummary
			var result any
			if all {
				for p, err := range client.Pages.ListAll(cmd.Context(), project, &opts) {
					if err != nil {
						return err
					}
					pages = append(pages, p)
				}
				result = pages
			} else {
				list, err := client.Pages.List(cmd.Context(), project, &opts)
				if err != nil {
					return err
				}
				pages = list.Pages
				result = list
			}

			return app.Printer().Print(result, func(w io.Writer) error {
				tw := output.Table(w)
				fmt.Fprintln(tw, "UPDATED\tVIEWS\tLINKED\tTITLE")
				for _, p := range pages {
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
	f.IntVar(&opts.Limit, "limit", 100, fmt.Sprintf("number of pages per request (max %d)", scrapbox.MaxListLimit))
	f.IntVar(&opts.Skip, "skip", 0, "number of pages to skip")
	f.StringVar(&opts.Sort, "sort", "", "sort key: updated|created|accessed|linked|views|title")
	f.BoolVar(&all, "all", false, "fetch all pages by paging through the results")
	return cmd
}

func newPageTitlesCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "titles",
		Short: "Print all page titles in a project",
		Long:  "Print all page titles in a project. Lighter than 'page list --all' and suitable for completion or grep.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			project, err := app.Project()
			if err != nil {
				return err
			}
			client, err := app.Client()
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
		Long:  "Print links written in a page, including links to empty pages.\n\n" + pageArgHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			page, err := fetchPage(cmd, app, args[0])
			if err != nil {
				return err
			}
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
		Short: "Print related pages (1-hop / 2-hop links)",
		Long:  "Print pages in the related page list of a page.\n\n" + pageArgHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			h := scrapbox.Hops(hops)
			if err := h.Validate(); err != nil {
				return err
			}
			page, err := fetchPage(cmd, app, args[0])
			if err != nil {
				return err
			}
			related := page.RelatedPages.Links1Hop
			if h == scrapbox.TwoHop {
				related = slices.Concat(related, page.RelatedPages.Links2Hop)
			}
			return app.Printer().Print(related, func(w io.Writer) error {
				tw := output.Table(w)
				fmt.Fprintln(tw, "UPDATED\tLINKED\tTITLE")
				for _, p := range related {
					fmt.Fprintf(tw, "%s\t%d\t%s\n", output.Time(p.Updated.Time()), p.Linked, p.Title)
				}
				return tw.Flush()
			})
		},
	}
	cmd.Flags().IntVar(&hops, "hops", 1, "range of related pages: 1 or 2")
	return cmd
}

func newPageCodeCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "code <title|url> <filename>",
		Short:   "Print a code block (code:<filename>) in a page",
		Example: `  sb page code "Page Title" main.go > main.go`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return printRaw(cmd, app, args[0], args[1], (*scrapbox.PagesService).Code)
		},
	}
}

func newPageTableCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "table <title|url> <filename>",
		Short:   "Print a table (table:<filename>) in a page as CSV",
		Example: `  sb page table "Page Title" members > members.csv`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return printRaw(cmd, app, args[0], args[1], (*scrapbox.PagesService).Table)
		},
	}
}

// fetchPage は引数のページを取得する。
func fetchPage(cmd *cobra.Command, app *App, arg string) (*scrapbox.Page, error) {
	ref, err := app.PageRef(arg)
	if err != nil {
		return nil, err
	}
	client, err := app.Client()
	if err != nil {
		return nil, err
	}
	return client.Pages.Get(cmd.Context(), ref.Project, ref.Title)
}

// rawFetcher はページ内のファイル (コードブロック・テーブル) を生テキストで取得する関数である。
type rawFetcher func(s *scrapbox.PagesService, ctx context.Context, project, title, filename string) (string, error)

// printRaw はページ内のファイルを取得し、出力形式に関わらずそのまま出力する。
// コードや CSV はリダイレクトしてファイルに保存する用途を想定するため、JSON には包まない。
func printRaw(cmd *cobra.Command, app *App, arg, filename string, fetch rawFetcher) error {
	ref, err := app.PageRef(arg)
	if err != nil {
		return err
	}
	client, err := app.Client()
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
