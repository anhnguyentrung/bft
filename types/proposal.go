package types

import (
	"time"
	"bft/crypto"
)

type View struct {
	Round 		uint64
	HeightId 	BlockHeightId
}

func (v View) Height() uint64 {
	return v.HeightId.Height
}

func (v View) Id() Hash {
	return v.HeightId.Id
}

func (v View) Next() View {
	return View{
		Round: v.Round + 1,
		HeightId: BlockHeightId{
			Height: v.Height() + 1,
		},
	}
}

func (v View) Compare(target View) int {
	if v.Height() < target.Height() {
		return -1
	}
	if v.Height() > target.Height() {
		return 1
	}
	if v.Round < target.Round {
		return -1
	}
	if v.Round > target.Round {
		return 1
	}
	return 0
}

type Proposal struct {
	View 			View
	ProposalBlock 	SignedBlock
	Timestamp 		time.Time
	Signature 		crypto.Signature
}