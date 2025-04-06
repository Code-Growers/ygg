package ygg

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/code-growers/ygg/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metriczComponent struct {
	config   config.MetricszConfig
	registry *prometheus.Registry

	// Server that's currently being used
	// This used for closing the component
	runningServer *http.Server

	// Channel to close when metricz is disabled
	exitChan chan (bool)
}

func (hc *metriczComponent) Hooks() []Hook {
	// TODO
	return nil
}

var _ Component = (*metriczComponent)(nil)

func NewMetriczComponent(registry *prometheus.Registry, config config.MetricszConfig) *metriczComponent {
	return &metriczComponent{
		config:   config,
		registry: registry,

		exitChan: make(chan bool, 1),
	}
}

func (hc *metriczComponent) Close(ctx context.Context) error {
	hc.exitChan <- true
	if hc.runningServer == nil {
		return nil
	}

	return hc.runningServer.Shutdown(ctx)
}

func (hc *metriczComponent) Startup() error {
	return nil
}

func (hc *metriczComponent) Run() error {
	if !hc.config.Enabled {
		<-hc.exitChan
		return nil
	}

	api := serveMetricsz(hc.config, hc.registry)
	hc.runningServer = api

	slog.Info(fmt.Sprintf("Starting healhz %s", api.Addr))
	return api.ListenAndServe()
}

func serveMetricsz(config config.MetricszConfig, reg *prometheus.Registry) *http.Server {
	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.InstrumentMetricHandler(
		reg, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
	))

	api := http.Server{
		Addr:         "0.0.0.0:" + config.Port,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		Handler:      router,
	}

	return &api
}

type healthzComponent struct {
	config    config.HealthzConfig
	providers []Provider

	// Server that's currently being used
	// This used for closing the component
	runningServer *http.Server

	// Channel to close when healhz is disabled
	exitChan chan (bool)
}

func (hc *healthzComponent) Hooks() []Hook {
	// TODO
	return nil
}

var _ Component = (*healthzComponent)(nil)

func NewHealthzComponent(providers []Provider, config config.HealthzConfig) *healthzComponent {
	return &healthzComponent{
		config:    config,
		providers: providers,

		exitChan: make(chan bool, 1),
	}
}

func (hc *healthzComponent) Close(ctx context.Context) error {
	hc.exitChan <- true
	if hc.runningServer == nil {
		return nil
	}

	return hc.runningServer.Shutdown(ctx)
}

func (hc *healthzComponent) Startup() error {
	return nil
}

func (hc *healthzComponent) Run() error {
	if !hc.config.Enabled {
		<-hc.exitChan
		return nil
	}

	healthzChecker := NewHealthChecker(hc.providers)
	api := healthzChecker.Serve(hc.config)
	hc.runningServer = api

	slog.Info(fmt.Sprintf("Starting metricz %s", api.Addr))
	return api.ListenAndServe()
}

// Checkable Makes sure the object has the ) function
type Checkable interface {
	Healthz(ctx context.Context) error
}

type CheckableFunc func(ctx context.Context) error

func (f CheckableFunc) Healthz(ctx context.Context) error {
	return f(ctx)
}

// Provider is a provder we can check for healthz
type Provider struct {
	Handle Checkable
	Name   string
}

func NewHealthChecker(providers []Provider) *HealthzChecker {
	return &HealthzChecker{
		Providers: providers,
	}
}

// HealthzChecker contains the instance
type HealthzChecker struct {
	Providers []Provider
}

// Error the structure of the Error object
type Error struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// Service struct reprecenting a healthz provider and it's status
type Service struct {
	Name         string `json:"name"`
	Healthy      bool   `json:"healthy"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// Response type, we return a json object with {healthy:bool, errors:[]}
type HealthzResponse struct {
	Services []Service `json:"services,omitempty"`
	Healthy  bool      `json:"healthy"`
}

// returns a http.HandlerFunc for the healthz service
func (h *HealthzChecker) Healthz() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		slog.DebugContext(ctx, "Handling the Healthz")
		w.Header().Set("Content-Type", "application/json")
		if h.Providers == nil {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("OK"))
			if err != nil {
				slog.ErrorContext(ctx, "Internal error", "err", err)
			}
			return
		}

		response := HealthzResponse{
			Services: []Service{},
			Healthy:  true,
		}

		for _, provider := range h.Providers {
			service := Service{
				Name:    provider.Name,
				Healthy: true,
			}
			err := provider.Handle.Healthz(r.Context())
			if err != nil {
				service.ErrorMessage = err.Error()
				service.Healthy = false
				response.Healthy = false
			}
			response.Services = append(response.Services, service)
		}

		if !response.Healthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		jsonBody, err := json.Marshal(response)
		if err != nil {
			slog.ErrorContext(ctx, "Unable to marshal errors", "err", err)
		}

		_, err = w.Write(jsonBody)
		if err != nil {
			slog.ErrorContext(ctx, "Internal error", "err", err)
		}
	})
}

func (h *HealthzChecker) Serve(config config.HealthzConfig) *http.Server {
	router := http.NewServeMux()
	router.Handle("/healthz", h.Healthz())
	router.Handle("/liveliness", Liveness())

	api := http.Server{
		Addr:         "0.0.0.0:" + config.Port,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		Handler:      router,
	}

	return &api
}

func Liveness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			slog.Error("Internal error", "err", err)
		}
	}
}
