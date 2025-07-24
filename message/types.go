package message

import (
	"github.com/google/uuid"
)

const (
	Invalid Type = iota
	Ping
	CreateRoomRequest
	CreateRoomResponse

	JoinRoomRequest

	IceCandidateForOwner
	IceCandidateForGuest
	IceAuthInitiate
	IceAuthResponse
	IceCandidatesEnd

	GuestDisconnected
	Kick
)

// connection creates a room
func CreateRoomMsg() Msg {
	return Msg{
		Type: CreateRoomRequest,
	}
}

// server responds with room created message and sends RoomID
func CreateRoomResponseMsg(RoomID string) Msg {
	return Msg{
		Type:   CreateRoomResponse,
		RoomID: RoomID,
	}
}

// sent to owner when a guest disconnects.
func GuestDisconnectedMsg(LostConnectionId uuid.UUID) Msg {
	return Msg{
		Type: GuestDisconnected,
		From: LostConnectionId,
	}
}

// which room to join + ice info
func JoinRoomRequestMsg(RoomID string) Msg {
	return Msg{
		Type:   JoinRoomRequest,
		RoomID: RoomID,
	}
}

// the guest initiates the ice auth
func IceAuthInitiateMsg(ufrag, pwd string) Msg {
	return Msg{
		Type:  IceAuthInitiate,
		Ufrag: ufrag, Pwd: pwd,
	}
}

// owner responds with its own credentials
func IceAuthResponseMsg(ufrag, pwd string, To uuid.UUID) Msg {
	return Msg{
		Type:  IceAuthResponse,
		To:    To,
		Ufrag: ufrag, Pwd: pwd,
	}
}
func IceCandidateForOwnerMsg(candidate string) Msg {
	return Msg{
		Type:      IceCandidateForOwner,
		Candidate: candidate,
	}
}
func IceCandidateForGuestMsg(candidate string, To uuid.UUID) Msg {
	return Msg{
		Type:      IceCandidateForGuest,
		To:        To,
		Candidate: candidate,
	}
}

// the owner tells the signaling server to kick this peer
func KickMsg(Target uuid.UUID) Msg {
	return Msg{
		To:   Target,
		Type: Kick,
	}
}
func PingMsg() Msg {
	return Msg{
		Type: Ping,
	}
}
