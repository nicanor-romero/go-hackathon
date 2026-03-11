package main

import (
	"bufio"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// analyzePod performs comprehensive diagnosis of a pod
func analyzePod(clientset *kubernetes.Clientset, namespace string, pod corev1.Pod) tea.Cmd {
	return func() tea.Msg {
		// Fetch logs (current and previous if crashed)
		logs, err := fetchPodLogs(clientset, namespace, pod.Name, false)
		if err != nil {
			// Try previous logs if current fails
			logs, _ = fetchPodLogs(clientset, namespace, pod.Name, true)
		}

		// Analyze pod
		diagnosis := performDiagnosis(pod, logs)

		return diagnosisReadyMsg{diagnosis: diagnosis}
	}
}

// performDiagnosis analyzes pod status and logs to detect issues
func performDiagnosis(pod corev1.Pod, logs string) *DiagnosisResult {
	result := &DiagnosisResult{
		PodName:   pod.Name,
		PodStatus: string(pod.Status.Phase),
		Errors:    []DetectedError{},
		Warnings:  []DetectedError{},
		Actions:   []string{},
	}

	// Analyze container statuses
	analyzeContainerStatus(pod, result)

	// Analyze logs
	analyzeLogs(logs, result)

	// Calculate health score
	result.HealthScore = calculateHealthScore(result)

	// Determine root cause
	result.RootCause = determineRootCause(result)

	// Generate recommended actions
	result.Actions = generateActions(result)

	return result
}

// analyzeContainerStatus checks container states for common issues
func analyzeContainerStatus(pod corev1.Pod, result *DiagnosisResult) {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		result.RestartCount = containerStatus.RestartCount

		// Check if container is waiting
		if containerStatus.State.Waiting != nil {
			reason := containerStatus.State.Waiting.Reason
			message := containerStatus.State.Waiting.Message

			switch reason {
			case "CrashLoopBackOff":
				result.Errors = append(result.Errors, DetectedError{
					Pattern:    "CrashLoopBackOff",
					Severity:   SeverityCritical,
					Category:   CategoryUnknown,
					Message:    fmt.Sprintf("Container is crash looping (%d restarts)", containerStatus.RestartCount),
					Suggestion: "Check logs with --previous flag to see why container crashed. Common causes: missing dependencies, configuration errors, or application bugs.",
					DebugCmds: []string{
						fmt.Sprintf("kubectl logs %s -n %s --previous", pod.Name, pod.Namespace),
						fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace),
					},
				})

			case "ImagePullBackOff", "ErrImagePull":
				result.Errors = append(result.Errors, DetectedError{
					Pattern:    reason,
					Severity:   SeverityCritical,
					Category:   CategoryImage,
					Message:    fmt.Sprintf("Cannot pull image: %s", message),
					Suggestion: "Verify image name and tag are correct. Check if imagePullSecrets are configured for private registries.",
					DebugCmds: []string{
						fmt.Sprintf("kubectl describe pod %s -n %s | grep -A 10 Events", pod.Name, pod.Namespace),
						fmt.Sprintf("kubectl get secret -n %s | grep docker", pod.Namespace),
					},
				})

			case "CreateContainerConfigError":
				result.Errors = append(result.Errors, DetectedError{
					Pattern:    reason,
					Severity:   SeverityCritical,
					Category:   CategoryConfig,
					Message:    fmt.Sprintf("Container config error: %s", message),
					Suggestion: "ConfigMap or Secret referenced in pod spec may be missing.",
					DebugCmds: []string{
						fmt.Sprintf("kubectl get configmap -n %s", pod.Namespace),
						fmt.Sprintf("kubectl get secret -n %s", pod.Namespace),
					},
				})
			}
		}

		// Check if container was terminated
		if containerStatus.State.Terminated != nil {
			exitCode := containerStatus.State.Terminated.ExitCode
			reason := containerStatus.State.Terminated.Reason
			result.ExitCode = exitCode

			if exitCode == 137 {
				result.Errors = append(result.Errors, DetectedError{
					Pattern:    "OOMKilled",
					Severity:   SeverityCritical,
					Category:   CategoryMemory,
					Message:    "Container was killed due to Out Of Memory (exit code 137)",
					Suggestion: "Increase memory limits in deployment spec. Current limits may be too low for the workload.",
					DebugCmds: []string{
						fmt.Sprintf("kubectl top pod %s -n %s", pod.Name, pod.Namespace),
						fmt.Sprintf("kubectl get pod %s -n %s -o jsonpath='{.spec.containers[*].resources.limits.memory}'", pod.Name, pod.Namespace),
					},
				})
			} else if exitCode != 0 {
				result.Errors = append(result.Errors, DetectedError{
					Pattern:    fmt.Sprintf("ExitCode%d", exitCode),
					Severity:   SeverityWarning,
					Category:   CategoryUnknown,
					Message:    fmt.Sprintf("Container exited with code %d (Reason: %s)", exitCode, reason),
					Suggestion: "Check application logs for the cause of non-zero exit code.",
					DebugCmds: []string{
						fmt.Sprintf("kubectl logs %s -n %s --previous", pod.Name, pod.Namespace),
					},
				})
			}
		}

		// Check for high restart count
		if containerStatus.RestartCount > 5 {
			result.Warnings = append(result.Warnings, DetectedError{
				Pattern:    "HighRestartCount",
				Severity:   SeverityWarning,
				Category:   CategoryUnknown,
				Message:    fmt.Sprintf("Container has restarted %d times", containerStatus.RestartCount),
				Suggestion: "Investigate why the container keeps restarting. Check liveness probe configuration and application stability.",
				DebugCmds: []string{
					fmt.Sprintf("kubectl describe pod %s -n %s | grep -A 10 Liveness", pod.Name, pod.Namespace),
				},
			})
		}
	}

	// Check pod conditions
	for _, condition := range pod.Status.Conditions {
		if condition.Status == corev1.ConditionFalse {
			if condition.Type == corev1.PodReady {
				result.Warnings = append(result.Warnings, DetectedError{
					Pattern:    "PodNotReady",
					Severity:   SeverityWarning,
					Category:   CategoryProbe,
					Message:    fmt.Sprintf("Pod not ready: %s", condition.Message),
					Suggestion: "Pod is not ready to receive traffic. Check readiness probe and application startup.",
					DebugCmds: []string{
						fmt.Sprintf("kubectl describe pod %s -n %s | grep -A 10 Readiness", pod.Name, pod.Namespace),
					},
				})
			}
		}
	}
}

// analyzeLogs scans logs for known error patterns
func analyzeLogs(logs string, result *DiagnosisResult) {
	scanner := bufio.NewScanner(strings.NewReader(logs))
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Check each error pattern
		for _, pattern := range errorPatterns {
			if pattern.Regex.MatchString(line) {
				error := DetectedError{
					Pattern:    pattern.Name,
					Severity:   pattern.Severity,
					Category:   pattern.Category,
					Message:    truncateString(line, 100),
					LineNumber: lineNumber,
					Suggestion: pattern.Suggestion,
					DebugCmds:  pattern.DebugCmds,
				}

				if pattern.Severity == SeverityCritical {
					result.Errors = append(result.Errors, error)
				} else {
					result.Warnings = append(result.Warnings, error)
				}

				// Only record first occurrence of each pattern type
				break
			}
		}
	}
}

// calculateHealthScore computes a health score (0-100) based on findings
func calculateHealthScore(result *DiagnosisResult) int {
	score := 100

	// Deduct points for errors and warnings
	score -= len(result.Errors) * 20
	score -= len(result.Warnings) * 10

	// Deduct for high restart count
	if result.RestartCount > 10 {
		score -= 20
	} else if result.RestartCount > 5 {
		score -= 10
	}

	// Deduct for non-zero exit code
	if result.ExitCode != 0 {
		score -= 15
	}

	if score < 0 {
		score = 0
	}

	return score
}

// determineRootCause attempts to identify the primary issue
func determineRootCause(result *DiagnosisResult) string {
	if len(result.Errors) == 0 {
		if len(result.Warnings) == 0 {
			return "No critical issues detected"
		}
		return "Minor issues detected - " + result.Warnings[0].Pattern
	}

	// Find the most severe/critical error
	for _, err := range result.Errors {
		if err.Severity == SeverityCritical {
			return err.Pattern + ": " + err.Message
		}
	}

	return result.Errors[0].Pattern
}

// generateActions creates a list of recommended actions based on diagnosis
func generateActions(result *DiagnosisResult) []string {
	actions := []string{}
	seen := make(map[string]bool)

	// Collect unique suggestions
	for _, err := range result.Errors {
		if !seen[err.Suggestion] {
			actions = append(actions, "🔴 "+err.Suggestion)
			seen[err.Suggestion] = true
		}
	}

	for _, warn := range result.Warnings {
		if !seen[warn.Suggestion] && len(actions) < 5 { // Limit to 5 actions
			actions = append(actions, "🟡 "+warn.Suggestion)
			seen[warn.Suggestion] = true
		}
	}

	if len(actions) == 0 {
		actions = append(actions, "✅ Pod appears healthy. No immediate actions required.")
	}

	return actions
}

// truncateString truncates a string to maxLen
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
