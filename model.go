package main

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type appState int

const (
	StateTeamSelect appState = iota
	StatePRList
	StatePRSubmenu
	StateReviewInput
	StateDiff
	StateLoading
	StateError
)

const pageSize = 30

type AppModel struct {
	state, prevState appState
	cursor           int
	submenuCursor    int
	teamPage         int
	prPage           int
	repo, org        string
	teams            []Team
	teamMembers      map[string]bool
	prs              []PR
	selectedTeam     Team
	selectedPR       PR
	spinner          spinner.Model
	textInput        textinput.Model // review comment input
	teamInput        textinput.Model // team filter input
	frame            int
	reviewMode       reviewMode
	parsedDiff       *ParsedDiff
	diffScrollY      int
	termWidth        int
	termHeight       int
	loadingMsg       string
	errorMsg         string
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func newAppModel(repo, org string) AppModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Leave a review comment (optional)..."
	ti.CharLimit = 500

	tf := textinput.New()
	tf.Placeholder = "Filter teams..."
	tf.CharLimit = 100

	return AppModel{
		state:      StateLoading,
		repo:       repo,
		org:        org,
		spinner:    s,
		textInput:  ti,
		teamInput:  tf,
		loadingMsg: "Fetching teams...",
		termWidth:  80,
		termHeight: 24,
	}
}

var submenuOptions = []struct {
	icon string
	name string
}{
	{"🌐", "Open in Browser"},
	{"✅", "Approve"},
	{"🔄", "Request Changes"},
	{"🔀", "Checkout"},
	{"📄", "View Diff"},
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		tick(),
		fetchTeamsCmd(m.org),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		return m, nil

	case tickMsg:
		m.frame++
		return m, tick()

	case teamsFetchedMsg:
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Failed to fetch teams: %v", msg.err)
			m.prevState = StateTeamSelect
			m.state = StateError
			return m, nil
		}
		m.teams = msg.teams
		m.state = StateTeamSelect
		m.cursor = 0
		return m, m.teamInput.Focus()

	case membersFetchedMsg:
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Failed to fetch members: %v", msg.err)
			m.prevState = StateTeamSelect
			m.state = StateError
			return m, nil
		}
		members := make(map[string]bool)
		for _, member := range msg.members {
			members[member.Login] = true
		}
		m.teamMembers = members
		m.loadingMsg = "Fetching pull requests..."
		return m, fetchPRsCmd(m.repo)

	case prsFetchedMsg:
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Failed to fetch PRs: %v", msg.err)
			m.prevState = StatePRList
			m.state = StateError
			return m, nil
		}
		m.prs = filterAndSortPRs(msg.prs, m.teamMembers)
		m.state = StatePRList
		m.cursor = 0
		m.prPage = 0
		return m, nil

	case actionDoneMsg:
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Action %q failed: %v", msg.action, msg.err)
			m.prevState = StatePRSubmenu
			m.state = StateError
			return m, nil
		}
		switch msg.action {
		case "checkout", "review":
			m.loadingMsg = "Refreshing pull requests..."
			m.state = StateLoading
			return m, tea.Batch(fetchPRsCmd(m.repo), m.spinner.Tick)
		}
		// "browser" — stay in current state
		return m, nil

	case diffFetchedMsg:
		if msg.err != nil {
			m.errorMsg = fmt.Sprintf("Failed to fetch diff: %v", msg.err)
			m.prevState = StatePRSubmenu
			m.state = StateError
			return m, nil
		}
		m.parsedDiff = msg.diff
		m.diffScrollY = 0
		m.state = StateDiff
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Update team filter input for non-key messages (cursor blink etc.)
	if m.state == StateTeamSelect {
		var cmd tea.Cmd
		m.teamInput, cmd = m.teamInput.Update(msg)
		return m, cmd
	}

	// Update spinner while loading (handles spinner tick messages)
	if m.state == StateLoading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update textinput for non-key messages (cursor blink etc.)
	if m.state == StateReviewInput {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.state {
	case StateTeamSelect:
		filtered := filterTeams(m.teams, m.teamInput.Value())
		pageStart := m.teamPage * pageSize
		pageCount := min(pageSize, len(filtered)-pageStart)
		switch key {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			// Only quit if the filter is empty; otherwise let 'q' be typed.
			if m.teamInput.Value() == "" {
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.teamInput, cmd = m.teamInput.Update(msg)
			m.cursor = 0
			m.teamPage = 0
			return m, cmd
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else if m.teamPage > 0 {
				m.teamPage--
				m.cursor = pageSize - 1
			}
		case "down", "j":
			if m.cursor < pageCount-1 {
				m.cursor++
			} else if pageStart+pageCount < len(filtered) {
				m.teamPage++
				m.cursor = 0
			}
		case "enter":
			if len(filtered) > 0 && pageStart+m.cursor < len(filtered) {
				m.selectedTeam = filtered[pageStart+m.cursor]
				m.state = StateLoading
				m.loadingMsg = fmt.Sprintf("Fetching members of %s...", m.selectedTeam.Name)
				return m, tea.Batch(fetchMembersCmd(m.org, m.selectedTeam.Slug), m.spinner.Tick)
			}
		default:
			// All other keys (typing, backspace, etc.) go to the filter input.
			prev := m.teamInput.Value()
			var cmd tea.Cmd
			m.teamInput, cmd = m.teamInput.Update(msg)
			if m.teamInput.Value() != prev {
				m.cursor = 0
				m.teamPage = 0
			}
			return m, cmd
		}

	case StatePRList:
		pageStart := m.prPage * pageSize
		pageCount := min(pageSize, len(m.prs)-pageStart)
		switch key {
		case "q":
			return m, tea.Quit
		case "esc":
			m.state = StateTeamSelect
			m.cursor = 0
			m.prPage = 0
			return m, m.teamInput.Focus()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else if m.prPage > 0 {
				m.prPage--
				m.cursor = pageSize - 1
			}
		case "down", "j":
			if m.cursor < pageCount-1 {
				m.cursor++
			} else if pageStart+pageCount < len(m.prs) {
				m.prPage++
				m.cursor = 0
			}
		case "enter":
			if len(m.prs) > 0 {
				m.selectedPR = m.prs[pageStart+m.cursor]
				m.state = StatePRSubmenu
				m.submenuCursor = 0
			}
		}

	case StatePRSubmenu:
		switch key {
		case "esc":
			m.state = StatePRList
			m.submenuCursor = 0
		case "up", "k":
			if m.submenuCursor > 0 {
				m.submenuCursor--
			}
		case "down", "j":
			if m.submenuCursor < len(submenuOptions)-1 {
				m.submenuCursor++
			}
		case "enter":
			return m.executeSubmenuAction()
		}

	case StateReviewInput:
		switch key {
		case "esc":
			m.state = StatePRSubmenu
			return m, nil
		case "enter":
			body := m.textInput.Value()
			m.textInput.Reset()
			m.state = StateLoading
			m.loadingMsg = "Submitting review..."
			return m, tea.Batch(reviewCmd(m.selectedPR.Number, m.repo, m.reviewMode, body), m.spinner.Tick)
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

	case StateDiff:
		totalLines := 0
		if m.parsedDiff != nil {
			totalLines = diffTotalLines(m.parsedDiff, m.termWidth)
		}
		visibleLines := m.termHeight - 5
		if visibleLines < 1 {
			visibleLines = 1
		}
		maxScroll := totalLines - visibleLines
		if maxScroll < 0 {
			maxScroll = 0
		}
		switch key {
		case "esc", "q":
			m.state = StatePRSubmenu
		case "up", "k":
			if m.diffScrollY > 0 {
				m.diffScrollY--
			}
		case "down", "j":
			if m.diffScrollY < maxScroll {
				m.diffScrollY++
			}
		case "pgup":
			m.diffScrollY -= visibleLines
			if m.diffScrollY < 0 {
				m.diffScrollY = 0
			}
		case "pgdown":
			m.diffScrollY += visibleLines
			if m.diffScrollY > maxScroll {
				m.diffScrollY = maxScroll
			}
		}

	case StateError:
		m.state = m.prevState
	}

	return m, nil
}

func (m AppModel) executeSubmenuAction() (tea.Model, tea.Cmd) {
	switch m.submenuCursor {
	case 0: // Open in Browser
		return m, openBrowserCmd(m.selectedPR.Number, m.repo)
	case 1: // Approve
		m.reviewMode = reviewModeApprove
		m.state = StateReviewInput
		m.textInput.Reset()
		focusCmd := m.textInput.Focus()
		return m, focusCmd
	case 2: // Request Changes
		m.reviewMode = reviewModeRequestChanges
		m.state = StateReviewInput
		m.textInput.Reset()
		focusCmd := m.textInput.Focus()
		return m, focusCmd
	case 3: // Checkout
		m.state = StateLoading
		m.loadingMsg = fmt.Sprintf("Checking out PR #%d...", m.selectedPR.Number)
		return m, tea.Batch(checkoutCmd(m.selectedPR.Number, m.repo), m.spinner.Tick)
	case 4: // View Diff
		m.state = StateLoading
		m.loadingMsg = fmt.Sprintf("Fetching diff for PR #%d...", m.selectedPR.Number)
		return m, tea.Batch(fetchDiffCmd(m.selectedPR.Number, m.repo), m.spinner.Tick)
	}
	return m, nil
}

// ─── Views ────────────────────────────────────────────────────────────────────

func (m AppModel) View() tea.View {
	var content string
	switch m.state {
	case StateTeamSelect:
		content = m.viewTeamSelect()
	case StateLoading:
		content = m.viewLoading()
	case StatePRList:
		content = m.viewPRList()
	case StatePRSubmenu:
		content = m.viewPRSubmenu()
	case StateReviewInput:
		content = m.viewReviewInput()
	case StateDiff:
		content = m.viewDiff()
	case StateError:
		content = m.viewError()
	}
	return tea.NewView(content)
}

func (m AppModel) rainbowTitle(title string) string {
	var b strings.Builder
	runes := []rune(title)
	for i, char := range runes {
		colorIdx := (m.frame + i) % len(titleColors)
		charStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(titleColors[colorIdx])).
			Bold(true)
		b.WriteString(charStyle.Render(string(char)))
	}
	return b.String()
}

func (m AppModel) viewTeamSelect() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.rainbowTitle("GitHub PR Dashboard"))
	b.WriteString("\n\n")

	// Filter input
	b.WriteString(m.teamInput.View())
	b.WriteString("\n\n")

	filtered := filterTeams(m.teams, m.teamInput.Value())

	totalPages := (len(filtered) + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	header := fmt.Sprintf("Teams (%d)", len(filtered))
	if totalPages > 1 {
		header += fmt.Sprintf("  %s", dimStyle.Render(fmt.Sprintf("page %d/%d", m.teamPage+1, totalPages)))
	}
	b.WriteString(labelStyle.Render(header))
	b.WriteString("\n\n")

	if len(filtered) == 0 {
		if len(m.teams) == 0 {
			b.WriteString(dimStyle.Render("  No teams found"))
		} else {
			b.WriteString(dimStyle.Render("  No teams match the filter"))
		}
		b.WriteString("\n")
	} else {
		pageStart := m.teamPage * pageSize
		pageEnd := min(pageStart+pageSize, len(filtered))
		for i, team := range filtered[pageStart:pageEnd] {
			if i == m.cursor {
				b.WriteString(selectedStyle.Render(fmt.Sprintf(" ▶ %s", team.Name)))
			} else {
				b.WriteString(regularStyle.Render(fmt.Sprintf("   %s", team.Name)))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	help := "type to filter  •  ↑/k up  •  ↓/j down  •  ⏎ select team  •  q quit"
	if totalPages > 1 {
		help += "  •  ↓/↑ past edge = next/prev page"
	}
	b.WriteString(helpStyle.Render(help))
	return b.String()
}

func (m AppModel) viewLoading() string {
	var b strings.Builder
	b.WriteString("\n\n")

	loadColors := []string{"220", "221", "227", "228", "227", "221"}
	colorIdx := m.frame % len(loadColors)
	animStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(loadColors[colorIdx])).Bold(true)
	b.WriteString(animStyle.Render(m.loadingMsg))
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View())
	b.WriteString(" ")
	b.WriteString(regularStyle.Render("Please wait..."))
	return b.String()
}

func (m AppModel) viewPRList() string {
	var b strings.Builder
	b.WriteString("\n")

	title := fmt.Sprintf("PRs for %s", m.selectedTeam.Name)
	if m.termWidth > 0 && len([]rune(title)) > m.termWidth-4 {
		runes := []rune(title)
		title = string(runes[:m.termWidth-5]) + "…"
	}
	b.WriteString(m.rainbowTitle(title))
	b.WriteString("\n\n")

	if len(m.prs) == 0 {
		b.WriteString(dimStyle.Render("  No PRs found for this team"))
		b.WriteString("\n")
	} else {
		// Compute title column width from available terminal space.
		// Fixed portions: cursor(2) + "#NNN "(6) + " [X]"(4) + " CI"(3) + extras(8) = ~23
		const fixedCols = 24
		titleWidth := m.termWidth - fixedCols
		if titleWidth < 10 {
			titleWidth = 10
		}

		pageStart := m.prPage * pageSize
		pageEnd := min(pageStart+pageSize, len(m.prs))
		for i, pr := range m.prs[pageStart:pageEnd] {
			selected := i == m.cursor

			numStr := fmt.Sprintf("#%-4d", pr.Number)

			titleStr := pr.Title
			titleRunes := []rune(titleStr)
			if len(titleRunes) > titleWidth {
				titleStr = string(titleRunes[:titleWidth-1]) + "…"
			}
			titleStr = fmt.Sprintf("%-*s", titleWidth, titleStr)

			// Author badge
			initial := "?"
			if len(pr.AuthorLogin) > 0 {
				initial = strings.ToUpper(string([]rune(pr.AuthorLogin)[0]))
			}
			colorCode := badgeColorForUsername(pr.AuthorLogin)
			badge := lipgloss.NewStyle().Foreground(lipgloss.Color(colorCode)).Bold(true).Render(fmt.Sprintf("[%s]", initial))

			// CI status
			var ciStr string
			switch pr.CIStatus {
			case CIPass:
				ciStr = statusPassStyle.Render("✓")
			case CIFail:
				ciStr = statusFailStyle.Render("✗")
			case CIPending:
				ciStr = statusPendingStyle.Render("⋯")
			default:
				ciStr = statusUnknownStyle.Render("?")
			}

			// Extras
			extras := ""
			if pr.IsDraft {
				extras += " " + draftBadgeStyle.Render("Draft")
			}
			if pr.Mergeable == "CONFLICTING" {
				extras += " " + statusFailStyle.Render("⚠")
			}

			var prefix, mainText string
			if selected {
				prefix = selectedStyle.Render("▶")
				mainText = selectedStyle.Render(numStr+" "+titleStr)
			} else {
				prefix = " "
				mainText = regularStyle.Render(numStr + " " + titleStr)
			}

			b.WriteString(prefix + " " + mainText + " " + badge + " " + ciStr + extras + "\n")
		}
	}

	b.WriteString("\n")
	totalPages := (len(m.prs) + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	prHelp := "↑/k up  •  ↓/j down  •  ⏎ open PR  •  Esc teams  •  q quit"
	if totalPages > 1 {
		prHelp = fmt.Sprintf("page %d/%d  •  ", m.prPage+1, totalPages) + prHelp +
			"  •  ↓/↑ past edge = next/prev page"
	}
	b.WriteString(helpStyle.Render(prHelp))
	return b.String()
}

func (m AppModel) viewPRSubmenu() string {
	var b strings.Builder
	b.WriteString("\n")

	prTitle := m.selectedPR.Title
	if len([]rune(prTitle)) > m.termWidth-12 && m.termWidth > 12 {
		runes := []rune(prTitle)
		prTitle = string(runes[:m.termWidth-13]) + "…"
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf("PR #%d: %s", m.selectedPR.Number, prTitle)))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Select an action:"))
	b.WriteString("\n\n")

	for i, opt := range submenuOptions {
		if i == m.submenuCursor {
			b.WriteString(selectedStyle.Render(fmt.Sprintf(" ▶ %s %s", opt.icon, opt.name)))
		} else {
			b.WriteString(regularStyle.Render(fmt.Sprintf("   %s %s", opt.icon, opt.name)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/k up  •  ↓/j down  •  ⏎ select  •  Esc back"))
	return b.String()
}

func (m AppModel) viewReviewInput() string {
	var b strings.Builder
	b.WriteString("\n")

	action := "Approve"
	if m.reviewMode == reviewModeRequestChanges {
		action = "Request Changes"
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf("%s PR #%d", action, m.selectedPR.Number)))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Review comment:"))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("⏎ submit  •  Esc cancel"))
	return b.String()
}

func (m AppModel) viewDiff() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(fmt.Sprintf("Diff: PR #%d — %s", m.selectedPR.Number, m.selectedPR.HeadRefName)))
	b.WriteString("\n")

	if m.parsedDiff == nil || len(m.parsedDiff.FileDiffs) == 0 {
		b.WriteString(dimStyle.Render("  No diff available"))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Esc back"))
		return b.String()
	}

	visibleHeight := m.termHeight - 6
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	var diffContent string
	if m.termWidth >= 160 {
		diffContent = renderSideBySideDiff(m.parsedDiff, m.diffScrollY, visibleHeight, m.termWidth)
	} else {
		diffContent = renderUnifiedDiff(m.parsedDiff, m.diffScrollY, visibleHeight, m.termWidth)
	}

	b.WriteString(diffContent)
	b.WriteString("\n")

	totalLines := diffTotalLines(m.parsedDiff, m.termWidth)
	pct := 0
	if totalLines > 0 {
		pct = ((m.diffScrollY + visibleHeight) * 100) / totalLines
		if pct > 100 {
			pct = 100
		}
	}
	modeLabel := "unified"
	if m.termWidth >= 160 {
		modeLabel = "side-by-side"
	}
	b.WriteString(diffSeparatorStyle.Render(
		fmt.Sprintf("── %s │ line %d/%d (%d%%)  j/k scroll  PgDn/PgUp page  Esc back ──",
			modeLabel, m.diffScrollY+1, totalLines, pct),
	))
	return b.String()
}

func (m AppModel) viewError() string {
	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(errorStyle.Render("  Error"))
	b.WriteString("\n\n")

	var msgBox strings.Builder
	msgBox.WriteString(regularStyle.Render(m.errorMsg))
	b.WriteString(boxStyle.Render(msgBox.String()))

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("  Press any key to continue..."))
	return b.String()
}
