package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// resolveContext finds the kubectl context that matches the cluster value
// from the alert details by checking against available contexts.
func resolveContext(cluster string) (string, error) {
	if cluster == "" {
		return "", fmt.Errorf("no cluster specified in alert details")
	}

	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("listing kubectl contexts: %w", err)
	}

	contexts := strings.Split(strings.TrimSpace(string(output)), "\n")
	clusterLower := strings.ToLower(cluster)

	for _, ctx := range contexts {
		if strings.Contains(strings.ToLower(ctx), clusterLower) {
			return ctx, nil
		}
	}

	return "", fmt.Errorf("no kubectl context found matching cluster %q", cluster)
}

func findPods(namespace, deployment, kubeContext string) ([]Pod, error) {
	pods, err := findPodsByLabel(namespace, deployment, kubeContext)
	if err == nil && len(pods) > 0 {
		return pods, nil
	}

	return findPodsByPrefix(namespace, deployment, kubeContext)
}

func kubectlArgs(kubeContext string, args ...string) []string {
	if kubeContext != "" {
		return append([]string{"--context", kubeContext}, args...)
	}
	return args
}

func findPodsByLabel(namespace, deployment, kubeContext string) ([]Pod, error) {
	args := kubectlArgs(kubeContext,
		"get", "pods",
		"-n", namespace,
		"-l", fmt.Sprintf("app=%s", deployment),
		"-o", "json",
	)
	cmd := exec.Command("kubectl", args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl get pods by label: %w", err)
	}

	return parsePodList(output)
}

func findPodsByPrefix(namespace, deployment, kubeContext string) ([]Pod, error) {
	args := kubectlArgs(kubeContext,
		"get", "pods",
		"-n", namespace,
		"-o", "json",
	)
	cmd := exec.Command("kubectl", args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl get pods: %w", err)
	}

	allPods, err := parsePodList(output)
	if err != nil {
		return nil, err
	}

	var matched []Pod
	for _, p := range allPods {
		if strings.HasPrefix(p.Name, deployment) {
			matched = append(matched, p)
		}
	}

	if len(matched) == 0 {
		return nil, fmt.Errorf("no pods found matching prefix %q in namespace %q", deployment, namespace)
	}

	return matched, nil
}

func parsePodList(data []byte) ([]Pod, error) {
	var result struct {
		Items []struct {
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			Status struct {
				Phase             string `json:"phase"`
				ContainerStatuses []struct {
					Ready bool `json:"ready"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing pod list: %w", err)
	}

	var pods []Pod
	for _, item := range result.Items {
		ready := 0
		total := len(item.Status.ContainerStatuses)
		for _, cs := range item.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
		}

		pods = append(pods, Pod{
			Name:      item.Metadata.Name,
			Namespace: item.Metadata.Namespace,
			Status:    item.Status.Phase,
			Ready:     fmt.Sprintf("%d/%d", ready, total),
		})
	}

	return pods, nil
}

func streamLogs(ctx context.Context, namespace, podName, kubeContext string, logChan chan<- string) {
	defer close(logChan)

	args := kubectlArgs(kubeContext,
		"logs", "-f",
		"-n", namespace,
		podName,
		"--tail=100",
	)
	cmd := exec.CommandContext(ctx, "kubectl", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logChan <- fmt.Sprintf("[error] Failed to create pipe: %v", err)
		return
	}

	if err := cmd.Start(); err != nil {
		logChan <- fmt.Sprintf("[error] Failed to start kubectl logs: %v", err)
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case logChan <- scanner.Text():
		}
	}

	cmd.Wait()
}

// Bubble Tea integration for log streaming

type logLineMsg string
type logDoneMsg struct{}

func waitForLogLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logDoneMsg{}
		}
		return logLineMsg(line)
	}
}
