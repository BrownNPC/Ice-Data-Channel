package client

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/BrownNPC/Ice-Data-Channel/message"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type Owner struct {
	connections map[uuid.UUID]*peerConnection
	connMu      sync.Mutex
	RoomID      string
	cfg         Config
	onConnect   func(conn Conn)

	ws ws
}

func NewOwner(ctx context.Context, onConnect func(conn Conn), cfg Config) (owner *Owner, err error) {
	conn, _, err := websocket.Dial(ctx, cfg.signalingServer.String(), nil)
	if err != nil {
		return
	}
	// owner just listens for other people trying to connect
	// just make a room and return.
	// handle connections in background
	owner = &Owner{
		ws:          ws{conn},
		cfg:         cfg,
		connections: map[uuid.UUID]*peerConnection{},
		onConnect:   onConnect,
		connMu:      sync.Mutex{},
	}
	err = owner.ws.WriteMsg(ctx, message.CreateRoomMsg())
	if err != nil {
		return
	}
	tctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	msg, err := owner.ws.ReadMsg(tctx)
	if err != nil {
		return
	}
	if msg.Type != message.CreateRoomResponse {
		return nil, fmt.Errorf("invalid response type from server")
	}
	owner.RoomID = msg.RoomID
	go owner.eventHandler(ctx)
	return
}
func (owner *Owner) eventHandler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			owner.ws.Close(websocket.StatusGoingAway, "room shut down")
			owner.disconnectAll()
			return
		default:
			msg, err := owner.ws.ReadMsg(ctx)
			if err != nil {
				slog.Error("failed to read message", "error", err)
				owner.disconnectAll()
				return
			}
			err = owner.handleMsg(ctx, msg)
			if err != nil {
				slog.Error("failed to handle message", "type", msg.Type, "error", err)
				owner.disconnectAll()
				return
			}
		}
	}
}

// only return error if state is unrecoverable
func (owner *Owner) handleMsg(ctx context.Context, msg message.Msg) error {
	switch msg.Type {
	case message.IceAuthInitiate:
		remoteUfrag, remotePwd := msg.Ufrag, msg.Pwd
		pc, ufrag, pwd, err := newPeerConnection(owner.cfg.agentCfg)
		if err != nil {
			return err
		}
		owner.addConnection(msg.From, pc)
		err = owner.ws.WriteMsg(ctx, message.IceAuthResponseMsg(ufrag, pwd, msg.From))
		if err != nil {
			owner.deleteConnection(msg.From)
			slog.Debug("failed to write to guest connection", "error", err)
			return nil
		}
		// dial in goroutine
		go func() {
			conn, err := pc.Dial(ctx, remoteUfrag, remotePwd)
			if err != nil {
				slog.Error("failed to dial", "error", err)
				return
			}
			owner.onConnect(newPacketConn(msg.From, conn))
		}()
		// forward locally gathered ice candidates
		go func() {
			for c := range pc.localCandidates {
				err = owner.ws.WriteMsg(ctx, message.IceCandidateForGuestMsg(c, msg.From))
				if err != nil {
					slog.Debug("error sending ice candidate", "error", err)
					return
				}
			}
		}()
		// receive remote candidates
	case message.IceCandidateForOwner:
		pc := owner.getConnection(msg.From)
		if pc == nil {
			slog.Debug("got ice candidates for a connection not in map", "id", msg.From)
			return nil
		}
		err := pc.AddRemoteCandidate(msg.Candidate)
		if err != nil {
			slog.Debug("failed to add remote candidate", "candidate", msg.Candidate)
		}
	case message.GuestDisconnected:
		pc := owner.getConnection(msg.From)
		if pc == nil {
			slog.Debug("ask to disconnect a non-connected peer")
			return nil
		}
		pc.agent.Close()

	case message.Ping:
		return nil
	default:
		return fmt.Errorf("unhandled message type")
	}
	return nil
}
func (owner *Owner) addConnection(id uuid.UUID, pc *peerConnection) {
	owner.connMu.Lock()
	owner.connections[id] = pc
	owner.connMu.Unlock()
}
func (owner *Owner) Kick(conn Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	owner.ws.WriteMsg(ctx, message.KickMsg(conn.iD))
	cancel()
	owner.deleteConnection(conn.iD)
}

// does not disconnect, only deletes from map
func (owner *Owner) deleteConnection(id uuid.UUID) {
	owner.connMu.Lock()
	delete(owner.connections, id)
	owner.connMu.Unlock()
}

// Could be nil
func (owner *Owner) getConnection(id uuid.UUID) *peerConnection {
	owner.connMu.Lock()
	defer owner.connMu.Unlock()

	return owner.connections[id]
}
func (owner *Owner) disconnectAll() {
	owner.connMu.Lock()
	defer owner.connMu.Unlock()
	for _, conn := range owner.connections {
		conn.agent.Close()
	}
}
