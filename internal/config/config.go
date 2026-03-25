package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
)

// Config holds poeusage configuration.
type Config struct {
	Timeout  int `toml:"timeout"`
	PageSize int `toml:"page_size"`
}

// DefaultConfigPath returns the XDG config path for the config file.
func DefaultConfigPath() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "poeusage", "config.toml")
}

// Load reads the config file (if present) and overlays environment variables.
func Load() (Config, error) {
	cfg := Config{
		Timeout:  30,
		PageSize: 100,
	}

	path := DefaultConfigPath()
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, &cfg); err != nil {
				return cfg, err
			}
		}
	}

	// Overlay env vars
	if v := os.Getenv("POEUSAGE_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Timeout = n
		}
	}
	if v := os.Getenv("POEUSAGE_PAGE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.PageSize = n
		}
	}

	return cfg, nil
}
