package cli

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
)

// configKey is a setting that the config command can read and write.
// Register new settings in configKeys. Credentials are handled by the auth command, not here.
type configKey struct {
	help string
	get  func(f *config.File) string
	set  func(f *config.File, v string)
}

var configKeys = map[string]configKey{
	"project": {
		help: "default project name",
		get:  func(f *config.File) string { return f.Project },
		set:  func(f *config.File, v string) { f.Project = v },
	},
}

func configKeyNames() []string { return slices.Sorted(maps.Keys(configKeys)) }

func lookupConfigKey(name string) (configKey, error) {
	k, ok := configKeys[name]
	if !ok {
		return configKey{}, fmt.Errorf("unknown config key %q (available: %s)", name, strings.Join(configKeyNames(), ", "))
	}
	return k, nil
}

func newConfigCmd(app *App) *cobra.Command {
	var keysHelp strings.Builder
	for _, name := range configKeyNames() {
		fmt.Fprintf(&keysHelp, "\n  %-10s %s", name, configKeys[name].help)
	}
	cmd := newGroupCmd("config", "Manage the config file (default project, etc.)",
		"Manage settings saved in the config file.\nFlags and environment variables take precedence over the config file.\nCredentials are managed by 'sb auth'.\n\nKeys:"+keysHelp.String())
	cmd.AddCommand(
		&cobra.Command{
			Use:     "get <key>",
			Short:   "Print a value in the config file",
			Args:    cobra.ExactArgs(1),
			Example: "  sb config get project",
			RunE: func(_ *cobra.Command, args []string) error {
				k, err := lookupConfigKey(args[0])
				if err != nil {
					return err
				}
				f, err := config.ReadFile(app.cfg.FilePath)
				if err != nil {
					return err
				}
				v := k.get(&f)
				return app.Printer().Print(v, func(w io.Writer) error {
					_, err := fmt.Fprintln(w, v)
					return err
				})
			},
		},
		&cobra.Command{
			Use:     "set <key> <value>",
			Short:   "Set a value in the config file",
			Args:    cobra.ExactArgs(2),
			Example: "  sb config set project my-project",
			RunE: func(_ *cobra.Command, args []string) error {
				k, err := lookupConfigKey(args[0])
				if err != nil {
					return err
				}
				return config.UpdateFile(app.cfg.FilePath, func(f *config.File) { k.set(f, args[1]) })
			},
		},
		&cobra.Command{
			Use:   "unset <key>",
			Short: "Remove a value from the config file",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				k, err := lookupConfigKey(args[0])
				if err != nil {
					return err
				}
				return config.UpdateFile(app.cfg.FilePath, func(f *config.File) { k.set(f, "") })
			},
		},
		&cobra.Command{
			Use:   "path",
			Short: "Print the config file path",
			Args:  cobra.NoArgs,
			RunE: func(_ *cobra.Command, _ []string) error {
				_, err := fmt.Fprintln(app.IO.Out, app.cfg.FilePath)
				return err
			},
		},
	)
	return cmd
}
