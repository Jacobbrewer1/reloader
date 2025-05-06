package main

import (
	"fmt"
	"log/slog"

	"github.com/caarlos0/env/v10"

	"github.com/jacobbrewer1/web"
	"github.com/jacobbrewer1/web/logging"
)

const (
	appName = "reloader"
)

type (
	// AppConfig is the configuration for the app.
	AppConfig struct{}

	// App is the main application struct.
	App struct {
		// base is the base web application.
		base *web.App

		// config is the application configuration.
		config *AppConfig
	}
)

func NewApp(l *slog.Logger) (*App, error) {
	base, err := web.NewApp(l)
	if err != nil {
		return nil, fmt.Errorf("failed to create base app: %w", err)
	}

	cfg := new(AppConfig)
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env vars: %w", err)
	}

	return &App{
		base:   base,
		config: cfg,
	}, nil
}

func (a *App) Start() error {
	if err := a.base.Start(
		web.WithInClusterKubeClient(),
		web.WithServiceEndpointHashBucket(appName),
		web.WithKubernetesSecretInformer(),
	); err != nil {
		return err
	}
	return nil
}

func (a *App) WaitForEnd() {
	a.base.WaitForEnd(a.Shutdown)
}

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
