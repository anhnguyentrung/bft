package types

import (
	"time"
	"bft/crypto"
)

type VoteType uint8

const (
	Prepare VoteType = iota
	Commit
	RoundChange
)

type Vote struct {
	Voter 			Validator
	Type 			VoteType
	View 			View
	ProposalBlockId Hash
	Timestamp 		time.Time
	Signature 		crypto.PublicKey
}
