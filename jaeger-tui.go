package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type jaegerState int

const (
	jaegerStateAuth jaegerState = iota
	jaegerStateLoading
	jaegerStateBrowsing
	jaegerStateLoadingDeps
	jaegerStateShowDeps
	jaegerStateError
)

type jaegerModel struct {
	state           jaegerState
	client          *JaegerClient
	services        []string
	filteredServices []string
	cursor          int
	authInput       textinput.Model
	filterInput     textinput.Model
	selectedService string
	dependencies    ServiceDependencies
	spinner         spinner.Model
	err             error
	frame           int
}

type jaegerServicesLoadedMsg struct {
	services []string
	err      error
}

type jaegerDepsLoadedMsg struct {
	deps ServiceDependencies
	err  error
}

type jaegerTickMsg time.Time

func jaegerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return jaegerTickMsg(t)
	})
}

func loadJaegerServices(client *JaegerClient) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		services, err := client.GetServices(ctx)
		return jaegerServicesLoadedMsg{services: services, err: err}
	}
}

func loadJaegerDeps(client *JaegerClient, serviceName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		deps, err := client.GetServiceDependencies(ctx, serviceName)
		return jaegerDepsLoadedMsg{deps: deps, err: err}
	}
}

func initialJaegerModel() jaegerModel {
	authInput := textinput.New()
	authInput.Placeholder = "Paste your auth cookie..."
	authInput.Focus()

	filterInput := textinput.New()
	filterInput.Placeholder = "Filter services..."

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return jaegerModel{
		state:       jaegerStateAuth,
		authInput:   authInput,
		filterInput: filterInput,
		spinner:     s,
	}
}

func (m jaegerModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		jaegerTick(),
	)
}

func (m jaegerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.state == jaegerStateBrowsing || m.state == jaegerStateAuth {
				return m, tea.Quit
			}
		case "esc":
			if m.state == jaegerStateShowDeps {
				m.state = jaegerStateBrowsing
				return m, nil
			}
		case "enter":
			if m.state == jaegerStateAuth {
				// Create client and load services
				authCookie := m.authInput.Value()
				m.client = NewJaegerClient(jaegerURL, authCookie)
				m.state = jaegerStateLoading
				m.filterInput.Focus()
				return m, loadJaegerServices(m.client)
			}
			if m.state == jaegerStateBrowsing && len(m.filteredServices) > 0 {
				// Load dependencies for selected service
				m.selectedService = m.filteredServices[m.cursor]
				m.state = jaegerStateLoadingDeps
				return m, loadJaegerDeps(m.client, m.selectedService)
			}
		case "up", "k":
			if m.state == jaegerStateBrowsing && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.state == jaegerStateBrowsing && m.cursor < len(m.filteredServices)-1 {
				m.cursor++
			}
		case "pgup":
			if m.state == jaegerStateBrowsing {
				m.cursor -= 10
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		case "pgdown":
			if m.state == jaegerStateBrowsing {
				m.cursor += 10
				if m.cursor >= len(m.filteredServices) {
					m.cursor = len(m.filteredServices) - 1
				}
			}
		case "home", "g":
			if m.state == jaegerStateBrowsing {
				m.cursor = 0
			}
		case "end", "G":
			if m.state == jaegerStateBrowsing {
				m.cursor = len(m.filteredServices) - 1
			}
		}

	case jaegerServicesLoadedMsg:
		if msg.err != nil {
			m.state = jaegerStateError
			m.err = msg.err
			return m, nil
		}
		m.services = msg.services
		m.filteredServices = msg.services
		m.state = jaegerStateBrowsing
		return m, nil

	case jaegerDepsLoadedMsg:
		if msg.err != nil {
			m.state = jaegerStateError
			m.err = msg.err
			return m, nil
		}
		m.dependencies = msg.deps
		m.state = jaegerStateShowDeps
		return m, nil

	case jaegerTickMsg:
		m.frame++
		if m.state == jaegerStateLoading || m.state == jaegerStateLoadingDeps ||
			m.state == jaegerStateBrowsing || m.state == jaegerStateAuth {
			return m, jaegerTick()
		}
		return m, nil
	}

	// Update spinner if loading
	if m.state == jaegerStateLoading || m.state == jaegerStateLoadingDeps {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update auth input
	if m.state == jaegerStateAuth {
		m.authInput, cmd = m.authInput.Update(msg)
		return m, cmd
	}

	// Update filter input
	if m.state == jaegerStateBrowsing {
		oldValue := m.filterInput.Value()
		m.filterInput, cmd = m.filterInput.Update(msg)

		if oldValue != m.filterInput.Value() {
			m.filterJaegerServices()
			m.cursor = 0
		}

		return m, cmd
	}

	return m, nil
}

func (m *jaegerModel) filterJaegerServices() {
	filter := strings.ToLower(m.filterInput.Value())
	if filter == "" {
		m.filteredServices = m.services
		return
	}

	filtered := []string{}
	for _, service := range m.services {
		if strings.Contains(strings.ToLower(service), filter) {
			filtered = append(filtered, service)
		}
	}
	m.filteredServices = filtered
}

func (m jaegerModel) View() tea.View {
	var content string

	switch m.state {
	case jaegerStateAuth:
		var b strings.Builder
		b.WriteString("\n\n")
		b.WriteString(titleStyle2.Render("🔍 Jaeger Tracing Browser"))
		b.WriteString("\n\n")
		b.WriteString("Authentication required (Google Cloud IAP).\n\n")

		instructStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
		b.WriteString(instructStyle.Render("Get cookies from browser:"))
		b.WriteString("\n")
		b.WriteString("1. Open https://tools.masstack.com/tracing\n")
		b.WriteString("2. Press F12 → Application/Storage → Cookies\n")
		b.WriteString("3. Copy both cookie values:\n")

		exampleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
		b.WriteString(exampleStyle.Render("   GCP_IAP_UID=<your-uid>; __Host-GCP_IAP_AUTH_TOKEN_=<your-token>"))
		b.WriteString("\n\n")

		b.WriteString("Paste here (both cookies, semicolon-separated):\n")
		b.WriteString(m.authInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press Enter to continue • Ctrl+C to quit"))
		content = b.String()

	case jaegerStateLoading:
		var b strings.Builder
		b.WriteString("\n\n")

		titleColors := []string{"205", "206", "207", "213", "219", "225", "219", "213", "207", "206"}
		colorIdx := m.frame % len(titleColors)
		animatedTitleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(titleColors[colorIdx])).
			Bold(true)
		b.WriteString(animatedTitleStyle.Render("🔍 Jaeger Tracing Browser"))
		b.WriteString("\n\n")

		b.WriteString(m.spinner.View())
		b.WriteString(" ")

		messages := []string{
			"Connecting to Jaeger...",
			"Fetching services...",
			"Loading data...",
		}
		msgIdx := (m.frame / 8) % len(messages)
		b.WriteString(subtitleStyle.Render(messages[msgIdx]))

		dots := strings.Repeat(".", (m.frame/3)%4)
		b.WriteString(subtitleStyle.Render(dots))

		b.WriteString("\n\n")
		content = b.String()

	case jaegerStateBrowsing:
		var b strings.Builder
		b.WriteString("\n")

		// Animated rainbow title
		titleColors := []string{"196", "202", "208", "214", "220", "226", "154", "118", "82", "46", "47", "48", "49", "50", "51", "45", "39", "33", "27", "21", "57", "93", "129", "165", "201", "200", "199", "198", "197"}
		title := "🔍 Jaeger Tracing - Services"
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

		// Filter input
		filterBox := filterBoxStyle.Render(
			subtitleStyle.Render("🔍 Filter: ") + m.filterInput.View(),
		)
		b.WriteString(filterBox)

		countStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true)
		b.WriteString("  ")
		b.WriteString(countStyle.Render(fmt.Sprintf("[%d/%d]", len(m.filteredServices), len(m.services))))
		b.WriteString("\n\n")

		// Services list (show max 20)
		start := m.cursor
		if start > 10 {
			start = m.cursor - 10
		}
		end := start + 20
		if end > len(m.filteredServices) {
			end = len(m.filteredServices)
		}

		if len(m.filteredServices) == 0 {
			b.WriteString(errorStyle.Render("⚠️  No services match filter"))
			b.WriteString("\n")
		} else {
			showScrollTop := start > 0
			showScrollBottom := end < len(m.filteredServices)

			if showScrollTop {
				scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
				b.WriteString(scrollStyle.Render("   ⬆️  more above..."))
				b.WriteString("\n")
			}

			for i := start; i < end; i++ {
				service := m.filteredServices[i]
				if i == m.cursor {
					b.WriteString(selectedMetricStyle.Render(fmt.Sprintf(" ▶ %s ", service)))
				} else {
					b.WriteString(regularMetricStyle.Render(fmt.Sprintf("   %s", service)))
				}
				b.WriteString("\n")
			}

			if showScrollBottom {
				scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
				b.WriteString(scrollStyle.Render("   ⬇️  more below..."))
				b.WriteString("\n")
			}
		}

		b.WriteString("\n")
		help1 := helpStyle.Render("↑/k up  •  ↓/j down  •  PgUp/PgDn jump  •  g/G top/bottom  •  ⏎ Enter view deps")
		help2 := helpStyle.Render("type to filter  •  q quit")
		b.WriteString(help1)
		b.WriteString("\n")
		b.WriteString(help2)

		content = b.String()

	case jaegerStateLoadingDeps:
		var b strings.Builder
		b.WriteString("\n\n")
		animatedTitleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)
		b.WriteString(animatedTitleStyle.Render(fmt.Sprintf("🔗 %s", m.selectedService)))
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(subtitleStyle.Render("Loading dependencies..."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press Esc to cancel"))
		content = b.String()

	case jaegerStateShowDeps:
		var b strings.Builder
		b.WriteString("\n")
		b.WriteString(titleStyle2.Render(fmt.Sprintf("🔗 %s - Dependencies", m.selectedService)))
		b.WriteString("\n\n")

		// Dependencies (services this service calls)
		var depsContent strings.Builder
		depsContent.WriteString(labelNameStyle.Render("⬇️  Dependencies (calls):"))
		depsContent.WriteString("\n\n")
		if len(m.dependencies.Dependencies) > 0 {
			for _, dep := range m.dependencies.Dependencies {
				depsContent.WriteString(regularMetricStyle.Render(fmt.Sprintf("  → %s", dep)))
				depsContent.WriteString("\n")
			}
		} else {
			depsContent.WriteString(statusStyle.Render("  (no dependencies)"))
			depsContent.WriteString("\n")
		}

		b.WriteString(detailBoxStyle.Render(depsContent.String()))
		b.WriteString("\n")

		// Parents (services that call this service)
		var parentsContent strings.Builder
		parentsContent.WriteString(labelNameStyle.Render("⬆️  Parents (called by):"))
		parentsContent.WriteString("\n\n")
		if len(m.dependencies.Parents) > 0 {
			for _, parent := range m.dependencies.Parents {
				parentsContent.WriteString(regularMetricStyle.Render(fmt.Sprintf("  ← %s", parent)))
				parentsContent.WriteString("\n")
			}
		} else {
			parentsContent.WriteString(statusStyle.Render("  (no parents)"))
			parentsContent.WriteString("\n")
		}

		b.WriteString(detailBoxStyle.Render(parentsContent.String()))

		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("⬅️  Esc back to services  •  q quit"))

		content = b.String()

	case jaegerStateError:
		var b strings.Builder
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("❌ Error\n\n%v", m.err)))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press q to quit"))
		content = b.String()
	}

	return tea.NewView(content)
}

func runJaegerTUI() {
	p := tea.NewProgram(
		initialJaegerModel(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
