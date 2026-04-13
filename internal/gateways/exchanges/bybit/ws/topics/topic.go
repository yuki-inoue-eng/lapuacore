package topics

// SubscribeMsgID and SubscribeMsg are not used by individual Bybit topics
// because subscription is handled as a batch by the Manager.
// These methods satisfy the gateways.Topic interface.

// topicBase provides default no-op implementations for subscribe methods.
type topicBase struct{}

func (topicBase) SubscribeMsgID() string { return "" }
func (topicBase) SubscribeMsg() []byte   { return nil }
