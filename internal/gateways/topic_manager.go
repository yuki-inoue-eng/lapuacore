package gateways

import "time"

// Topic is the interface for a WebSocket channel topic.
type Topic interface {
	TopicName() string
	SubscribeMsgID() string
	SubscribeMsg() []byte
	MsgHandler(timestamp *time.Time, rawMsg []byte) error
}

// TopicManager orchestrates topic subscriptions and message routing.
type TopicManager interface {
	SetTopics(topics []Topic)
	SubscribeRequests() [][]byte
	HandleTopicMessage(timestamp *time.Time, rawMsg []byte) error
	HandleSubscribeResp(rawMsg []byte) error
	MeasureLatency(rawMsg []byte) (string, time.Duration, error)
}
