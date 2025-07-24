package client

import (
	"context"
	"github.com/pion/ice/v4"
)

// a peer connection is responsible for making
// an Ice UDP connection
type peerConnection struct {
	localCandidates chan string // candidates gathered
	connectionState chan ice.ConnectionState

	agent *ice.Agent
}

func newPeerConnection(cfg ice.AgentConfig) (pc *peerConnection, ufrag, pwd string, err error) {
	agent, err := ice.NewAgent(&cfg)
	if err != nil {
		return
	}
	ufrag, pwd, err = agent.GetLocalUserCredentials()
	if err != nil {
		return
	}
	pc = &peerConnection{
		agent:           agent,
		localCandidates: make(chan string, 50),
		connectionState: make(chan ice.ConnectionState, 10),
	}

	agent.OnCandidate(func(c ice.Candidate) {
		if c == nil {
			close(pc.localCandidates)
			return
		}
		pc.localCandidates <- c.Marshal()
	})
	agent.OnConnectionStateChange(func(cs ice.ConnectionState) {
		pc.connectionState <- cs
	})

	// start gathering candidates to channel
	err = agent.GatherCandidates()
	if err != nil {
		return
	}
	return
}

// ICE Candidates gathered by this peer connection that need to be forwarded to remote
func (pc *peerConnection) LocalCandidates() <-chan string {
	return pc.localCandidates
}

// AddRemoteCandidate adds a new remote candidate.
func (pc *peerConnection) AddRemoteCandidate(c string) error {
	cand, err := ice.UnmarshalCandidate(c)
	if err != nil {
		return err
	}
	return pc.agent.AddRemoteCandidate(cand)
}

// get ice connectionState
func (pc *peerConnection) ConnectionState() <-chan ice.ConnectionState {
	return pc.connectionState
}

// Dial connects to the remote agent, acting as the controlling ice agent.
// Dial blocks until at least one ice candidate pair has successfully connected.
func (pc *peerConnection) Dial(ctx context.Context, remoteUfrag, remotePwd string) (*ice.Conn, error) {
	return pc.agent.Dial(ctx, remoteUfrag, remotePwd)
}

// Accept connects to the remote agent, acting as the controlled ice agent.
// Accept blocks until at least one ice candidate pair has successfully connected.
func (pc *peerConnection) Accept(ctx context.Context, remoteUfrag, remotePwd string) (*ice.Conn, error) {
	return pc.agent.Accept(ctx, remoteUfrag, remotePwd)
}
