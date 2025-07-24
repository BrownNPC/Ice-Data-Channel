package server

import (
	"context"
	"crypto/rand"
	"github.com/BrownNPC/Ice-Data-Channel/message"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type Room struct {
	ID          string
	OwnerID     uuid.UUID
	Connections map[uuid.UUID]*Connection
	sync.Mutex
	// room shuts down when we are unable to write something to the owner
	shutdown context.CancelFunc
	// kick all guests when this is closed
	shutdownCtx context.Context // when done, room is shut shutdown
	Ready       chan struct{}
}

func (room *Room) GetConnection(id uuid.UUID) (*Connection, bool) {
	room.Lock()
	defer room.Unlock()
	connection, ok := room.Connections[id]
	return connection, ok
}
func (room *Room) Delete(id uuid.UUID) {
	room.Lock()
	defer room.Unlock()
	delete(room.Connections, id)
}

// write a message to the target's websocket.
// if we are sending to the owner, and the write fails, we shutdown the room
func (room *Room) WriteToWebsocket(id uuid.UUID, msg message.Msg) {
	connection, ok := room.GetConnection(id)
	if !ok {
		slog.Debug("room.WriteToWebsocket(): connection not found", "id", id)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := connection.conn.Write(ctx, websocket.MessageBinary, msg.Encode())
	if err != nil {
		slog.Debug("error received while writing", "id", id)
		if id == room.OwnerID {
			slog.Debug("shutting down room")
			room.shutdown()
			connection.conn.Close(websocket.StatusGoingAway, "unreachable")
			room.Delete(room.OwnerID)
		} else {
			room.WriteToWebsocket(room.OwnerID, message.GuestDisconnectedMsg(id))
			room.Delete(id)
		}
	}
}
func NewRoom() *Room {
	room := Room{
		ID:          rand.Text()[:6],
		OwnerID:     uuid.UUID{},
		Connections: map[uuid.UUID]*Connection{},
		Mutex:       sync.Mutex{},
		shutdown:    nil,
		shutdownCtx: nil,
		Ready:       make(chan struct{}),
	}
	return &room
}

type Connection struct {
	ID     uuid.UUID
	RoomID string
	conn   *websocket.Conn
}

// making a new owner connection marks the room as ready
func (room *Room) NewConnection(owner bool, conn *websocket.Conn) {
	connection := Connection{
		ID:     uuid.New(),
		RoomID: room.ID,
		conn:   conn,
	}
	if owner {
		ctx, shutdown := context.WithCancel(context.Background())

		room.Lock()
		room.OwnerID = connection.ID
		room.shutdownCtx = ctx
		room.shutdown = shutdown
		room.Connections[connection.ID] = &connection
		room.Unlock()
		close(room.Ready)
		go connection.PingLoop(room, room.shutdownCtx)
		// blocking
		room.WriteToWebsocket(room.OwnerID, message.CreateRoomResponseMsg(room.ID))
		connection.Listen(room, room.shutdownCtx)
	} else if !owner {
		room.Lock()
		room.Connections[connection.ID] = &connection
		room.Unlock()
		go connection.PingLoop(room, room.shutdownCtx)
		// blocking
		connection.Listen(room, room.shutdownCtx)
	}
}
func (connection *Connection) PingLoop(room *Room, shutdownCtx context.Context) {
	for {
		select {
		case <-shutdownCtx.Done():
			return
		default:
			time.Sleep(time.Second * 10)
			room.WriteToWebsocket(connection.ID, message.PingMsg())
		}
	}
}
func (connection *Connection) Listen(room *Room, shutdownCtx context.Context) {
	IsOwner := connection.ID == room.OwnerID
	for {
		typ, payload, err := connection.conn.Read(shutdownCtx)
		if err != nil {
			slog.Debug("failed to read", "error", err)
			return
		}
		if typ == websocket.MessageBinary {
			msg := message.Decode(payload)
			if IsOwner {
				// blindly forward the message
				if OwnerMsgTypesToForwardToGuest(msg.Type) {
					room.WriteToWebsocket(msg.To, msg)
				} else {
					slog.Debug("unallowed message type sent by owner")
				}
			} else { // not owner
				if !GuestMsgTypesToForwardToOwner(msg.Type) {
					slog.Debug("unallowed message type sent by guest", "type", msg.Type)
					continue
				}
				// forward it to the owner
				msg.From = connection.ID
				room.WriteToWebsocket(room.OwnerID, msg)
			}
		}
	}
}
