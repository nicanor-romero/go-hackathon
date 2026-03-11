package main

import (
	"time"

	"charm.land/bubbles/v2/spinner"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Application states
type appState int

const (
	stateLoading appState = iota
	stateNamespaceSelector
	stateDeploymentList
	stateDeploymentDetail
	statePodDiagnosis
	stateError
)

// Model represents the application state
type model struct {
	// Application state
	state appState

	// Kubernetes client
	clientset *kubernetes.Clientset
	namespace string
	context   string

	// Data
	namespaces         []string
	deployments        []appsv1.Deployment
	pods               []corev1.Pod
	selectedDeployment int
	selectedPod        int
	diagnosis          *DiagnosisResult

	// UI state
	cursor int
	width  int
	height int

	// Components
	spinner spinner.Model
	err     error
}

// Messages for Bubbletea
type namespacesLoadedMsg struct {
	namespaces []string
}

type deploymentsLoadedMsg struct {
	deployments []appsv1.Deployment
}

type podsLoadedMsg struct {
	pods []corev1.Pod
}

type diagnosisReadyMsg struct {
	diagnosis *DiagnosisResult
}

type errorMsg struct {
	err error
}

type tickMsg time.Time

// Severity levels for errors
type Severity int

const (
	SeverityCritical Severity = iota
	SeverityWarning
	SeverityInfo
)

func (s Severity) String() string {
	switch s {
	case SeverityCritical:
		return "CRITICAL"
	case SeverityWarning:
		return "WARNING"
	case SeverityInfo:
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

func (s Severity) Icon() string {
	switch s {
	case SeverityCritical:
		return "🔴"
	case SeverityWarning:
		return "🟡"
	case SeverityInfo:
		return "🔵"
	default:
		return "⚪"
	}
}

// Error categories
type ErrorCategory int

const (
	CategoryMemory ErrorCategory = iota
	CategoryNetwork
	CategoryConfig
	CategoryAuth
	CategoryImage
	CategoryProbe
	CategoryResource
	CategoryUnknown
)

func (c ErrorCategory) String() string {
	switch c {
	case CategoryMemory:
		return "Memory"
	case CategoryNetwork:
		return "Network"
	case CategoryConfig:
		return "Configuration"
	case CategoryAuth:
		return "Authentication"
	case CategoryImage:
		return "Image"
	case CategoryProbe:
		return "Health Probe"
	case CategoryResource:
		return "Resource"
	default:
		return "Unknown"
	}
}

// DetectedError represents an error found in logs or pod status
type DetectedError struct {
	Pattern    string
	Severity   Severity
	Category   ErrorCategory
	Message    string
	LineNumber int
	Suggestion string
	DebugCmds  []string
}

// DiagnosisResult contains the analysis of a pod
type DiagnosisResult struct {
	PodName      string
	PodStatus    string
	RestartCount int32
	ExitCode     int32
	Errors       []DetectedError
	Warnings     []DetectedError
	HealthScore  int
	RootCause    string
	Actions      []string
}
