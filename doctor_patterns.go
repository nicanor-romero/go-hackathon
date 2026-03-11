package main

import "regexp"

// ErrorPattern represents a detectable error pattern in logs or pod status
type ErrorPattern struct {
	Name       string
	Regex      *regexp.Regexp
	Severity   Severity
	Category   ErrorCategory
	Suggestion string
	DebugCmds  []string
}

// errorPatterns contains all known error patterns for diagnosis
var errorPatterns = []ErrorPattern{
	// Memory errors
	{
		Name:       "OOM Killed",
		Regex:      regexp.MustCompile(`(?i)(oom|out of memory|killed process)`),
		Severity:   SeverityCritical,
		Category:   CategoryMemory,
		Suggestion: "Increase memory limits in deployment spec. Current pod was killed due to memory exhaustion.",
		DebugCmds: []string{
			"kubectl top pod <pod-name> -n <namespace>",
			"kubectl describe pod <pod-name> -n <namespace> | grep -i memory",
			"kubectl get pod <pod-name> -n <namespace> -o jsonpath='{.spec.containers[*].resources.limits.memory}'",
		},
	},
	{
		Name:       "Memory Allocation Error",
		Regex:      regexp.MustCompile(`(?i)(cannot allocate memory|bad_alloc|OutOfMemoryError)`),
		Severity:   SeverityCritical,
		Category:   CategoryMemory,
		Suggestion: "Application unable to allocate memory. Increase memory limits or investigate memory leaks.",
		DebugCmds: []string{
			"kubectl top pod <pod-name> -n <namespace>",
			"kubectl logs <pod-name> -n <namespace> --previous | grep -i memory",
		},
	},

	// Database connection errors
	{
		Name:       "PostgreSQL Connection Refused",
		Regex:      regexp.MustCompile(`(?i)(connection refused|could not connect to server).*postgres`),
		Severity:   SeverityCritical,
		Category:   CategoryNetwork,
		Suggestion: "Cannot connect to PostgreSQL database. Verify database service is running and accessible.",
		DebugCmds: []string{
			"kubectl get svc -n <namespace> | grep postgres",
			"kubectl get endpoints -n <namespace> | grep postgres",
			"kubectl describe pod <pod-name> -n <namespace> | grep -i env",
		},
	},
	{
		Name:       "MySQL Connection Error",
		Regex:      regexp.MustCompile(`(?i)(can't connect to mysql|error.*:3306|access denied for user)`),
		Severity:   SeverityCritical,
		Category:   CategoryNetwork,
		Suggestion: "Cannot connect to MySQL database. Check service availability and credentials.",
		DebugCmds: []string{
			"kubectl get svc -n <namespace> | grep mysql",
			"kubectl get secret -n <namespace> | grep mysql",
		},
	},
	{
		Name:       "MongoDB Connection Failure",
		Regex:      regexp.MustCompile(`(?i)(failed to connect to.*mongo|connection.*:27017)`),
		Severity:   SeverityCritical,
		Category:   CategoryNetwork,
		Suggestion: "MongoDB connection failed. Verify MongoDB service and credentials.",
		DebugCmds: []string{
			"kubectl get svc -n <namespace> | grep mongo",
			"kubectl get endpoints -n <namespace> | grep mongo",
		},
	},

	// Network errors
	{
		Name:       "Connection Timeout",
		Regex:      regexp.MustCompile(`(?i)(connection.*timed? ?out|dial tcp.*i/o timeout|context deadline exceeded)`),
		Severity:   SeverityWarning,
		Category:   CategoryNetwork,
		Suggestion: "Network timeout detected. Check network policies, service endpoints, or increase timeout values.",
		DebugCmds: []string{
			"kubectl get networkpolicy -n <namespace>",
			"kubectl get svc -n <namespace>",
			"kubectl describe pod <pod-name> -n <namespace>",
		},
	},
	{
		Name:       "DNS Resolution Failure",
		Regex:      regexp.MustCompile(`(?i)(no such host|dns.*resolution.*failed|lookup.*no such host)`),
		Severity:   SeverityWarning,
		Category:   CategoryNetwork,
		Suggestion: "DNS resolution failed. Verify service names and CoreDNS functionality.",
		DebugCmds: []string{
			"kubectl get svc kube-dns -n kube-system",
			"kubectl logs -n kube-system -l k8s-app=kube-dns",
		},
	},

	// Configuration errors
	{
		Name:       "ConfigMap Not Found",
		Regex:      regexp.MustCompile(`(?i)(configmap.*not found|failed to load config)`),
		Severity:   SeverityCritical,
		Category:   CategoryConfig,
		Suggestion: "Required ConfigMap is missing. Create the ConfigMap or verify mount paths.",
		DebugCmds: []string{
			"kubectl get configmap -n <namespace>",
			"kubectl describe pod <pod-name> -n <namespace> | grep -A 10 Volumes",
		},
	},
	{
		Name:       "Environment Variable Missing",
		Regex:      regexp.MustCompile(`(?i)(environment variable.*not set|missing required.*env|undefined.*variable)`),
		Severity:   SeverityCritical,
		Category:   CategoryConfig,
		Suggestion: "Required environment variable is not set. Update deployment with missing env vars.",
		DebugCmds: []string{
			"kubectl describe pod <pod-name> -n <namespace> | grep -A 20 Environment",
			"kubectl get deployment <deployment> -n <namespace> -o yaml | grep -A 10 env:",
		},
	},
	{
		Name:       "File/Config Not Found",
		Regex:      regexp.MustCompile(`(?i)(no such file|cannot open.*config|failed to read.*yaml|config.*does not exist)`),
		Severity:   SeverityWarning,
		Category:   CategoryConfig,
		Suggestion: "Configuration file not found. Verify ConfigMap is mounted correctly.",
		DebugCmds: []string{
			"kubectl exec <pod-name> -n <namespace> -- ls -la /etc/config",
			"kubectl describe configmap <configmap> -n <namespace>",
		},
	},

	// Authentication errors
	{
		Name:       "Authentication Failure",
		Regex:      regexp.MustCompile(`(?i)(authentication failed|unauthorized|401|invalid credentials|access denied)`),
		Severity:   SeverityCritical,
		Category:   CategoryAuth,
		Suggestion: "Authentication failed. Verify credentials in secrets and service account permissions.",
		DebugCmds: []string{
			"kubectl get secret -n <namespace>",
			"kubectl get serviceaccount -n <namespace>",
			"kubectl describe pod <pod-name> -n <namespace> | grep 'Service Account'",
		},
	},
	{
		Name:       "Permission Denied",
		Regex:      regexp.MustCompile(`(?i)(permission denied|forbidden|403|not authorized)`),
		Severity:   SeverityCritical,
		Category:   CategoryAuth,
		Suggestion: "Insufficient permissions. Check RBAC roles and bindings for service account.",
		DebugCmds: []string{
			"kubectl describe rolebinding -n <namespace>",
			"kubectl describe clusterrolebinding",
		},
	},

	// Image errors
	{
		Name:       "Image Pull Error",
		Regex:      regexp.MustCompile(`(?i)(failed to pull image|image.*not found|manifest.*not found)`),
		Severity:   SeverityCritical,
		Category:   CategoryImage,
		Suggestion: "Cannot pull container image. Verify image name/tag and registry credentials.",
		DebugCmds: []string{
			"kubectl describe pod <pod-name> -n <namespace> | grep -A 5 Events",
			"kubectl get secret -n <namespace> | grep docker",
		},
	},

	// Probe failures
	{
		Name:       "Liveness Probe Failed",
		Regex:      regexp.MustCompile(`(?i)liveness probe failed`),
		Severity:   SeverityWarning,
		Category:   CategoryProbe,
		Suggestion: "Liveness probe is failing. Container will be restarted. Check health endpoint and increase timeouts if needed.",
		DebugCmds: []string{
			"kubectl describe pod <pod-name> -n <namespace> | grep -A 10 Liveness",
			"kubectl logs <pod-name> -n <namespace> | grep -i health",
		},
	},
	{
		Name:       "Readiness Probe Failed",
		Regex:      regexp.MustCompile(`(?i)readiness probe failed`),
		Severity:   SeverityWarning,
		Category:   CategoryProbe,
		Suggestion: "Readiness probe failing. Pod will not receive traffic. Verify health endpoint response.",
		DebugCmds: []string{
			"kubectl describe pod <pod-name> -n <namespace> | grep -A 10 Readiness",
			"kubectl get endpoints -n <namespace>",
		},
	},

	// Resource errors
	{
		Name:       "CPU Throttling",
		Regex:      regexp.MustCompile(`(?i)(cpu.*throttl|throttling.*cpu)`),
		Severity:   SeverityWarning,
		Category:   CategoryResource,
		Suggestion: "CPU throttling detected. Consider increasing CPU limits.",
		DebugCmds: []string{
			"kubectl top pod <pod-name> -n <namespace>",
			"kubectl get pod <pod-name> -n <namespace> -o jsonpath='{.spec.containers[*].resources}'",
		},
	},

	// Application-specific errors
	{
		Name:       "Port Already in Use",
		Regex:      regexp.MustCompile(`(?i)(address already in use|bind.*address already in use|port.*already allocated)`),
		Severity:   SeverityCritical,
		Category:   CategoryConfig,
		Suggestion: "Port conflict detected. Check if another process is using the same port or if multiple containers are binding to the same port.",
		DebugCmds: []string{
			"kubectl describe pod <pod-name> -n <namespace> | grep -A 5 Ports",
			"kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A 10 ports:",
		},
	},
}
