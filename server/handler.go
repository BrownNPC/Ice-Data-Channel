package server

import (
	"context"
	"dc/message"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

var (
	storage = struct {
		sync.RWMutex
		rooms map[string]*Room
	}{rooms: map[string]*Room{}}
)

func Serve(l net.Listener, path string) {
	mux := http.NewServeMux()
	mux.HandleFunc(path, WsHandler)
	if err := http.Serve(l, mux); err != nil {
		log.Printf("http.Serve error: %v", err)
	}
}

func WsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	msg := ReadInitialMessage(conn)
	if msg == nil {
		return
	}
	switch msg.Type {
	case message.CreateRoomRequest:
		room := NewRoom()
		go StoreRoom(room)
		// blocking
		room.NewConnection(true, conn)
		DeleteRoom(room)
	case message.JoinRoomRequest:
		room := GetRoom(msg.RoomID)
		if room == nil {
			conn.Close(websocket.StatusNormalClosure, "room does not exist")
			return
		}
		<-room.Ready
		// blocking
		room.NewConnection(false, conn)
	}
}

// message types that will be forwarded from owner to a specific guest
func OwnerMsgTypesToForwardToGuest(typ message.Type) bool {
	switch typ {
	case message.IceAuthResponse,
		message.IceCandidatesEnd,
		message.IceCandidateForGuest:
		return true
	default:
		return false
	}
}

// message types that will be forwarded from a specific guest to owner
func GuestMsgTypesToForwardToOwner(typ message.Type) bool {
	switch typ {
	case
		message.IceAuthInitiate,
		message.IceCandidatesEnd,
		message.IceCandidateForOwner:
		return true
	default:
		return false
	}
}
func ReadInitialMessage(conn *websocket.Conn) *message.Msg {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*20)
	// conn must send a message telling us what it wants to do
	typ, payload, err := conn.Read(ctx)
	cancel()
	if err != nil {
		conn.Close(websocket.StatusNormalClosure, "unable to read")
		return nil
	}
	if typ != websocket.MessageBinary {
		conn.Close(websocket.StatusNormalClosure, "incorrect message")
		return nil
	}
	msg := message.Decode(payload)
	if msg.Type == message.Invalid {
		conn.Close(websocket.StatusNormalClosure, "failed to decode message")
		return nil
	}
	switch msg.Type {
	case message.CreateRoomRequest, message.JoinRoomRequest:
		return &msg
	default:
		conn.Close(websocket.StatusNormalClosure, "invalid message type")
		return nil
	}
}
func StoreRoom(room *Room) {
	storage.Lock()
	storage.rooms[room.ID] = room
	storage.Unlock()
}
func DeleteRoom(room *Room) {
	storage.Lock()
	delete(storage.rooms, room.ID)
	storage.Unlock()
}
func GetRoom(id string) *Room {
	storage.Lock()
	defer storage.Unlock()

	return storage.rooms[id]
}
