package nntpcli

import (
	"log/slog"
	"time"
)

type Config struct {
	log     *slog.Logger
	timeout time.Duration
}

type Option func(*Config)

var configDefault = Config{
	timeout: time.Duration(5) * time.Second,
	log:     slog.Default(),
}

func mergeWithDefault(config ...Config) Config {
	if len(config) == 0 {
		return configDefault
	}

	cfg := config[0]

	if cfg.timeout == 0 {
		cfg.timeout = configDefault.timeout
	}

	if cfg.log == nil {
		cfg.log = configDefault.log
	}

	return cfg
}
