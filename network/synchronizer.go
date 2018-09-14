package network

type SyncState uint8

const (
	Catchup SyncState = iota
	InSync
)

type Synchronizer struct {
	knownHeight uint64
	lastRequestedHeight uint64
	expectedHeight uint64
	source *Connection
	state SyncState
}
