package config

import (
	"time"
)

type CorsConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
	AllowedHeaders   []string
}

type ServerConfig struct {
	Port            string
	Enabled         bool
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	Cors            *CorsConfig
}
