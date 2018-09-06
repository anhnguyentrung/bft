package consensus

import (
	"sync"
	"bft/types"
)

type ConsensusStateType uint8

const (
	NewRound ConsensusStateType = iota
	PrePrepared
	Prepared
	Committed
	FinalCommitted
	RoundChange
)

type ConsensusState struct {
	mutex sync.Mutex
	stateType ConsensusStateType
	view types.View
	lockedView types.View
	proposal types.Proposal
	voteStorage map[ConsensusStateType]types.BlockVotes
}

func NewConsensusState() *ConsensusState {
	cs := &ConsensusState{}
	cs.stateType = NewRound
	cs.view.Round = 1
	cs.view.HeightId.Height = 1
	cs.voteStorage = make(map[ConsensusStateType]types.BlockVotes, 0)
	states := []ConsensusStateType{NewRound, PrePrepared, Prepared, Committed, FinalCommitted, RoundChange}
	for _, state := range states {
		cs.voteStorage[state] = make(types.BlockVotes, 0)
	}
	return cs
}

func (cs *ConsensusState) setProposal(proposal types.Proposal) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.proposal = proposal
}

func (cs *ConsensusState) setSate(state ConsensusStateType) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	if cs.stateType != state {
		cs.stateType = state
	}
	//TODO: process pending requests or backlogs
}