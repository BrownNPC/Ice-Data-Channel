package client

import (
	"context"
	"dc/message"
	"fmt"
	"log/slog"

	"github.com/coder/websocket"
	"github.com/pion/ice/v4"
)

type Guest struct {
	pc   *peerConnection
	ws   ws
	conn *ice.Conn
}

func NewGuest(ctx context.Context, roomID string, cfg Config) (guest *Guest, err error) {
	conn, _, err := websocket.Dial(ctx, cfg.signalingServer.String(), nil)
	if err != nil {
		return
	}
	pc, ufrag, pwd, err := newPeerConnection(cfg.agentCfg)
	if err != nil {
		return
	}
	guest = &Guest{
		ws: ws{conn},
		pc: pc,
	}
	err = guest.ws.WriteMsg(ctx, message.JoinRoomRequestMsg(roomID))
	if err != nil {
		return
	}
	// initiate ice auth
	err = guest.ws.WriteMsg(ctx, message.IceAuthInitiateMsg(ufrag, pwd))
	if err != nil {
		return
	}
	// wait for response
	msg, err := guest.ws.ReadMsg(ctx)
	if err != nil {
		return
	}
	if msg.Type != message.IceAuthResponse {
		guest.ws.Close(websocket.StatusProtocolError, "wrong message type sent. expected IceAuthResponse")
		return nil, fmt.Errorf("invalid response type from owner %s", msg.Type)
	}
	remoteUfrag, remotePwd := msg.Ufrag, msg.Pwd
	go guest.CandidateListener(ctx)

	ice_conn, err := guest.pc.Accept(ctx, remoteUfrag, remotePwd)
	if err != nil {
		return
	}
	guest.conn = ice_conn
	return
}
func (guest *Guest) Conn() *ice.Conn { return guest.conn }

// listen for ice candidates over websocket
func (guest *Guest) CandidateListener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cs := <-guest.pc.connectionState:
			switch cs {
			case ice.ConnectionStateConnected,
				ice.ConnectionStateCompleted,
				ice.ConnectionStateFailed,
				ice.ConnectionStateDisconnected,
				ice.ConnectionStateClosed:
				slog.Info("stopping listening for candidates", "state", cs.String())
				return // stop
			}
		default:
			msg, err := guest.ws.ReadMsg(ctx)
			if err != nil {
				slog.Error("failed to read message", "error", err)
				guest.ws.Close(websocket.StatusNormalClosure, "failed to read message")
				return
			}
			if msg.Type == message.Ping {
				continue
			}
			if msg.Type != message.IceCandidateForGuest {
				slog.Error("invalid message type received", "type", msg.Type.String())
				guest.ws.Close(websocket.StatusProtocolError, "invalid message type received, was expecting ice candidate")
				return
			}
			err = guest.pc.AddRemoteCandidate(msg.Candidate)
			if err != nil {
				guest.ws.Close(websocket.StatusProtocolError, "invalid ice candidate received")
				slog.Error("invalid ice candidate", "error", err)
				return
			}
		}
	}

}
