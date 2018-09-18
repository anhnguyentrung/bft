package network

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


