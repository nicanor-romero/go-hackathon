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
	shieldStateIntro        shieldState = iota // welcome screen
	shieldStateInput                           // type a command
	shieldStateAnalyzing                       // spinner while Claude analyzes original
	shieldStateResult                          // show risk analysis, wait for action
	shieldStateAnalyzingAlt                    // spinner while Claude analyzes alternative
	shieldStateResultAlt                       // show pre-analysis of selected alternative
	shieldStateExecuting                       // spinner while command runs
	shieldStateDone                            // show output (success or failure)
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type analysisMsg struct {
	risk         string   // "LOW" | "MEDIUM" | "HIGH" | "CRITICAL"
	explanation  string
	warning      string
	alternative  string
	invalid      bool     // command looks wrong/has typos
	alternatives []string // suggested corrections when invalid=true
	err          error
}

type execMsg struct {
	output string
	err    error
}

type errorInterpMsg struct {
	text string
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type shieldModel struct {
	state        shieldState
	input        textinput.Model
	spinner      spinner.Model
	command      string
	analysis     analysisMsg
	altCursor    int    // which alternative is selected
	altAnalysis  analysisMsg
	execOutput   string
	execErr      string
	errorInterp  string
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
	shSelectStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("51"))
	shDimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	shInvalidStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
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
		case strings.HasPrefix(line, "INVALID:"):
			val := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "INVALID:")))
			result.invalid = val == "true" || val == "yes"
		case strings.HasPrefix(line, "SUGGESTIONS:"):
			raw := strings.TrimSpace(strings.TrimPrefix(line, "SUGGESTIONS:"))
			for _, s := range strings.Split(raw, "|") {
				s = strings.TrimSpace(s)
				if s != "" && strings.ToLower(s) != "none" {
					result.alternatives = append(result.alternatives, s)
				}
			}
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

	return shieldModel{state: shieldStateIntro, input: ti, spinner: s}
}

func (m shieldModel) Init() tea.Cmd {
	return m.spinner.Tick
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

		case "esc":
			switch m.state {
			case shieldStateResult:
				// Cancel — back to input, clear command
				m.state = shieldStateInput
				m.input.SetValue("")
				return m, m.input.Focus()
			case shieldStateResultAlt:
				// Back to original result
				m.state = shieldStateResult
				return m, nil
			}

		case "enter":
			switch m.state {
			case shieldStateIntro:
				m.state = shieldStateInput
				return m, m.input.Focus()

			case shieldStateInput:
				command := strings.TrimSpace(m.input.Value())
				if command == "" {
					return m, nil
				}
				m.command = command
				m.state = shieldStateAnalyzing
				return m, tea.Batch(m.spinner.Tick, analyzeCommand(command))

			case shieldStateResult:
				if m.analysis.invalid && len(m.analysis.alternatives) > 0 {
					// Enter on result with invalid command selects the highlighted alternative
					selected := m.analysis.alternatives[m.altCursor]
					m.state = shieldStateAnalyzingAlt
					return m, tea.Batch(m.spinner.Tick, analyzeCommandAsAlt(selected))
				}
				// Normal: execute original
				m.state = shieldStateExecuting
				return m, tea.Batch(m.spinner.Tick, executeCommand(m.command))

			case shieldStateResultAlt:
				// Execute the alternative
				selected := m.altAnalysis.alternative
				if selected == "" || strings.ToLower(selected) == "none" {
					// fall back to the raw alt text
					selected = m.analysis.alternatives[m.altCursor]
				}
				m.command = selected
				m.state = shieldStateExecuting
				return m, tea.Batch(m.spinner.Tick, executeCommand(selected))
			}

		case "y", "Y":
			switch m.state {
			case shieldStateResult:
				if !m.analysis.invalid {
					m.state = shieldStateExecuting
					return m, tea.Batch(m.spinner.Tick, executeCommand(m.command))
				}
			case shieldStateResultAlt:
				selected := m.analysis.alternatives[m.altCursor]
				m.command = selected
				m.state = shieldStateExecuting
				return m, tea.Batch(m.spinner.Tick, executeCommand(selected))
			}

		case "n", "N":
			switch m.state {
			case shieldStateResult, shieldStateResultAlt:
				m.state = shieldStateInput
				m.input.SetValue("")
				return m, m.input.Focus()
			}

		case "c":
			// Change — go back to input keeping the current command text
			if m.state == shieldStateResult || m.state == shieldStateResultAlt {
				m.state = shieldStateInput
				m.input.SetValue(m.command)
				return m, m.input.Focus()
			}

		case "up", "k":
			if m.state == shieldStateResult && m.analysis.invalid && m.altCursor > 0 {
				m.altCursor--
			}

		case "down", "j":
			if m.state == shieldStateResult && m.analysis.invalid {
				if m.altCursor < len(m.analysis.alternatives)-1 {
					m.altCursor++
				}
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
		if msg.err != nil {
			m.analysis = msg
			m.state = shieldStateResult
			return m, nil
		}
		m.analysis = msg
		m.altCursor = 0
		m.state = shieldStateResult
		return m, nil

	case altAnalysisMsg:
		m.altAnalysis = analysisMsg(msg)
		m.state = shieldStateResultAlt
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
	case shieldStateAnalyzing, shieldStateAnalyzingAlt, shieldStateExecuting:
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
	case shieldStateIntro:
		introBox := shBoxStyle.BorderForeground(lipgloss.Color("205"))
		var intro strings.Builder
		intro.WriteString(shNormalStyle.Width(56).Render(
			"Antes de ejecutar cualquier comando, esta herramienta lo analiza con IA para detectar riesgos, errores y alternativas más seguras.",
		))
		intro.WriteString("\n\n")
		intro.WriteString(shNormalStyle.Width(56).Render("  • Detecta comandos incorrectos o con typos"))
		intro.WriteString("\n")
		intro.WriteString(shNormalStyle.Width(56).Render("  • Sugiere alternativas y las pre-analiza"))
		intro.WriteString("\n")
		intro.WriteString(shNormalStyle.Width(56).Render("  • Muestra nivel de riesgo antes de ejecutar"))
		b.WriteString(introBox.Render(intro.String()))
		b.WriteString("\n\n")
		b.WriteString(shHelpStyle.Render("⏎ comenzar  •  ctrl+c salir"))

	case shieldStateInput:
		b.WriteString(shLabelStyle.Render("Comando:"))
		b.WriteString("\n")
		b.WriteString(shInputStyle.Render(m.input.View()))
		b.WriteString("\n\n")
		if os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" {
			b.WriteString(shHelpStyle.Render("⚠️  ANTHROPIC_AUTH_TOKEN no está definido — se omitirá el análisis"))
			b.WriteString("\n\n")
		}
		b.WriteString(shHelpStyle.Render("⏎ analizar  •  ctrl+c salir"))

	case shieldStateAnalyzing:
		b.WriteString(shNormalStyle.Render("$ " + m.command))
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(shNormalStyle.Render("Analizando con IA..."))

	case shieldStateResult:
		b.WriteString(shNormalStyle.Render("$ " + m.command))
		b.WriteString("\n")

		if m.analysis.invalid {
			// Command looks wrong — show invalid notice + alternatives
			invalidBox := shBoxStyle.BorderForeground(lipgloss.Color("196"))
			var boxContent strings.Builder
			boxContent.WriteString(shInvalidStyle.Render("⚠️  Comando incorrecto o con errores"))
			boxContent.WriteString("\n\n")
			boxContent.WriteString(shLabelStyle.Render("Problema detectado:"))
			boxContent.WriteString("\n")
			boxContent.WriteString(shNormalStyle.Width(58).Render("  " + m.analysis.explanation))

			if len(m.analysis.alternatives) > 0 {
				boxContent.WriteString("\n\n")
				boxContent.WriteString(shLabelStyle.Render("💡 ¿Quisiste decir?"))
				boxContent.WriteString("\n")
				for i, alt := range m.analysis.alternatives {
					if i == m.altCursor {
						boxContent.WriteString(shSelectStyle.Render(fmt.Sprintf("  ▶ %s", alt)))
					} else {
						boxContent.WriteString(shDimStyle.Render(fmt.Sprintf("    %s", alt)))
					}
					boxContent.WriteString("\n")
				}
			}

			b.WriteString(invalidBox.Render(boxContent.String()))
			b.WriteString("\n\n")
			if len(m.analysis.alternatives) > 0 {
				b.WriteString(shHelpStyle.Render("↑/↓ seleccionar  •  ⏎ pre-analizar  •  c cambiar  •  esc cancelar"))
			} else {
				b.WriteString(shHelpStyle.Render("c cambiar  •  esc cancelar"))
			}
		} else {
			// Valid command — show normal risk result
			riskBoxStyle := shBoxStyle.BorderForeground(riskColor(m.analysis.risk))
			var boxContent strings.Builder
			boxContent.WriteString(
				lipgloss.NewStyle().Bold(true).Foreground(riskColor(m.analysis.risk)).Render(
					fmt.Sprintf("%s Riesgo: %s", riskIcon(m.analysis.risk), m.analysis.risk),
				),
			)
			boxContent.WriteString("\n\n")
			boxContent.WriteString(shLabelStyle.Render("¿Qué hace?"))
			boxContent.WriteString("\n")
			boxContent.WriteString(shNormalStyle.Width(58).Render("  " + m.analysis.explanation))

			if m.analysis.warning != "" && strings.ToLower(m.analysis.warning) != "none" {
				boxContent.WriteString("\n\n")
				boxContent.WriteString(shWarnStyle.Render("⚠️  Advertencia:"))
				boxContent.WriteString("\n")
				boxContent.WriteString(shNormalStyle.Width(58).Render("  " + m.analysis.warning))
			}

			if m.analysis.alternative != "" && strings.ToLower(m.analysis.alternative) != "none" {
				boxContent.WriteString("\n\n")
				boxContent.WriteString(shLabelStyle.Render("💡 Alternativa más segura:"))
				boxContent.WriteString("\n")
				boxContent.WriteString(shNormalStyle.Width(58).Render("  " + m.analysis.alternative))
			}

			b.WriteString(riskBoxStyle.Render(boxContent.String()))
			b.WriteString("\n\n")
			b.WriteString(shHelpStyle.Render("y/⏎ ejecutar  •  c cambiar  •  esc cancelar"))
		}

	case shieldStateAnalyzingAlt:
		selected := ""
		if m.altCursor < len(m.analysis.alternatives) {
			selected = m.analysis.alternatives[m.altCursor]
		}
		b.WriteString(shNormalStyle.Render("$ " + selected))
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(shNormalStyle.Render("Pre-analizando sugerencia con IA..."))

	case shieldStateResultAlt:
		selected := m.analysis.alternatives[m.altCursor]
		b.WriteString(shNormalStyle.Render("$ " + selected))
		b.WriteString("\n")

		previewBox := shBoxStyle.BorderForeground(lipgloss.Color("51"))
		var boxContent strings.Builder
		boxContent.WriteString(shSelectStyle.Render("🔍 Pre-análisis de la sugerencia"))
		boxContent.WriteString("\n\n")
		boxContent.WriteString(
			lipgloss.NewStyle().Bold(true).Foreground(riskColor(m.altAnalysis.risk)).Render(
				fmt.Sprintf("%s Riesgo: %s", riskIcon(m.altAnalysis.risk), m.altAnalysis.risk),
			),
		)
		boxContent.WriteString("\n\n")
		boxContent.WriteString(shLabelStyle.Render("¿Qué hace?"))
		boxContent.WriteString("\n")
		boxContent.WriteString(shNormalStyle.Width(58).Render("  " + m.altAnalysis.explanation))

		if m.altAnalysis.warning != "" && strings.ToLower(m.altAnalysis.warning) != "none" {
			boxContent.WriteString("\n\n")
			boxContent.WriteString(shWarnStyle.Render("⚠️  Advertencia:"))
			boxContent.WriteString("\n")
			boxContent.WriteString(shNormalStyle.Width(58).Render("  " + m.altAnalysis.warning))
		}

		b.WriteString(previewBox.Render(boxContent.String()))
		b.WriteString("\n\n")
		b.WriteString(shHelpStyle.Render("y/⏎ ejecutar  •  esc volver  •  n cancelar"))

	case shieldStateExecuting:
		b.WriteString(shNormalStyle.Render("$ " + m.command))
		b.WriteString("\n\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(shNormalStyle.Render("Ejecutando..."))

	case shieldStateDone:
		b.WriteString(shNormalStyle.Render("$ " + m.command))
		b.WriteString("\n")

		if m.execErr == "" {
			output := m.execOutput
			if output == "" {
				output = "(sin salida)"
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
				shSuccessStyle.Render("✅ Salida") + "\n" + shNormalStyle.Render(output),
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
				b.WriteString(shNormalStyle.Render("Interpretando el error..."))
			} else {
				interpBox := shBoxStyle.BorderForeground(lipgloss.Color("205"))
				b.WriteString(interpBox.Render(
					shLabelStyle.Render("🤖 Explicación IA") + "\n\n" +
						shNormalStyle.Width(58).Render(m.errorInterp),
				))
			}
		}

		b.WriteString("\n\n")
		b.WriteString(shHelpStyle.Render("r ejecutar otro  •  q salir"))
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
				explanation: "ANTHROPIC_AUTH_TOKEN no está definido.",
				warning:     "none",
				alternative: "none",
			}
		}

		client := newAnthropicClient()
		prompt := fmt.Sprintf(`You are a shell command safety analyzer. Analyze the following shell command.

First, check if the command appears to be INVALID — for example, it contains typos, misspelled program names, wrong syntax, or doesn't make sense as a shell command.

Command: %s

Respond ONLY in this exact format (no markdown, no extra text):
RISK: LOW|MEDIUM|HIGH|CRITICAL
EXPLANATION: <one sentence: what this command does>
WARNING: <one sentence: what could go wrong, or "none">
ALTERNATIVE: <safer equivalent command, or "none">
INVALID: true|false
SUGGESTIONS: <if INVALID=true: up to 3 corrected command variants separated by " | ", otherwise "none">`, command)

		msg, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
			Model:     defaultModel(),
			MaxTokens: 300,
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

// altAnalysisMsg wraps analysisMsg so Update can distinguish it.
type altAnalysisMsg analysisMsg

func analyzeCommandAsAlt(command string) tea.Cmd {
	return func() tea.Msg {
		if os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" {
			return altAnalysisMsg{
				risk:        "UNKNOWN",
				explanation: "ANTHROPIC_AUTH_TOKEN no está definido.",
				warning:     "none",
				alternative: "none",
			}
		}

		client := newAnthropicClient()
		prompt := fmt.Sprintf(`You are a shell command safety analyzer. Analyze this command briefly.

Command: %s

Respond ONLY in this exact format (no markdown, no extra text):
RISK: LOW|MEDIUM|HIGH|CRITICAL
EXPLANATION: <one sentence: what this command does>
WARNING: <one sentence: what could go wrong, or "none">
ALTERNATIVE: none
INVALID: false
SUGGESTIONS: none`, command)

		msg, err := client.Messages.New(context.TODO(), anthropic.MessageNewParams{
			Model:     defaultModel(),
			MaxTokens: 200,
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		if err != nil {
			return altAnalysisMsg{err: err}
		}
		if len(msg.Content) == 0 {
			return altAnalysisMsg{err: fmt.Errorf("empty response from Claude")}
		}
		parsed := parseAnalysis(msg.Content[0].Text)
		return altAnalysisMsg(parsed)
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
			return errorInterpMsg{text: "Define ANTHROPIC_AUTH_TOKEN para obtener explicaciones de errores."}
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
			return errorInterpMsg{text: fmt.Sprintf("No se pudo interpretar el error: %v", err)}
		}
		if len(msg.Content) == 0 {
			return errorInterpMsg{text: "Respuesta vacía de Claude."}
		}
		return errorInterpMsg{text: msg.Content[0].Text}
	}
}
