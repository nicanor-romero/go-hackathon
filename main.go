package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type mainMenuState int

const (
	mainMenuStateChoosing mainMenuState = iota
)

type mainMenuModel struct {
	cursor int
	choice string
}

func (m mainMenuModel) Init() tea.Cmd {
	return nil
}

func (m mainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < 1 {
				m.cursor++
			}
		case "enter":
			// Store choice and quit
			if m.cursor == 0 {
				m.choice = "prometheus"
			} else {
				m.choice = "jaeger"
			}
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m mainMenuModel) View() tea.View {
	var b strings.Builder

	b.WriteString("\n")

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	b.WriteString(titleStyle.Render("🔧 Monitoring & Tracing Tools"))
	b.WriteString("\n\n")

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	regularStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	options := []struct {
		icon  string
		label string
		desc  string
	}{
		{"📊", "Prometheus Metrics", "Browse metrics, stats, and PromQL queries"},
		{"🔍", "Jaeger Tracing", "Browse services and dependencies"},
	}

	for i, opt := range options {
		if i == m.cursor {
			b.WriteString(selectedStyle.Render(fmt.Sprintf(" ▶ %s %s", opt.icon, opt.label)))
			b.WriteString("\n")
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
			b.WriteString(helpStyle.Render(fmt.Sprintf("     %s", opt.desc)))
		} else {
			b.WriteString(regularStyle.Render(fmt.Sprintf("   %s %s", opt.icon, opt.label)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	b.WriteString(helpStyle.Render("↑/k up  •  ↓/j down  •  ⏎ Enter select  •  q quit"))

	return tea.NewView(b.String())
}

func main() {
	// Show main menu
	menu := mainMenuModel{}
	p := tea.NewProgram(
		menu,
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get choice from final model
	result := finalModel.(mainMenuModel)

	// Run selected tool
	switch result.choice {
	case "prometheus":
		runPromTUI()
	case "jaeger":
		runJaegerTUI()
	default:
		// User quit without selecting
		os.Exit(0)
	}
}
