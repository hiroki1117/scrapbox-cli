package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
	"github.com/hiroki1117/scrapbox-cli/internal/output"
)

func newAuthCmd(app *App) *cobra.Command {
	cmd := newGroupCmd("auth", "Manage authentication for private projects", `Manage authentication for private projects and Smart Context.

Credentials are saved in the config file (permission 0600).
Environment variables take precedence over it (see 'sb help environment').

Auth types:
  pat              Personal Access Token (recommended). Issue at https://scrapbox.io/settings/personal-access-tokens
  service-account  Service Account access key (Business plan, single project)
  sid              connect.sid cookie value of a logged-in browser`)
	cmd.AddCommand(newAuthLoginCmd(app), newAuthStatusCmd(app), newAuthLogoutCmd(app))
	return cmd
}

func newAuthLoginCmd(app *App) *cobra.Command {
	var (
		authType  string
		withToken bool
		noVerify  bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save credentials to the config file",
		Long: `Save credentials to the config file.

The credential is read from a hidden prompt, or from standard input with --with-token.
It is never accepted as a command line argument to keep it out of shell history.`,
		Example: `  sb auth login
  sb auth login --type sid
  pbpaste | sb auth login --with-token`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			typ, err := config.ParseAuthType(authType)
			if err != nil {
				return err
			}
			secret, err := readSecret(app, withToken, fmt.Sprintf("Paste your credential (%s): ", typ))
			if err != nil {
				return err
			}
			creds := config.NewCredentials(typ, secret)

			if !noVerify {
				if err := verifyCredentials(cmd, app, typ, creds); err != nil {
					return err
				}
			}
			if err := config.UpdateFile(app.cfg.FilePath, func(f *config.File) { f.Credentials = creds }); err != nil {
				return err
			}
			fmt.Fprintf(app.IO.ErrOut, "Saved %s credential to %s\n", typ, app.cfg.FilePath)
			if app.cfg.CredentialSource == config.SourceEnv {
				fmt.Fprintln(app.IO.ErrOut, "warning: credentials in environment variables take precedence over the saved credential.")
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&authType, "type", string(config.AuthPersonalAccessToken), "auth type: pat|service-account|sid")
	f.BoolVar(&withToken, "with-token", false, "read the credential from standard input")
	f.BoolVar(&noVerify, "no-verify", false, "save without verifying the credential")
	return cmd
}

// readSecret は認証情報を読み込む。端末ならエコーなしのプロンプトで、それ以外は標準入力の 1 行目を読む。
func readSecret(app *App, fromStdin bool, prompt string) (string, error) {
	var secret string
	if f, ok := app.IO.In.(*os.File); ok && !fromStdin && term.IsTerminal(int(f.Fd())) {
		fmt.Fprint(app.IO.ErrOut, prompt)
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(app.IO.ErrOut)
		if err != nil {
			return "", fmt.Errorf("read credential: %w", err)
		}
		secret = string(b)
	} else {
		line, err := bufio.NewReader(app.IO.In).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read credential: %w", err)
		}
		secret = line
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", errors.New("credential is empty")
	}
	return secret, nil
}

// verifyCredentials は認証情報で API にアクセスできるかを確認する。
func verifyCredentials(cmd *cobra.Command, app *App, typ config.AuthType, creds config.Credentials) error {
	client, err := app.NewClient(creds.Authenticator())
	if err != nil {
		return err
	}
	// Service Account はユーザーに紐づかないため、プロジェクトへのアクセスで確認する。
	if typ == config.AuthServiceAccount {
		project, err := app.Project()
		if err != nil {
			return fmt.Errorf("service account is verified against a project: %w (or use --no-verify)", err)
		}
		if _, err := client.Pages.List(cmd.Context(), project, nil); err != nil {
			return fmt.Errorf("verify credential: %w", err)
		}
		fmt.Fprintf(app.IO.ErrOut, "Verified access to project %q\n", project)
		return nil
	}
	u, err := client.Users.Me(cmd.Context())
	if err != nil {
		return fmt.Errorf("verify credential: %w", err)
	}
	if u.IsGuest {
		return errors.New("verify credential: authentication failed (the credential may be invalid or expired)")
	}
	fmt.Fprintf(app.IO.ErrOut, "Logged in as %s\n", u.Name)
	return nil
}

// authStatus は auth status の出力である。認証情報そのものは含めない。
type authStatus struct {
	ConfigFile string        `json:"configFile"`
	Source     config.Source `json:"source"`
	Type       string        `json:"type"`
	Credential string        `json:"credential"`
	User       string        `json:"user,omitempty"`
}

func newAuthStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current authentication state",
		Long: `Show which credential is used, where it comes from (env or file), and the logged-in user.
The credential itself is masked.

JSON output: {configFile, source, type, credential, user}`,
		Example: `  sb auth status
  sb auth status -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			typ, secret := app.cfg.Credentials.Active()
			st := authStatus{
				ConfigFile: app.cfg.FilePath,
				Source:     app.cfg.CredentialSource,
				Type:       string(typ),
				Credential: mask(secret),
			}
			if typ == config.AuthPersonalAccessToken || typ == config.AuthSessionID {
				client, err := app.Client()
				if err != nil {
					return err
				}
				u, err := client.Users.Me(cmd.Context())
				if err != nil {
					return err
				}
				if !u.IsGuest {
					st.User = u.Name
				}
			}
			return app.Printer().Print(st, func(w io.Writer) error {
				tw := output.Table(w)
				fmt.Fprintf(tw, "Config file:\t%s\n", st.ConfigFile)
				if typ == "" {
					fmt.Fprintln(tw, "Auth:\tnot configured (run 'sb auth login')")
					return tw.Flush()
				}
				fmt.Fprintf(tw, "Auth:\t%s (from %s)\n", st.Type, st.Source)
				fmt.Fprintf(tw, "Credential:\t%s\n", st.Credential)
				switch {
				case st.User != "":
					fmt.Fprintf(tw, "User:\t%s\n", st.User)
				case typ != config.AuthServiceAccount:
					fmt.Fprintln(tw, "User:\tguest (the credential may be invalid or expired)")
				}
				return tw.Flush()
			})
		},
	}
}

func newAuthLogoutCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove credentials from the config file",
		Long:  "Remove credentials from the config file. Credentials in environment variables are not affected.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			f, err := config.ReadFile(app.cfg.FilePath)
			if err != nil {
				return err
			}
			if f.Credentials.IsZero() {
				fmt.Fprintln(app.IO.ErrOut, "No credentials saved in the config file")
			} else {
				f.Credentials = config.Credentials{}
				if err := config.WriteFile(app.cfg.FilePath, f); err != nil {
					return err
				}
				fmt.Fprintf(app.IO.ErrOut, "Removed credentials from %s\n", app.cfg.FilePath)
			}
			if app.cfg.CredentialSource == config.SourceEnv {
				fmt.Fprintln(app.IO.ErrOut, "note: credentials in environment variables are still used. unset them to log out completely.")
			}
			return nil
		},
	}
}

// mask は認証情報の先頭 4 文字だけを残して伏せる。
func mask(s string) string {
	if s == "" {
		return ""
	}
	const visible = 4
	if len(s) <= visible*2 {
		return strings.Repeat("*", len(s))
	}
	return s[:visible] + strings.Repeat("*", 8)
}
