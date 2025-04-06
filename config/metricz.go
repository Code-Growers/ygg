package config

import "time"

type MetricszConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Enabled      bool
}

var DefaultMetricszConfig = MetricszConfig{
	Port:         "60004",
	ReadTimeout:  5 * time.Second,
	WriteTimeout: 5 * time.Second,
	Enabled:      true,
}
