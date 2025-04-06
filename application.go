package ygg

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Logger interface {
	Debug(mgs string, params ...any)
	Info(mgs string, params ...any)
}

type slogLogger struct{}

func (l *slogLogger) Debug(msg string, params ...any) {
	slog.Debug(msg, params...)
}

func (l *slogLogger) Info(msg string, params ...any) {
	slog.Info(msg, params...)
}

type Component interface {
	// Synchronous startup, container runs this one by one
	Startup() error
	// Asynchronous run,  container runs this in parrallel
	// Has to block until it's done
	// When one of componenst exits this func, container will start shutting down
	Run() error
	// Cleanup function
	Close(ctx context.Context) error
	// Retrieve hook functions
	Hooks() []Hook
}

type HookType = uint

const (
	HookTypeBeforeStartup HookType = iota
	HookTypeAfterStartup

	HookTypeBeforeRun
	HookTypeAfterRun

	HookTypeBeforeClose
)

type Hook interface {
	Type() HookType
	Run() error
}

var defaultLogger = &slogLogger{}

type Application struct {
	components []Component
	timeout    time.Duration
	logger     Logger
}

func NewApplications(components ...Component) *Application {
	return &Application{
		components: components,
		timeout:    10 * time.Second,
		logger:     defaultLogger,
	}
}

func (app Application) WithTimeout(timeout time.Duration) Application {
	app.timeout = timeout
	return app
}

func (app Application) WithLogger(logger Logger) Application {
	app.logger = logger

	return app
}

func (app *Application) AddComponent(c Component) {
	if app.components == nil {
		app.components = []Component{}
	}

	app.components = append(app.components, c)
}

type collectedHooks struct {
	beforeStartup []Hook
	afterStartup  []Hook

	beforeRun []Hook
	afterRun  []Hook

	beforeClose []Hook
}

func (app *Application) Run() error {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	serverErrors := make(chan error, 1)

	hooks := collectedHooks{
		beforeStartup: []Hook{},
		afterStartup:  []Hook{},
		beforeRun:     []Hook{},
		afterRun:      []Hook{},
		beforeClose:   []Hook{},
	}

	for _, component := range app.components {
		for _, hook := range component.Hooks() {
			switch hook.Type() {
			case HookTypeBeforeStartup:
				hooks.beforeStartup = append(hooks.beforeStartup, hook)
			case HookTypeAfterStartup:
				hooks.afterStartup = append(hooks.afterStartup, hook)
			case HookTypeBeforeRun:
				hooks.beforeRun = append(hooks.beforeRun, hook)
			case HookTypeAfterRun:
				hooks.afterRun = append(hooks.afterRun, hook)
			case HookTypeBeforeClose:
				hooks.beforeClose = append(hooks.beforeClose, hook)
			}
		}
	}

	err := runHooks(hooks.beforeStartup)
	if err != nil {
		return fmt.Errorf("beforeStartup hooks error: %w", err)
	}

	for _, component := range app.components {
		component := component
		app.logger.Debug("Starting component %T", component)
		err := component.Startup()
		if err != nil {
			return fmt.Errorf("startup error: %w", err)
		}
	}

	err = runHooks(hooks.afterStartup)
	if err != nil {
		return fmt.Errorf("afterStartup hooks error: %w", err)
	}

	err = runHooks(hooks.beforeRun)
	if err != nil {
		return fmt.Errorf("beforeRun hooks error: %w", err)
	}
	for _, component := range app.components {
		component := component
		go func() {
			serverErrors <- component.Run()
		}()
	}
	err = runHooks(hooks.afterRun)
	if err != nil {
		return fmt.Errorf("afterRun hooks error: %w", err)
	}

	select {
	case err := <-serverErrors:
		app.logger.Info("Application Error : %v", err)
		// TODO add shutdown time config
		ctx, cancel := context.WithTimeout(context.Background(), app.timeout)
		defer cancel()
		for _, component := range app.components {
			app.logger.Info("Closing component %T", component)
			err := component.Close(ctx)
			if err != nil {
				app.logger.Info("Component %T Closed with error %s", component, err.Error())
				return err
			}
		}
		return err

	case sig := <-shutdown:
		app.logger.Info("%v : Shuting down gracefully", sig)
		exitChan := make(chan error, 1)
		// TODO add shutdown time config
		ctx, cancel := context.WithTimeout(context.Background(), app.timeout)
		defer cancel()

		go func() {
			err := runHooks(hooks.beforeClose)
			if err != nil {
				exitChan <- fmt.Errorf("beforeClose hooks error: %w", err)
				return
			}

			for _, component := range app.components {
				app.logger.Info("Closing component %T", component)
				err := component.Close(ctx)
				if err != nil {
					app.logger.Info("Shuting down did not complete, %v", err)
				}
			}
			exitChan <- err
		}()

		select {
		case err := <-exitChan:
			return err
		case <-ctx.Done():
			return fmt.Errorf("killed after %s", app.timeout)
		}
	}
}

func runHooks(hooks []Hook) error {
	var masterErr error
	for _, hook := range hooks {
		err := hook.Run()
		if err != nil {
			masterErr = errors.Join(masterErr, err)
		}
	}

	return masterErr
}
