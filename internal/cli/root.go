package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
	"github.com/hiroki1117/scrapbox-cli/internal/output"
)

// globalFlags はすべてのサブコマンドで使えるフラグである。
type globalFlags struct {
	project string
	output  string
	baseURL string
	timeout time.Duration
}

// NewRootCmd はルートコマンドを生成する。
func NewRootCmd(app *App) *cobra.Command {
	var gf globalFlags

	cmd := &cobra.Command{
		Use:   "sb",
		Short: "Command line client for Scrapbox (Cosense)",
		Long: `sb is a command line client for Scrapbox (Cosense).

Environment variables:
  ` + config.EnvProject + `         default project name (or 'sb config set project <name>')
  ` + config.EnvPersonalToken + `                  Personal Access Token (recommended for private projects)
  ` + config.EnvServiceAccountKey + `  Service Account access key (Business plan)
  ` + config.EnvSessionID + `                  connect.sid cookie value
  ` + config.EnvBaseURL + `             API base URL (default: https://scrapbox.io/api)
  ` + config.EnvTimeout + `             request timeout (e.g. 30s)
  ` + config.EnvConfigFile + `                    config file path (default: ~/.config/sb/config.json)

Precedence: flags > environment variables > config file.
Run 'sb auth login' to save credentials for private projects.`,
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
	pf.StringVarP(&gf.project, "project", "p", "", "project name (env: "+config.EnvProject+")")
	pf.StringVarP(&gf.output, "output", "o", string(output.Text), "output format: text|json")
	pf.StringVar(&gf.baseURL, "base-url", "", "API base URL (env: "+config.EnvBaseURL+")")
	pf.DurationVar(&gf.timeout, "timeout", 0, "request timeout (env: "+config.EnvTimeout+")")

	cmd.AddCommand(
		newPageCmd(app),
		newSearchCmd(app),
		newContextCmd(app),
		newProjectCmd(app),
		newAuthCmd(app),
		newConfigCmd(app),
		newWhoamiCmd(app),
		newVersionCmd(app),
	)
	return cmd
}

// init は環境変数とフラグから設定を解決する。フラグは環境変数より優先する。
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

	format, err := output.ParseFormat(gf.output)
	if err != nil {
		return err
	}
	a.cfg = cfg
	a.format = format
	return nil
}
