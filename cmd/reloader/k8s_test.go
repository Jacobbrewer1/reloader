package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func testablePod(t *testing.T) *corev1.Pod {
	t.Helper()
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}
}

func Test_KillPod(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		pod := testablePod(t)
		kubeClient := fake.NewClientset(pod)

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		err := killPods(ctx, kubeClient, []*corev1.Pod{pod})
		require.NoError(t, err)
	})

	t.Run("pod not found", func(t *testing.T) {
		t.Parallel()
		kubeClient := fake.NewClientset()

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		err := killPods(ctx, kubeClient, []*corev1.Pod{testablePod(t)})
		require.Error(t, err)
		require.Contains(t, err.Error(), "pods \"test-pod\" not found")
	})

	t.Run("multiple pods not found", func(t *testing.T) {
		t.Parallel()
		pod1 := testablePod(t)
		pod2 := testablePod(t)
		kubeClient := fake.NewClientset()

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		err := killPods(ctx, kubeClient, []*corev1.Pod{pod1, pod2})
		require.EqualError(t, err, "pods \"test-pod\" not found; pods \"test-pod\" not found")
	})
}
