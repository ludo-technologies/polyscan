package config

import (
	_ "embed"
	"encoding/json"
)

// DefaultConfigJSON contains the embedded default configuration file
//
//go:embed default_config.json
var DefaultConfigJSON string

// LoadDefaultConfig parses the embedded default config and returns the full Config struct
func LoadDefaultConfig() (*Config, error) {
	var cfg Config
	if err := json.Unmarshal([]byte(DefaultConfigJSON), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
