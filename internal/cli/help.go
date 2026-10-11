package cli

import (
	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
)

// Command group IDs, used to group commands by purpose in the root help.
const (
	groupRead     = "read"
	groupSearch   = "search"
	groupSettings = "settings"
)

const rootLong = `sb reads Scrapbox (Cosense) pages from the command line.

Quick start:
  sb config set project help-jp     set the default project
  sb page get "Smart Context"       print a page
  sb search text keyword            full-text search
  sb auth login                     authenticate for private projects

Conventions:
  <title|url>  a page title in the current project, or a page URL
               (https://scrapbox.io/<project>/<title>; its project is used)
  project      resolved from --project, ` + config.EnvProject + `, then 'sb config set project'
  -o json      machine-readable output; recommended for scripts and AI agents
  stdout       results only; counts, hints and errors are written to stderr
  exit code    0 on success, 1 on any error

Run 'sb help environment' for environment variables and credentials.`

// pageArgHelp is appended to the Long help of commands that take a page argument.
const pageArgHelp = `<title|url> is a page title in the current project, or a page URL like
https://scrapbox.io/<project>/<title>.`

const environmentLong = `Environment variables (flags take precedence over environment variables,
which take precedence over the config file):

  ` + config.EnvProject + `         default project name
  ` + config.EnvPersonalToken + `                  Personal Access Token (recommended for private projects)
  ` + config.EnvServiceAccountKey + `  Service Account access key (Business plan)
  ` + config.EnvSessionID + `                  connect.sid cookie value
  ` + config.EnvBaseURL + `             API base URL (default: https://scrapbox.io/api)
  ` + config.EnvTimeout + `             request timeout, e.g. 30s (default: 30s)
  ` + config.EnvMaxRetries + `         max retries on 429/5xx/network errors (default: 3)
  ` + config.EnvConfigFile + `                    config file path (default: ~/.config/sb/config.json)

If any credential variable is set, credentials in the config file are ignored.
If several credentials are set, PAT > Service Account > connect.sid is used.`

// newEnvironmentTopic is the help topic shown by 'sb help environment'.
// cobra lists a command without Run under "Additional help topics".
func newEnvironmentTopic() *cobra.Command {
	return &cobra.Command{
		Use:   "environment",
		Short: "Environment variables and configuration precedence",
		Long:  environmentLong,
	}
}

// newGroupCmd creates a command that only groups subcommands.
// It prints help without arguments and fails on unknown subcommands (by default cobra prints help and exits 0).
func newGroupCmd(use, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
}
