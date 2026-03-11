package main

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// loadPodsForDeployment fetches pods for a specific deployment
func loadPodsForDeployment(clientset *kubernetes.Clientset, namespace, deploymentName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Get deployment to find label selector
		deployment, err := clientset.AppsV1().
			Deployments(namespace).
			Get(ctx, deploymentName, metav1.GetOptions{})

		if err != nil {
			return errorMsg{err: err}
		}

		// Build label selector from deployment
		labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)

		// List pods with matching labels
		pods, err := clientset.CoreV1().
			Pods(namespace).
			List(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})

		if err != nil {
			return errorMsg{err: err}
		}

		return podsLoadedMsg{pods: pods.Items}
	}
}
