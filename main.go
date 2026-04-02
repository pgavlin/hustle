package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/urfave/cli/v3"

	logpkg "github.com/pgavlin/hustle/log"
	"github.com/pgavlin/hustle/ui"
)

func main() {
	cmd := &cli.Command{
		Name:      "hustle",
		Usage:     "A terminal viewer for slog JSON logs",
		ArgsUsage: "[logfile]",
		Action:    run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	var records []logpkg.LogRecord
	var skipped int
	var err error

	if cmd.NArg() > 0 {
		records, skipped, err = logpkg.Load(cmd.Args().First())
	} else {
		records, skipped, err = logpkg.LoadReader(os.Stdin)
	}
	if err != nil {
		return err
	}

	m := ui.New(records, skipped)
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}
