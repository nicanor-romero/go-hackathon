# Generic Menu/Submenu TUI Example

This is a reusable template for building menu-based TUI applications with Bubble Tea v2.

## File: `example.go`

A complete, working example demonstrating:

### Features

✅ **Main Menu** - 4 items with icons
✅ **Submenu Navigation** - 3 actions per item
✅ **Loading State** - Animated spinner
✅ **Detail View** - Boxed information display
✅ **Animations** - Rainbow title, pulsing colors
✅ **Consistent Styling** - Matches prom-metrics look & feel

### Pattern Structure

```
┌─ Main Menu ───────────────┐
│ 📊 Dashboard              │
│ ⚙️  Settings              │
│ 📈 Reports                │
│ ❓ Help                   │
└───────────────────────────┘
         ↓ Enter
┌─ Submenu ────────────────┐
│ 📋 Dashboard              │
│                           │
│ 👁️  View Details          │
│ ✏️  Edit                   │
│ 🗑️  Delete                 │
└───────────────────────────┘
         ↓ Enter
┌─ Loading ────────────────┐
│ ⏳ Loading Dashboard      │
│ ⠋ Please wait...         │
└───────────────────────────┘
         ↓ Auto
┌─ Detail ─────────────────┐
│ 📄 Dashboard - Details    │
│                           │
│ ╭────────────────────╮   │
│ │ Information:       │   │
│ │   Item: Dashboard  │   │
│ │   Action: View     │   │
│ │   Status: Active   │   │
│ ╰────────────────────╯   │
└───────────────────────────┘
```

## Code Structure

### States
```go
type exampleState int

const (
    exampleStateMainMenu
    exampleStateSubmenu
    exampleStateLoading
    exampleStateDetail
)
```

### Model
```go
type exampleModel struct {
    state         exampleState
    mainCursor    int          // Main menu position
    submenuCursor int          // Submenu position
    selectedItem  string       // Currently selected item
    spinner       spinner.Model
    frame         int          // Animation frame
}
```

### Key Methods

**Init()** - Start spinner and animation tick
**Update()** - Handle keyboard input and state transitions
**View()** - Render UI based on current state

## Customization Guide

### 1. Change Menu Items

Edit the items array in the main menu view:
```go
items := []struct {
    icon  string
    label string
}{
    {"📊", "Dashboard"},
    {"⚙️", "Settings"},
    // Add your items here
}
```

### 2. Change Submenu Actions

Edit the options array in submenu view:
```go
options := []struct {
    icon string
    name string
    desc string
}{
    {"👁️", "View Details", "See detailed information"},
    // Add your actions here
}
```

### 3. Customize Colors

All styles are defined at the top:
```go
exampleTitleStyle    // Title color (pink)
exampleSelectedStyle // Selected item (green)
exampleRegularStyle  // Regular items (gray)
exampleBoxStyle      // Box borders (blue)
exampleLabelStyle    // Labels (yellow)
exampleHelpStyle     // Help text (dark gray)
```

### 4. Add Real Functionality

Replace the mock loading with actual logic:
```go
case "enter":
    if m.state == exampleStateSubmenu {
        m.state = exampleStateLoading
        return m, func() tea.Msg {
            // Do real work here
            result := doSomething(m.selectedItem)
            return result
        }
    }
```

## Running the Example

Simply run with the `example` argument:

```bash
./prom-metrics example
```

Or to run the Prometheus TUI (default):

```bash
./prom-metrics
```

## Key Concepts

### State Management
Each screen is a state. Transitions happen on key events.

### Cursor Management
Two cursors track position in main menu and submenu independently.

### Animation
`frame` counter increments every 100ms for rainbow effects.

### Styling
Lip Gloss styles are defined once and reused throughout.

## Tips

- **Keep states simple** - One state = one screen
- **Use descriptive names** - Make code self-documenting
- **Separate concerns** - View logic in View(), state in Update()
- **Reuse styles** - Define once, apply everywhere
- **Add context** - Description text helps users understand options

## Next Steps

Use this as a foundation to build:
- Configuration tools
- Dashboard viewers
- Interactive installers
- Admin panels
- File browsers
- Any menu-driven TUI!

Happy building! 🚀
