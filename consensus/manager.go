package consensus

import (
	"bft/types"
	"sync"
	"log"
	"crypto/sha256"
	"bft/crypto"
	"math"
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
	cm.enterPrePrepared(proposal)
}

func (cm *ConsensusManager) onPrepare(vote types.Vote) {
	if !cm.verifyPrepare(vote) {
		return
	}
	cm.currentState.applyVote(vote)
	proposalHeightId := cm.currentState.getProposalHeightId()
	// if the validator have a locked block, she should broadcast COMMIT on the locked block and enter prepared
	if cm.currentState.isLocked() && proposalHeightId.Equals(cm.currentState.getLockedHeightId()) {
		cm.enterPrepared()
	}
	// the validator received +2/3 prepare
	if cm.canEnterPrepared() {
		cm.enterPrepared()
	}
}

func (cm *ConsensusManager) onCommit(vote types.Vote) {
	if !cm.verifyCommit(vote) {
		return
	}
	cm.currentState.applyVote(vote)
}

func (cm *ConsensusManager) onRoundChange(vote types.Vote) {

}

// check whether the validator received +2/3 prepare
func (cm *ConsensusManager) canEnterPrepared() bool {
	currentState := cm.currentState
	if currentState.stateType >= Prepared {
		log.Printf("current state %s is greater than prepared", currentState.stateType.String())
		return false
	}
	if currentState.voteStorage[types.Prepare].Size() < int(math.Floor(float64(cm.validatorSet.Size()*2)/3)) + 1 {
		return false
	}
	return true
}

// check whether the validator received +2/3 commit
func (cm *ConsensusManager) canEnterCommitted() bool {
	currentState := cm.currentState
	if currentState.stateType >= Committed {
		log.Printf("current state %s is greater than prepared", currentState.stateType.String())
		return false
	}
	if currentState.voteStorage[types.Commit].Size() < int(math.Floor(float64(cm.validatorSet.Size()*2)/3)) + 1 {
		return false
	}
	return true
}

func (cm *ConsensusManager) enterPrePrepared(proposal types.Proposal) {
	currentState := cm.currentState
	if currentState.stateType == NewRound {
		if currentState.isLocked() {
			if currentState.proposal.BlockHeightId().Equals(currentState.lockedHeightId) {
				currentState.setSate(Prepared)
				cm.broadCast(types.Commit)
			}
		} else {
			currentState.setProposal(proposal)
			currentState.setSate(PrePrepared)
			cm.broadCast(types.Prepare)
		}
	}
}

func (cm *ConsensusManager) enterPrepared() {
	currentState := cm.currentState
	// lock proposal block
	currentState.lock()
	currentState.setSate(Prepared)
	cm.broadCast(types.Commit)
}

func (cm *ConsensusManager) enterCommitted() {
	currentState := cm.currentState
	// lock proposal block
	currentState.lock()
	currentState.setSate(Committed)
	//TODO: Commit proposal block
}

func (cm *ConsensusManager) broadCast(voteType types.VoteType) {
	voter := cm.validatorSet.Self()
	view := cm.currentState.proposal.View
	blockId := cm.currentState.proposal.BlockId()
	vote := types.Vote {
		types.Hash{},
		voter.Address,
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
	vote.Hash = hash
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
