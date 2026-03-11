# AGENTS.md - Go Hackathon TUI Project

This document provides coding guidelines for AI agents working on this Bubbletea v2 TUI application.

## Project Overview

**Type:** Go terminal user interface (TUI) application using Bubbletea v2 framework  
**Module:** `github.com/masmovil/mm-monorepo/tools/swe/bubble-cli-test`  
**Go Version:** 1.25.0+  
**Architecture:** Elm-based state management with view/update/model pattern

## Build, Test, and Lint Commands

### Building
```bash
# Build the application
go build -o example-tui

# Run the application
./example-tui

# Build and run in one command
go build -o example-tui && ./example-tui
```

### Testing
```bash
# Run all tests (when tests exist)
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a single test file
go test -v ./path/to/file_test.go

# Run a specific test function
go test -v -run TestFunctionName

# Run tests with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting and Formatting
```bash
# Format code (always run before committing)
go fmt ./...

# Run go vet
go vet ./...

# Install and run golangci-lint (recommended)
golangci-lint run
golangci-lint run --fix

# Check imports
goimports -l .
goimports -w .
```

### Dependencies
```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify

# Update dependency
go get -u charm.land/bubbletea/v2@latest
```

## Code Style Guidelines

### File Organization

Projects should follow this structure:
```
your-app/
├── main.go              # Entry point only (~5-21 lines)
├── types.go             # Type definitions, structs, enums, constants
├── model.go             # Model initialization & layout calculation
├── update.go            # Message dispatcher
├── update_keyboard.go   # Keyboard event handling
├── update_mouse.go      # Mouse event handling
├── view.go              # View rendering & layouts
├── styles.go            # Lipgloss style definitions
├── config.go            # Configuration management (optional)
├── go.mod               # Go module definition
└── .agents/skills/      # Bundled AI skills
```

**Rules:**
- Keep `main.go` minimal (entry point only)
- One file, one responsibility
- Maximum file size: 800 lines (ideally <500)
- Group related functionality into separate files

### Imports

**Standard order (groups separated by blank lines):**
```go
package main

import (
	// Standard library
	"fmt"
	"os"
	"strings"
	"time"

	// External dependencies
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)
```

**Key dependencies:**
- `charm.land/bubbletea/v2` - TUI framework (alias as `tea`)
- `charm.land/lipgloss/v2` - Terminal styling
- `charm.land/bubbles/v2` - Pre-built components (spinner, list, etc.)

### Naming Conventions

**Types:**
- Use PascalCase for exported types: `ExampleModel`, `ExampleState`
- Use camelCase for unexported types: `exampleModel`, `exampleState`
- State types as enums with prefix: `exampleStateMainMenu`, `exampleStateLoading`

**Variables:**
- Use camelCase: `mainCursor`, `submenuCursor`, `selectedItem`
- Single letter OK in tight loops: `for i, item := range items`
- Descriptive names for clarity: `maxTextWidth` not `mtw`

**Functions:**
- Use camelCase for unexported: `initialExampleModel()`, `exampleTick()`
- Use PascalCase for exported: `Init()`, `Update()`, `View()`
- Bubbletea interface methods: `Init()`, `Update(msg tea.Msg)`, `View()`

**Constants:**
- Use camelCase for unexported: `const maxRetries = 3`
- Use PascalCase for exported: `const MaxConnections = 100`

**Styles:**
- Prefix with component name: `exampleTitleStyle`, `exampleSelectedStyle`
- Descriptive suffixes: `Style`, `Border`, `Color`

### Types and Structs

**State enums using iota:**
```go
type exampleState int

const (
	exampleStateMainMenu exampleState = iota
	exampleStateSubmenu
	exampleStateLoading
	exampleStateDetail
)
```

**Model struct:**
```go
type exampleModel struct {
	// State
	state         exampleState
	
	// UI state
	mainCursor    int
	submenuCursor int
	selectedItem  string
	
	// Bubbletea components
	spinner       spinner.Model
	
	// Animation
	frame         int
}
```

**Custom messages:**
```go
type exampleTickMsg time.Time

func exampleTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return exampleTickMsg(t)
	})
}
```

### Error Handling

**Basic pattern:**
```go
if err != nil {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
```

**In Update() method:**
```go
case errMsg:
	m.err = msg
	return m, nil
```

**Never panic** - always handle errors gracefully and provide user feedback

## Bubbletea-Specific Guidelines

### The Elm Architecture

**Three core methods:**
```go
func (m model) Init() tea.Cmd {
	// Initialize and return commands
	return tea.Batch(m.spinner.Tick, tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle messages and update state
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle keyboard
	case tea.MouseMsg:
		// Handle mouse
	}
	return m, nil
}

func (m model) View() tea.View {
	// Render the UI
	return tea.NewView(content)
}
```

### The 4 Golden Rules for TUI Layout

**CRITICAL:** Follow these rules to prevent layout bugs (from `.agents/skills/bubbletea/references/golden-rules.md`):

1. **Always Account for Borders** - Subtract 2 from height BEFORE rendering panels
   ```go
   contentHeight := m.height - titleLines - statusLines - 2 // -2 for borders!
   ```

2. **Never Auto-Wrap in Bordered Panels** - Always truncate text explicitly
   ```go
   maxTextWidth := panelWidth - 4 // -2 borders, -2 padding
   title = truncateString(title, maxTextWidth)
   ```

3. **Match Mouse Detection to Layout** - Use X for horizontal, Y for vertical
   ```go
   if m.shouldUseVerticalStack() {
       // Use msg.Y coordinates
   } else {
       // Use msg.X coordinates
   }
   ```

4. **Use Weights, Not Pixels** - Proportional layouts scale perfectly
   ```go
   leftWidth := (availableWidth * leftWeight) / totalWeight
   rightWidth := availableWidth - leftWidth
   ```

### Lipgloss Styling

**Define styles as package variables:**
```go
var (
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	boxStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		MarginTop(1)
)
```

**Common color palette:**
- Pink/Magenta: `"205"` (titles, highlights)
- Green: `"42"` (selected items, success)
- Gray: `"252"` (regular text)
- Blue: `"63"` (borders, accents)
- Yellow: `"220"` (labels, warnings)
- Dim: `"241"` (help text)

## Best Practices

### State Management
- Use typed enums for states (not strings)
- Keep state transitions explicit in `Update()`
- One state = one view mode

### Keyboard Handling
```go
case tea.KeyMsg:
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit  // Always handle quit
	case "q":
		// Quit from main menu only
	case "esc":
		// Navigate back
	case "enter":
		// Confirm/select
	case "up", "k":
		// Vim-style navigation
	case "down", "j":
		// Vim-style navigation
	}
```

### Animation
```go
// Tick-based animation
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// In Update()
case tickMsg:
	m.frame++
	return m, tick()

// In View()
colorIdx := m.frame % len(colors)
```

### Async Operations
```go
case "enter":
	m.state = stateLoading
	return m, func() tea.Msg {
		time.Sleep(1 * time.Second)
		// Perform operation
		return loadedMsg{data: result}
	}
```

## Common Patterns

### Rainbow/Animated Text
```go
titleColors := []string{"196", "202", "208", "214", "220", "226"}
for i, char := range text {
	colorIdx := (m.frame + i) % len(titleColors)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(titleColors[colorIdx]))
	animatedText.WriteString(style.Render(string(char)))
}
```

### Menu Navigation
```go
// Cursor movement
case "up", "k":
	if m.cursor > 0 {
		m.cursor--
	}
case "down", "j":
	if m.cursor < maxItems-1 {
		m.cursor++
	}

// Selection indicator
if i == m.cursor {
	fmt.Fprintf(&b, " ▶ %s", item)  // Selected
} else {
	fmt.Fprintf(&b, "   %s", item)  // Regular
}
```

### Loading States with Spinner
```go
s := spinner.New()
s.Spinner = spinner.Dot
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

// In Init()
return m.spinner.Tick

// In Update() during loading
m.spinner, cmd = m.spinner.Update(msg)
return m, cmd

// In View()
fmt.Fprintf(&b, "%s Loading...", m.spinner.View())
```

## Additional Resources

- **Bubbletea Skill:** `.agents/skills/bubbletea/SKILL.md` - Comprehensive TUI development guide
- **Golden Rules:** `.agents/skills/bubbletea/references/golden-rules.md` - Critical layout patterns
- **Components:** `.agents/skills/bubbletea/references/components.md` - Reusable component catalog
- **Troubleshooting:** `.agents/skills/bubbletea/references/troubleshooting.md` - Common issues & fixes

## Notes for Agents

- This is a hackathon/example project - prioritize working code over perfection
- No tests currently exist - add them if time permits
- No CI/CD setup - run `go build` locally to verify
- README mentions `prom-browser/` but it doesn't exist - ignore those references
- Always check `.agents/skills/bubbletea/` for TUI-specific guidance before implementing layouts
- Use the existing `example.go` as a reference for coding style and patterns
