package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
)

// configKey は config コマンドで読み書きできる設定項目である。
// 項目を追加する場合は configKeys に登録する。認証情報は auth コマンドで扱うため登録しない。
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

func configKeyNames() []string {
	names := make([]string, 0, len(configKeys))
	for k := range configKeys {
		names = append(names, k)
	}
	slices.Sort(names)
	return names
}

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
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the config file",
		Long:  "Manage the config file. Environment variables and flags take precedence over the config file.\n\nKeys:" + keysHelp.String(),
	}
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
