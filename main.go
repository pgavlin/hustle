package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
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
			&cli.StringFlag{
				Name:  "cpuprofile",
				Usage: "Write CPU profile to this path",
			},
			&cli.StringFlag{
				Name:  "memprofile",
				Usage: "Write memory profile to this path on exit",
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
	if cpuProfile := cmd.String("cpuprofile"); cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return fmt.Errorf("creating CPU profile: %w", err)
		}
		defer f.Close()

		if err = pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("starting CPU profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	var format logpkg.Format
	if name := cmd.String("format"); name != "auto" {
		format = logpkg.FormatByName(name)
		if format == nil {
			return fmt.Errorf("unknown format %q; valid formats: %s",
				name, strings.Join(logpkg.FormatNames(), ", "))
		}
	}

	var lf *logpkg.LogFile
	var err error

	if cmd.NArg() > 0 {
		lf, err = logpkg.Load(cmd.Args().First(), format)
	} else {
		lf, err = logpkg.LoadReader(os.Stdin, format)
	}
	if err != nil {
		return err
	}
	defer lf.Close()

	m := ui.New(lf.Records, lf.Skipped, lf.Format.Name())
	p := tea.NewProgram(m)
	_, err = p.Run()

	if memProfile := cmd.String("memprofile"); memProfile != "" {
		runtime.GC()
		f, ferr := os.Create(memProfile)
		if ferr != nil {
			return fmt.Errorf("creating memory profile: %w", ferr)
		}
		defer f.Close()
		if ferr = pprof.WriteHeapProfile(f); ferr != nil {
			return fmt.Errorf("writing memory profile: %w", ferr)
		}
	}

	return err
}
