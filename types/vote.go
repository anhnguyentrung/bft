package types

import (
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
	Signature 	crypto.Signature
}

type Votes = []Vote
type BlockVotes = map[string]Votes // string: block's height + block's id
