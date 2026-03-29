package configs

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// RawStrategy holds the strategy name from the YAML config file.
type RawStrategy struct {
	Name string `yaml:"name"`
}

// RawConfig represents the raw parsed YAML configuration.
type RawConfig struct {
	Strategy RawStrategy       `yaml:"strategy"`
	Params   map[string]string `yaml:"params"`
}

func (c *RawConfig) validate() error {
	if c.Strategy.Name == "" {
		return errors.New("strategy name is required")
	}

	return nil
}

func readRawConfig(filePath string) (*RawConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config RawConfig
	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if err = config.validate(); err != nil {
		return nil, err
	}

	return &config, nil
}
