package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	StoragePath       string `yaml:"storage_path"`
	ScheduleKey       string `yaml:"schedule_key"`
	SubstitutionsKey  string `yaml:"substitutions_key"`
	SyncIntervalHours int    `yaml:"sync_interval_hours"`
	SeedOnStart       bool   `yaml:"seed_on_start"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse: %w", err)
	}

	if cfg.SyncIntervalHours == 0 {
		cfg.SyncIntervalHours = 6
	}

	if v := os.Getenv("STORAGE_PATH"); v != "" {
		cfg.StoragePath = v
	}
	if v := os.Getenv("SCHEDULE_KEY"); v != "" {
		cfg.ScheduleKey = v
	}
	if v := os.Getenv("SUBSTITUTIONS_KEY"); v != "" {
		cfg.SubstitutionsKey = v
	}
	if v := os.Getenv("SYNC_INTERVAL_HOURS"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			cfg.SyncIntervalHours = val
		}
	}
	if v := os.Getenv("SEED_ON_START"); v != "" {
		cfg.SeedOnStart = v == "true" || v == "1"
	}

	return &cfg, nil
}
