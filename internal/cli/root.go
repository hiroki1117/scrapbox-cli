package cli

import (
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
	"github.com/hiroki1117/scrapbox-cli/internal/output"
)

// globalFlags are flags available to all subcommands.
type globalFlags struct {
	project string
	output  string
	baseURL string
	timeout time.Duration
	retries int
}

// NewRootCmd creates the root command.
func NewRootCmd(app *App) *cobra.Command {
	var gf globalFlags

	cmd := &cobra.Command{
		Use:           "sb",
		Short:         "Command line client for Scrapbox (Cosense)",
		Long:          rootLong,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return app.init(cmd, gf)
		},
	}
	cmd.SetIn(app.IO.In)
	cmd.SetOut(app.IO.Out)
	cmd.SetErr(app.IO.ErrOut)

	pf := cmd.PersistentFlags()
	pf.StringVarP(&gf.project, "project", "p", "", "project name (default: $"+config.EnvProject+" or config file)")
	pf.StringVarP(&gf.output, "output", "o", string(output.Text), "output format: text|json")
	pf.StringVar(&gf.baseURL, "base-url", "", "API base URL (default: https://scrapbox.io/api)")
	pf.DurationVar(&gf.timeout, "timeout", 0, "request timeout, e.g. 10s (default: 30s)")
	pf.IntVar(&gf.retries, "retries", 0, "max retries on 429/5xx/network errors (default: 3)")

	cmd.AddGroup(
		&cobra.Group{ID: groupRead, Title: "Read commands:"},
		&cobra.Group{ID: groupSearch, Title: "Search commands:"},
		&cobra.Group{ID: groupSettings, Title: "Auth and settings commands:"},
	)
	addToGroup(cmd, groupRead, newPageCmd(app), newContextCmd(app), newProjectCmd(app))
	addToGroup(cmd, groupSearch, newSearchCmd(app))
	addToGroup(cmd, groupSettings, newAuthCmd(app), newConfigCmd(app), newWhoamiCmd(app))
	cmd.AddCommand(newVersionCmd(app), newEnvironmentTopic())
	return cmd
}

// addToGroup adds commands to parent under the given group.
func addToGroup(parent *cobra.Command, groupID string, cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.GroupID = groupID
		parent.AddCommand(c)
	}
}

// init resolves the config from env and flags. Flags take precedence over env.
func (a *App) init(cmd *cobra.Command, gf globalFlags) error {
	cfg, err := config.Load(a.Getenv)
	if err != nil {
		return err
	}
	flags := cmd.Flags()
	if flags.Changed("project") {
		cfg.Project = gf.project
	}
	if flags.Changed("base-url") {
		cfg.BaseURL = gf.baseURL
	}
	if flags.Changed("timeout") {
		cfg.Timeout = gf.timeout
	}
	if flags.Changed("retries") {
		if gf.retries < 0 {
			return errors.New("--retries must not be negative")
		}
		cfg.MaxRetries = gf.retries
	}

	format, err := output.ParseFormat(gf.output)
	if err != nil {
		return err
	}
	a.cfg = cfg
	a.format = format
	return nil
}
