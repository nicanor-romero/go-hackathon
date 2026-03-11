package main

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// newK8sClient creates a Kubernetes clientset using the default kubeconfig
func newK8sClient() (*kubernetes.Clientset, string, string, error) {
	// Get kubeconfig path
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, "", "", err
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, "", "", err
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, "", "", err
	}

	// Get current context and namespace
	rawConfig, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, "", "", err
	}

	currentContext := rawConfig.CurrentContext
	namespace := "default"
	if ctx, ok := rawConfig.Contexts[currentContext]; ok {
		if ctx.Namespace != "" {
			namespace = ctx.Namespace
		}
	}

	return clientset, currentContext, namespace, nil
}
