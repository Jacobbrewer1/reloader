package main

import (
	"context"

	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func killPods(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	pods []*corev1.Pod,
) error {
	var multiErr error
	for _, pod := range pods {
		if err := kubeClient.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
			multiErr = multierr.Append(multiErr, err)
		}
	}
	return multiErr
}
