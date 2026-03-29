package metrics

import (
	"log/slog"

	"github.com/InfluxCommunity/influxdb3-go/influxdb3"
	"github.com/hashicorp/go-retryablehttp"
)

func NewInfluxDBClient(url, bucketName, token string) (*influxdb3.Client, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.Logger = slog.Default()
	retryClient.RetryMax = 10

	client, err := influxdb3.New(influxdb3.ClientConfig{
		Host:       url,
		Token:      token,
		Database:   bucketName,
		HTTPClient: retryClient.StandardClient(),
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}
