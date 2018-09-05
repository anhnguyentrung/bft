package consensus

import (
	"bft/types"
	"sync"
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
	state ConsensusStateType
	view types.View
	lockedView types.View
	voteStorage map[ConsensusStateType]types.BlockVotes
}

func NewConsensusState() *ConsensusState {
	cs := &ConsensusState{}
	cs.state = NewRound
	cs.voteStorage = make(map[ConsensusStateType]types.BlockVotes, 0)
	states := []ConsensusStateType{NewRound, PrePrepared, Prepared, Committed, FinalCommitted, RoundChange}
	for _, state := range states {
		cs.voteStorage[state] = make(types.BlockVotes, 0)
	}
	return cs
}

type ConsensusManager struct {
	mutex sync.Mutex
	state *ConsensusState
	validators []types.Validator
}

func NewConsensusManager() *ConsensusManager {
	cm := &ConsensusManager{}
	cm.state = NewConsensusState()
	return cm
}

func (cm *ConsensusManager) getVotes(blockHeightId types.BlockHeightId, state ConsensusStateType) []types.Vote {
	heightId := blockHeightId.String()
	return cm.state.voteStorage[state][heightId]
}

func (cm *ConsensusManager) addVote(vote types.Vote, blockHeightId types.BlockHeightId, state ConsensusStateType) {
	cm.mutex.Lock()
	heightId := blockHeightId.String()
	if _, ok := cm.state.voteStorage[state][heightId]; !ok {
		cm.state.voteStorage[state][heightId] = types.Votes{}
	}
	cm.state.voteStorage[state][heightId] = append(cm.state.voteStorage[state][heightId], vote)
	cm.mutex.Unlock()
}

func (cm *ConsensusManager) removeVotes(blockHeightId types.BlockHeightId) {
	cm.mutex.Lock()
	heightId := blockHeightId.String()
	states := []ConsensusStateType{NewRound, PrePrepared, Prepared, Committed, FinalCommitted, RoundChange}
	for _, state := range states {
		if _, ok := cm.state.voteStorage[state][heightId]; ok {
			delete(cm.state.voteStorage[state], heightId)
		}
	}
	cm.mutex.Unlock()
}

func (cm *ConsensusManager) SetValidators(validators []types.Validator) {
	cm.validators = validators
}

func (cm *ConsensusManager) Receive(vote types.Vote) {
	vote := types.Vote{}
	err = Unmark([]byte(str), &message)
}
