package main

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// initialModel creates the initial application model
func initialModel() (model, error) {
	// Create Kubernetes client
	clientset, context, namespace, err := newK8sClient()
	if err != nil {
		return model{}, err
	}

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := model{
		state:              stateLoading,
		clientset:          clientset,
		namespace:          namespace,
		context:            context,
		deployments:        []appsv1.Deployment{},
		pods:               []corev1.Pod{},
		selectedDeployment: 0,
		selectedPod:        0,
		cursor:             0,
		spinner:            s,
	}

	return m, nil
}

// Init initializes the Bubbletea program
func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadDeployments(m.clientset, m.namespace),
	)
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case namespacesLoadedMsg:
		m.namespaces = msg.namespaces
		m.state = stateNamespaceSelector
		m.cursor = 0
		// Find current namespace in list
		for i, ns := range m.namespaces {
			if ns == m.namespace {
				m.cursor = i
				break
			}
		}
		return m, nil

	case deploymentsLoadedMsg:
		m.deployments = msg.deployments
		m.state = stateDeploymentList
		return m, nil

	case podsLoadedMsg:
		m.pods = msg.pods
		m.state = stateDeploymentDetail
		return m, nil

	case diagnosisReadyMsg:
		m.diagnosis = msg.diagnosis
		m.state = statePodDiagnosis
		return m, nil

	case errorMsg:
		m.err = msg.err
		m.state = stateError
		return m, nil

	case tea.KeyMsg:
		return handleKeyPress(m, msg)
	}

	// Update spinner if loading
	if m.state == stateLoading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the UI
func (m model) View() tea.View {
	var content string

	switch m.state {
	case stateLoading:
		content = renderLoading(m)
	case stateNamespaceSelector:
		content = renderNamespaceSelector(m)
	case stateDeploymentList:
		content = renderDeploymentList(m)
	case stateDeploymentDetail:
		content = renderDeploymentDetail(m)
	case statePodDiagnosis:
		content = renderPodDiagnosis(m)
	case stateError:
		content = renderError(m)
	default:
		content = "Unknown state"
	}

	return tea.NewView(content)
}
