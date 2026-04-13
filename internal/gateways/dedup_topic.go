package gateways

import "time"

// dedupTopic wraps a Topic and skips duplicate messages based on MessageID.
type dedupTopic struct {
	inner Topic
	cache *TTLCache
}

func newDedupTopic(inner Topic, cache *TTLCache) Topic {
	return &dedupTopic{inner: inner, cache: cache}
}

func (d *dedupTopic) TopicName() string        { return d.inner.TopicName() }
func (d *dedupTopic) SubscribeMsgID() string    { return d.inner.SubscribeMsgID() }
func (d *dedupTopic) SubscribeMsg() []byte      { return d.inner.SubscribeMsg() }
func (d *dedupTopic) MessageID(rawMsg []byte) string { return d.inner.MessageID(rawMsg) }

func (d *dedupTopic) MsgHandler(timestamp *time.Time, rawMsg []byte) error {
	id := d.inner.MessageID(rawMsg)
	if id != "" && !d.cache.AddIfAbsent(id) {
		return nil // duplicate, discard
	}
	return d.inner.MsgHandler(timestamp, rawMsg)
}
