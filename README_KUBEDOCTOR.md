# KubeDoctor 🩺

**Terminal UI for Kubernetes Troubleshooting during On-Call**

KubeDoctor is an interactive TUI (Terminal User Interface) built with Bubbletea v2 that helps you quickly diagnose Kubernetes pod failures during on-call shifts.

## Features

### 🎯 Core Features (MVP)
- ✅ **Namespace Selector** - Switch between namespaces without leaving the TUI ⭐ NEW
- ✅ **Deployment Overview** - View all deployments in your namespace with health indicators
- ✅ **Pod Inspection** - Drill down into pods for each deployment
- ✅ **Intelligent Log Analysis** - Automatic detection of 15+ common error patterns
- ✅ **KubeDoctor Diagnosis** - Smart analysis with suggestions and debug commands
- ✅ **Quick Navigation** - Vim-style keyboard shortcuts (j/k/enter/esc)
- ✅ **Beautiful UI** - Color-coded status indicators and clean layouts

### 🔍 Error Patterns Detected

KubeDoctor automatically detects and provides suggestions for:

**Memory Issues:**
- OOMKilled (Exit code 137)
- Memory allocation failures
- Memory exhaustion

**Network Problems:**
- PostgreSQL/MySQL/MongoDB connection failures
- Connection timeouts
- DNS resolution failures

**Configuration Errors:**
- Missing ConfigMaps or Secrets
- Missing environment variables
- File/config not found

**Authentication:**
- Authentication failures
- Permission denied / RBAC issues

**Image Issues:**
- ImagePullBackOff
- Image not found
- Registry authentication

**Health Probes:**
- Liveness probe failures
- Readiness probe failures

**Resource Issues:**
- CPU throttling
- Insufficient resources (Pending pods)
- Node evictions

**Application Errors:**
- CrashLoopBackOff
- Port conflicts
- Exit code analysis

## Installation

### Prerequisites
- Go 1.25.0 or later
- kubectl configured with access to a Kubernetes cluster
- Valid kubeconfig file (~/.kube/config)

### Build from Source

```bash
# Clone or navigate to project
cd go-hackathon

# Build
go build -o kubedoctor

# Run
./kubedoctor
```

## Usage

### Starting KubeDoctor

```bash
./kubedoctor
```

The application will automatically:
1. Connect to your current kubectl context
2. Use the namespace from your current context
3. Load deployments from that namespace

### Navigation

**Deployment List View:**
- `↑` or `k` - Move up
- `↓` or `j` - Move down
- `Enter` - View pods for selected deployment
- `n` - Open namespace selector ⭐ NEW
- `r` - Refresh deployment list
- `q` - Quit application

**Namespace Selector View:** ⭐ NEW
- `↑` or `k` - Move up
- `↓` or `j` - Move down
- `Enter` - Switch to selected namespace
- `Esc` - Back to deployment list
- `r` - Refresh namespace list

**Pod Detail View:**
- `↑` or `k` - Move up
- `↓` or `j` - Move down
- `d` or `Enter` - Diagnose selected pod
- `Esc` - Back to deployment list

**Pod Diagnosis View:**
- `Esc` - Back to pod list
- `q` - Quit application

### Example Workflow

1. **Start KubeDoctor**
   ```bash
   ./kubedoctor
   ```

2. **Switch Namespace (Optional)** ⭐ NEW
   - Press `n` to open namespace selector
   - Navigate with `j`/`k` or arrow keys
   - Press `Enter` to switch to selected namespace
   - Deployments automatically reload

3. **View Deployments**
   - You'll see a list of all deployments with health indicators:
     - ✅ Green = All replicas ready
     - ⚠️ Yellow = Some replicas not ready
     - ❌ Red = No replicas ready

4. **Select a Deployment**
   - Navigate with `j`/`k` or arrow keys
   - Press `Enter` to view pods

5. **Select a Pod**
   - See all pods for the deployment
   - Status indicators show pod health
   - Restart counts are displayed
   - Press `d` or `Enter` to diagnose

6. **View Diagnosis**
   - Health Score (0-100)
   - Root Cause analysis
   - Critical Issues (🔴)
   - Warnings (🟡)
   - Recommended Actions
   - Debug kubectl commands (ready to copy)

## Example Output

### Deployment List
```
🚀 KubeDoctor - mas-billing-prod @ mas-billing-prod-es

📦 DEPLOYMENTS (8)

▶ api-gateway                              (3/3) ✅
  billing-service                          (2/3) ⚠️
  payment-processor                        (0/3) ❌
  notification-service                     (5/5) ✅
  user-service                             (4/4) ✅
  ...

 ↑/k up │ ↓/j down │ ⏎ view pods │ r refresh │ q quit
```

### Pod Diagnosis
```
🔍 Diagnosis: billing-service-7d9f8-xkj2p

Health Score: 25/100

Root Cause: CrashLoopBackOff: Container is crash looping (15 restarts)

🔴 CRITICAL ISSUES:

  1. PostgreSQL Connection Refused: connection refused postgres:5432
     Category: Network | Line: 42
     💡 Cannot connect to PostgreSQL database. Verify database service 
        is running and accessible.

  2. OOMKilled: Container was killed due to Out Of Memory (exit code 137)
     Category: Memory | Line: 0
     💡 Increase memory limits in deployment spec. Current limits may 
        be too low for the workload.

📋 RECOMMENDED ACTIONS:

  1. 🔴 Cannot connect to PostgreSQL database. Verify database service...
  2. 🔴 Increase memory limits in deployment spec...
  3. 🔴 Check logs with --previous flag to see why container crashed...

🛠️  DEBUG COMMANDS:

  1. kubectl get svc -n mas-billing-prod | grep postgres
  2. kubectl get endpoints -n mas-billing-prod | grep postgres
  3. kubectl describe pod billing-service-7d9f8-xkj2p -n mas-billing-prod

 esc back to pods │ q quit
```

## Architecture

### File Structure

```
go-hackathon/
├── main.go                # Entry point
├── types.go              # Type definitions
├── model.go              # Model initialization
├── update_keyboard.go    # Keyboard handling
├── view.go               # UI rendering
├── styles.go             # Lipgloss styles
├── k8s.go                # Kubernetes client
├── k8s_deployments.go    # Deployment operations
├── k8s_pods.go           # Pod operations
├── k8s_logs.go           # Log fetching
├── doctor.go             # Analysis logic
└── doctor_patterns.go    # Error patterns
```

### Technology Stack

- **Bubbletea v2** - TUI framework (Elm architecture)
- **Lipgloss v2** - Terminal styling
- **Bubbles v2** - TUI components (spinner)
- **client-go** - Official Kubernetes Go client
- **k8s.io/api** - Kubernetes resource types

## Configuration

KubeDoctor uses your existing kubectl configuration:

- **Kubeconfig**: `~/.kube/config` (or `$KUBECONFIG` env variable)
- **Context**: Current context from kubeconfig
- **Namespace**: Namespace from current context (or "default")

To switch contexts/namespaces before running:

```bash
# List contexts
kubectl config get-contexts

# Switch context
kubectl config use-context my-context

# Set namespace
kubectl config set-context --current --namespace=my-namespace

# Run KubeDoctor
./kubedoctor
```

## Troubleshooting

### "Error initializing KubeDoctor: ..."

**Solution:**
1. Verify kubectl is configured: `kubectl cluster-info`
2. Check you have access: `kubectl get deployments`
3. Verify kubeconfig exists: `ls ~/.kube/config`

### "No deployments found"

**Possible causes:**
- Wrong namespace selected
- No deployments exist in namespace
- RBAC permissions insufficient

**Solution:**
```bash
# Check current namespace
kubectl config view --minify | grep namespace

# List deployments
kubectl get deployments

# Switch namespace if needed
kubectl config set-context --current --namespace=<namespace>
```

### Connection timeouts

**Solution:**
- Verify cluster is accessible
- Check VPN connection if required
- Increase timeout in code if needed (default: 15s)

## Contributing

This is a hackathon project. Contributions welcome!

### Adding New Error Patterns

Edit `doctor_patterns.go` and add to the `errorPatterns` slice:

```go
{
    Name:     "Your Error Pattern",
    Regex:    regexp.MustCompile(`(?i)your-regex-pattern`),
    Severity: SeverityCritical,  // or SeverityWarning, SeverityInfo
    Category: CategoryNetwork,    // or other category
    Suggestion: "What to do to fix this issue",
    DebugCmds: []string{
        "kubectl command to run",
        "another helpful command",
    },
},
```

## License

MIT

## Authors

Built for hackathon with ❤️ using Bubbletea v2

## Acknowledgments

- [Charm](https://charm.sh/) for the amazing Bubbletea framework
- [K9s](https://k9scli.io/) for inspiration
- Kubernetes community for client-go

---

**Happy Debugging! 🩺🚀**
