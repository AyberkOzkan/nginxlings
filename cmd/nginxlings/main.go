package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		fmt.Println("Hata:", err)
		os.Exit(1)
	}
}
