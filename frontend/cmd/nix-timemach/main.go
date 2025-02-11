package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"nix-timemach/internal/backend"
	"nix-timemach/internal/ui"
)

func main() {
	client := backend.NewClient("../backend/target/release/nix-timemach-backend")
	app := ui.NewApp(client)
	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
