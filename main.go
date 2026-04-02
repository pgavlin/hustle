package main

import (
	"context"
	"fmt"
	"os"
	"strings"

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
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "auto",
				Usage:   fmt.Sprintf("Log format: auto, %s", strings.Join(logpkg.FormatNames(), ", ")),
			},
		},
		Action: run,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	var format logpkg.Format
	if name := cmd.String("format"); name != "auto" {
		format = logpkg.FormatByName(name)
		if format == nil {
			return fmt.Errorf("unknown format %q; valid formats: %s",
				name, strings.Join(logpkg.FormatNames(), ", "))
		}
	}

	var records []logpkg.LogRecord
	var skipped int
	var detected logpkg.Format
	var err error

	if cmd.NArg() > 0 {
		records, skipped, detected, err = logpkg.Load(cmd.Args().First(), format)
	} else {
		records, skipped, detected, err = logpkg.LoadReader(os.Stdin, format)
	}
	if err != nil {
		return err
	}

	m := ui.New(records, skipped, detected.Name())
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}
