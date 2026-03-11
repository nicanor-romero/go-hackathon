package main

import (
	"bytes"
	"context"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// fetchPodLogs retrieves logs from a pod
func fetchPodLogs(clientset *kubernetes.Clientset, namespace, podName string, previous bool) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tailLines := int64(500) // Last 500 lines
	opts := &corev1.PodLogOptions{
		Previous:  previous,
		TailLines: &tailLines,
	}

	req := clientset.CoreV1().
		Pods(namespace).
		GetLogs(podName, opts)

	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
