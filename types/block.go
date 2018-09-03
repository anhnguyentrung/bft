package types

import (
	"time"
	"bft/crypto"
)

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

type SignedBlock struct {
	Block Block
	Signature crypto.Signature
}
