package types

import (
	"time"
	"bft/crypto"
)

type View struct {
	Round 		uint64
	HeightId 	BlockHeightId
}

type PrePrepare struct {
	View 			View
	ProposalBlock 	Block
	Timestamp 		time.Time
	Signature 		crypto.Signature
}