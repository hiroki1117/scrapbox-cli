// Command sb は Scrapbox (Cosense) の CLI である。
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/hiroki1117/scrapbox-cli/internal/cli"
)

// ビルド時に -ldflags "-X main.version=..." で埋め込む。
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	app := &cli.App{
		IO:        cli.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
		Getenv:    os.Getenv,
		BuildInfo: cli.BuildInfo{Version: version, Commit: commit, Date: date},
	}
	if err := cli.NewRootCmd(app).ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		if hint := cli.ErrorHint(err); hint != "" {
			fmt.Fprintln(os.Stderr, hint)
		}
		return 1
	}
	return 0
}
