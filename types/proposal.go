package types

import (
	"time"
	"bft/crypto"
)

type View struct {
	Round uint64
	Height uint64
}

type PrePrepare struct {
	View View
	ProposalBlock Block
	Timestamp time.Time
	Signature crypto.Signature
}