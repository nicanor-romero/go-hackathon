package main

import (
	"fmt"
	"strings"
)

// renderLoading renders the loading state
func renderLoading(m model) string {
	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("🚀 KubeDoctor"))
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View())
	b.WriteString(" ")
	b.WriteString(regularStyle.Render("Loading..."))
	b.WriteString("\n")
	return b.String()
}

// renderError renders the error state
func renderError(m model) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(errorStyle.Render("❌ Error"))
	b.WriteString("\n\n")
	b.WriteString(regularStyle.Render(fmt.Sprintf("Failed to connect to Kubernetes cluster:\n%v", m.err)))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Make sure your kubeconfig is configured correctly and the cluster is accessible."))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press 'q' to quit"))
	b.WriteString("\n")
	return b.String()
}

// renderNamespaceSelector renders the namespace selector view
func renderNamespaceSelector(m model) string {
	var b strings.Builder

	// Title bar
	b.WriteString("\n")
	title := fmt.Sprintf("🔧 Namespace Selector - Current: %s", m.namespace)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Label
	b.WriteString(labelStyle.Render("📁 NAMESPACES"))
	b.WriteString(fmt.Sprintf(" (%d)\n\n", len(m.namespaces)))

	// Namespace list
	if len(m.namespaces) == 0 {
		b.WriteString(dimStyle.Render("  No namespaces found"))
		b.WriteString("\n")
	} else {
		// Calculate scrolling window (show 20 items at a time)
		maxVisible := 20
		start := 0
		end := len(m.namespaces)

		if len(m.namespaces) > maxVisible {
			// Keep cursor in view
			if m.cursor >= maxVisible {
				start = m.cursor - maxVisible + 1
			}
			end = start + maxVisible
			if end > len(m.namespaces) {
				end = len(m.namespaces)
				start = end - maxVisible
				if start < 0 {
					start = 0
				}
			}
		}

		for i := start; i < end; i++ {
			ns := m.namespaces[i]

			// Selection indicator
			cursor := "  "
			if i == m.cursor {
				cursor = selectedStyle.Render("▶ ")
			}

			// Current namespace indicator
			currentMarker := " "
			if ns == m.namespace {
				currentMarker = successStyle.Render("✓ ")
			}

			// Namespace name
			name := ns
			if len(name) > 50 {
				name = name[:47] + "..."
			}

			// Render line
			if i == m.cursor {
				b.WriteString(cursor)
				b.WriteString(selectedStyle.Render(fmt.Sprintf("%-52s %s", name, currentMarker)))
			} else {
				b.WriteString(cursor)
				b.WriteString(regularStyle.Render(fmt.Sprintf("%-52s %s", name, currentMarker)))
			}
			b.WriteString("\n")
		}

		// Show scroll indicator if there are more items
		if len(m.namespaces) > maxVisible {
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(fmt.Sprintf("  Showing %d-%d of %d", start+1, end, len(m.namespaces))))
			b.WriteString("\n")
		}
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(statusBarStyle.Render(" ↑/k up │ ↓/j down │ ⏎ select namespace │ esc back │ r refresh "))
	b.WriteString("\n")

	return b.String()
}

// renderDeploymentList renders the deployment list view
func renderDeploymentList(m model) string {
	var b strings.Builder

	// Title bar
	b.WriteString("\n")
	title := fmt.Sprintf("🚀 KubeDoctor - %s @ %s", m.namespace, m.context)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Label
	b.WriteString(labelStyle.Render("📦 DEPLOYMENTS"))
	b.WriteString(fmt.Sprintf(" (%d)\n\n", len(m.deployments)))

	// Deployments list
	if len(m.deployments) == 0 {
		b.WriteString(dimStyle.Render("  No deployments found in this namespace"))
		b.WriteString("\n")
	} else {
		for i, deployment := range m.deployments {
			ready := deployment.Status.ReadyReplicas
			desired := *deployment.Spec.Replicas

			// Selection indicator
			cursor := "  "
			if i == m.cursor {
				cursor = selectedStyle.Render("▶ ")
			}

			// Health icon
			healthIcon := deploymentHealthIcon(ready, desired)

			// Deployment name and status
			name := deployment.Name
			if len(name) > 40 {
				name = name[:37] + "..."
			}

			status := fmt.Sprintf("(%d/%d)", ready, desired)

			// Render line
			if i == m.cursor {
				b.WriteString(cursor)
				b.WriteString(selectedStyle.Render(fmt.Sprintf("%-42s %s %s", name, status, healthIcon)))
			} else {
				b.WriteString(cursor)
				b.WriteString(regularStyle.Render(fmt.Sprintf("%-42s %s %s", name, status, healthIcon)))
			}
			b.WriteString("\n")
		}
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(statusBarStyle.Render(" ↑/k up │ ↓/j down │ ⏎ view pods │ n namespace │ r refresh │ q quit "))
	b.WriteString("\n")

	return b.String()
}

// renderDeploymentDetail renders the deployment detail with pod list
func renderDeploymentDetail(m model) string {
	var b strings.Builder

	if m.selectedDeployment >= len(m.deployments) {
		return "Invalid deployment selected"
	}

	deployment := m.deployments[m.selectedDeployment]

	// Title
	b.WriteString("\n")
	title := fmt.Sprintf("📦 %s - Pods", deployment.Name)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Deployment info
	ready := deployment.Status.ReadyReplicas
	desired := *deployment.Spec.Replicas
	healthIcon := deploymentHealthIcon(ready, desired)
	b.WriteString(labelStyle.Render("Status: "))
	b.WriteString(regularStyle.Render(fmt.Sprintf("%d/%d replicas ready %s", ready, desired, healthIcon)))
	b.WriteString("\n\n")

	// Pods list
	b.WriteString(labelStyle.Render("PODS:"))
	b.WriteString("\n\n")

	if len(m.pods) == 0 {
		b.WriteString(dimStyle.Render("  No pods found for this deployment"))
		b.WriteString("\n")
	} else {
		for i, pod := range m.pods {
			// Selection indicator
			cursor := "  "
			if i == m.cursor {
				cursor = selectedStyle.Render("▶ ")
			}

			// Pod status
			status := string(pod.Status.Phase)
			if len(pod.Status.ContainerStatuses) > 0 {
				containerStatus := pod.Status.ContainerStatuses[0]
				if containerStatus.State.Waiting != nil {
					status = containerStatus.State.Waiting.Reason
				}
			}

			statusIcon := podStatusIcon(status)

			// Restart count
			restarts := int32(0)
			if len(pod.Status.ContainerStatuses) > 0 {
				restarts = pod.Status.ContainerStatuses[0].RestartCount
			}

			// Pod name
			name := pod.Name
			if len(name) > 35 {
				name = name[:32] + "..."
			}

			// Render line
			podLine := fmt.Sprintf("%-37s %-20s [%d restarts] %s", name, status, restarts, statusIcon)
			if i == m.cursor {
				b.WriteString(cursor)
				b.WriteString(selectedStyle.Render(podLine))
			} else {
				b.WriteString(cursor)
				b.WriteString(regularStyle.Render(podLine))
			}
			b.WriteString("\n")
		}
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(statusBarStyle.Render(" ↑/k up │ ↓/j down │ d diagnose │ ⏎ diagnose │ esc back "))
	b.WriteString("\n")

	return b.String()
}

// renderPodDiagnosis renders the pod diagnosis view with KubeDoctor analysis
func renderPodDiagnosis(m model) string {
	var b strings.Builder

	if m.diagnosis == nil {
		return "No diagnosis available"
	}

	d := m.diagnosis

	// Title
	b.WriteString("\n")
	title := fmt.Sprintf("🔍 Diagnosis: %s", d.PodName)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Health score
	healthColor := successStyle
	if d.HealthScore < 50 {
		healthColor = errorStyle
	} else if d.HealthScore < 80 {
		healthColor = warningStyle
	}
	b.WriteString(labelStyle.Render("Health Score: "))
	b.WriteString(healthColor.Render(fmt.Sprintf("%d/100", d.HealthScore)))
	b.WriteString("\n\n")

	// Root cause
	b.WriteString(labelStyle.Render("Root Cause: "))
	if len(d.Errors) > 0 {
		b.WriteString(errorStyle.Render(d.RootCause))
	} else {
		b.WriteString(successStyle.Render(d.RootCause))
	}
	b.WriteString("\n\n")

	// Critical errors
	if len(d.Errors) > 0 {
		b.WriteString(errorStyle.Render("🔴 CRITICAL ISSUES:"))
		b.WriteString("\n\n")
		for i, err := range d.Errors {
			if i >= 3 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more errors\n", len(d.Errors)-3)))
				break
			}
			b.WriteString(fmt.Sprintf("  %d. ", i+1))
			b.WriteString(errorStyle.Render(err.Pattern))
			b.WriteString(": ")
			b.WriteString(regularStyle.Render(err.Message))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(fmt.Sprintf("     Category: %s | Line: %d\n", err.Category, err.LineNumber)))
			b.WriteString(regularStyle.Render(fmt.Sprintf("     💡 %s\n", err.Suggestion)))
			b.WriteString("\n")
		}
	}

	// Warnings
	if len(d.Warnings) > 0 {
		b.WriteString(warningStyle.Render("🟡 WARNINGS:"))
		b.WriteString("\n\n")
		for i, warn := range d.Warnings {
			if i >= 2 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more warnings\n", len(d.Warnings)-2)))
				break
			}
			b.WriteString(fmt.Sprintf("  %d. ", i+1))
			b.WriteString(warningStyle.Render(warn.Pattern))
			b.WriteString(": ")
			b.WriteString(regularStyle.Render(warn.Message))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(fmt.Sprintf("     💡 %s\n", warn.Suggestion)))
			b.WriteString("\n")
		}
	}

	// Recommended actions
	b.WriteString(labelStyle.Render("📋 RECOMMENDED ACTIONS:"))
	b.WriteString("\n\n")
	for i, action := range d.Actions {
		if i >= 5 {
			break
		}
		b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, action))
	}

	// Debug commands (from first error)
	if len(d.Errors) > 0 && len(d.Errors[0].DebugCmds) > 0 {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("🛠️  DEBUG COMMANDS:"))
		b.WriteString("\n\n")
		for i, cmd := range d.Errors[0].DebugCmds {
			if i >= 3 {
				break
			}
			// Replace placeholders
			cmd = strings.ReplaceAll(cmd, "<pod-name>", d.PodName)
			cmd = strings.ReplaceAll(cmd, "<namespace>", m.namespace)
			if m.selectedDeployment < len(m.deployments) {
				cmd = strings.ReplaceAll(cmd, "<deployment>", m.deployments[m.selectedDeployment].Name)
			}
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %d. %s\n", i+1, cmd)))
		}
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(statusBarStyle.Render(" esc back to pods │ q quit "))
	b.WriteString("\n")

	return b.String()
}
