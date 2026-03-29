package configs

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Exchange holds raw credentials for supported exchanges.
type Exchange struct {
	CoinEx RawCredential `yaml:"coinex"`
}

// RawCredential holds the raw API credentials parsed from YAML.
type RawCredential struct {
	APIKey string `yaml:"api_key"`
	Secret string `yaml:"secret"`
}

// RawAuthConfig holds the raw authentication config for services like InfluxDB.
type RawAuthConfig struct {
	Url   string `yaml:"url"`
	Token string `yaml:"token"`
}

// RawSecret represents the raw parsed YAML secret file.
type RawSecret struct {
	Exchange Exchange      `yaml:"exchanges"`
	InfluxDB RawAuthConfig `yaml:"influxdb"`
}

func readRawSecret(filePath string) (*RawSecret, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var secret RawSecret
	if err = yaml.Unmarshal(data, &secret); err != nil {
		return nil, err
	}
	return &secret, nil
}
