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

func (synchronizer *Synchronizer) setState(state SyncState) {
	if synchronizer.state == state {
		return
	}
	synchronizer.state = state
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
			synchronizer.sendSyncRequest(c, start, end)
			synchronizer.lastRequestedHeight = end
		}
	}
}

func (synchronizer *Synchronizer) sendSyncRequest(c *Connection, start, end uint64) {
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


