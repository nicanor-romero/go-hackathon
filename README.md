# Monitoring & Tracing Tools TUI 🚀

Beautiful, interactive TUI for browsing Prometheus metrics and Jaeger traces using Bubble Tea v2.

**Note:** Requires VPN connection to access internal services.

## Features

### 🔍 Jaeger Tracing Browser

- 🎨 Browse all microservices from Jaeger
- 🔗 View first-level dependencies for each service
- ⬇️  See which services a service depends on (downstream)
- ⬆️  See which services depend on it (upstream)
- 🔍 Real-time filtering by service name
- ⌨️  Keyboard navigation (vim-style)
- 🎨 Animated UI with rainbow titles

### 📊 Prometheus Metrics Browser

- 🎨 **Beautiful UI** with colors, gradients, and animations
- 🔄 **Animated spinners** during loading states
- 📊 Browse all available metrics from Prometheus
- 🔍 Real-time filtering by metric name
- ⌨️  Keyboard navigation (vim-style)
- 📋 **Interactive submenu** for each metric with 3 options:
  - 🏷️  **View Labels** - Show all available fields/labels
  - 📈 **View Statistics** - Min/Max/Average, cardinality, rate
  - 📝 **PromQL Queries** - Generate ready-to-use queries for Grafana
- ✨ Async loading with visual feedback
- 🎯 Highlighted selection with custom styling
- ⚠️  Smart metric type detection (summaries vs counters/gauges)

## Libraries Used

- **Bubble Tea v2**: TUI framework - provides the event loop, state management, and rendering engine
- **Lip Gloss v2**: Terminal styling - creates beautiful colors, gradients, borders, boxes, and layouts
- **Bubbles v2**: Pre-built components - text input for filtering and animated spinner for loading states

## Run

```bash
go build -o prom-metrics
./prom-metrics
```

On startup, you'll see a menu to choose between:
- 📊 **Prometheus Metrics** - Browse metrics, stats, and PromQL queries
- 🔍 **Jaeger Tracing** - Browse services and dependencies

### Authentication

**Jaeger** requires Google Cloud IAP authentication. To get your cookies:
1. Open https://tools.masstack.com/tracing in your browser
2. Press F12 → Application (or Storage) → Cookies
3. Find these two cookies:
   - `GCP_IAP_UID`
   - `__Host-GCP_IAP_AUTH_TOKEN_`
4. Format them as: `GCP_IAP_UID=<value>; __Host-GCP_IAP_AUTH_TOKEN_=<value>`
5. Paste when prompted in the TUI

**Example:**
```
GCP_IAP_UID=117869924448184822733; __Host-GCP_IAP_AUTH_TOKEN_=ATJDtjRMPPwJU3FwKkY32GKx2UXixZOkmAdNZ78CYLOKKaLietdv5xID5vW7eEVfRorz8jub7vi5vedhZ
```

**Prometheus** works without authentication (VPN only).

📖 **See [JAEGER-AUTH.md](./JAEGER-AUTH.md) for detailed authentication guide with examples.**

## Keybindings

**Main Menu:**
- `↑/k` - Move up
- `↓/j` - Move down
- `Enter` - Select tool
- `q` - Quit

**Jaeger - Services List:**
- `↑/k` - Move cursor up
- `↓/j` - Move cursor down
- `PgUp/PgDn` - Jump 10 items
- `g/G` - Jump to top/bottom
- `Enter` - View dependencies
- `type` - Filter services (live search)
- `q` - Quit

**Jaeger - Dependencies View:**
- `Esc` - Go back to services list
- `q` / `Ctrl+C` - Quit

**Prometheus - Metrics browser:**
- `↑/k` - Move cursor up
- `↓/j` - Move cursor down
- `PgUp/PgDn` - Jump 10 items up/down
- `g/G` - Jump to top/bottom
- `Enter` - Select metric (opens submenu)
- `type` - Filter metrics (live search)
- `q` - Quit

**Submenu (after selecting a metric):**
- `↑/k` - Move up
- `↓/j` - Move down
- `Enter` - Select option
- `Esc` - Go back to metrics list

**Detail views (Labels/Stats/PromQL):**
- `Esc` - Go back to submenu
- `q` / `Ctrl+C` - Quit

## PromQL Queries

The PromQL view generates 6 useful query patterns:
1. **Basic Query** - Current values
2. **Rate (5m)** - Per-second rate for counters
3. **Sum** - Aggregate across all labels
4. **Average by Label** - Group by instance
5. **Top 10** - Highest values
6. **Increase (1h)** - Total increase for counters

Copy any query and paste directly into Grafana!

## How it works

1. **main.go**: Main menu to choose between tools
2. **jaeger.go**: HTTP client for Jaeger API queries
3. **jaeger-tui.go**: Bubble Tea TUI for browsing services and dependencies
4. **prometheus.go**: HTTP client for Prometheus API queries
5. **prom-tui.go**: Bubble Tea TUI for browsing metrics

## Architecture

The application uses a state machine pattern with these states:
- `stateLoading` - Initial metrics fetch
- `stateBrowsing` - Main metrics list with filtering
- `stateSubmenu` - Option selection menu
- `stateLoadingLabels/Stats` - Async data fetching
- `stateShowLabels/Stats/PromQL` - Detail views

Each state has its own view rendering and keyboard handling logic.
