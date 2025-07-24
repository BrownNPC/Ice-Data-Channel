package server_test

import (
	"github.com/BrownNPC/Ice-Data-Channel/message"
	"github.com/BrownNPC/Ice-Data-Channel/server"
	"net"
	"net/url"
	"testing"

	"github.com/coder/websocket"
)

func TestCreateRoom(t *testing.T) {
	u := url.URL{Scheme: "ws", Path: "/ws", Host: "localhost:9009"}
	l, err := net.Listen("tcp", u.Host)
	if err != nil {
		t.Error(err)
	}
	go server.Serve(l, u.Path)
	ctx := t.Context()
	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		t.Error(err)
	}
	{
		msg := message.CreateRoomMsg()
		err = conn.Write(ctx, websocket.MessageBinary, msg.Encode())
		if err != nil {
			t.Error(err)
		}
	}
	typ, payload, err := conn.Read(ctx)
	if err != nil {
		t.Error(err)
	}
	if typ != websocket.MessageBinary {
		t.Error("server did not respond with binary message")
	}
	msg := message.Decode(payload)
	if msg.Type != message.CreateRoomResponse {
		t.Error("server did not send create room response")
	}
	if msg.RoomID == "" {
		t.Error("server sent empty room id")
	}
}
func TestCreateRoomAndJoin(t *testing.T) {
	u := url.URL{Scheme: "ws", Path: "/ws", Host: "localhost:9010"}
	l, err := net.Listen("tcp", u.Host)
	if err != nil {
		t.Error(err)
	}
	go server.Serve(l, u)
	ctx := t.Context()
	// owner conn
	conn, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		t.Error(err)
	}
	// send create room request
	{
		msg := message.CreateRoomMsg()
		err = conn.Write(ctx, websocket.MessageBinary, msg.Encode())
		if err != nil {
			t.Error(err)
		}
	}

	var roomId string // get create room response
	{
		typ, payload, err := conn.Read(ctx)
		if err != nil {
			t.Error(err)
		}
		if typ != websocket.MessageBinary {
			t.Error("server did not respond with binary message")
		}
		msg := message.Decode(payload)
		if msg.Type != message.CreateRoomResponse {
			t.Error("server did not send create room response")
		}
		if msg.RoomID == "" {
			t.Error("server sent empty room id")
		}
		roomId = msg.RoomID
	}
	{
		// try to join the room
		conn, _, err := websocket.Dial(ctx, u.String(), nil)
		if err != nil {
			t.Error(err)
		}
		err = conn.Write(ctx, websocket.MessageBinary, message.JoinRoomRequestMsg(roomId).Encode())
		if err != nil {
			t.Error(err)
		}

		// forward a message to room owner
		err = conn.Write(ctx, websocket.MessageBinary, message.IceAuthInitiateMsg("", "").Encode())
		if err != nil {
			t.Error(err)
		}
	}
	// check if message was received on owner side
	typ, payload, err := conn.Read(ctx)
	if err != nil {
		t.Error(err)
	}
	if typ != websocket.MessageBinary {
		t.Error("message type is non-binary")
	}
	msg := message.Decode(payload)
	if msg.Type != message.IceAuthInitiate {
		t.Error("wrong message type was received")
	}
}
