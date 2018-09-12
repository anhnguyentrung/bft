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
	proposal *types.Proposal
	pending *types.Proposal
	prepareCommits map[types.VoteType]*types.VoteSet // include prepare, commit
	roundChanges map[uint64]*types.VoteSet
}

func NewConsensusState(view types.View, validatorSet *types.ValidatorSet) *ConsensusState {
	cs := &ConsensusState{}
	cs.view = view
	cs.prepareCommits = make(map[types.VoteType]*types.VoteSet, 0)
	voteTypes := []types.VoteType{types.Prepare, types.Commit}
	for _, voteType := range voteTypes {
		cs.prepareCommits[voteType] = types.NewVoteSet(view, voteType, validatorSet)
	}
	cs.roundChanges = make(map[uint64]*types.VoteSet, 0)
	return cs
}

func (cs *ConsensusState) setProposal(proposal *types.Proposal) {
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
	if err := cs.prepareCommits[vote.Type].AddVote(vote, true); err != nil {
		log.Println(err)
		return
	}
	if vote.Type == types.Commit {
		cs.prepareCommits[types.Prepare].AddVote(vote, false)
	}
}

func (cs *ConsensusState) applyRoundChange(vote types.Vote, validatorSet *types.ValidatorSet) {
	view := vote.View
	round := view.Round
	if _, ok := cs.roundChanges[round]; !ok {
		cs.roundChanges[round] = types.NewVoteSet(view, vote.Type, validatorSet)
	}
	cs.roundChanges[round].AddVote(vote, true)
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

func (cs *ConsensusState) updateRound(round uint64) {
	cs.view.Round = round
	if !cs.isLocked() {
		cs.proposal = nil
	}
	cs.clearSmallerRound()
}

func (cs *ConsensusState) clearSmallerRound() {
	cs.rwMutex.Lock()
	defer cs.rwMutex.Unlock()
	for round, _ := range cs.roundChanges {
		if round < cs.view.Round {
			delete(cs.roundChanges, round)
		}
	}
}