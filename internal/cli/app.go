// Package cli defines the subcommands of sb.
//
// Each subcommand gets the Client, Printer and config from App.
// To add a command, define newXxxCmd(app *App) *cobra.Command and AddCommand it to its parent.
package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/hiroki1117/scrapbox-cli/internal/config"
	"github.com/hiroki1117/scrapbox-cli/internal/output"
	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

// IOStreams are the command's input and outputs. Tests replace them with buffers.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// BuildInfo is the version information embedded at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// App holds the dependencies shared by commands.
// The config and output format are resolved in the root PersistentPreRunE; Printer and Client are created on first use.
type App struct {
	IO        IOStreams
	Getenv    func(string) string
	BuildInfo BuildInfo
	// If HTTPClient is nil, the scrapbox package default is used.
	HTTPClient *http.Client

	cfg     config.Config
	format  output.Format
	printer *output.Printer
	client  *scrapbox.Client
}

// Printer returns the Printer for the output format.
func (a *App) Printer() *output.Printer {
	if a.printer == nil {
		a.printer = output.NewPrinter(a.IO.Out, a.format)
	}
	return a.printer
}

// Client returns the API client using the configured credentials, created on first call.
func (a *App) Client() (*scrapbox.Client, error) {
	if a.client != nil {
		return a.client, nil
	}
	c, err := a.NewClient(a.cfg.Credentials.Authenticator())
	if err != nil {
		return nil, err
	}
	a.client = c
	return c, nil
}

// NewClient creates an API client with the given authenticator.
// auth login uses it to verify a credential before saving it.
func (a *App) NewClient(auth scrapbox.Authenticator) (*scrapbox.Client, error) {
	// Copy the injected *http.Client before setting the timeout so it is not modified.
	var hc http.Client
	if a.HTTPClient != nil {
		hc = *a.HTTPClient
	}
	hc.Timeout = a.cfg.Timeout

	ua := "scrapbox-cli"
	if a.BuildInfo.Version != "" {
		ua += "/" + a.BuildInfo.Version
	}
	retry := scrapbox.DefaultRetryPolicy
	retry.MaxRetries = a.cfg.MaxRetries
	return scrapbox.NewClient(
		scrapbox.WithBaseURL(a.cfg.BaseURL),
		scrapbox.WithRetryPolicy(retry),
		scrapbox.WithHTTPClient(&hc),
		scrapbox.WithAuthenticator(auth),
		scrapbox.WithUserAgent(ua),
	)
}

// errNoProject is returned when no project is specified.
var errNoProject = fmt.Errorf("project is not specified: use --project, set %s, or run 'sb config set project <name>'", config.EnvProject)

// Project returns the target project name, or an error if it is not set.
func (a *App) Project() (string, error) {
	if a.cfg.Project == "" {
		return "", errNoProject
	}
	return a.cfg.Project, nil
}

// PageRef converts a command argument to a page reference.
// A page URL uses its own project; a title uses the configured project.
func (a *App) PageRef(arg string) (scrapbox.PageRef, error) {
	if scrapbox.IsURL(arg) {
		return scrapbox.ParsePageURL(arg)
	}
	project, err := a.Project()
	if err != nil {
		return scrapbox.PageRef{}, err
	}
	return scrapbox.PageRef{Project: project, Title: arg}, nil
}

// projectClient returns the target project and the API client.
func (a *App) projectClient() (string, *scrapbox.Client, error) {
	project, err := a.Project()
	if err != nil {
		return "", nil, err
	}
	client, err := a.Client()
	return project, client, err
}

// pageClient resolves a page argument and returns it with the API client.
func (a *App) pageClient(arg string) (scrapbox.PageRef, *scrapbox.Client, error) {
	ref, err := a.PageRef(arg)
	if err != nil {
		return ref, nil, err
	}
	client, err := a.Client()
	return ref, client, err
}

// ErrorHint returns a hint for fixing err, or an empty string if there is none.
func ErrorHint(err error) string {
	switch {
	case errors.Is(err, scrapbox.ErrUnauthorized), errors.Is(err, scrapbox.ErrForbidden):
		return fmt.Sprintf("hint: this request requires authentication (private projects, Smart Context, etc.). run 'sb auth login', or set %s, %s or %s.",
			config.EnvPersonalToken, config.EnvServiceAccountKey, config.EnvSessionID)
	case errors.Is(err, scrapbox.ErrNotFound):
		return "hint: check the project name and page title."
	}
	return ""
}
