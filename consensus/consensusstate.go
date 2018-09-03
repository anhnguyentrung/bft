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
	mutex sync.Mutex
	state ConsensusStateType
	view types.View
	lockedView types.View
	voteStorage map[ConsensusStateType]types.BlockVotes
}

func (cs *ConsensusState) votes(blockHeightId types.BlockHeightId, state ConsensusStateType) []types.Vote {
	heightId := blockHeightId.String()
	return cs.voteStorage[state][heightId]
}

func (cs *ConsensusState) addVote(vote types.Vote, blockHeightId types.BlockHeightId, state ConsensusStateType) {
	cs.mutex.Lock()
	heightId := blockHeightId.String()
	if _, ok := cs.voteStorage[state][heightId]; !ok {
		cs.voteStorage[state][heightId] = types.Votes{}
	}
	cs.voteStorage[state][heightId] = append(cs.voteStorage[state][heightId], vote)
	cs.mutex.Unlock()
}

func (cs *ConsensusState) removeVotes(blockHeightId types.BlockHeightId) {
	cs.mutex.Lock()
	heightId := blockHeightId.String()
	states := []ConsensusStateType{NewRound, PrePrepared, Prepared, Committed, FinalCommitted, RoundChange}
	for _, state := range states {
		if _, ok := cs.voteStorage[state][heightId]; ok {
			delete(cs.voteStorage[state], heightId)
		}
	}
	cs.mutex.Unlock()
}
