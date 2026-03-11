package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	// Title
	titleStyle2 = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginBottom(1)

	// Subtitle style
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Italic(true)

	// Status bar at bottom
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	// Error messages
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Padding(1, 2).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196"))

	// Selected metric (cursor)
	selectedMetricStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Bold(true)

	// Regular metric
	regularMetricStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	// Detail view box
	detailBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			MarginTop(1)

	// Label style (field names)
	labelNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)

	// Stats style
	statsKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")).
			Bold(true)

	statsValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("155"))

	// Filter input style
	filterBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	// Keybindings help
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type state int

const (
	stateLoading state = iota
	stateBrowsing
	stateSubmenu
	stateLoadingLabels
	stateShowLabels
	stateLoadingStats
	stateShowStats
	stateShowPromQL
	stateError
)

type submenuOption int

const (
	optionLabels submenuOption = iota
	optionStats
	optionPromQL
)

type promModel struct {
	state           state
	client          *PrometheusClient
	metrics         []string
	filteredMetrics []string
	cursor          int
	submenuCursor   int
	filterInput     textinput.Model
	selectedMetric  string
	metricStats     MetricStats
	metricLabels    []string
	spinner         spinner.Model
	err             error
	frame           int
}

type metricsLoadedMsg struct {
	metrics []string
	err     error
}

type statsLoadedMsg struct {
	stats MetricStats
}

type labelsLoadedMsg struct {
	labels []string
	err    error
}

type tickMsg time.Time

func loadMetrics(client *PrometheusClient) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		metrics, err := client.GetMetricNames(ctx)
		return metricsLoadedMsg{metrics: metrics, err: err}
	}
}

func loadStats(client *PrometheusClient, metricName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		stats := client.GetMetricStats(ctx, metricName)
		return statsLoadedMsg{stats: stats}
	}
}

func loadLabels(client *PrometheusClient, metricName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		labels, err := client.GetMetricLabels(ctx, metricName)
		return labelsLoadedMsg{labels: labels, err: err}
	}
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func initialPromModel() promModel {
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter metrics..."
	filterInput.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return promModel{
		state:       stateLoading,
		client:      NewPrometheusClient(prometheusURL),
		filterInput: filterInput,
		spinner:     s,
	}
}

func (m promModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		loadMetrics(m.client),
		tick(),
	)
}

func (m promModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// Always allow ctrl+c to quit
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			// 'q' quits unless in submenu or detail views
			if m.state == stateBrowsing || m.state == stateLoading {
				return m, tea.Quit
			}
		case "esc":
			if m.state == stateSubmenu {
				// Go back to browsing
				m.state = stateBrowsing
				m.submenuCursor = 0
				return m, nil
			}
			if m.state == stateShowLabels || m.state == stateShowStats || m.state == stateShowPromQL {
				// Go back to submenu
				m.state = stateSubmenu
				return m, nil
			}
		case "enter":
			if m.state == stateBrowsing && len(m.filteredMetrics) > 0 {
				// Show submenu
				m.selectedMetric = m.filteredMetrics[m.cursor]
				m.state = stateSubmenu
				m.submenuCursor = 0
				return m, nil
			}
			if m.state == stateSubmenu {
				// Handle submenu selection
				switch m.submenuCursor {
				case int(optionLabels):
					m.state = stateLoadingLabels
					return m, loadLabels(m.client, m.selectedMetric)
				case int(optionStats):
					m.state = stateLoadingStats
					return m, loadStats(m.client, m.selectedMetric)
				case int(optionPromQL):
					m.state = stateShowPromQL
					return m, nil
				}
			}
		case "up", "k":
			if m.state == stateBrowsing && m.cursor > 0 {
				m.cursor--
			}
			if m.state == stateSubmenu && m.submenuCursor > 0 {
				m.submenuCursor--
			}
		case "down", "j":
			if m.state == stateBrowsing && m.cursor < len(m.filteredMetrics)-1 {
				m.cursor++
			}
			if m.state == stateSubmenu && m.submenuCursor < 2 {
				m.submenuCursor++
			}
		case "pgup":
			if m.state == stateBrowsing {
				m.cursor -= 10
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		case "pgdown":
			if m.state == stateBrowsing {
				m.cursor += 10
				if m.cursor >= len(m.filteredMetrics) {
					m.cursor = len(m.filteredMetrics) - 1
				}
			}
		case "home", "g":
			if m.state == stateBrowsing {
				m.cursor = 0
			}
		case "end", "G":
			if m.state == stateBrowsing {
				m.cursor = len(m.filteredMetrics) - 1
			}
		}

	case metricsLoadedMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.metrics = msg.metrics
		m.filteredMetrics = msg.metrics
		m.state = stateBrowsing
		return m, nil

	case statsLoadedMsg:
		m.metricStats = msg.stats
		if msg.stats.Error != nil {
			m.state = stateError
			m.err = msg.stats.Error
			return m, nil
		}
		m.state = stateShowStats
		return m, nil

	case labelsLoadedMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.metricLabels = msg.labels
		m.state = stateShowLabels
		return m, nil

	case tickMsg:
		m.frame++
		// Continue ticking during loading states and browsing for title animation
		if m.state == stateLoading || m.state == stateLoadingStats ||
		   m.state == stateLoadingLabels ||
		   m.state == stateBrowsing || m.state == stateSubmenu {
			return m, tick()
		}
		return m, nil
	}

	// Update spinner if loading
	if m.state == stateLoading || m.state == stateLoadingStats ||
		m.state == stateLoadingLabels {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update filter input
	if m.state == stateBrowsing {
		oldValue := m.filterInput.Value()
		m.filterInput, cmd = m.filterInput.Update(msg)

		// Re-filter if input changed
		if oldValue != m.filterInput.Value() {
			m.filterMetrics()
			m.cursor = 0
		}

		return m, cmd
	}

	return m, nil
}

func (m *promModel) filterMetrics() {
	filter := strings.ToLower(m.filterInput.Value())
	if filter == "" {
		m.filteredMetrics = m.metrics
		return
	}

	filtered := []string{}
	for _, metric := range m.metrics {
		if strings.Contains(strings.ToLower(metric), filter) {
			filtered = append(filtered, metric)
		}
	}
	m.filteredMetrics = filtered
}

func formatValue(val float64) string {
	if math.IsNaN(val) {
		return "N/A"
	}
	if math.IsInf(val, 1) {
		return "+∞"
	}
	if math.IsInf(val, -1) {
		return "-∞"
	}
	return fmt.Sprintf("%.2f", val)
}

func (m promModel) View() tea.View {
	var content string

	switch m.state {
	case stateLoading:
		var b strings.Builder
		b.WriteString("\n\n")

		// Animated title with pulsing effect
		titleColors := []string{"205", "206", "207", "213", "219", "225", "219", "213", "207", "206"}
		colorIdx := m.frame % len(titleColors)
		animatedTitleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(titleColors[colorIdx])).
			Bold(true)
		b.WriteString(animatedTitleStyle.Render("🚀 Prometheus Metrics Browser"))
		b.WriteString("\n\n")

		// Animated progress bar
		progressChars := []string{"▱", "▰"}
		var progressBar strings.Builder
		for i := 0; i < 30; i++ {
			if (m.frame+i)%3 == 0 {
				progressBar.WriteString(progressChars[1])
			} else {
				progressBar.WriteString(progressChars[0])
			}
		}
		progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
		b.WriteString(progressStyle.Render(progressBar.String()))
		b.WriteString("\n\n")

		// Spinner with message
		b.WriteString(m.spinner.View())
		b.WriteString(" ")

		// Cycling loading messages
		messages := []string{
			"Connecting to Prometheus...",
			"Fetching metric names...",
			"Loading data...",
			"Almost there...",
		}
		msgIdx := (m.frame / 8) % len(messages)
		b.WriteString(subtitleStyle.Render(messages[msgIdx]))

		// Animated dots
		dots := strings.Repeat(".", (m.frame/3)%4)
		b.WriteString(subtitleStyle.Render(dots))

		b.WriteString("\n\n")
		content = b.String()

	case stateSubmenu:
		var b strings.Builder
		b.WriteString("\n\n")

		// Title
		b.WriteString(titleStyle2.Render(fmt.Sprintf("📊 %s", m.selectedMetric)))
		b.WriteString("\n\n")

		// Submenu title
		b.WriteString(labelNameStyle.Render("Choose an option:"))
		b.WriteString("\n\n")

		// Menu options
		options := []struct {
			icon  string
			label string
			desc  string
		}{
			{"🏷️", "View Labels", "Show all available fields/labels"},
			{"📈", "View Statistics", "Show min/max/avg and cardinality"},
			{"📝", "PromQL Queries", "Generate queries for Grafana"},
		}

		for i, opt := range options {
			if i == m.submenuCursor {
				b.WriteString(selectedMetricStyle.Render(fmt.Sprintf(" ▶ %s %s", opt.icon, opt.label)))
				b.WriteString("\n")
				b.WriteString(statusStyle.Render(fmt.Sprintf("     %s", opt.desc)))
			} else {
				b.WriteString(regularMetricStyle.Render(fmt.Sprintf("   %s %s", opt.icon, opt.label)))
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/k up  •  ↓/j down  •  ⏎ Enter select  •  Esc back"))

		content = b.String()

	case stateLoadingLabels:
		var b strings.Builder
		b.WriteString("\n\n")
		animatedTitleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)
		b.WriteString(animatedTitleStyle.Render(fmt.Sprintf("🏷️ %s", m.selectedMetric)))
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(subtitleStyle.Render("Loading labels..."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press Esc to cancel"))
		content = b.String()

	case stateLoadingStats:
		var b strings.Builder
		b.WriteString("\n\n")
		animatedTitleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)
		b.WriteString(animatedTitleStyle.Render(fmt.Sprintf("📈 %s", m.selectedMetric)))
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(subtitleStyle.Render("Calculating statistics..."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press Esc to cancel"))
		content = b.String()

	case stateShowLabels:
		var b strings.Builder
		b.WriteString("\n")
		b.WriteString(titleStyle2.Render(fmt.Sprintf("🏷️ %s - Labels", m.selectedMetric)))
		b.WriteString("\n\n")

		// Labels box
		var labelsContent strings.Builder
		labelsContent.WriteString(labelNameStyle.Render("Available Fields:"))
		labelsContent.WriteString("\n\n")
		if len(m.metricLabels) > 0 {
			for _, label := range m.metricLabels {
				labelsContent.WriteString(regularMetricStyle.Render(fmt.Sprintf("  • %s", label)))
				labelsContent.WriteString("\n")
			}
		} else {
			labelsContent.WriteString(statusStyle.Render("  (no labels)"))
			labelsContent.WriteString("\n")
		}

		b.WriteString(detailBoxStyle.Render(labelsContent.String()))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("⬅️  Esc back to menu  •  q quit"))

		content = b.String()

	case stateShowStats:
		var b strings.Builder

		// Title
		b.WriteString("\n")
		b.WriteString(titleStyle2.Render(fmt.Sprintf("📈 %s - Statistics", m.selectedMetric)))
		b.WriteString("\n\n")

		// Stats box
		var statsContent strings.Builder
		statsContent.WriteString(labelNameStyle.Render("📈 Statistics:"))
		statsContent.WriteString("\n\n")

		// Show metric type
		if m.metricStats.IsSummary {
			typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Italic(true)
			statsContent.WriteString(fmt.Sprintf("  %s\n\n",
				typeStyle.Render("⚠️  Summary metric (use quantiles for percentiles)")))
		}

		// Cardinality
		statsContent.WriteString(fmt.Sprintf("  %s  %s\n",
			statsKeyStyle.Render("Series: "),
			statsValueStyle.Render(fmt.Sprintf("%d time series", m.metricStats.Count))))

		// Min/Max always shown
		statsContent.WriteString(fmt.Sprintf("  %s  %s\n",
			statsKeyStyle.Render("Min:    "),
			statsValueStyle.Render(formatValue(m.metricStats.Min))))
		statsContent.WriteString(fmt.Sprintf("  %s  %s\n",
			statsKeyStyle.Render("Max:    "),
			statsValueStyle.Render(formatValue(m.metricStats.Max))))

		// Average only for non-summaries
		if m.metricStats.HasAvg {
			statsContent.WriteString(fmt.Sprintf("  %s  %s\n",
				statsKeyStyle.Render("Average:"),
				statsValueStyle.Render(formatValue(m.metricStats.Avg))))
		}

		// Rate only for non-summaries
		if m.metricStats.HasRate {
			rateStr := formatValue(m.metricStats.Rate)
			if rateStr != "N/A" {
				rateStr = rateStr + "/s"
			}
			statsContent.WriteString(fmt.Sprintf("  %s  %s\n",
				statsKeyStyle.Render("Rate:   "),
				statsValueStyle.Render(rateStr)))
		}

		b.WriteString(detailBoxStyle.Render(statsContent.String()))

		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("⬅️  Esc back to menu  •  q quit"))

		content = b.String()

	case stateShowPromQL:
		var b strings.Builder
		b.WriteString("\n")
		b.WriteString(titleStyle2.Render(fmt.Sprintf("📝 %s - PromQL Queries", m.selectedMetric)))
		b.WriteString("\n\n")

		// Generate useful PromQL queries
		queries := []struct {
			title string
			query string
			desc  string
		}{
			{
				"Basic Query",
				m.selectedMetric,
				"Current values for all time series",
			},
			{
				"Rate (5m)",
				fmt.Sprintf("rate(%s[5m])", m.selectedMetric),
				"Per-second rate over last 5 minutes (for counters)",
			},
			{
				"Sum",
				fmt.Sprintf("sum(%s)", m.selectedMetric),
				"Sum across all labels",
			},
			{
				"Average by Label",
				fmt.Sprintf("avg by(instance) (%s)", m.selectedMetric),
				"Average grouped by instance",
			},
			{
				"Top 10",
				fmt.Sprintf("topk(10, %s)", m.selectedMetric),
				"Top 10 time series by value",
			},
			{
				"Increase (1h)",
				fmt.Sprintf("increase(%s[1h])", m.selectedMetric),
				"Total increase over last hour (for counters)",
			},
		}

		// Render queries
		for _, q := range queries {
			var queryContent strings.Builder
			queryContent.WriteString(labelNameStyle.Render(q.title))
			queryContent.WriteString("\n")
			queryContent.WriteString(statusStyle.Render(q.desc))
			queryContent.WriteString("\n\n")

			// Query in a box
			queryStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("155")).
				Background(lipgloss.Color("235")).
				Padding(0, 1)
			queryContent.WriteString(queryStyle.Render(q.query))
			queryContent.WriteString("\n")

			b.WriteString(detailBoxStyle.Render(queryContent.String()))
			b.WriteString("\n")
		}

		// Instructions
		b.WriteString("\n")
		instructStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Italic(true)
		b.WriteString(instructStyle.Render("💡 Copy any query above and paste into Grafana panel query editor"))

		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("⬅️  Esc back to menu  •  q quit"))

		content = b.String()

	case stateError:
		var b strings.Builder
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("❌ Error\n\n%v", m.err)))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press q to quit"))
		content = b.String()

	case stateBrowsing:
		var b strings.Builder

		// Animated title with color cycling
		b.WriteString("\n")

		// Create rainbow gradient effect
		titleColors := []string{"196", "202", "208", "214", "220", "226", "154", "118", "82", "46", "47", "48", "49", "50", "51", "45", "39", "33", "27", "21", "57", "93", "129", "165", "201", "200", "199", "198", "197"}

		title := "🚀 Prometheus Metrics Browser"
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

		// Filter input with box
		filterBox := filterBoxStyle.Render(
			subtitleStyle.Render("🔍 Filter: ") + m.filterInput.View(),
		)
		b.WriteString(filterBox)

		// Count badge
		countStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true)
		b.WriteString("  ")
		b.WriteString(countStyle.Render(fmt.Sprintf("[%d/%d]", len(m.filteredMetrics), len(m.metrics))))
		b.WriteString("\n\n")

		// Metrics list (show max 20)
		start := m.cursor
		if start > 10 {
			start = m.cursor - 10
		}
		end := start + 20
		if end > len(m.filteredMetrics) {
			end = len(m.filteredMetrics)
		}

		if len(m.filteredMetrics) == 0 {
			b.WriteString(errorStyle.Render("⚠️  No metrics match filter"))
			b.WriteString("\n")
		} else {
			// Show scroll indicator if there are more items
			showScrollTop := start > 0
			showScrollBottom := end < len(m.filteredMetrics)

			if showScrollTop {
				scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
				b.WriteString(scrollStyle.Render("   ⬆️  more above..."))
				b.WriteString("\n")
			}

			for i := start; i < end; i++ {
				metric := m.filteredMetrics[i]
				if i == m.cursor {
					b.WriteString(selectedMetricStyle.Render(fmt.Sprintf(" ▶ %s ", metric)))
				} else {
					b.WriteString(regularMetricStyle.Render(fmt.Sprintf("   %s", metric)))
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

		// Two-line help for all the controls
		help1 := helpStyle.Render("↑/k up  •  ↓/j down  •  PgUp/PgDn jump  •  g/G top/bottom  •  ⏎ Enter select metric")
		help2 := helpStyle.Render("type to filter  •  q quit")
		b.WriteString(help1)
		b.WriteString("\n")
		b.WriteString(help2)

		content = b.String()
	}

	return tea.NewView(content)
}

func runPromTUI() {
	p := tea.NewProgram(
		initialPromModel(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
