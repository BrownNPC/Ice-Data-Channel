package client

import (
	"net/url"

	"github.com/pion/ice/v4"
	"github.com/pion/stun/v3"
)

type Config struct {
	// config to pass to PeerConnection
	AgentCfg ice.AgentConfig
	// where to dial the signaling server
	SignalingServer url.URL
}

func DefaultConfig(SignalingServerAddr, path string) Config {
	return Config{
		SignalingServer: url.URL{
			Scheme: "ws", Path: path,
			Host: SignalingServerAddr,
		},
		AgentCfg: ice.AgentConfig{
			NetworkTypes:     []ice.NetworkType{ice.NetworkTypeUDP4, ice.NetworkTypeUDP6},
			MulticastDNSMode: ice.MulticastDNSModeQueryAndGather,
			Urls: []*stun.URI{
				{Scheme: stun.SchemeTypeSTUN,
					Host:  "stun.l.google.com",
					Port:  19302,
					Proto: stun.ProtoTypeUDP,
				},
			}},
	}
}
