package gateways

import (
	"context"
	"time"
)

// ChannelGroup manages N redundant WebSocket channels to the same endpoint.
// Messages are deduplicated across all channels: only the first arrival of
// each message is processed, the rest are discarded.
type ChannelGroup struct {
	channels []*Channel
	topicMgs []TopicManager
	cache    *TTLCache
}

// NewChannelGroup creates a group of N redundant channels.
// channelFactory creates a new Channel (called N times).
// topicMgFactory creates a new TopicManager (called N times).
func NewChannelGroup(
	n int,
	channelFactory func() *Channel,
	topicMgFactory func() TopicManager,
	dedupTTL time.Duration,
) *ChannelGroup {
	cache := NewTTLCache(dedupTTL)
	g := &ChannelGroup{
		cache: cache,
	}
	for i := 0; i < n; i++ {
		ch := channelFactory()
		mg := topicMgFactory()
		ch.SetTopicMg(mg)
		g.channels = append(g.channels, ch)
		g.topicMgs = append(g.topicMgs, mg)
	}
	return g
}

// SetTopics wraps each topic with dedup filtering and sets them on
// all internal topic managers.
func (g *ChannelGroup) SetTopics(topics []Topic) {
	var wrapped []Topic
	for _, t := range topics {
		wrapped = append(wrapped, newDedupTopic(t, g.cache))
	}
	for _, mg := range g.topicMgs {
		mg.SetTopics(wrapped)
	}
}

// Start launches all channels and the cache cleanup goroutine in parallel.
// Blocks until ctx is cancelled or a channel returns an error.
// If a channel fails, the error is returned immediately.
func (g *ChannelGroup) Start(ctx context.Context) error {
	go g.cache.StartCleanup(ctx)

	errCh := make(chan error, len(g.channels))
	for _, ch := range g.channels {
		ch := ch
		go func() {
			if err := ch.Start(ctx); err != nil {
				errCh <- err
			}
		}()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}
