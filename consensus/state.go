package consensus

import (
	"sync"
	"bft/types"
	"log"
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

func (cst ConsensusStateType) String() string {
	switch cst {
	case NewRound:
		return "new round"
	case PrePrepared:
		return "pre-prepared"
	case Prepared:
		return "prepared"
	case Committed:
		return "committed"
	case FinalCommitted:
		return "final commited"
	case RoundChange:
		return "round change"
	default:
		return ""
	}
}

type ConsensusState struct {
	rwMutex sync.RWMutex
	stateType ConsensusStateType
	view types.View
	lockedHeightId types.BlockHeightId
	proposal types.Proposal
	voteStorage map[types.VoteType]*types.VoteSet
}

func NewConsensusState(view types.View, validatorSet types.ValidatorSet) *ConsensusState {
	cs := &ConsensusState{}
	cs.view = view
	cs.voteStorage = make(map[types.VoteType]*types.VoteSet, 0)
	voteTypes := []types.VoteType{types.Prepare, types.Commit, types.RoundChange}
	for _, voteType := range voteTypes {
		cs.voteStorage[voteType] = types.NewVoteSet(view, voteType, validatorSet)
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
	if err := cs.voteStorage[vote.Type].AddVote(vote, true); err != nil {
		log.Println(err)
		return
	}
	if vote.Type == types.Commit {
		cs.voteStorage[types.Prepare].AddVote(vote, false)
	}
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