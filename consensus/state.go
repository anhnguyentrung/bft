package consensus

import (
	"sync"
	"bft/types"
	"log"
	"math"
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
	pendingProposal *types.Proposal
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

func (cs *ConsensusState) round() uint64 {
	return cs.view.Round
}

func (cs *ConsensusState) height() uint64 {
	return cs.view.Height
}

func (cs *ConsensusState) setProposal(proposal *types.Proposal) {
	cs.proposal = proposal
}

func (cs *ConsensusState) setSate(state ConsensusStateType) {
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
	err := cs.roundChanges[round].AddVote(vote, true)
	if err != nil {
		log.Println(err)
	}
}

func (cs *ConsensusState) proposalHeightId() types.BlockHeightId {
	return cs.proposal.BlockHeightId()
}

func (cs *ConsensusState) lock() {
	proposalHeightId := cs.proposalHeightId()
	if proposalHeightId.IsValid() {
		cs.lockedHeightId = proposalHeightId
	}
}

func (cs *ConsensusState) unLock() {
	cs.lockedHeightId = types.BlockHeightId{}
}

func (cs *ConsensusState) isLocked() bool {
	if !cs.lockedHeightId.IsValid() {
		return false
	}
	return true
}

func (cs *ConsensusState) updateView(v types.View) {
	if cs.view.Compare(v) != 0 {
		cs.view = v
		for _, votSet := range cs.roundChanges {
			votSet.ChangeView(v)
		}
		for _, votSet := range cs.prepareCommits {
			votSet.ChangeView(v)
		}
		if !cs.isLocked() {
			cs.proposal = nil
		}
	}
}

func (cs *ConsensusState) clearSmallerRound() {
	for round, _ := range cs.roundChanges {
		if round < cs.view.Round {
			delete(cs.roundChanges, round)
		}
	}
}

func (cs *ConsensusState) getMaxRound(threshold int) (maxRound uint64) {
	maxRound = uint64(math.MaxUint64)
	for round, voteSet := range cs.roundChanges {
		voteNum := voteSet.Size()
		if voteNum >= threshold && maxRound < uint64(voteNum) {
			maxRound = round
		}
	}
	return
}

func (cs *ConsensusState) prepares() *types.VoteSet {
	cs.rwMutex.RLock()
	defer cs.rwMutex.RUnlock()
	return cs.prepareCommits[types.Prepare]
}

func (cs *ConsensusState) commits() *types.VoteSet {
	cs.rwMutex.RLock()
	defer cs.rwMutex.RUnlock()
	return cs.prepareCommits[types.Commit]
}