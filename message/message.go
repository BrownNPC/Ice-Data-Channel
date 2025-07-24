package message

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack"
)

//go:generate stringer -type=Type
type Type int

type Msg struct {
	_msgpack struct{} `msgpack:",omitempty"` // msgpack will ignore empty fields
	Type

	Success bool
	Cause   string // why is Success false

	RoomID string
	// Id of the connection this message is related to in some way
	From, To uuid.UUID //role depends on message type

	// ICE
	Ufrag, Pwd, Candidate string
}

func Decode(b []byte) (msg Msg) {
	err := msgpack.Unmarshal(b, &msg)
	if err != nil {
		return Msg{Type: Invalid}
	}
	return msg
}
func (m Msg) Encode() []byte {
	b, err := msgpack.Marshal(&m)
	if err != nil {
		panic(fmt.Errorf("Msg.Encode() error: %w", err))
	}
	return b
}
