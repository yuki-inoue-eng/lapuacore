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

// RawDiscord holds the raw Discord webhook URLs parsed from YAML.
type RawDiscord struct {
	InfoUrl      string `yaml:"info_url"`
	WarnUrl      string `yaml:"warn_url"`
	EmergencyUrl string `yaml:"emergency_url"`
}

// RawSecret represents the raw parsed YAML secret file.
type RawSecret struct {
	Exchange Exchange      `yaml:"exchanges"`
	InfluxDB RawAuthConfig `yaml:"influxdb"`
	Discord  RawDiscord    `yaml:"discord"`
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
