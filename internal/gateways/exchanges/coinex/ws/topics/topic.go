package topics

import "time"

// Topic は websocket channel topic のインターフェース
type Topic interface {
	TopicName() string
	SubscribeMsgID() string
	SubscribeMsg() []byte
	MsgHandler(timestamp *time.Time, rawMsg []byte) error
}
