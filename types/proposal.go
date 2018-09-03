package types

import "time"

type View struct {
	Round uint64
	Height uint64
}

type PrePrepare struct {
	View View
	ProposalBlock Block
	Timestamp time.Time
	Signature []byte
}
