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
	lastHeight := blockStore.LastHeight()
	if lastHeight < synchronizer.lastRequestedHeight && c.IsAvailable() {
		return
	}
	if !c.IsAvailable() {
		log.Println("This connection is not available to sync")
		synchronizer.knownHeight = blockStore.LastHeight()
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

func (synchronizer *Synchronizer) updateKnownHeight(connection *Connection) {
	if connection.IsAvailable() {
		if connection.lastReceivedHandshake.Height() > synchronizer.knownHeight {
			synchronizer.knownHeight = connection.lastReceivedHandshake.Height()
		}
	}
}

func (synchronizer *Synchronizer) shouldSync() bool {
	blockStore := database.GetBlockStore()
	return synchronizer.lastRequestedHeight < synchronizer.knownHeight || blockStore.LastHeight() < synchronizer.lastRequestedHeight
}

func (synchronizer *Synchronizer) startSync(connection *Connection, localLastHeight uint64, remoteLastHeight uint64) {
	if remoteLastHeight > synchronizer.knownHeight {
		synchronizer.knownHeight = remoteLastHeight
	}
	if !synchronizer.shouldSync() {
		return
	}
	if synchronizer.state == InSync {
		synchronizer.setState(Catchup)
		synchronizer.expectedHeight = localLastHeight + 1
	}
	synchronizer.requestBlocks(connection)
}

func (synchronizer *Synchronizer) handleHandshake(handshake *types.Handshake, connection *Connection) {
	blockStore := database.GetBlockStore()
	localLastHeight := blockStore.LastHeight()
	remoteLastHeight := handshake.LastHeightId.Height
	synchronizer.updateKnownHeight(connection)
	connection.Sync(false)
	if localLastHeight < remoteLastHeight {
		synchronizer.startSync(connection, localLastHeight, remoteLastHeight)
		return
	}
}


