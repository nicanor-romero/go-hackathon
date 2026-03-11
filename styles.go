package main

import "charm.land/lipgloss/v2"

var (
	// Title styles
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginBottom(1)

	// Selection styles
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	regularStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// Status styles
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	criticalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Background(lipgloss.Color("52"))

	// Box styles
	boxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	focusedBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(1, 2)

	// Label styles
	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true)

	// Help style
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
)

// Deployment health indicator
func deploymentHealthIcon(ready, desired int32) string {
	if ready == 0 {
		return errorStyle.Render("❌")
	} else if ready < desired {
		return warningStyle.Render("⚠️")
	}
	return successStyle.Render("✅")
}

// Pod status icon
func podStatusIcon(status string) string {
	switch status {
	case "Running":
		return successStyle.Render("✅")
	case "Pending":
		return warningStyle.Render("⏳")
	case "Failed", "Error":
		return errorStyle.Render("❌")
	case "CrashLoopBackOff":
		return criticalStyle.Render("🔄💥")
	case "ImagePullBackOff", "ErrImagePull":
		return errorStyle.Render("📦❌")
	default:
		return dimStyle.Render("⚪")
	}
}
