package main

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// loadDeployments fetches deployments from the current namespace
func loadDeployments(clientset *kubernetes.Clientset, namespace string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		deployments, err := clientset.AppsV1().
			Deployments(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			return errorMsg{err: err}
		}

		return deploymentsLoadedMsg{deployments: deployments.Items}
	}
}
