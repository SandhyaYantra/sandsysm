package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Interval     int    `yaml:"interval"`
	Theme        string `yaml:"theme"`
	DefaultIface string `yaml:"default_iface"`
	MaxProcs     int    `yaml:"max_procs"`
}

func Load() Config {
	config := Config{
		Interval:     1,
		Theme:        "dark",
		DefaultIface: "auto",
		MaxProcs:     10,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return config
	}
	configDir := filepath.Join(home, ".config", "sandsysm")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return config
	}

	data, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		return config
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{
			Interval:     1,
			Theme:        "dark",
			DefaultIface: "auto",
			MaxProcs:     10,
		}
	}
	return config
}
