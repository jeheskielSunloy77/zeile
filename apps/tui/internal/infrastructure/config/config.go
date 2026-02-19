package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DataDir            string `toml:"data_dir"`
	ManagedCopyDefault bool   `toml:"managed_copy_default"`
	MinSpreadWidth     int    `toml:"min_spread_width"`
}

type Loaded struct {
	Config Config
	Path   string
}

func Load() (Loaded, error) {
	configPath, err := defaultConfigPath()
	if err != nil {
		return Loaded{}, err
	}

	cfg := defaults()

	if _, statErr := os.Stat(configPath); statErr == nil {
		if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
			return Loaded{}, fmt.Errorf("decode config: %w", err)
		}
	} else if !os.IsNotExist(statErr) {
		return Loaded{}, fmt.Errorf("stat config: %w", statErr)
	} else {
		if err := writeDefaults(configPath, cfg); err != nil {
			return Loaded{}, err
		}
	}

	if cfg.DataDir == "" {
		cfg.DataDir = defaults().DataDir
	}
	if cfg.MinSpreadWidth <= 0 {
		cfg.MinSpreadWidth = defaults().MinSpreadWidth
	}

	return Loaded{Config: cfg, Path: configPath}, nil
}

func defaults() Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return Config{
		DataDir:            filepath.Join(homeDir, ".zeile"),
		ManagedCopyDefault: true,
		MinSpreadWidth:     120,
	}
}

func defaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "zeile", "config.toml"), nil
}

func writeDefaults(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(cfg); err != nil {
		return fmt.Errorf("write config defaults: %w", err)
	}

	return nil
}
