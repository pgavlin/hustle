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
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: hustle <logfile>\n")
		os.Exit(1)
	}

	path := os.Args[1]
	records, skipped, err := logpkg.Load(path)
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
