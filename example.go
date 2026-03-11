package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Example TUI demonstrating menu/submenu navigation with Bubble Tea v2
// Uses the same styling as prom-metrics for consistency

// Styles
var (
	exampleTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true).
				MarginBottom(1)

	exampleSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Bold(true)

	exampleRegularStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	exampleBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			MarginTop(1)

	exampleLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")).
				Bold(true)

	exampleHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
)

// States
type exampleState int

const (
	exampleStateMainMenu exampleState = iota
	exampleStateSubmenu
	exampleStateLoading
	exampleStateDetail
)

// Example model
type exampleModel struct {
	state         exampleState
	mainCursor    int
	submenuCursor int
	selectedItem  string
	spinner       spinner.Model
	frame         int
}

type exampleTickMsg time.Time

func exampleTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return exampleTickMsg(t)
	})
}

func initialExampleModel() exampleModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return exampleModel{
		state:   exampleStateMainMenu,
		spinner: s,
	}
}

func (m exampleModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		exampleTick(),
	)
}

func (m exampleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.state == exampleStateMainMenu {
				return m, tea.Quit
			}
		case "esc":
			if m.state == exampleStateSubmenu {
				m.state = exampleStateMainMenu
				m.submenuCursor = 0
			} else if m.state == exampleStateDetail {
				m.state = exampleStateSubmenu
			}
		case "enter":
			if m.state == exampleStateMainMenu {
				// Store selected item and show submenu
				items := []string{"Dashboard", "Settings", "Reports", "Help"}
				m.selectedItem = items[m.mainCursor]
				m.state = exampleStateSubmenu
				m.submenuCursor = 0
			} else if m.state == exampleStateSubmenu {
				// Show loading then detail
				m.state = exampleStateLoading
				return m, func() tea.Msg {
					time.Sleep(1 * time.Second)
					return "loaded"
				}
			}
		case "up", "k":
			if m.state == exampleStateMainMenu && m.mainCursor > 0 {
				m.mainCursor--
			} else if m.state == exampleStateSubmenu && m.submenuCursor > 0 {
				m.submenuCursor--
			}
		case "down", "j":
			if m.state == exampleStateMainMenu && m.mainCursor < 3 {
				m.mainCursor++
			} else if m.state == exampleStateSubmenu && m.submenuCursor < 2 {
				m.submenuCursor++
			}
		}

	case string:
		if msg == "loaded" {
			m.state = exampleStateDetail
		}

	case exampleTickMsg:
		m.frame++
		if m.state == exampleStateMainMenu || m.state == exampleStateLoading || m.state == exampleStateSubmenu {
			return m, exampleTick()
		}
	}

	// Update spinner if loading
	if m.state == exampleStateLoading {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m exampleModel) View() tea.View {
	var content string

	switch m.state {
	case exampleStateMainMenu:
		var b strings.Builder
		b.WriteString("\n")

		// Animated rainbow title
		titleColors := []string{"196", "202", "208", "214", "220", "226", "154", "118", "82", "46"}
		title := "🎯 Example Menu TUI"
		var animatedTitle strings.Builder
		for i, char := range title {
			colorIdx := (m.frame + i) % len(titleColors)
			charStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(titleColors[colorIdx])).
				Bold(true)
			animatedTitle.WriteString(charStyle.Render(string(char)))
		}
		b.WriteString(animatedTitle.String())
		b.WriteString("\n\n")

		// Menu items
		items := []struct {
			icon  string
			label string
		}{
			{"📊", "Dashboard"},
			{"⚙️", "Settings"},
			{"📈", "Reports"},
			{"❓", "Help"},
		}

		for i, item := range items {
			if i == m.mainCursor {
				b.WriteString(exampleSelectedStyle.Render(fmt.Sprintf(" ▶ %s %s", item.icon, item.label)))
			} else {
				b.WriteString(exampleRegularStyle.Render(fmt.Sprintf("   %s %s", item.icon, item.label)))
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(exampleHelpStyle.Render("↑/k up  •  ↓/j down  •  ⏎ Enter select  •  q quit"))

		content = b.String()

	case exampleStateSubmenu:
		var b strings.Builder
		b.WriteString("\n")
		b.WriteString(exampleTitleStyle.Render(fmt.Sprintf("📋 %s", m.selectedItem)))
		b.WriteString("\n\n")

		b.WriteString(exampleLabelStyle.Render("Choose an action:"))
		b.WriteString("\n\n")

		options := []struct {
			icon string
			name string
			desc string
		}{
			{"👁️", "View Details", "See detailed information"},
			{"✏️", "Edit", "Modify settings or data"},
			{"🗑️", "Delete", "Remove this item"},
		}

		for i, opt := range options {
			if i == m.submenuCursor {
				b.WriteString(exampleSelectedStyle.Render(fmt.Sprintf(" ▶ %s %s", opt.icon, opt.name)))
				b.WriteString("\n")
				b.WriteString(exampleHelpStyle.Render(fmt.Sprintf("     %s", opt.desc)))
			} else {
				b.WriteString(exampleRegularStyle.Render(fmt.Sprintf("   %s %s", opt.icon, opt.name)))
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(exampleHelpStyle.Render("↑/k up  •  ↓/j down  •  ⏎ Enter select  •  Esc back"))

		content = b.String()

	case exampleStateLoading:
		var b strings.Builder
		b.WriteString("\n\n")

		titleColors := []string{"220", "221", "227", "228", "227", "221"}
		colorIdx := m.frame % len(titleColors)
		animatedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(titleColors[colorIdx])).
			Bold(true)
		b.WriteString(animatedStyle.Render(fmt.Sprintf("⏳ Loading %s", m.selectedItem)))
		b.WriteString("\n\n")

		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(exampleRegularStyle.Render("Please wait..."))

		content = b.String()

	case exampleStateDetail:
		var b strings.Builder
		b.WriteString("\n")
		b.WriteString(exampleTitleStyle.Render(fmt.Sprintf("📄 %s - Details", m.selectedItem)))
		b.WriteString("\n\n")

		var detailContent strings.Builder
		detailContent.WriteString(exampleLabelStyle.Render("Information:"))
		detailContent.WriteString("\n\n")

		actions := []string{"View Details", "Edit", "Delete"}
		detailContent.WriteString(exampleRegularStyle.Render(fmt.Sprintf("  Item: %s\n", m.selectedItem)))
		detailContent.WriteString(exampleRegularStyle.Render(fmt.Sprintf("  Action: %s\n", actions[m.submenuCursor])))
		detailContent.WriteString(exampleRegularStyle.Render("  Status: Active\n"))
		detailContent.WriteString(exampleRegularStyle.Render("  Last Modified: 2026-03-11\n"))

		b.WriteString(exampleBoxStyle.Render(detailContent.String()))

		b.WriteString("\n\n")
		b.WriteString(exampleHelpStyle.Render("⬅️  Esc back to menu  •  q quit"))

		content = b.String()
	}

	return tea.NewView(content)
}

func runExampleTUI() {
	p := tea.NewProgram(
		initialExampleModel(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
