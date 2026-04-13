package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type messageReceiveHandleFunc func(rawMsg []byte) error

type messageReceiver struct {
	conn     *websocket.Conn
	handlers []messageReceiveHandleFunc

	chanCloseOnce sync.Once
	msgChan       chan []byte
	errChan       chan error
}

func newMessageReceiver(conn *websocket.Conn) *messageReceiver {
	return &messageReceiver{
		conn:    conn,
		msgChan: make(chan []byte, 300),
		errChan: make(chan error),
	}
}

// SetHandler registers a handler invoked on each received message.
func (r *messageReceiver) SetHandler(handler messageReceiveHandleFunc) {
	r.handlers = append(r.handlers, handler)
}

func (r *messageReceiver) GetMsgChan() chan []byte {
	return r.msgChan
}

func (r *messageReceiver) GetErrChan() chan error {
	return r.errChan
}

func (r *messageReceiver) chanClose() {
	r.chanCloseOnce.Do(func() {
		close(r.msgChan)
		close(r.errChan)
	})
}

func (r *messageReceiver) start() {
	defer r.chanClose()
	for {
		_, rawMsg, err := r.conn.ReadMessage()
		if err != nil {
			r.errChan <- err
			break
		}

		for _, handler := range r.handlers {
			if err = handler(rawMsg); err != nil {
				r.errChan <- err
				break
			}
		}

		r.msgChan <- rawMsg
	}
}
