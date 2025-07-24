package client

import (
	"net"

	"github.com/google/uuid"
	"github.com/pion/ice/v4"
)

// Conn implements [net.PacketConn] as well as [net.Conn]
// Although the message reliability depends on configuration.
// By default it's UDP hence it's unreliable
type Conn struct {
	*ice.Conn

	iD uuid.UUID
}

func newPacketConn(ID uuid.UUID, conn *ice.Conn) Conn {
	return Conn{iD: ID, Conn: conn}
}

func (conn Conn) ReadFrom(p []byte) (int, net.Addr, error) {
	n, err := conn.Read(p)
	if err != nil {
		return n, nil, err
	}
	return n, conn.RemoteAddr(), nil
}
func (conn Conn) WriteTo(p []byte, addr net.Addr) (int, error) {
	return conn.Write(p)
}
