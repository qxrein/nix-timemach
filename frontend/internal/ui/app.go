package ui

import (
	"fmt"
	"nix-timemach/internal/backend"
	"nix-timemach/internal/models"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateGenerations state = iota
	stateDiff
)

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Back   key.Binding
	Quit   key.Binding
	Reload key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.Back, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select},
		{k.Back, k.Reload, k.Quit},
	}
}

type App struct {
	keys        keyMap
	help        help.Model
	viewport    viewport.Model
	spinner     spinner.Model
	client      *backend.Client // Add this
	state       state
	generations []models.Generation
	cursor      int
	selected    *models.Generation
	diff        *models.GenerationDiff
	err         error
	ready       bool
	loading     bool
	width       int
	height      int
}

func NewApp(client *backend.Client) *App {
	keys := keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Reload: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reload"),
		),
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &App{
		keys:    keys,
		help:    help.New(),
		spinner: sp,
		client:  client, // Pass the client here
		state:   stateGenerations,
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Tick,
		a.fetchGenerations,
	)
}

func (a *App) fetchGenerations() tea.Msg {
	generations, err := a.client.GetGenerations()
	if err != nil {
		return errMsg{err}
	}
	return generationsMsg(generations)
}

func (a *App) fetchDiff(from, to string) tea.Msg {
	diff, err := a.client.GetDiff(from, to)
	if err != nil {
		return errMsg{err}
	}
	return diffMsg(diff)
}

type generationsMsg []models.Generation
type diffMsg models.GenerationDiff
type errMsg struct{ error }

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, a.keys.Quit):
			return a, tea.Quit

		case key.Matches(msg, a.keys.Back):
			if a.state == stateDiff {
				a.state = stateGenerations
				a.selected = nil
				a.diff = nil
			}

		case key.Matches(msg, a.keys.Up):
			if a.state == stateGenerations && a.cursor > 0 {
				a.cursor--
			}

		case key.Matches(msg, a.keys.Down):
			if a.state == stateGenerations && a.cursor < len(a.generations)-1 {
				a.cursor++
			}

		case key.Matches(msg, a.keys.Select):
			if a.state == stateGenerations {
				if a.selected == nil {
					a.selected = &a.generations[a.cursor]
					a.generations[a.cursor].Selected = true
				} else {
					a.state = stateDiff
					cmds = append(cmds, func() tea.Msg {
						return a.fetchDiff(a.selected.ID, a.generations[a.cursor].ID)
					})
				}
			}

		case key.Matches(msg, a.keys.Reload):
			a.loading = true
			cmds = append(cmds, a.fetchGenerations)
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.viewport = viewport.New(msg.Width, msg.Height-4) // Account for help menu
		a.help.Width = msg.Width
		a.ready = true

	case generationsMsg:
		a.loading = false
		a.generations = msg
		a.cursor = 0

	case diffMsg:
		a.loading = false
		a.diff = (*models.GenerationDiff)(&msg)

	case errMsg:
		a.err = msg.error
		a.loading = false

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a *App) View() string {
	if !a.ready {
		return "Initializing..."
	}

	if a.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'r' to retry or 'q' to quit", a.err)
	}

	var content string

	switch a.state {
	case stateGenerations:
		content = a.renderGenerations()
	case stateDiff:
		content = a.renderDiff()
	}

	if a.loading {
		content = fmt.Sprintf("%s Loading...", a.spinner.View())
	}

	return fmt.Sprintf("%s\n\n%s", content, a.help.View(a.keys))
}

func (a *App) renderGenerations() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("nix-timemach"))
	b.WriteString("\n\n")

	for i, gen := range a.generations {
		item := fmt.Sprintf("%s - %s", gen.Timestamp.Format("2006-01-02 15:04:05"), gen.Description)

		style := itemStyle
		if i == a.cursor {
			item = "> " + item
		} else {
			item = "  " + item
		}

		if gen.Selected {
			style = selectedItemStyle
		}

		b.WriteString(style.Render(item))
		b.WriteString("\n")
	}

	return b.String()
}

func (a *App) renderDiff() string {
	if a.diff == nil {
		return "Loading diff..."
	}

	var b strings.Builder

	fromTime := a.selected.Timestamp.Format("2006-01-02 15:04:05")
	toTime := a.generations[a.cursor].Timestamp.Format("2006-01-02 15:04:05")

	b.WriteString(titleStyle.Render(fmt.Sprintf("Diff: %s → %s", fromTime, toTime)))
	b.WriteString("\n\n")

	if len(a.diff.Added) > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(special).Render("Added:"))
		b.WriteString("\n")
		for _, item := range a.diff.Added {
			b.WriteString(fmt.Sprintf("  + %s\n", item))
		}
		b.WriteString("\n")
	}

	if len(a.diff.Removed) > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Removed:"))
		b.WriteString("\n")
		for _, item := range a.diff.Removed {
			b.WriteString(fmt.Sprintf("  - %s\n", item))
		}
		b.WriteString("\n")
	}

	if len(a.diff.Modified) > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Modified:"))
		b.WriteString("\n")
		for _, item := range a.diff.Modified {
			b.WriteString(fmt.Sprintf("  ~ %s\n", item))
		}
	}

	return b.String()
}
