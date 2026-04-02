// main.go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	logpkg "github.com/pgavlin/hustle/log"
	"github.com/pgavlin/hustle/ui"
)

func main() {
	var records []logpkg.LogRecord
	var skipped int
	var err error

	if len(os.Args) >= 2 {
		records, skipped, err = logpkg.Load(os.Args[1])
	} else {
		records, skipped, err = logpkg.LoadReader(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	m := ui.New(records, skipped)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
