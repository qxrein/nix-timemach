package ui

import "github.com/charmbracelet/lipgloss"

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			MarginLeft(2)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	selectedItemStyle = itemStyle.Copy().
				Foreground(special).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtle).
			PaddingLeft(4).
			PaddingBottom(1)
)
