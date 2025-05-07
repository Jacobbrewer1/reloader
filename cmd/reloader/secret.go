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

// watchSecrets starts watching for secret updates and deletes.
func (a *App) watchSecrets(ctx context.Context) {
	informer := a.base.SecretInformer()

	handler := kubecache.ResourceEventHandlerFuncs{
		UpdateFunc: onSecretUpdate(
			ctx,
			logging.LoggerWithComponent(a.base.Logger(), "secrets"),
			a.base.ServiceEndpointHashBucket(),
			a.base.KubeClient(),
			a.base.PodLister(),
		),
	}

	if a.config.KillOnDelete {
		handler.DeleteFunc = onSecretDelete(
			ctx,
			logging.LoggerWithComponent(a.base.Logger(), "secrets"),
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

	a.base.Logger().Info("watching secrets")
	<-ctx.Done()
}

// onSecretUpdate is called when a secret is updated. It checks if the secret is in the
func onSecretUpdate(
	ctx context.Context,
	l *slog.Logger,
	bucket cache.HashBucket,
	kubeClient kubernetes.Interface,
	podLister listersv1.PodLister,
) func(any, any) {
	return func(oldObj, newObj any) {
		secret, ok := newObj.(*corev1.Secret)
		if !ok {
			return
		}

		l.Debug("secret updated", slog.String("name", secret.Name), slog.String("namespace", secret.Namespace))

		if !bucket.InBucket(secret.Name) {
			return
		}

		l.Debug("handling secret update", slog.String("name", secret.Name), slog.String("namespace", secret.Namespace))

		// Get all pods that use this secret. This is specified with the label
		// "reloader/secret": "<secret-name>".
		pods, err := podLister.Pods(secret.Namespace).List(labels.SelectorFromSet(map[string]string{
			"reloader/secret": secret.Name,
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

// onSecretDelete is called when a secret is deleted. It checks if the secret is in the
func onSecretDelete(
	ctx context.Context,
	l *slog.Logger,
	bucket cache.HashBucket,
	kubeClient kubernetes.Interface,
	podLister listersv1.PodLister,
) func(any) {
	return func(obj any) {
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			return
		}

		l.Debug("secret deleted", slog.String("name", secret.Name), slog.String("namespace", secret.Namespace))

		if !bucket.InBucket(secret.Name) {
			return
		}

		l.Debug("handling secret delete", slog.String("name", secret.Name), slog.String("namespace", secret.Namespace))

		pods, err := podLister.Pods(secret.Namespace).List(labels.SelectorFromSet(map[string]string{
			"reloader/secret": secret.Name,
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
