package main

import "github.com/charmbracelet/lipgloss"

// Styles: TUI'de kullandığımız tüm görsel stiller tek yerde toplu.
type Styles struct {
	Title                 lipgloss.Style
	ListTitle             lipgloss.Style
	SelectedItem          lipgloss.Style
	Item                  lipgloss.Style
	CompletedItem         lipgloss.Style
	SelectedCompletedItem lipgloss.Style
	Box                   lipgloss.Style
	StatusBar             lipgloss.Style
	Help                  lipgloss.Style
}

// DefaultStyles: koyu tema terminal için varsayılan görünüm.
func DefaultStyles() Styles {
	var s Styles

	s.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")) // mavimsi başlık

	s.ListTitle = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	s.SelectedItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")). // mor ton
		Bold(true)

	s.Item = lipgloss.NewStyle()

	s.CompletedItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")) // yeşil

	s.SelectedCompletedItem = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // parlak yeşil
		Bold(true)

	s.Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("60")).
		Padding(0, 1)

	s.StatusBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")). // açık gri yazı
		Padding(0, 1)

	s.Help = lipgloss.NewStyle().
		Faint(true).
		Padding(0, 1)

	return s
}
