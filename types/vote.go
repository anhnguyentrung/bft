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

func (voteType VoteType) String() string {
	switch voteType {
	case Prepare:
		return "Prepare"
	case Commit:
		return "Commit"
	case RoundChange:
		return "RoundChange"
	default:
		return ""
	}
}

type Vote struct {
	Hash 		Hash
	Address 	string // voter's address
	Type 		VoteType
	View 		View
	BlockId 	Hash
	Signature 	crypto.Signature
}
