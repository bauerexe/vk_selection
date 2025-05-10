package config

import (
	"errors"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type GRPC struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}
type Log struct {
	Level string `yaml:"level"`
}
type Config struct {
	GRPC GRPC `yaml:"grpc"`
	Log  Log  `yaml:"log"`
}

func Load() (*Config, error) {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if v := os.Getenv("GRPC_HOST"); v != "" {
		cfg.GRPC.Host = v
	}
	if v := os.Getenv("GRPC_PORT"); v != "" {
		if p, _ := strconv.Atoi(v); p != 0 {
			cfg.GRPC.Port = p
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}

	if cfg.GRPC.Host == "" {
		cfg.GRPC.Host = "0.0.0.0"
	}
	if cfg.GRPC.Port == 0 {
		cfg.GRPC.Port = 8080
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}

	return cfg, nil
}
