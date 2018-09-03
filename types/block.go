package types

import "time"

type BlockHeader struct {
	Id Hash
	Height uint64
	PreviousId Hash
	Proposer string
	Timestamp time.Time
}

type Block struct {
	Header BlockHeader
}
