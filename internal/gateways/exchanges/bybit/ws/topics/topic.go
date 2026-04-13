package topics

import "time"

// Topic is the interface for WebSocket channel topics.
type Topic interface {
	TopicName() string
	MsgHandler(timestamp *time.Time, rawMsg []byte) error
}
