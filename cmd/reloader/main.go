package main

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/caarlos0/env/v10"

	"github.com/jacobbrewer1/web"
	"github.com/jacobbrewer1/web/logging"
)

const (
	appName = "reloader"
)

type (
	// AppConfig is the configuration for the app.
	AppConfig struct {
		// KillOnDelete is the flag to kill the pods on dependent deletion.
		KillOnDeleteStr string `env:"KILL_ON_DELETE" envDefault:"false"`
		// KillOnDelete is the flag to kill
		KillOnDelete bool
	}

	// App is the main application struct.
	App struct {
		// base is the base web application.
		base *web.App

		// config is the application configuration.
		config *AppConfig
	}
)

// NewApp creates a new application instance.
func NewApp(l *slog.Logger) (*App, error) {
	base, err := web.NewApp(l)
	if err != nil {
		return nil, fmt.Errorf("failed to create base app: %w", err)
	}

	cfg := new(AppConfig)
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env vars: %w", err)
	}
	cfg.KillOnDelete, _ = strconv.ParseBool(cfg.KillOnDeleteStr)

	return &App{
		base:   base,
		config: cfg,
	}, nil
}

// Start starts the application.
func (a *App) Start() error {
	a.base.Logger().Info("starting reloader", slog.String("KILL_ON_DELETE", fmt.Sprintf("%t", a.config.KillOnDelete)))

	if err := a.base.Start(
		web.WithInClusterKubeClient(),
		web.WithServiceEndpointHashBucket(appName),
		web.WithKubernetesPodInformer(),
		web.WithKubernetesConfigMapInformer(),
		web.WithKubernetesSecretInformer(),
		web.WithIndefiniteAsyncTask("configmaps-reload", a.watchConfigMaps),
		web.WithIndefiniteAsyncTask("secrets-reload", a.watchSecrets),
	); err != nil {
		return err
	}
	return nil
}

// WaitForEnd waits for the application to end.
func (a *App) WaitForEnd() {
	a.base.WaitForEnd(a.Shutdown)
}

// Shutdown shuts down the application.
func (a *App) Shutdown() {
	a.base.Shutdown()
}

func main() {
	l := logging.NewLogger(
		logging.WithAppName(appName),
	)

	app, err := NewApp(l)
	if err != nil {
		l.Error("failed to create app", slog.String(logging.KeyError, err.Error()))
		panic(err)
	}

	if err := app.Start(); err != nil {
		l.Error("failed to start app", slog.String(logging.KeyError, err.Error()))
		panic(err)
	}

	app.WaitForEnd()
}
