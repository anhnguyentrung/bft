package network

import (
	"bft/database"
	"log"
	"bft/types"
	"bft/encoding"
)

type SyncState uint8

const (
	Catchup SyncState = iota
	InSync
)

func (syncState SyncState) String() string {
	switch syncState {
	case Catchup:
		return "catchup"
	case InSync:
		return "in sync"
	default:
		return "unknown"
	}
}

type Synchronizer struct {
	knownHeight uint64
	lastRequestedHeight uint64
	expectedHeight uint64
	state SyncState
}

func NewSynchronizer() *Synchronizer {
	return &Synchronizer{
		knownHeight:0,
		lastRequestedHeight:0,
		expectedHeight:1,
		state:InSync,
	}
}

func (s *Synchronizer) setState(state SyncState) {
	if s.state == state {
		return
	}
	s.state = state
}

func (s *Synchronizer) requestBlocks(c *Connection) {
	blockStore := database.GetBlockStore()
	lastHeight := blockStore.LastHeight()
	if lastHeight < s.lastRequestedHeight && c.IsAvailable() {
		return
	}
	if !c.IsAvailable() {
		log.Println("This connection is not available to sync")
		s.knownHeight = blockStore.LastHeight()
		s.lastRequestedHeight = 0
		s.setState(InSync)
		return
	}
	if s.lastRequestedHeight != s.knownHeight {
		start := s.expectedHeight
		end := s.knownHeight
		if end > 0 && end >= start {
			s.sendSyncRequest(c, start, end)
			s.lastRequestedHeight = end
		}
	}
}

func (s *Synchronizer) sendSyncRequest(c *Connection, start, end uint64) {
	syncRequest := types.SyncRequest{
		StartHeight: start,
		EndHeight: end,
	}
	payload, err := encoding.MarshalBinary(syncRequest)
	if err != nil {
		log.Println(err)
		return
	}
	message := types.NewMessage(types.SyncRequestMessage, payload)
	c.Send(message)
}

func (s *Synchronizer) updateKnownHeight(connection *Connection) {
	if connection.IsAvailable() {
		if connection.lastReceivedHandshake.Height() > s.knownHeight {
			s.knownHeight = connection.lastReceivedHandshake.Height()
		}
	}
}

func (s *Synchronizer) shouldSync() bool {
	blockStore := database.GetBlockStore()
	return s.lastRequestedHeight < s.knownHeight || blockStore.LastHeight() < s.lastRequestedHeight
}

func (s *Synchronizer) startSync(connection *Connection, localLastHeight uint64, remoteLastHeight uint64) {
	if remoteLastHeight > s.knownHeight {
		s.knownHeight = remoteLastHeight
	}
	if !s.shouldSync() {
		return
	}
	if s.state == InSync {
		s.setState(Catchup)
		s.expectedHeight = localLastHeight + 1
	}
	s.requestBlocks(connection)
}

func (s *Synchronizer) handleHandshake(handshake *types.Handshake, connection *Connection) {
	blockStore := database.GetBlockStore()
	localLastHeight := blockStore.LastHeight()
	remoteLastHeight := handshake.LastHeightId.Height
	s.updateKnownHeight(connection)
	connection.Sync(false)
	if localLastHeight < remoteLastHeight {
		s.startSync(connection, localLastHeight, remoteLastHeight)
		return
	}
}


