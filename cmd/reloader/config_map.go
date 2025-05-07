package main

import (
	"context"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	kubecache "k8s.io/client-go/tools/cache"

	"github.com/jacobbrewer1/web/cache"
	"github.com/jacobbrewer1/web/logging"
)

// watchConfigMaps starts watching for configMap updates and deletes.
func (a *App) watchConfigMaps(ctx context.Context) {
	informer := a.base.ConfigMapInformer()

	handler := kubecache.ResourceEventHandlerFuncs{
		UpdateFunc: onConfigMapUpdate(
			ctx,
			logging.LoggerWithComponent(a.base.Logger(), "configmaps"),
			a.base.ServiceEndpointHashBucket(),
			a.base.KubeClient(),
			a.base.PodLister(),
		),
	}

	if a.config.KillOnDelete {
		a.base.Logger().Debug("watching configmaps for delete events")

		handler.DeleteFunc = onConfigMapDelete(
			ctx,
			logging.LoggerWithComponent(a.base.Logger(), "configmaps"),
			a.base.ServiceEndpointHashBucket(),
			a.base.KubeClient(),
			a.base.PodLister(),
		)
	}

	_, err := informer.AddEventHandler(handler)
	if err != nil {
		a.base.Logger().Error("failed to add event handler", slog.String(logging.KeyError, err.Error()))
		return
	}

	a.base.Logger().Info("watching configmaps")
	informer.Run(ctx.Done())
}

// onConfigMapUpdate is called when a configMap is updated. It checks if the configMap is in the
func onConfigMapUpdate(
	ctx context.Context,
	l *slog.Logger,
	bucket cache.HashBucket,
	kubeClient kubernetes.Interface,
	podLister listersv1.PodLister,
) func(any, any) {
	return func(oldObj, newObj any) {
		configMap, ok := newObj.(*corev1.ConfigMap)
		if !ok {
			return
		}

		l.Debug("configmap updated", slog.String("name", configMap.Name), slog.String("namespace", configMap.Namespace))

		if !bucket.InBucket(configMap.Name) {
			return
		}

		l.Debug("handling configmap update", slog.String("name", configMap.Name), slog.String("namespace", configMap.Namespace))

		// Get all pods that use this configMap. This is specified with the label
		// "reloader/configmap": "<configmap-name>".
		pods, err := podLister.Pods(configMap.Namespace).List(labels.SelectorFromSet(map[string]string{
			"reloader/configmap": configMap.Name,
		}))
		if err != nil {
			l.Error("failed to list pods", slog.String(logging.KeyError, err.Error()))
			return
		}

		if err := killPods(ctx, kubeClient, pods); err != nil { // nolint:revive // Traditional error handling
			l.Error("failed to kill pods", slog.String(logging.KeyError, err.Error()))
			return
		}
	}
}

// onConfigMapDelete is called when a configMap is deleted. It checks if the configMap is in the
func onConfigMapDelete(
	ctx context.Context,
	l *slog.Logger,
	bucket cache.HashBucket,
	kubeClient kubernetes.Interface,
	podLister listersv1.PodLister,
) func(any) {
	return func(obj any) {
		configMap, ok := obj.(*corev1.ConfigMap)
		if !ok {
			return
		}

		l.Debug("configmap deleted", slog.String("name", configMap.Name), slog.String("namespace", configMap.Namespace))

		if !bucket.InBucket(configMap.Name) {
			return
		}

		l.Debug("handling configmap delete", slog.String("name", configMap.Name), slog.String("namespace", configMap.Namespace))

		// Get all pods that use this configMap. This is specified with the label
		// "reloader/configmap": "<configmap-name>".
		pods, err := podLister.Pods(configMap.Namespace).List(labels.SelectorFromSet(map[string]string{
			"reloader/configmap": configMap.Name,
		}))
		if err != nil {
			l.Error("failed to list pods", slog.String(logging.KeyError, err.Error()))
			return
		}

		if err := killPods(ctx, kubeClient, pods); err != nil { // nolint:revive // Traditional error handling
			l.Error("failed to kill pods", slog.String(logging.KeyError, err.Error()))
			return
		}
	}
}
