package client

import (
	"context"
	"github.com/BrownNPC/Ice-Data-Channel/message"
	"fmt"

	"github.com/coder/websocket"
)

type ws struct{ *websocket.Conn }

// Send a message over the websocket
func (ws ws) WriteMsg(ctx context.Context, msg message.Msg) error {
	return ws.Write(ctx, websocket.MessageBinary, msg.Encode())
}

// Read a message from the websocket
func (ws ws) ReadMsg(ctx context.Context) (msg message.Msg, err error) {
	typ, payload, err := ws.Read(ctx)
	if err != nil {
		return
	}
	if typ != websocket.MessageBinary {
		return msg, fmt.Errorf("non-binary message received")
	}
	return message.Decode(payload), nil
}
