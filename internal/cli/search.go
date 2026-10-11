package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

func newSearchCmd(app *App) *cobra.Command {
	var opts scrapbox.SearchOptions
	cmd := &cobra.Command{
		Use:   "search <query>...",
		Short: "Full-text search pages in a project",
		Long: `Full-text search pages in a project.

Multiple arguments are joined with spaces (AND search). Prefix a word with "-" to exclude it.`,
		Example: `  sb search golang cli
  sb search -- golang -python
  sb search golang -o json | jq -r '.pages[].title'`,
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
	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "maximum number of pages (default: server default)")
	return cmd
}
