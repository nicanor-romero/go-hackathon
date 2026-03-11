package main

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ---------------------------------------------------------------------------
// States
// ---------------------------------------------------------------------------

type shieldState int

const (
	shieldStateInput     shieldState = iota
	shieldStateAnalyzing             // spinner while Claude analyzes
	shieldStateResult                // show risk analysis, wait for y/n
	shieldStateExecuting             // spinner while command runs
	shieldStateDone                  // show output (success or failure)
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type analysisMsg struct {
	risk        string // "LOW" | "MEDIUM" | "HIGH" | "CRITICAL"
	explanation string
	warning     string
	alternative string
	err         error
}

type execMsg struct {
	output string // combined stdout+stderr
	err    error
}

type errorInterpMsg struct {
	text string
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type shieldModel struct {
	state       shieldState
	input       textinput.Model
	spinner     spinner.Model
	command     string
	analysis    analysisMsg
	execOutput  string // stdout+stderr from the command
	execErr     string // non-empty if command failed
	errorInterp string // Claude's error explanation (populated async)
}

// ---------------------------------------------------------------------------
// Styles
// ---------------------------------------------------------------------------

var (
	shTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).MarginBottom(1)
	shInputStyle   = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63")).Padding(0, 1)
	shBoxStyle     = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(1, 2).MarginTop(1)
	shLabelStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
	shNormalStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	shHelpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	shSuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	shErrorStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	shWarnStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
)

func riskColor(risk string) color.Color {
	switch risk {
	case "LOW":
		return lipgloss.Color("42")
	case "MEDIUM":
		return lipgloss.Color("220")
	case "HIGH":
		return lipgloss.Color("208")
	case "CRITICAL":
		return lipgloss.Color("196")
	default:
		return lipgloss.Color("252")
	}
}

func riskIcon(risk string) string {
	switch risk {
	case "LOW":
		return "✅"
	case "MEDIUM":
		return "⚠️ "
	case "HIGH":
		return "🔶"
	case "CRITICAL":
		return "🚨"
	default:
		return "❓"
	}
}

// ---------------------------------------------------------------------------
// Parsing
// ---------------------------------------------------------------------------

func parseAnalysis(text string) analysisMsg {
	result := analysisMsg{}
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "RISK:"):
			result.risk = strings.TrimSpace(strings.TrimPrefix(line, "RISK:"))
		case strings.HasPrefix(line, "EXPLANATION:"):
			result.explanation = strings.TrimSpace(strings.TrimPrefix(line, "EXPLANATION:"))
		case strings.HasPrefix(line, "WARNING:"):
			result.warning = strings.TrimSpace(strings.TrimPrefix(line, "WARNING:"))
		case strings.HasPrefix(line, "ALTERNATIVE:"):
			result.alternative = strings.TrimSpace(strings.TrimPrefix(line, "ALTERNATIVE:"))
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func initialShieldModel() shieldModel {
	ti := textinput.New()
	ti.Placeholder = "type a shell command..."
	ti.SetWidth(60)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return shieldModel{state: shieldStateInput, input: ti, spinner: s}
}

func (m shieldModel) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), m.spinner.Tick)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m shieldModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			switch m.state {
			case shieldStateInput:
				command := strings.TrimSpace(m.input.Value())
				if command == "" {
					return m, nil
				}
				m.command = command
				m.state = shieldStateAnalyzing
				return m, tea.Batch(m.spinner.Tick, analyzeCommand(command))
			case shieldStateResult:
				m.state = shieldStateExecuting
				return m, tea.Batch(m.spinner.Tick, executeCommand(m.command))
			}

		case "y", "Y":
			if m.state == shieldStateResult {
				m.state = shieldStateExecuting
				return m, tea.Batch(m.spinner.Tick, executeCommand(m.command))
			}

		case "n", "N", "esc":
			if m.state == shieldStateResult {
				m.state = shieldStateInput
				m.input.SetValue("")
				return m, m.input.Focus()
			}

		case "q":
			if m.state == shieldStateDone {
				return m, tea.Quit
			}

		case "r":
			if m.state == shieldStateDone {
				newM := initialShieldModel()
				return newM, newM.Init()
			}
		}

	case analysisMsg:
		m.analysis = msg
		m.state = shieldStateResult
		return m, nil

	case execMsg:
		if msg.err != nil {
			m.execErr = msg.err.Error()
			m.execOutput = msg.output
			m.state = shieldStateDone
			return m, tea.Batch(m.spinner.Tick, interpretError(m.command, msg.output))
		}
		m.execOutput = msg.output
		m.state = shieldStateDone
		return m, nil

	case errorInterpMsg:
		m.errorInterp = msg.text
		return m, nil
	}

	// Sub-component updates
	switch m.state {
	case shieldStateInput:
		m.input, cmd = m.input.Update(msg)
	case shieldStateAnalyzing, shieldStateExecuting:
		m.spinner, cmd = m.spinner.Update(msg)
	case shieldStateDone:
		if m.execErr != "" && m.errorInterp == "" {
			m.spinner, cmd = m.spinner.Update(msg)
		}
	}

	return m, cmd
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m shieldModel) View() tea.View {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(shTitleStyle.Render("🛡️  Command Risk Shield"))
	b.WriteString("\n\n")

	switch m.state {
	case shieldStateInput:
		b.WriteString(shLabelStyle.Render("Command:"))
		b.WriteString("\n")
		b.WriteString(shInputStyle.Render(m.input.View()))
		b.WriteString("\n\n")
		if os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" {
			b.WriteString(shHelpStyle.Render("⚠️  ANTHROPIC_AUTH_TOKEN not set — analysis will be skipped"))
			b.WriteString("\n\n")
		}
		b.WriteString(shHelpStyle.Render("⏎ analyze  •  ctrl+c quit"))

	case shieldStateAnalyzing:
		b.WriteString(shNormalStyle.Render("$ " + m.command))
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(shNormalStyle.Render("Analyzing with AI..."))

	case shieldStateResult:
		b.WriteString(shNormalStyle.Render("$ " + m.command))
		b.WriteString("\n")

		riskBoxStyle := shBoxStyle.BorderForeground(riskColor(m.analysis.risk))

		var boxContent strings.Builder
		boxContent.WriteString(
			lipgloss.NewStyle().Bold(true).Foreground(riskColor(m.analysis.risk)).Render(
				fmt.Sprintf("%s Risk: %s", riskIcon(m.analysis.risk), m.analysis.risk),
			),
		)
		boxContent.WriteString("\n\n")
		boxContent.WriteString(shLabelStyle.Render("What it does:"))
		boxContent.WriteString("\n")
		boxContent.WriteString(shNormalStyle.Width(58).Render("  " + m.analysis.explanation))

		if m.analysis.warning != "" && strings.ToLower(m.analysis.warning) != "none" {
			boxContent.WriteString("\n\n")
			boxContent.WriteString(shWarnStyle.Render("⚠️  Warning:"))
			boxContent.WriteString("\n")
			boxContent.WriteString(shNormalStyle.Width(58).Render("  " + m.analysis.warning))
		}

		if m.analysis.alternative != "" && strings.ToLower(m.analysis.alternative) != "none" {
			boxContent.WriteString("\n\n")
			boxContent.WriteString(shLabelStyle.Render("💡 Safer alternative:"))
			boxContent.WriteString("\n")
			boxContent.WriteString(shNormalStyle.Width(58).Render("  " + m.analysis.alternative))
		}

		b.WriteString(riskBoxStyle.Render(boxContent.String()))
		b.WriteString("\n\n")
		b.WriteString(shHelpStyle.Render("y/⏎ execute  •  n/esc cancel"))

	case shieldStateExecuting:
		b.WriteString(shNormalStyle.Render("$ " + m.command))
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(shNormalStyle.Render("Running..."))

	case shieldStateDone:
		b.WriteString(shNormalStyle.Render("$ " + m.command))
		b.WriteString("\n")

		if m.execErr == "" {
			output := m.execOutput
			if output == "" {
				output = "(no output)"
			} else {
				lines := strings.Split(output, "\n")
				if len(lines) > 10 {
					lines = lines[:10]
					lines = append(lines, "...")
				}
				output = strings.Join(lines, "\n")
			}
			successBox := shBoxStyle.BorderForeground(lipgloss.Color("42"))
			b.WriteString(successBox.Render(
				shSuccessStyle.Render("✅ Output") + "\n" + shNormalStyle.Render(output),
			))
		} else {
			errLines := strings.Split(m.execOutput, "\n")
			if len(errLines) > 5 {
				errLines = errLines[:5]
			}
			errOutput := strings.Join(errLines, "\n")
			if errOutput == "" {
				errOutput = m.execErr
			}
			errorBox := shBoxStyle.BorderForeground(lipgloss.Color("196"))
			b.WriteString(errorBox.Render(
				shErrorStyle.Render("❌ Error") + "\n" +
					shNormalStyle.Render(m.execErr) + "\n" +
					shNormalStyle.Render(errOutput),
			))

			if m.errorInterp == "" {
				b.WriteString("\n\n")
				b.WriteString(m.spinner.View())
				b.WriteString(" ")
				b.WriteString(shNormalStyle.Render("Interpreting error..."))
			} else {
				interpBox := shBoxStyle.BorderForeground(lipgloss.Color("205"))
				b.WriteString(interpBox.Render(
					shLabelStyle.Render("🤖 AI Explanation") + "\n\n" +
						shNormalStyle.Width(58).Render(m.errorInterp),
				))
			}
		}

		b.WriteString("\n\n")
		b.WriteString(shHelpStyle.Render("r run another  •  q quit"))
	}

	return tea.NewView(b.String())
}

// ---------------------------------------------------------------------------
// Claude helpers
// ---------------------------------------------------------------------------

func newAnthropicClient() anthropic.Client {
	opts := []option.RequestOption{
		option.WithAPIKey(os.Getenv("ANTHROPIC_AUTH_TOKEN")),
	}
	if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return anthropic.NewClient(opts...)
}

func defaultModel() anthropic.Model {
	if m := os.Getenv("ANTHROPIC_DEFAULT_SONNET_MODEL"); m != "" {
		return anthropic.Model(m)
	}
	return anthropic.ModelClaudeHaiku4_5_20251001
}

// ---------------------------------------------------------------------------
// Claude commands
// ---------------------------------------------------------------------------

func analyzeCommand(command string) tea.Cmd {
	return func() tea.Msg {
		if os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" {
			return analysisMsg{
				risk:        "UNKNOWN",
				explanation: "ANTHROPIC_AUTH_TOKEN not set.",
				warning:     "none",
				alternative: "none",
			}
		}

		client := newAnthropicClient()
		prompt := fmt.Sprintf(`You are a shell command safety analyzer. Analyze the following command.

Command: %s

Respond ONLY in this exact format (no markdown, no extra text):
RISK: LOW|MEDIUM|HIGH|CRITICAL
EXPLANATION: <one sentence: what this command does>
WARNING: <one sentence: what could go wrong, or "none">
ALTERNATIVE: <safer equivalent, or "none">`, command)

		msg, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
			Model:     defaultModel(),
			MaxTokens: 256,
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		if err != nil {
			return analysisMsg{err: err}
		}
		if len(msg.Content) == 0 {
			return analysisMsg{err: fmt.Errorf("empty response from Claude")}
		}
		return parseAnalysis(msg.Content[0].Text)
	}
}

func executeCommand(command string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return execMsg{}
		}
		cmd := exec.Command(parts[0], parts[1:]...)
		output, err := cmd.CombinedOutput()
		return execMsg{output: strings.TrimSpace(string(output)), err: err}
	}
}

func interpretError(command, errOutput string) tea.Cmd {
	return func() tea.Msg {
		if os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" {
			return errorInterpMsg{text: "Set ANTHROPIC_AUTH_TOKEN to get error explanations."}
		}

		client := newAnthropicClient()
		prompt := fmt.Sprintf(`A shell command failed. Explain why and how to fix it in 2-3 plain sentences. Be direct and specific. Do not use markdown.

Command: %s
Error output:
%s`, command, errOutput)

		msg, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
			Model:     defaultModel(),
			MaxTokens: 256,
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		if err != nil {
			return errorInterpMsg{text: fmt.Sprintf("Could not interpret error: %v", err)}
		}
		if len(msg.Content) == 0 {
			return errorInterpMsg{text: "Empty response from Claude."}
		}
		return errorInterpMsg{text: msg.Content[0].Text}
	}
}
