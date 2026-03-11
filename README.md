# Bubble Tea v2 Examples 🎨

Two complete TUI applications demonstrating Bubble Tea v2 capabilities.

## 📁 Project Structure

```
bubble-cli-test/
├── main.go           # Generic menu/submenu example (default)
├── example.go        # Example implementation
├── EXAMPLE.md        # Example documentation
│
└── prom-browser/     # Prometheus metrics browser
    ├── main.go
    ├── prom-tui.go
    ├── prometheus.go
    └── README.md
```

## 🎯 Generic Menu/Submenu Example (Root Directory)

A reusable template for building menu-based TUI applications.

### Features
- ✅ Main menu with 4 items
- ✅ Submenu navigation (3 actions per item)
- ✅ Loading state with spinner
- ✅ Detail view with boxed content
- ✅ Rainbow title animation
- ✅ Same styling as prom-browser

### Run
```bash
go build -o example-tui
./example-tui
```

See [EXAMPLE.md](./EXAMPLE.md) for complete documentation and customization guide.

## 🚀 Prometheus Metrics Browser (`prom-browser/`)

Interactive TUI for browsing Prometheus metrics with advanced features.

### Features
- 🎨 Beautiful UI with colors and animations
- 📊 Browse all available metrics
- 🔍 Real-time filtering
- 📋 Interactive submenu for each metric:
  - 🏷️ **View Labels** - All available fields
  - 📈 **View Statistics** - Min/Max/Avg, cardinality
  - 📝 **PromQL Queries** - Ready-to-use Grafana queries
- ⚠️ Smart metric type detection (summaries vs counters)

### Run
```bash
cd prom-browser
go build -o prom-metrics
./prom-metrics
```

**Note:** Requires VPN connection to access Prometheus.

See [prom-browser/README.md](./prom-browser/README.md) for details.

## 📚 Libraries Used

- **Bubble Tea v2** (`charm.land/bubbletea/v2` v2.0.2) - TUI framework
- **Lip Gloss v2** (`charm.land/lipgloss/v2` v2.0.1) - Terminal styling
- **Bubbles v2** (`charm.land/bubbles/v2` v2.0.0) - Pre-built components

## 🎓 Learning Resources

Both examples demonstrate:
- State management patterns
- Menu/submenu navigation
- Keyboard event handling
- Animation techniques
- Async operations with spinners
- Consistent styling with Lip Gloss

Start with the generic example to learn the basics, then explore prom-browser for advanced patterns!

## 🛠️ Development

Both applications are independent Go modules with their own `go.mod` files.

```bash
# Build generic example
go build -o example-tui

# Build Prometheus browser
cd prom-browser && go build -o prom-metrics
```
