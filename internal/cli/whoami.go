package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newWhoamiCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Print the authenticated user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := app.Client()
			if err != nil {
				return err
			}
			u, err := client.Users.Me(cmd.Context())
			if err != nil {
				return err
			}
			return app.Printer().Print(u, func(w io.Writer) error {
				if u.IsGuest {
					_, err := fmt.Fprintln(w, "not logged in (guest)")
					return err
				}
				_, err := fmt.Fprintf(w, "%s (%s)\n", u.Name, u.DisplayName)
				return err
			})
		},
	}
}

func newVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			bi := app.BuildInfo
			return app.Printer().Print(bi, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "sb %s (commit: %s, built: %s)\n", bi.Version, bi.Commit, bi.Date)
				return err
			})
		},
	}
}
