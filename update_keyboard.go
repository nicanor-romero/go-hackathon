package main

import (
	tea "charm.land/bubbletea/v2"
)

// handleKeyPress processes keyboard input based on current state
func handleKeyPress(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		// Quit from main menu or error state
		if m.state == stateDeploymentList || m.state == stateError {
			return m, tea.Quit
		}
		// In other states, 'q' does nothing (use esc to go back)

	case "esc":
		// Navigate back
		switch m.state {
		case stateNamespaceSelector:
			// Go back to deployment list with current namespace
			m.state = stateLoading
			m.cursor = 0
			return m, loadDeployments(m.clientset, m.namespace)
		case stateDeploymentDetail:
			m.state = stateDeploymentList
			m.pods = nil
			m.selectedPod = 0
			m.cursor = m.selectedDeployment // Restore cursor to deployment
		case statePodDiagnosis:
			m.state = stateDeploymentDetail
			m.diagnosis = nil
			m.cursor = m.selectedPod // Restore cursor to pod
		}
		return m, nil

	case "r":
		// Refresh
		if m.state == stateDeploymentList {
			m.state = stateLoading
			return m, loadDeployments(m.clientset, m.namespace)
		} else if m.state == stateNamespaceSelector {
			m.state = stateLoading
			return m, loadNamespaces(m.clientset)
		}

	case "n":
		// Open namespace selector
		if m.state == stateDeploymentList {
			m.state = stateLoading
			return m, loadNamespaces(m.clientset)
		}

	case "?":
		// TODO: Show help screen
		return m, nil
	}

	// State-specific keyboard handling
	switch m.state {
	case stateNamespaceSelector:
		return handleNamespaceSelectorKeys(m, msg)
	case stateDeploymentList:
		return handleDeploymentListKeys(m, msg)
	case stateDeploymentDetail:
		return handleDeploymentDetailKeys(m, msg)
	case statePodDiagnosis:
		return handlePodDiagnosisKeys(m, msg)
	}

	return m, nil
}

// handleNamespaceSelectorKeys handles keyboard input in namespace selector view
func handleNamespaceSelectorKeys(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.namespaces)-1 {
			m.cursor++
		}

	case "enter":
		// Switch to selected namespace
		if len(m.namespaces) > 0 && m.cursor < len(m.namespaces) {
			m.namespace = m.namespaces[m.cursor]
			m.state = stateLoading
			m.cursor = 0
			return m, loadDeployments(m.clientset, m.namespace)
		}
	}

	return m, nil
}

// handleDeploymentListKeys handles keyboard input in deployment list view
func handleDeploymentListKeys(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.deployments)-1 {
			m.cursor++
		}

	case "enter":
		// Load pods for selected deployment
		if len(m.deployments) > 0 && m.cursor < len(m.deployments) {
			m.selectedDeployment = m.cursor
			m.state = stateLoading
			m.cursor = 0 // Reset cursor for pod list
			deploymentName := m.deployments[m.selectedDeployment].Name
			return m, loadPodsForDeployment(m.clientset, m.namespace, deploymentName)
		}
	}

	return m, nil
}

// handleDeploymentDetailKeys handles keyboard input in deployment detail view
func handleDeploymentDetailKeys(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.pods)-1 {
			m.cursor++
		}

	case "d", "enter":
		// Diagnose selected pod
		if len(m.pods) > 0 && m.cursor < len(m.pods) {
			m.selectedPod = m.cursor
			m.state = stateLoading
			m.cursor = 0 // Reset cursor for diagnosis view
			pod := m.pods[m.selectedPod]
			return m, analyzePod(m.clientset, m.namespace, pod)
		}
	}

	return m, nil
}

// handlePodDiagnosisKeys handles keyboard input in pod diagnosis view
func handlePodDiagnosisKeys(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Currently just esc to go back (handled in main handleKeyPress)
	// Future: add scrolling, copy commands, etc.
	return m, nil
}
