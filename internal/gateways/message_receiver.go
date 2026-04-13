package gateways

import (
	"bytes"
	"compress/gzip"
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

// MessageReceiveHandleFunc is a handler invoked on each received message.
type MessageReceiveHandleFunc func(rawMsg []byte) error

// MessageReceiver reads messages from a WebSocket connection and dispatches
// them through registered handlers before forwarding to MsgChan.
type MessageReceiver struct {
	conn     *websocket.Conn
	handlers []MessageReceiveHandleFunc

	chanCloseOnce sync.Once
	msgChan       chan []byte
	errChan       chan error
}

// NewMessageReceiver creates a new message receiver for the given connection.
func NewMessageReceiver(conn *websocket.Conn) *MessageReceiver {
	return &MessageReceiver{
		conn:    conn,
		msgChan: make(chan []byte, 300),
		errChan: make(chan error),
	}
}

// SetHandler registers a handler invoked on each received message.
func (r *MessageReceiver) SetHandler(handler MessageReceiveHandleFunc) {
	r.handlers = append(r.handlers, handler)
}

// GetMsgChan returns the channel that receives decoded messages.
func (r *MessageReceiver) GetMsgChan() <-chan []byte {
	return r.msgChan
}

// GetErrChan returns the channel that receives errors.
func (r *MessageReceiver) GetErrChan() <-chan error {
	return r.errChan
}

func (r *MessageReceiver) chanClose() {
	r.chanCloseOnce.Do(func() {
		close(r.msgChan)
		close(r.errChan)
	})
}

// Start reads messages in a loop until an error occurs or the connection closes.
func (r *MessageReceiver) Start() {
	defer r.chanClose()
	for {
		messageType, rawMsg, err := r.conn.ReadMessage()
		if err != nil {
			r.errChan <- err
			return
		}
		if messageType == websocket.BinaryMessage {
			rawMsg, err = decodeGzip(rawMsg)
			if err != nil {
				r.errChan <- err
				return
			}
		}
		for _, handler := range r.handlers {
			if err = handler(rawMsg); err != nil {
				r.errChan <- err
				return
			}
		}
		r.msgChan <- rawMsg
	}
}

func decodeGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
