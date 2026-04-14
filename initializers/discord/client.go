package discord

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

type Client struct {
	strategyName string
	infoUrl      string
	warnUrl      string
	emergencyUrl string
	noopMode     bool
}

// NewClient creates a Discord webhook client.
// If strategyName is empty or all URLs are empty, it runs in noop mode.
func NewClient(strategyName, infoUrl, warnUrl, emergencyUrl string) *Client {
	if strategyName == "" || (infoUrl == "" && warnUrl == "" && emergencyUrl == "") {
		slog.Info("noop discord client is initialized.")
		return &Client{noopMode: true}
	}
	return &Client{
		strategyName: strategyName,
		infoUrl:      infoUrl,
		warnUrl:      warnUrl,
		emergencyUrl: emergencyUrl,
		noopMode:     false,
	}
}

type dto struct {
	UserName string `json:"username"`
	Content  string `json:"content"`
}

func (c *Client) SendInfo(msg string) {
	if c.noopMode {
		return
	}
	go sendMsg(c.infoUrl, c.strategyName, msg)
}

func (c *Client) SyncSendInfo(msg string) {
	if c.noopMode {
		return
	}
	sendMsg(c.infoUrl, c.strategyName, msg)
}

func (c *Client) SendEmergency(msg string) {
	if c.noopMode {
		return
	}
	go sendMsg(c.emergencyUrl, c.strategyName, msg)
}

func (c *Client) SyncSendEmergency(msg string) {
	if c.noopMode {
		return
	}
	sendMsg(c.emergencyUrl, c.strategyName, msg)
}

func (c *Client) SendWarn(msg string) {
	if c.noopMode {
		return
	}
	go sendMsg(c.warnUrl, c.strategyName, msg)
}

func (c *Client) SyncSendWarn(msg string) {
	if c.noopMode {
		return
	}
	sendMsg(c.warnUrl, c.strategyName, msg)
}

func sendMsg(url, strategyName, msg string) {
	msgDto := dto{
		UserName: strategyName,
		Content:  msg,
	}
	rawMsg, err := json.Marshal(&msgDto)
	if err != nil {
		slog.Error("failed to json marshal", "error", err)
		return
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(rawMsg))
	if err != nil {
		slog.Error("failed to create discord request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to send discord message", "error", err)
	} else if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		slog.Error("failed to send discord message", "status", resp.StatusCode, "message", msg)
	}
}
