package consensus

import (
	"bft/types"
	"sync"
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

type ConsensusState struct {
	state ConsensusStateType
	view types.View
	lockedView types.View
	voteStorage map[ConsensusStateType]types.BlockVotes
}

func NewConsensusState() *ConsensusState {
	cs := &ConsensusState{}
	cs.state = NewRound
	cs.view.Round = 1
	cs.view.HeightId.Height = 1
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

func (cm *ConsensusManager) Receive(message types.Message, decoder types.DeserializeFunc) {
	messageType := message.Header.Type
	switch messageType {
	case types.VoteMessage:
		vote := types.Vote{}
		err := decoder(message.Payload, &vote)
		if err != nil {
			log.Fatal(err)
		}
		cm.onVote(vote)
	case types.ProposalMessage:
		proposal := types.Proposal{}
		err := decoder(message.Payload, &proposal)
		if err != nil {
			log.Fatal(err)
		}
		cm.onProposal(proposal)
	}
}

func (cm *ConsensusManager) onVote(vote types.Vote) {
	switch vote.Type {
	case types.Prepare:
		cm.onPrepare(vote)
	case types.Commit:
		cm.onCommit(vote)
	case types.RoundChange:
		cm.onRoundChange(vote)
	}
}

func (cm *ConsensusManager) onProposal(proposal types.Proposal) {

}

func (cm *ConsensusManager) onPrepare(vote types.Vote) {

}

func (cm *ConsensusManager) onCommit(vote types.Vote) {

}

func (cm *ConsensusManager) onRoundChange(vote types.Vote) {

}
