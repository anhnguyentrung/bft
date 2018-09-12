package consensus

import (
	"bft/types"
	"sync"
	"log"
	"crypto/sha256"
	"bft/crypto"
	"math"
	"time"
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
	roundChangeTimer *time.Timer
	f int // maximum number of faults
}

func NewConsensusManager(validators types.Validators, address string) *ConsensusManager {
	cm := &ConsensusManager{}
	cm.validatorSet = types.NewValidatorSet(validators, address)
	cm.f = int(math.Floor(float64(cm.validatorSet.Size())/3))
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
		cm.onProposal(&proposal)
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

func (cm *ConsensusManager) onProposal(proposal *types.Proposal) {
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
	if !cm.verifyRoundChange(vote) {
		return
	}
	cm.currentState.applyRoundChange(vote, cm.validatorSet)
	if cm.shouldChangeRound(vote.View.Round) {
		cm.changeRound(vote.View.Round)
	}
}

func (cm *ConsensusManager) shouldChangeRound(round uint64) bool {
	currentState := cm.currentState
	voteCount := currentState.roundChanges[round].Size()
	if currentState.stateType == RoundChange && voteCount == cm.f + 1 {
		if currentState.round() < round {
			return true
		}
	}
	return false
}

func (cm *ConsensusManager) shouldStartNewRound(round uint64) bool {
	currentState := cm.currentState
	stateType := currentState.stateType
	currentRound := currentState.round()
	voteCount := currentState.roundChanges[round].Size()
	if voteCount == 2*cm.f + 1 && (stateType == RoundChange || currentRound < round) {
		return true
	}
	return false
}

func (cm *ConsensusManager) isProposer() bool {
	if cm.validatorSet == nil || cm.validatorSet.Size() == 0 {
		log.Println("validator set should not be nil or empty")
		return false
	}
	self := cm.validatorSet.Self()
	return cm.validatorSet.IsProposer(self)
}

func (cm *ConsensusManager) startNewRound(round uint64) {
	if cm.head == nil {
		log.Fatal("blockchain must have a head")
	}
	newView := types.View{
		0,
		cm.head.Height() + 1,
	}
	if cm.currentState == nil {
		log.Println("initial round")
	} else if cm.head.Height() >= cm.currentState.height() {
		log.Println("catch up latest proposal")
	} else if cm.head.Height() == cm.currentState.height() - 1 {
		if round == 0 {
			return
		}
		if round < cm.currentState.round() {
			log.Println("new round should be greater than current round")
			return
		}
		newView.Round = round
	} else {
		log.Println("new height should be greater than current height")
	}
	// delete all old votes
	for k, _ := range cm.currentState.roundChanges {
		delete(cm.currentState.roundChanges, k)
	}
	cm.changeRound(round)
	cm.validatorSet.CalculateProposer(round)
	cm.currentState.setSate(NewRound)
	if newView.Round != 0 && cm.isProposer() {
		if cm.currentState.isLocked() {
			if cm.currentState.proposal != nil {
				cm.sendProposal(*cm.currentState.proposal)
			}
		} else {
			if cm.currentState.pendingProposal != nil {
				cm.sendProposal(*cm.currentState.pendingProposal)
			}
		}
	}
	cm.newRoundChangeTimer()
}

// check whether the validator received +2/3 prepare
func (cm *ConsensusManager) canEnterPrepared() bool {
	currentState := cm.currentState
	if currentState.stateType >= Prepared {
		log.Printf("current state %s is greater than prepared", currentState.stateType.String())
		return false
	}
	if currentState.prepareCommits[types.Prepare].Size() < 2*cm.f + 1 {
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
	if currentState.prepareCommits[types.Commit].Size() < 2*cm.f + 1 {
		return false
	}
	return true
}

func (cm *ConsensusManager) enterPrePrepared(proposal *types.Proposal) {
	currentState := cm.currentState
	if currentState.stateType == NewRound {
		if currentState.isLocked() {
			if currentState.proposal.BlockHeightId().Equals(currentState.lockedHeightId) {
				currentState.setSate(Prepared)
				cm.sendVote(types.Commit)
			}
		} else {
			currentState.setProposal(proposal)
			currentState.setSate(PrePrepared)
			cm.sendVote(types.Prepare)
		}
	}
}

func (cm *ConsensusManager) enterPrepared() {
	currentState := cm.currentState
	// lock proposal block
	currentState.lock()
	currentState.setSate(Prepared)
	cm.sendVote(types.Commit)
}

func (cm *ConsensusManager) enterCommitted() {
	currentState := cm.currentState
	// lock proposal block
	currentState.lock()
	currentState.setSate(Committed)
	//TODO: Commit proposal block
}

func (cm *ConsensusManager) changeRound(round uint64) {
	cm.currentState.setSate(RoundChange)
	cm.currentState.updateRound(round)
	cm.newRoundChangeTimer()
	cm.sendVote(types.RoundChange)
}

func (cm *ConsensusManager) stopRoundChangeTimer() {
	if cm.roundChangeTimer != nil {
		cm.roundChangeTimer.Stop()
	}
}

func (cm *ConsensusManager) newRoundChangeTimer() {
	cm.stopRoundChangeTimer()
	timeout := types.RequestTimeout * time.Millisecond
	cm.roundChangeTimer = time.AfterFunc(timeout, cm.handleTimeout)
}

func (cm *ConsensusManager) handleTimeout() {
	if cm.currentState.stateType != RoundChange {
		threshold := 2*cm.f + 1
		maxRound := cm.currentState.getMaxRound(threshold)
		if maxRound != math.MaxUint64 && maxRound > cm.currentState.round() {
			cm.changeRound(maxRound)
			return
		}
	}
	if cm.head != nil && cm.head.Height() >= cm.currentState.height() {
		cm.startNewRound(0)
	} else {
		cm.changeRound(cm.currentState.round() + 1)
	}
}

func (cm *ConsensusManager) sendVote(voteType types.VoteType) {
	voter := cm.validatorSet.Self()
	view := cm.currentState.proposal.View
	blockId := types.Hash{}
	if voteType != types.RoundChange {
		blockId = cm.currentState.proposal.BlockId()
	}
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
	if err != nil {
		log.Println(err)
		return
	}
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

func (cm *ConsensusManager) sendProposal(proposal types.Proposal) {
	payload, err := cm.enDecoder.Encode(proposal)
	if err != nil {
		log.Println(err)
		return
	}
	length := uint32(len(payload))
	message := types.Message{
		types.MessageHeader{
			types.ProposalMessage,
			length,
		},
		payload,
	}
	cm.broadcaster(message)
}
