package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/output"
)

func newProjectCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Read projects",
	}
	cmd.AddCommand(newProjectGetCmd(app), newProjectListCmd(app))
	return cmd
}

func newProjectGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get [name]",
		Short: "Print project information",
		Long:  "Print project information. If name is omitted, the current project is used.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) == 1 {
				name = args[0]
			} else {
				p, err := app.Project()
				if err != nil {
					return err
				}
				name = p
			}
			client, err := app.Client()
			if err != nil {
				return err
			}
			p, err := client.Projects.Get(cmd.Context(), name)
			if err != nil {
				return err
			}
			return app.Printer().Print(p, func(w io.Writer) error {
				tw := output.Table(w)
				fmt.Fprintf(tw, "Name:\t%s\n", p.Name)
				fmt.Fprintf(tw, "Display name:\t%s\n", p.DisplayName)
				fmt.Fprintf(tw, "Public:\t%t\n", p.PublicVisible)
				fmt.Fprintf(tw, "Member:\t%t\n", p.IsMember)
				fmt.Fprintf(tw, "Created:\t%s\n", output.Time(p.Created.Time()))
				fmt.Fprintf(tw, "Updated:\t%s\n", output.Time(p.Updated.Time()))
				return tw.Flush()
			})
		},
	}
}

func newProjectListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects you belong to (requires authentication)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.Client()
			if err != nil {
				return err
			}
			list, err := client.Projects.List(cmd.Context())
			if err != nil {
				return err
			}
			return app.Printer().Print(list, func(w io.Writer) error {
				tw := output.Table(w)
				fmt.Fprintln(tw, "NAME\tDISPLAY NAME\tPUBLIC\tUPDATED")
				for _, p := range list.Projects {
					fmt.Fprintf(tw, "%s\t%s\t%t\t%s\n", p.Name, p.DisplayName, p.PublicVisible, output.Time(p.Updated.Time()))
				}
				return tw.Flush()
			})
		},
	}
}
