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
	Voter 		Validator
	Type 		VoteType
	View 		View
	BlockId 	Hash
	Timestamp 	time.Time
	Signature 	crypto.PublicKey
}

type Votes = []Vote
type BlockVotes = map[string]Votes // string: block's height + block's id
