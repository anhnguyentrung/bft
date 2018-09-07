package consensus

import (
	"bft/types"
	"sync"
	"log"
	"crypto/sha256"
	"bft/crypto"
)

type BroadcastFunc func(message types.Message)

type ConsensusManager struct {
	mutex sync.Mutex
	currentState *ConsensusState
	validatorSet *types.ValidatorSet
	head *types.BlockHeader
	enDecoder types.EnDecoder
	signer crypto.SignFunc
	broadcaster BroadcastFunc
}

func NewConsensusManager(validators types.Validators, address string) *ConsensusManager {
	cm := &ConsensusManager{}
	cm.currentState = NewConsensusState()
	cm.validatorSet = types.NewValidatorSet(validators, address)
	return cm
}

func (cm *ConsensusManager) SetEnDecoder(enDecoder types.EnDecoder) {
	cm.enDecoder = enDecoder
}

func (cm *ConsensusManager) SetSigner(signer crypto.SignFunc) {
	cm.signer = signer
}

func (cm *ConsensusManager) SetBroadcaster(broadcaster BroadcastFunc) {
	cm.broadcaster = broadcaster
}

func (cm *ConsensusManager) getVotes(blockId types.Hash, voteType types.VoteType) []types.Vote {
	return cm.currentState.voteStorage[voteType][blockId.String()]
}

func (cm *ConsensusManager) addVote(vote types.Vote, blockId types.Hash, voteType types.VoteType) {
	cm.mutex.Lock()
	if _, ok := cm.currentState.voteStorage[voteType][blockId.String()]; !ok {
		cm.currentState.voteStorage[voteType][blockId.String()] = types.Votes{}
	}
	cm.currentState.voteStorage[voteType][blockId.String()] = append(cm.currentState.voteStorage[voteType][blockId.String()], vote)
	cm.mutex.Unlock()
}

func (cm *ConsensusManager) removeVotes(blockId types.Hash) {
	cm.mutex.Lock()
	voteTypes := []types.VoteType{types.Prepare, types.Commit, types.RoundChange}
	for _, voteType := range voteTypes {
		if _, ok := cm.currentState.voteStorage[voteType][blockId.String()]; ok {
			delete(cm.currentState.voteStorage[voteType], blockId.String())
		}
	}
	cm.mutex.Unlock()
}

func (cm *ConsensusManager) Receive(message types.Message) {
	messageType := message.Header.Type
	switch messageType {
	case types.VoteMessage:
		vote := types.Vote{}
		err := cm.enDecoder.Decode(message.Payload, &vote)
		if err != nil {
			log.Fatal(err)
		}
		cm.onVote(vote)
	case types.ProposalMessage:
		proposal := types.Proposal{}
		err := cm.enDecoder.Decode(message.Payload, &proposal)
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
	// check proposal's round and height
	if result := proposal.View.Compare(cm.currentState.view); result != 0 {
		// if proposal is an existing block, broadcast commit
		if result < 0 {
			//TODO: handle the existing block
		}
	}
	proposer := proposal.ProposalBlock.Header().Proposer
	// Is proposal from valid proposer
	if !cm.validatorSet.IsProposer(proposer) {
		log.Println("Don't accept a proposal from unknown proposer")
		return
	}
	// check proposal
	if !cm.verifyProposal(proposal) {
		return
	}
	//TODO: handle the future block
	cm.currentState.enterPrePrepared(proposal)
}

func (cm *ConsensusManager) onPrepare(vote types.Vote) {
	if !cm.verifyPrepare(vote) {
		return
	}
	cm.currentState.applyVote(vote)
	proposalHeightId := cm.currentState.getProposalHeightId()
	// if the validator have a locked block, she should broadcast COMMIT on the locked block and enter prepared
	if cm.currentState.isLocked() && proposalHeightId.Equals(cm.currentState.getLockedHeightId()) {
		cm.currentState.enterPrepared()
	}
	// the validator received +2/3 prepare
	if
}

func (cm *ConsensusManager) onCommit(vote types.Vote) {

}

func (cm *ConsensusManager) onRoundChange(vote types.Vote) {

}

func (cm *ConsensusManager) broadCast(voteType types.VoteType) {
	voter := cm.validatorSet.Self()
	view := cm.currentState.proposal.View
	blockId := cm.currentState.proposal.BlockId()
	vote := types.Vote {
		voter,
		voteType,
		view,
		blockId,
		crypto.Signature{},
	}
	buf, err := cm.enDecoder.Encode(vote)
	if err != nil {
		log.Println(err)
		return
	}
	hash := sha256.Sum256(buf)
	sig, err := cm.signer(hash[:])
	if err != nil {
		log.Println(err)
		return
	}
	vote.Signature = sig
	payload, err := cm.enDecoder.Encode(vote)
	length := uint32(len(payload))
	message := types.Message{
		types.MessageHeader{
			types.VoteMessage,
			length,
		},
		payload,
	}
	cm.broadcaster(message)
}
