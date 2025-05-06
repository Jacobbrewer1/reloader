package main

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/jacobbrewer1/web/cache"
)

func Test_OnConfigMapUpdate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		logger := slog.New(slog.DiscardHandler)
		bucket := cache.NewFixedHashBucket(1)

		pods := []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod3",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "not-in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
		}

		kubeClient := fake.NewClientset()
		for _, pod := range pods {
			_, err := kubeClient.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)
		}

		informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
		podLister := informerFactory.Core().V1().Pods().Lister()
		informerFactory.Start(ctx.Done())
		informerFactory.WaitForCacheSync(ctx.Done())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "in-bucket",
				Namespace: "default",
			},
			Data: map[string]string{"key": "value"},
		}

		_, err := kubeClient.CoreV1().ConfigMaps("default").Create(ctx, cm, metav1.CreateOptions{})
		require.NoError(t, err)

		handler := onConfigMapUpdate(ctx, logger, bucket, kubeClient, podLister)
		handler(nil, cm)

		// Check that the pods were killed
		for _, pod := range pods[:2] {
			_, err = kubeClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			require.EqualError(t, err, fmt.Sprintf("pods %q not found", pod.Name))
		}

		// Check that the pod that was not in the bucket was not killed
		_, err = kubeClient.CoreV1().Pods(pods[2].Namespace).Get(ctx, pods[2].Name, metav1.GetOptions{})
		require.NoError(t, err)
	})

	t.Run("not-configmap", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		logger := slog.New(slog.DiscardHandler)
		bucket := cache.NewFixedHashBucket(1)

		pods := []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod3",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "not-in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
		}

		kubeClient := fake.NewClientset()
		for _, pod := range pods {
			_, err := kubeClient.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)
		}

		informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
		podLister := informerFactory.Core().V1().Pods().Lister()
		informerFactory.Start(ctx.Done())
		informerFactory.WaitForCacheSync(ctx.Done())

		handler := onConfigMapUpdate(ctx, logger, bucket, kubeClient, podLister)
		handler(nil, testablePod(t))

		for _, pod := range pods {
			p, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, corev1.PodRunning, p.Status.Phase)
		}
	})

	t.Run("not-in-bucket", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		logger := slog.New(slog.DiscardHandler)
		bucket := cache.NewFixedHashBucket(2)
		bucket.Advance() // Advance the bucket to ensure the configmap is not in it

		pods := []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod3",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "not-in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
		}

		kubeClient := fake.NewClientset()
		for _, pod := range pods {
			_, err := kubeClient.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)
		}

		informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
		podLister := informerFactory.Core().V1().Pods().Lister()
		informerFactory.Start(ctx.Done())
		informerFactory.WaitForCacheSync(ctx.Done())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "in-bucket",
				Namespace: "default",
			},
			Data: map[string]string{"key": "value"},
		}

		handler := onConfigMapUpdate(ctx, logger, bucket, kubeClient, podLister)

		handler(nil, cm)

		for _, pod := range pods {
			p, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, corev1.PodRunning, p.Status.Phase)
		}
	})
}

func Test_OnConfigMapDelete(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		logger := slog.New(slog.DiscardHandler)
		bucket := cache.NewFixedHashBucket(1)

		pods := []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod3",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "not-in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
		}

		kubeClient := fake.NewClientset()
		for _, pod := range pods {
			_, err := kubeClient.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)
		}

		informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
		podLister := informerFactory.Core().V1().Pods().Lister()
		informerFactory.Start(ctx.Done())
		informerFactory.WaitForCacheSync(ctx.Done())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "in-bucket",
				Namespace: "default",
			},
			Data: map[string]string{"key": "value"},
		}

		_, err := kubeClient.CoreV1().ConfigMaps("default").Create(ctx, cm, metav1.CreateOptions{})
		require.NoError(t, err)

		handler := onConfigMapDelete(ctx, logger, bucket, kubeClient, podLister)
		handler(cm)

		// Check that the pods were killed
		for _, pod := range pods[:2] {
			_, err = kubeClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			require.EqualError(t, err, fmt.Sprintf("pods %q not found", pod.Name))
		}

		// Check that the pod that was not in the bucket was not killed
		_, err = kubeClient.CoreV1().Pods(pods[2].Namespace).Get(ctx, pods[2].Name, metav1.GetOptions{})
		require.NoError(t, err)
	})

	t.Run("not-configmap", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		logger := slog.New(slog.DiscardHandler)
		bucket := cache.NewFixedHashBucket(1)

		pods := []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod3",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "not-in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
		}

		kubeClient := fake.NewClientset()
		for _, pod := range pods {
			_, err := kubeClient.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)
		}

		informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
		podLister := informerFactory.Core().V1().Pods().Lister()
		informerFactory.Start(ctx.Done())
		informerFactory.WaitForCacheSync(ctx.Done())

		handler := onConfigMapDelete(ctx, logger, bucket, kubeClient, podLister)
		handler(testablePod(t))

		for _, pod := range pods {
			p, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, corev1.PodRunning, p.Status.Phase)
		}
	})

	t.Run("not-in-bucket", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		logger := slog.New(slog.DiscardHandler)
		bucket := cache.NewFixedHashBucket(2)
		bucket.Advance() // Advance the bucket to ensure the configmap is not in it

		pods := []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod3",
					Namespace: "default",
					Labels:    map[string]string{"reloader/configmap": "not-in-bucket"},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
		}

		kubeClient := fake.NewClientset()
		for _, pod := range pods {
			_, err := kubeClient.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			require.NoError(t, err)
		}

		informerFactory := informers.NewSharedInformerFactory(kubeClient, 0)
		podLister := informerFactory.Core().V1().Pods().Lister()
		informerFactory.Start(ctx.Done())
		informerFactory.WaitForCacheSync(ctx.Done())

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "in-bucket",
				Namespace: "default",
			},
			Data: map[string]string{"key": "value"},
		}

		handler := onConfigMapDelete(ctx, logger, bucket, kubeClient, podLister)

		handler(cm)

		for _, pod := range pods {
			p, err := kubeClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, corev1.PodRunning, p.Status.Phase)
		}
	})
}
