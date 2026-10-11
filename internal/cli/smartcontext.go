package cli

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

func newContextCmd(app *App) *cobra.Command {
	var hops int
	cmd := &cobra.Command{
		Use:   "context <title|url>",
		Short: "Export a page and its related pages as AI-ready text (Smart Context)",
		Long: `Export a page and its related pages as a single text formatted for LLMs.
This is the same as "Export for AI" (Smart Context) in the Scrapbox page menu.

Note that 2-hop export can exceed 1MB in large projects.

` + pageArgHelp,
		Example: `  sb context "Page Title" | pbcopy
  sb context "Page Title" --hops 2 > context.txt`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			h := scrapbox.Hops(hops)
			if err := h.Validate(); err != nil {
				return err
			}
			ref, err := app.PageRef(args[0])
			if err != nil {
				return err
			}
			client, err := app.Client()
			if err != nil {
				return err
			}
			text, err := client.SmartContext.Export(cmd.Context(), ref.Project, ref.Title, h)
			if err != nil {
				return err
			}
			result := struct {
				Project string `json:"project"`
				Title   string `json:"title"`
				Hops    int    `json:"hops"`
				Text    string `json:"text"`
			}{ref.Project, ref.Title, hops, text}
			return app.Printer().Print(result, func(w io.Writer) error {
				_, err := io.WriteString(w, text)
				return err
			})
		},
	}
	cmd.Flags().IntVar(&hops, "hops", 1, "range of related pages: 1 or 2")
	return cmd
}
