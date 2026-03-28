package ws

import (
	"bytes"
	"compress/gzip"
	"io"
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

func (r *messageReceiver) setHandler(handler messageReceiveHandleFunc) {
	r.handlers = append(r.handlers, handler)
}

func (r *messageReceiver) getMsgChan() chan []byte {
	return r.msgChan
}

func (r *messageReceiver) getErrChan() chan error {
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