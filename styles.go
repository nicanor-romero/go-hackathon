package main

import "charm.land/lipgloss/v2"

var (
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).MarginBottom(1)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	regularStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	boxStyle      = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			MarginTop(1)
	labelStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	helpStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	loadingStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	errorStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	badgeStyle          = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	statusPassStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	statusFailStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	statusPendingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	statusUnknownStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	draftBadgeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	diffAddedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	diffRemovedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	diffHunkHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	diffSeparatorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

var titleColors = []string{"196", "202", "208", "214", "220", "226", "154", "118", "82", "46"}

// badgeColorForUsername returns an ANSI 256-color code string for the given login.
func badgeColorForUsername(login string) string {
	palette := []string{"196", "202", "208", "226", "46", "51", "21", "201"}
	h := 0
	for _, c := range login {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return palette[h%len(palette)]
}
