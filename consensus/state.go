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
	rwMutex sync.RWMutex
	stateType ConsensusStateType
	view types.View
	lockedView types.View
	lockedHeightId types.BlockHeightId
	proposal types.Proposal
	voteStorage map[types.VoteType]types.BlockVotes
}

func NewConsensusState() *ConsensusState {
	cs := &ConsensusState{}
	cs.stateType = NewRound
	cs.view.Round = 1
	cs.view.Height = 1
	cs.voteStorage = make(map[types.VoteType]types.BlockVotes, 0)
	voteTypes := []types.VoteType{types.Prepare, types.Commit, types.RoundChange}
	for _, voteType := range voteTypes {
		cs.voteStorage[voteType] = make(types.BlockVotes, 0)
	}
	return cs
}

func (cs *ConsensusState) setProposal(proposal types.Proposal) {
	cs.rwMutex.Lock()
	defer cs.rwMutex.Unlock()
	cs.proposal = proposal
}

func (cs *ConsensusState) setSate(state ConsensusStateType) {
	cs.rwMutex.Lock()
	defer cs.rwMutex.Unlock()
	if cs.stateType != state {
		cs.stateType = state
	}
	//TODO: process pending requests or backlogs
}

func (cs *ConsensusState) applyVote(vote types.Vote) {
	cs.voteStorage[vote.Type][vote.BlockId.String()] = append(cs.voteStorage[vote.Type][vote.BlockId.String()], vote)
}

func (cs *ConsensusState) lock() {
	cs.rwMutex.Lock()
	defer cs.rwMutex.Unlock()
	if cs.getProposalHeightId().IsValid() {
		cs.lockedHeightId = cs.getProposalHeightId()
	}
}

func (cs *ConsensusState) isLocked() bool {
	cs.rwMutex.RLock()
	defer cs.rwMutex.RUnlock()
	if !cs.lockedHeightId.IsValid() {
		return false
	}
	return true
}

func (cs *ConsensusState) getLockedHeightId() types.BlockHeightId {
	cs.rwMutex.RLock()
	defer cs.rwMutex.RUnlock()
	return cs.lockedHeightId
}

func (cs *ConsensusState) getProposalHeightId() types.BlockHeightId {
	cs.rwMutex.RLock()
	defer cs.rwMutex.RUnlock()
	return cs.proposal.BlockHeightId()
}