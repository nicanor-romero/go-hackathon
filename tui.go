package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// --- States ---

type warState int

const (
	stateLoading warState = iota
	stateIncidentList
	stateIncidentDetail
	stateEscalateInput
	stateEscalateResult
	stateFindPods
	statePodList
	stateLogStream
	stateWarInput
	stateWarResult
)

// --- Messages ---

type autoRefreshMsg struct{}

type incidentsLoadedMsg struct {
	incidents []Incident
	err       error
}

type alertDetailMsg struct {
	incident *Incident
	err      error
}

type bulkDetailsMsg struct {
	details map[string]*Incident // alertID -> full incident
}

type escalateResultMsg struct{ err error }
type podsFoundMsg struct {
	pods []Pod
	err  error
}
type warResultMsg struct {
	channelURL string
	err        error
}

// --- Styles ---

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	regularStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	boxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			MarginTop(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	p1Style = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	p2Style = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	p3Style = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	p4Style = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	p5Style = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
)

func priorityStyle(p string) lipgloss.Style {
	switch p {
	case "P1":
		return p1Style
	case "P2":
		return p2Style
	case "P3":
		return p3Style
	case "P4":
		return p4Style
	default:
		return p5Style
	}
}

// --- Model ---

type warModel struct {
	state  warState
	err    error
	width  int
	height int

	// Config
	opsgenieAPIKey string
	opsgenieAPIURL string
	slackBotToken  string

	// Incident list
	incidents  []Incident
	listCursor int
	listOffset int
	refreshing bool

	// Filters
	filterActive         bool
	filterTeamInput      textinput.Model
	filterNamespaceInput textinput.Model
	filterFocus          int // 0=team, 1=namespace

	// Incident detail
	selected     *Incident
	actionCursor int

	// Escalate
	teamInput      textinput.Model
	escalateErr    error
	escalateResult string

	// WAR room
	warInput      textinput.Model
	warErr        error
	warChannelURL string

	// Pods / Logs
	kubeContext string
	pods        []Pod
	podCursor   int
	podOffset   int
	logLines  []string
	logChan   chan string
	logCancel context.CancelFunc
	logScroll int

	// UI components
	spinner spinner.Model
}

func initialWarModel(apiKey, apiURL, slackToken string) warModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "team-name"
	ti.Prompt = "Target team: "
	ti.CharLimit = 64

	wi := textinput.New()
	wi.Placeholder = "channel-suffix"
	wi.Prompt = "Channel suffix: "
	wi.CharLimit = 64

	ft := textinput.New()
	ft.Placeholder = "filter by team..."
	ft.Prompt = "Team: "
	ft.CharLimit = 64

	fn := textinput.New()
	fn.Placeholder = "filter by namespace..."
	fn.Prompt = "Namespace: "
	fn.CharLimit = 64

	return warModel{
		state:                stateLoading,
		opsgenieAPIKey:       apiKey,
		opsgenieAPIURL:       apiURL,
		slackBotToken:        slackToken,
		spinner:              s,
		teamInput:            ti,
		warInput:             wi,
		filterTeamInput:      ft,
		filterNamespaceInput: fn,
	}
}

func scheduleAutoRefresh() tea.Cmd {
	return tea.Tick(10*time.Second, func(time.Time) tea.Msg {
		return autoRefreshMsg{}
	})
}

func (m warModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadIncidents(),
		scheduleAutoRefresh(),
	)
}

func (m warModel) loadIncidents() tea.Cmd {
	return func() tea.Msg {
		incidents, err := fetchIncidents(m.opsgenieAPIKey, m.opsgenieAPIURL)
		return incidentsLoadedMsg{incidents: incidents, err: err}
	}
}

// --- Update ---

func (m warModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case autoRefreshMsg:
		m.refreshing = true
		return m, tea.Batch(m.loadIncidents(), scheduleAutoRefresh())

	case incidentsLoadedMsg:
		m.refreshing = false
		if msg.err != nil {
			// Only show error if we're on the loading/list screen
			if m.state == stateLoading || m.state == stateIncidentList {
				m.err = msg.err
				m.state = stateIncidentList
			}
			return m, nil
		}
		m.incidents = msg.incidents
		if m.state == stateLoading {
			m.state = stateIncidentList
		}
		m.err = nil
		// Clamp cursor if list shrank
		filtered := m.filteredIncidents()
		if m.listCursor >= len(filtered) {
			m.listCursor = max(0, len(filtered)-1)
		}
		// Fetch details for all incidents to populate team/namespace for filtering
		return m, m.fetchBulkDetailsCmd(msg.incidents)

	case bulkDetailsMsg:
		for i := range m.incidents {
			if detail, ok := msg.details[m.incidents[i].ID]; ok {
				m.incidents[i].Team = detail.Team
				m.incidents[i].Namespace = detail.Namespace
				m.incidents[i].Deployment = detail.Deployment
				m.incidents[i].Cluster = detail.Cluster
			}
		}
		return m, nil

	case alertDetailMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateIncidentList
		} else {
			m.selected = msg.incident
			m.actionCursor = 0
			m.state = stateIncidentDetail
		}
		return m, nil

	case escalateResultMsg:
		m.state = stateEscalateResult
		m.escalateErr = msg.err
		if msg.err == nil {
			m.escalateResult = fmt.Sprintf("Escalated to team %q", m.teamInput.Value())
		}
		return m, nil

	case contextResolvedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateIncidentDetail
		} else {
			m.kubeContext = msg.kubeContext
			m.pods = msg.pods
			m.podCursor = 0
			m.podOffset = 0
			m.state = statePodList
		}
		return m, nil

	case podsFoundMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateIncidentDetail
		} else {
			m.pods = msg.pods
			m.podCursor = 0
			m.podOffset = 0
			m.state = statePodList
		}
		return m, nil

	case logLineMsg:
		m.logLines = append(m.logLines, string(msg))
		// Auto-scroll to bottom
		maxVisible := m.height - 6
		if maxVisible < 1 {
			maxVisible = 10
		}
		if len(m.logLines) > maxVisible {
			m.logScroll = len(m.logLines) - maxVisible
		}
		return m, waitForLogLine(m.logChan)

	case logDoneMsg:
		return m, nil

	case warResultMsg:
		m.state = stateWarResult
		m.warErr = msg.err
		if msg.err == nil {
			m.warChannelURL = msg.channelURL
		}
		return m, nil
	}

	// Update spinner when loading or refreshing
	if m.state == stateLoading || m.state == stateFindPods || m.refreshing {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update text inputs when active
	if m.state == stateEscalateInput {
		var cmd tea.Cmd
		m.teamInput, cmd = m.teamInput.Update(msg)
		return m, cmd
	}
	if m.state == stateWarInput {
		var cmd tea.Cmd
		m.warInput, cmd = m.warInput.Update(msg)
		return m, cmd
	}
	if m.state == stateIncidentList && m.filterActive {
		var cmd tea.Cmd
		if m.filterFocus == 0 {
			m.filterTeamInput, cmd = m.filterTeamInput.Update(msg)
		} else {
			m.filterNamespaceInput, cmd = m.filterNamespaceInput.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m warModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit
	if key == "ctrl+c" {
		m.cancelLogs()
		return m, tea.Quit
	}

	switch m.state {
	case stateIncidentList:
		if m.filterActive {
			return m.handleFilterKey(key, msg)
		}
		return m.handleIncidentListKey(key)
	case stateIncidentDetail:
		return m.handleIncidentDetailKey(key)
	case stateEscalateInput:
		return m.handleEscalateInputKey(key, msg)
	case stateEscalateResult:
		return m.handleResultScreenKey(key, stateIncidentDetail)
	case statePodList:
		return m.handlePodListKey(key)
	case stateLogStream:
		return m.handleLogStreamKey(key)
	case stateWarInput:
		return m.handleWarInputKey(key, msg)
	case stateWarResult:
		return m.handleResultScreenKey(key, stateIncidentDetail)
	}

	return m, nil
}

func (m warModel) listVisibleLines() int {
	// header(2) + footer(2) + margins
	v := m.height - 5
	hasFilters := m.filterTeamInput.Value() != "" || m.filterNamespaceInput.Value() != ""
	if m.filterActive {
		v -= 3
	} else if hasFilters {
		v -= 1
	}
	if v < 3 {
		v = 3
	}
	return v
}

func (m warModel) handleIncidentListKey(key string) (tea.Model, tea.Cmd) {
	filtered := m.filteredIncidents()
	switch key {
	case "q":
		return m, tea.Quit
	case "r":
		m.state = stateLoading
		m.err = nil
		return m, tea.Batch(m.spinner.Tick, m.loadIncidents())
	case "f", "/":
		m.filterActive = true
		m.filterFocus = 0
		m.listCursor = 0
		m.listOffset = 0
		return m, m.filterTeamInput.Focus()
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
			if m.listCursor < m.listOffset {
				m.listOffset = m.listCursor
			}
		}
	case "down", "j":
		if m.listCursor < len(filtered)-1 {
			m.listCursor++
			visible := m.listVisibleLines()
			if m.listCursor >= m.listOffset+visible {
				m.listOffset = m.listCursor - visible + 1
			}
		}
	case "enter":
		if len(filtered) > 0 {
			inc := filtered[m.listCursor]
			m.selected = &inc
			m.actionCursor = 0
			m.state = stateLoading
			return m, tea.Batch(m.spinner.Tick, m.fetchDetailCmd(inc.ID))
		}
	}
	return m, nil
}

func (m warModel) handleFilterKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.filterActive = false
		m.filterTeamInput.Blur()
		m.filterNamespaceInput.Blur()
		return m, nil
	case "enter":
		m.filterActive = false
		m.filterTeamInput.Blur()
		m.filterNamespaceInput.Blur()
		m.listCursor = 0
		m.listOffset = 0
		return m, nil
	case "tab", "shift+tab":
		if m.filterFocus == 0 {
			m.filterFocus = 1
			m.filterTeamInput.Blur()
			return m, m.filterNamespaceInput.Focus()
		}
		m.filterFocus = 0
		m.filterNamespaceInput.Blur()
		return m, m.filterTeamInput.Focus()
	}

	// Update the focused input
	var cmd tea.Cmd
	if m.filterFocus == 0 {
		m.filterTeamInput, cmd = m.filterTeamInput.Update(msg)
	} else {
		m.filterNamespaceInput, cmd = m.filterNamespaceInput.Update(msg)
	}

	// Reset cursor when filter changes
	m.listCursor = 0
	m.listOffset = 0

	return m, cmd
}

func (m warModel) handleIncidentDetailKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.state = stateIncidentList
		m.err = nil
	case "up", "k":
		if m.actionCursor > 0 {
			m.actionCursor--
		}
	case "down", "j":
		if m.actionCursor < 2 {
			m.actionCursor++
		}
	case "enter":
		switch m.actionCursor {
		case 0: // Escalate
			m.teamInput.SetValue("")
			m.state = stateEscalateInput
			return m, m.teamInput.Focus()
		case 1: // Investigate
			m.state = stateFindPods
			return m, tea.Batch(m.spinner.Tick, m.findPodsCmd())
		case 2: // Create WAR
			m.warInput.SetValue("")
			m.state = stateWarInput
			return m, m.warInput.Focus()
		}
	}
	return m, nil
}

func (m warModel) handleEscalateInputKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.state = stateIncidentDetail
		return m, nil
	case "enter":
		team := m.teamInput.Value()
		if team == "" {
			return m, nil
		}
		m.state = stateLoading
		return m, tea.Batch(m.spinner.Tick, m.escalateCmd(team))
	}
	var cmd tea.Cmd
	m.teamInput, cmd = m.teamInput.Update(msg)
	return m, cmd
}

func (m warModel) handlePodListKey(key string) (tea.Model, tea.Cmd) {
	visible := m.listVisibleLines()
	switch key {
	case "esc":
		m.state = stateIncidentDetail
	case "up", "k":
		if m.podCursor > 0 {
			m.podCursor--
			if m.podCursor < m.podOffset {
				m.podOffset = m.podCursor
			}
		}
	case "down", "j":
		if m.podCursor < len(m.pods)-1 {
			m.podCursor++
			if m.podCursor >= m.podOffset+visible {
				m.podOffset = m.podCursor - visible + 1
			}
		}
	case "enter":
		if len(m.pods) > 0 {
			pod := m.pods[m.podCursor]
			m.logLines = nil
			m.logScroll = 0
			m.logChan = make(chan string, 100)
			ctx, cancel := context.WithCancel(context.Background())
			m.logCancel = cancel
			m.state = stateLogStream
			go streamLogs(ctx, pod.Namespace, pod.Name, m.kubeContext, m.logChan)
			return m, waitForLogLine(m.logChan)
		}
	}
	return m, nil
}

func (m warModel) handleLogStreamKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.cancelLogs()
		m.state = statePodList
	case "up", "k":
		if m.logScroll > 0 {
			m.logScroll--
		}
	case "down", "j":
		maxVisible := m.height - 6
		if maxVisible < 1 {
			maxVisible = 10
		}
		if m.logScroll < len(m.logLines)-maxVisible {
			m.logScroll++
		}
	}
	return m, nil
}

func (m warModel) handleWarInputKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.state = stateIncidentDetail
		return m, nil
	case "enter":
		suffix := m.warInput.Value()
		if suffix == "" {
			return m, nil
		}
		team := m.selected.Team
		if team == "" {
			team = "ops"
		}
		channelName := fmt.Sprintf("war-%s-%s", strings.ToLower(team), strings.ToLower(suffix))
		m.state = stateLoading
		return m, tea.Batch(m.spinner.Tick, m.createWarCmd(channelName))
	}
	var cmd tea.Cmd
	m.warInput, cmd = m.warInput.Update(msg)
	return m, cmd
}

func (m warModel) handleResultScreenKey(key string, backState warState) (tea.Model, tea.Cmd) {
	if key == "esc" || key == "enter" {
		m.state = backState
		m.err = nil
	}
	return m, nil
}

func (m *warModel) cancelLogs() {
	if m.logCancel != nil {
		m.logCancel()
		m.logCancel = nil
	}
}

// --- Commands ---

func (m warModel) fetchDetailCmd(alertID string) tea.Cmd {
	return func() tea.Msg {
		inc, err := fetchAlertDetail(m.opsgenieAPIKey, m.opsgenieAPIURL, alertID)
		return alertDetailMsg{incident: inc, err: err}
	}
}

func (m warModel) escalateCmd(team string) tea.Cmd {
	return func() tea.Msg {
		err := escalateAlert(m.opsgenieAPIKey, m.opsgenieAPIURL, *m.selected, team)
		return escalateResultMsg{err: err}
	}
}

type contextResolvedMsg struct {
	kubeContext string
	pods        []Pod
	err         error
}

func (m warModel) findPodsCmd() tea.Cmd {
	return func() tea.Msg {
		kubeCtx, err := resolveContext(m.selected.Cluster)
		if err != nil {
			return podsFoundMsg{err: err}
		}
		pods, err := findPods(m.selected.Namespace, m.selected.Deployment, kubeCtx)
		return contextResolvedMsg{kubeContext: kubeCtx, pods: pods, err: err}
	}
}

func (m warModel) createWarCmd(channelName string) tea.Cmd {
	return func() tea.Msg {
		url, err := createSlackChannel(m.slackBotToken, channelName)
		return warResultMsg{channelURL: url, err: err}
	}
}

func (m warModel) fetchBulkDetailsCmd(incidents []Incident) tea.Cmd {
	return func() tea.Msg {
		details := make(map[string]*Incident)
		for _, inc := range incidents {
			detail, err := fetchAlertDetail(m.opsgenieAPIKey, m.opsgenieAPIURL, inc.ID)
			if err == nil {
				details[inc.ID] = detail
			}
		}
		return bulkDetailsMsg{details: details}
	}
}

// --- View ---

func (m warModel) View() tea.View {
	var content string

	switch m.state {
	case stateLoading:
		content = m.viewLoading()
	case stateIncidentList:
		content = m.viewIncidentList()
	case stateIncidentDetail:
		content = m.viewIncidentDetail()
	case stateEscalateInput:
		content = m.viewEscalateInput()
	case stateEscalateResult:
		content = m.viewEscalateResult()
	case stateFindPods:
		content = m.viewLoading()
	case statePodList:
		content = m.viewPodList()
	case stateLogStream:
		content = m.viewLogStream()
	case stateWarInput:
		content = m.viewWarInput()
	case stateWarResult:
		content = m.viewWarResult()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m warModel) viewLoading() string {
	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View())
	b.WriteString(" ")
	b.WriteString(regularStyle.Render("Loading..."))
	return b.String()
}

func (m warModel) viewIncidentList() string {
	var b strings.Builder
	b.WriteString("\n")
	title := "WAR Operator - Open Incidents"
	if m.refreshing {
		title += "  " + m.spinner.View()
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Filter bar
	filtered := m.filteredIncidents()
	hasFilters := m.filterTeamInput.Value() != "" || m.filterNamespaceInput.Value() != ""

	if m.filterActive {
		b.WriteString("\n")
		focusIndicator := func(focused bool) string {
			if focused {
				return selectedStyle.Render("▸ ")
			}
			return "  "
		}
		b.WriteString(focusIndicator(m.filterFocus == 0))
		b.WriteString(m.filterTeamInput.View())
		b.WriteString("\n")
		b.WriteString(focusIndicator(m.filterFocus == 1))
		b.WriteString(m.filterNamespaceInput.View())
		b.WriteString("\n")
		if hasFilters {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  Showing %d of %d incidents", len(filtered), len(m.incidents))))
			b.WriteString("\n")
		}
	} else if hasFilters {
		filters := []string{}
		if v := m.filterTeamInput.Value(); v != "" {
			filters = append(filters, fmt.Sprintf("team:%s", v))
		}
		if v := m.filterNamespaceInput.Value(); v != "" {
			filters = append(filters, fmt.Sprintf("ns:%s", v))
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Filters: %s (%d/%d)", strings.Join(filters, " "), len(filtered), len(m.incidents))))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("r refresh  •  f filter  •  q quit"))
		return b.String()
	}

	if len(filtered) == 0 {
		if hasFilters {
			b.WriteString(dimStyle.Render("  No incidents match the current filters."))
		} else {
			b.WriteString(dimStyle.Render("  No open incidents. All clear!"))
		}
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("r refresh  •  f filter  •  q quit"))
		return b.String()
	}

	visible := m.listVisibleLines()
	end := m.listOffset + visible
	if end > len(filtered) {
		end = len(filtered)
	}

	if m.listOffset > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("   ... %d more above", m.listOffset)))
		b.WriteString("\n")
	}

	for i := m.listOffset; i < end; i++ {
		inc := filtered[i]
		pStyle := priorityStyle(inc.Priority)
		priority := pStyle.Render(fmt.Sprintf("[%s]", inc.Priority))
		age := formatAge(inc.StartTime)

		if i == m.listCursor {
			b.WriteString(selectedStyle.Render(fmt.Sprintf(" > %s %s", priority, inc.Title)))
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %s", age)))
		} else {
			b.WriteString(regularStyle.Render(fmt.Sprintf("   %s %s", priority, inc.Title)))
		}
		b.WriteString("\n")
	}

	if end < len(filtered) {
		b.WriteString(dimStyle.Render(fmt.Sprintf("   ... %d more below", len(filtered)-end)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.filterActive {
		b.WriteString(helpStyle.Render("Tab switch field  •  Enter apply  •  Esc close filter"))
	} else {
		b.WriteString(helpStyle.Render("j/k navigate  •  Enter select  •  f filter  •  r refresh  •  q quit"))
	}

	return b.String()
}

func (m warModel) viewIncidentDetail() string {
	inc := m.selected
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(fmt.Sprintf("Incident: %s", inc.Title)))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// Metadata
	var detail strings.Builder
	detail.WriteString(labelStyle.Render("Priority:   "))
	detail.WriteString(priorityStyle(inc.Priority).Render(inc.Priority))
	detail.WriteString("\n")
	detail.WriteString(labelStyle.Render("Namespace:  "))
	detail.WriteString(regularStyle.Render(valueOr(inc.Namespace, "n/a")))
	detail.WriteString("\n")
	detail.WriteString(labelStyle.Render("Deployment: "))
	detail.WriteString(regularStyle.Render(valueOr(inc.Deployment, "n/a")))
	detail.WriteString("\n")
	detail.WriteString(labelStyle.Render("Team:       "))
	detail.WriteString(regularStyle.Render(valueOr(inc.Team, "n/a")))
	detail.WriteString("\n")
	detail.WriteString(labelStyle.Render("Cluster:    "))
	detail.WriteString(regularStyle.Render(valueOr(inc.Cluster, "n/a")))
	detail.WriteString("\n")
	detail.WriteString(labelStyle.Render("Started:    "))
	detail.WriteString(regularStyle.Render(inc.StartTime.Format(time.RFC3339)))
	detail.WriteString("\n")
	if inc.Description != "" {
		detail.WriteString(labelStyle.Render("Description: "))
		detail.WriteString("\n")
		detail.WriteString(regularStyle.Render("  " + inc.Description))
	}

	b.WriteString(boxStyle.Render(detail.String()))
	b.WriteString("\n\n")

	// Actions
	b.WriteString(labelStyle.Render("Actions:"))
	b.WriteString("\n\n")

	actions := []struct {
		icon string
		name string
		desc string
	}{
		{"!", "Escalate", "Escalate to another team"},
		{"?", "Investigate", "Find pods and stream logs"},
		{"#", "Create WAR Room", "Create a Slack war room channel"},
	}

	for i, a := range actions {
		if i == m.actionCursor {
			b.WriteString(selectedStyle.Render(fmt.Sprintf(" > [%s] %s", a.icon, a.name)))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(fmt.Sprintf("     %s", a.desc)))
		} else {
			b.WriteString(regularStyle.Render(fmt.Sprintf("   [%s] %s", a.icon, a.name)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k navigate  •  Enter select  •  Esc back"))

	return b.String()
}

func (m warModel) viewEscalateInput() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Escalate Incident"))
	b.WriteString("\n\n")
	b.WriteString(regularStyle.Render(fmt.Sprintf("Incident: %s", m.selected.Title)))
	b.WriteString("\n\n")
	b.WriteString(m.teamInput.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Enter submit  •  Esc cancel"))
	return b.String()
}

func (m warModel) viewEscalateResult() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Escalation Result"))
	b.WriteString("\n\n")

	if m.escalateErr != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Failed: %v", m.escalateErr)))
	} else {
		b.WriteString(successStyle.Render(m.escalateResult))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Enter/Esc to go back"))
	return b.String()
}

func (m warModel) viewPodList() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(fmt.Sprintf("Pods - %s/%s", m.selected.Namespace, m.selected.Deployment)))
	b.WriteString("\n\n")

	if len(m.pods) == 0 {
		b.WriteString(dimStyle.Render("  No pods found."))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Esc back"))
		return b.String()
	}

	visible := m.listVisibleLines()
	end := m.podOffset + visible
	if end > len(m.pods) {
		end = len(m.pods)
	}

	if m.podOffset > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("   ... %d more above", m.podOffset)))
		b.WriteString("\n")
	}

	for i := m.podOffset; i < end; i++ {
		pod := m.pods[i]
		line := fmt.Sprintf("%s  [%s]  Ready: %s", pod.Name, pod.Status, pod.Ready)
		if i == m.podCursor {
			b.WriteString(selectedStyle.Render(fmt.Sprintf(" > %s", line)))
		} else {
			b.WriteString(regularStyle.Render(fmt.Sprintf("   %s", line)))
		}
		b.WriteString("\n")
	}

	if end < len(m.pods) {
		b.WriteString(dimStyle.Render(fmt.Sprintf("   ... %d more below", len(m.pods)-end)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k navigate  •  Enter stream logs  •  Esc back"))

	return b.String()
}

func (m warModel) viewLogStream() string {
	var b strings.Builder
	pod := m.pods[m.podCursor]
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(fmt.Sprintf("Logs: %s", pod.Name)))
	b.WriteString("\n")

	maxVisible := m.height - 6
	if maxVisible < 1 {
		maxVisible = 20
	}

	start := m.logScroll
	end := start + maxVisible
	if end > len(m.logLines) {
		end = len(m.logLines)
	}

	if len(m.logLines) == 0 {
		b.WriteString(dimStyle.Render("  Waiting for logs..."))
		b.WriteString("\n")
	} else {
		for i := start; i < end; i++ {
			b.WriteString(dimStyle.Render(m.logLines[i]))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(fmt.Sprintf("j/k scroll  •  Esc back  •  Lines: %d", len(m.logLines))))

	return b.String()
}

func (m warModel) viewWarInput() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Create WAR Room"))
	b.WriteString("\n\n")
	b.WriteString(regularStyle.Render(fmt.Sprintf("Incident: %s", m.selected.Title)))
	b.WriteString("\n")

	team := m.selected.Team
	if team == "" {
		team = "ops"
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("Channel will be: war-%s-<suffix>", strings.ToLower(team))))
	b.WriteString("\n\n")
	b.WriteString(m.warInput.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Enter submit  •  Esc cancel"))
	return b.String()
}

func (m warModel) viewWarResult() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("WAR Room Created"))
	b.WriteString("\n\n")

	if m.warErr != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Failed: %v", m.warErr)))
	} else {
		b.WriteString(successStyle.Render("Channel created successfully!"))
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("URL: "))
		b.WriteString(regularStyle.Render(m.warChannelURL))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Enter/Esc to go back"))
	return b.String()
}

// --- Filtering ---

func (m warModel) filteredIncidents() []Incident {
	teamFilter := strings.ToLower(strings.TrimSpace(m.filterTeamInput.Value()))
	nsFilter := strings.ToLower(strings.TrimSpace(m.filterNamespaceInput.Value()))

	if teamFilter == "" && nsFilter == "" {
		return m.incidents
	}

	var filtered []Incident
	for _, inc := range m.incidents {
		if teamFilter != "" && !strings.Contains(strings.ToLower(inc.Team), teamFilter) {
			continue
		}
		if nsFilter != "" && !strings.Contains(strings.ToLower(inc.Namespace), nsFilter) {
			continue
		}
		filtered = append(filtered, inc)
	}
	return filtered
}

// --- Helpers ---

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
