package network

import (
	"bft/database"
	"log"
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
	source *Connection
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

func (synchronizer *Synchronizer) setState(state SyncState) {
	if synchronizer.state == state {
		return
	}
	synchronizer.state = state
}

func (synchronizer *Synchronizer) shouldSync(c *Connection) bool {
	blockStore := database.GetBlockStore()
	if c != nil && synchronizer.state == Catchup {
		return c.lastHeightId.IsValid() && c.lastHeightId.Height < blockStore.Head().Height()
	}
	return false
}

func (synchronizer *Synchronizer) requestBlocks(c *Connection) {
	blockStore := database.GetBlockStore()
	lastHeight := blockStore.Head().Height()
	if lastHeight < synchronizer.lastRequestedHeight && c.IsAvailable() {
		return
	}
	if !c.IsAvailable() {
		log.Println("This connection is not available to sync")
		synchronizer.knownHeight = blockStore.Head().Height()
		synchronizer.lastRequestedHeight = 0
		synchronizer.setState(InSync)
		return
	}
	if synchronizer.lastRequestedHeight != synchronizer.knownHeight {
		start := synchronizer.expectedHeight
		end := synchronizer.knownHeight
		if end > 0 && end >= start {

			synchronizer.lastRequestedHeight = end
		}
	}
}

func (synchronizer *Synchronizer) sendSyncRequest(c *Connection, start, end uint32) {

}


