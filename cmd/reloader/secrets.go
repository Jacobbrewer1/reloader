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

func (a *App) watchSecrets(ctx context.Context) {
	informer := a.base.SecretInformer()

	_, err := informer.AddEventHandler(kubecache.ResourceEventHandlerFuncs{
		UpdateFunc: onSecretUpdate(
			ctx,
			logging.LoggerWithComponent(a.base.Logger(), "secrets"),
			a.base.ServiceEndpointHashBucket(),
			a.base.KubeClient(),
			a.base.PodLister(),
		),
	})
	if err != nil {
		a.base.Logger().Error("failed to add event handler", slog.String(logging.KeyError, err.Error()))
		return
	}

	a.base.Logger().Info("watching secrets")
	<-ctx.Done()
}

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

		if !bucket.InBucket(secret.Name) {
			return
		}

		// Get all pods that use this secret. This is specified with the label
		// "reloader/secret": "<secret-name>".
		pods, err := podLister.Pods(secret.Namespace).List(labels.SelectorFromSet(map[string]string{
			"reloader/secret": secret.Name,
		}))
		if err != nil {
			l.Error("failed to list pods", slog.String(logging.KeyError, err.Error()))
			return
		}

		if err := killPods(ctx, kubeClient, pods); err != nil {
			l.Error("failed to kill pods", slog.String(logging.KeyError, err.Error()))
			return
		}
	}
}
