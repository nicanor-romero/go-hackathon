package main

import (
	"context"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// loadNamespaces fetches all namespaces from the cluster
func loadNamespaces(clientset *kubernetes.Clientset) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		namespaceList, err := clientset.CoreV1().
			Namespaces().
			List(ctx, metav1.ListOptions{})

		if err != nil {
			return errorMsg{err: err}
		}

		// Extract namespace names
		namespaces := make([]string, 0, len(namespaceList.Items))
		for _, ns := range namespaceList.Items {
			namespaces = append(namespaces, ns.Name)
		}

		// Sort alphabetically
		sort.Strings(namespaces)

		return namespacesLoadedMsg{namespaces: namespaces}
	}
}
