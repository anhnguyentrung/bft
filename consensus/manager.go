package consensus

import (
	"bft/types"
	"sync"
	"log"
	"crypto/sha256"
	"bft/crypto"
	"math"
	"time"
	"bft/database"
	"bft/encoding"
	"fmt"
)

type BroadcastFunc func(message types.Message)

type ConsensusManager struct {
	mutex sync.Mutex
	currentState *ConsensusState
	validatorSet *types.ValidatorSet
	blockStore *database.BlockStore
	signer crypto.SignFunc
	broadcaster BroadcastFunc
	roundChangeTimer *time.Timer
	f int // maximum number of faults
}

func NewConsensusManager(validators types.Validators, address string) *ConsensusManager {
	cm := &ConsensusManager{}
	cm.validatorSet = types.NewValidatorSet(validators, address)
	cm.blockStore = database.GetBlockStore()
	cm.f = int(math.Floor(float64(cm.validatorSet.Size())/3))
	return cm
}

func (cm *ConsensusManager) SetSigner(signer crypto.SignFunc) {
	cm.signer = signer
}

func (cm *ConsensusManager) SetBroadcaster(broadcaster BroadcastFunc) {
	cm.broadcaster = broadcaster
}

func (cm *ConsensusManager) head() *types.Block {
	return cm.blockStore.Head()
}

func (cm *ConsensusManager) Receive(message types.Message) {
	messageType := message.Type
	switch messageType {
	case types.VoteMessage:
		vote := message.ToVote(encoding.UnmarshalBinary)
		cm.onVote(vote)
	case types.ProposalMessage:
		proposal := message.ToProposal(encoding.UnmarshalBinary)
		cm.onProposal(proposal)
	}
}

func (cm *ConsensusManager) onVote(vote *types.Vote) {
	if vote == nil {
		log.Println("unable to parse vote")
		return
	}
	switch vote.Type {
	case types.Prepare:
		cm.onPrepare(*vote)
	case types.Commit:
		cm.onCommit(*vote)
	case types.RoundChange:
		cm.onRoundChange(*vote)
	}
}

func (cm *ConsensusManager) onProposal(proposal *types.Proposal) {
	if proposal == nil {
		log.Println("unable to parse proposal")
		return
	}
	// check proposal's round and height
	if result := proposal.View.Compare(cm.currentState.view); result != 0 {
		// if proposal is an existing block, broadcast commit
		if result < 0 {
			//TODO: handle the existing block
		}
		return
	}
	proposer := proposal.Proposer()
	// Is proposal from valid proposer
	if !cm.validatorSet.IsProposer(proposer) {
		log.Println("Don't accept a proposal from unknown proposer")
		return
	}
	// check proposal
	if err := cm.verifyProposal(proposal); err != nil {
		log.Println(err)
		//TODO: handle the future block
		cm.sendRoundChange(cm.currentState.round() + 1)
		return
	}
	cm.enterPrePrepared(proposal)
}

func (cm *ConsensusManager) enterPrePrepared(proposal *types.Proposal) {
	currentState := cm.currentState
	if currentState.stateType == NewRound {
		if currentState.isLocked() {
			if currentState.proposal.BlockHeightId().Equals(currentState.lockedHeightId) {
				currentState.setSate(Prepared)
				cm.sendVote(types.Commit)
			} else {
				// should go to next round
				cm.sendRoundChange(cm.currentState.round() + 1)
			}
		} else {
			currentState.setProposal(proposal)
			currentState.setSate(PrePrepared)
			cm.sendVote(types.Prepare)
		}
	}
}

func (cm *ConsensusManager) onPrepare(vote types.Vote) {
	if err := cm.verifyVote(vote); err != nil {
		return
	}
	cm.currentState.applyVote(vote)
	proposalHeightId := cm.currentState.proposalHeightId()
	// if the validator have a locked block, she should broadcast COMMIT on the locked block and enter prepared
	// or the validator received +2/3 prepare
	if (cm.currentState.isLocked() && proposalHeightId.Equals(cm.currentState.lockedHeightId)) || cm.canEnterPrepared() {
		cm.enterPrepared()
	}
}

func (cm *ConsensusManager) enterPrepared() {
	currentState := cm.currentState
	// lock proposal block
	currentState.lock()
	currentState.setSate(Prepared)
	cm.sendVote(types.Commit)
}

// check whether the validator received +2/3 prepare
func (cm *ConsensusManager) canEnterPrepared() bool {
	currentState := cm.currentState
	if currentState.stateType >= Prepared {
		log.Printf("current state %s is greater than prepared", currentState.stateType.String())
		return false
	}
	if currentState.prepares().Size() < 2*cm.f + 1 {
		return false
	}
	return true
}

func (cm *ConsensusManager) onCommit(vote types.Vote) {
	if err := cm.verifyVote(vote); err != nil {
		return
	}
	cm.currentState.applyVote(vote)
	if cm.canEnterCommitted() {
		cm.enterCommitted()
	}
}

// check whether the validator received +2/3 commit
func (cm *ConsensusManager) canEnterCommitted() bool {
	currentState := cm.currentState
	if currentState.stateType >= Committed {
		log.Printf("current state %s is greater than prepared", currentState.stateType.String())
		return false
	}
	if currentState.commits().Size() < 2*cm.f + 1 {
		return false
	}
	return true
}

func (cm *ConsensusManager) enterCommitted() {
	currentState := cm.currentState
	// lock proposal block
	currentState.lock()
	currentState.setSate(Committed)
	proposal := currentState.proposal
	if proposal != nil {
		//TODO: Commit proposal block
		commits := make([]types.Vote, 0)
		for _, vote := range currentState.commits().Votes() {
			commits = append(commits, vote)
		}
		if err := cm.commitBlock(&proposal.Block, commits); err != nil {
			currentState.unLock()
			cm.sendRoundChange(currentState.round() + 1)
			return
		}
		cm.startNewRound(0)
	}
}

func (cm *ConsensusManager) commitBlock(block *types.Block, commits []types.Vote) error {
	if !block.IsValid() {
		return fmt.Errorf("block is invalid")
	}
	header := block.Header()
	if len(commits) < 2*cm.f + 1 {
		return fmt.Errorf("there are not enough commit votes")
	}
	header.Commits = commits
	return cm.blockStore.AddBlock(block)
}

func (cm *ConsensusManager) onRoundChange(vote types.Vote) {
	if !cm.verifyRoundChange(vote) {
		return
	}
	cm.currentState.applyRoundChange(vote, cm.validatorSet)
	if cm.shouldChangeRound(vote.View.Round) {
		cm.sendRoundChange(vote.View.Round)
		return
	}
	if cm.shouldStartNewRound(vote.View.Round) {
		cm.startNewRound(vote.View.Round)
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
	head := cm.head()
	if head == nil {
		log.Fatal("blockchain must have a head")
	}
	newView := types.View{
		0,
		head.Height() + 1,
	}
	if cm.currentState == nil {
		log.Println("initial round")
		cm.currentState = NewConsensusState(newView, cm.validatorSet)
	} else if head.Height() >= cm.currentState.height() {
		log.Println("catch up latest proposal")
		cm.currentState = NewConsensusState(newView, cm.validatorSet)
	} else if head.Height() == cm.currentState.height() - 1 {
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
	cm.currentState.updateView(newView)
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

func (cm *ConsensusManager) changeView(view types.View) {
	cm.currentState.setSate(RoundChange)
	cm.currentState.updateView(view)
	cm.newRoundChangeTimer()
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
		threshold := cm.f + 1
		maxRound := cm.currentState.getMaxRound(threshold)
		if maxRound != math.MaxUint64 && maxRound > cm.currentState.round() {
			cm.sendRoundChange(maxRound)
			return
		}
	}
	head := cm.head()
	if head != nil && head.Height() >= cm.currentState.height() {
		cm.startNewRound(0)
	} else {
		cm.sendRoundChange(cm.currentState.round() + 1)
	}
}

func (cm *ConsensusManager) sendVote(voteType types.VoteType) {
	voter := cm.validatorSet.Self()
	view := cm.currentState.view
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
	buf, err := encoding.MarshalBinary(vote)
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
	// self-processing
	cm.onVote(&vote)
	// send to others
	payload, err := encoding.MarshalBinary(vote)
	if err != nil {
		log.Println(err)
		return
	}
	message := types.NewMessage(types.VoteMessage, payload)
	cm.broadcaster(message)
}

func (cm *ConsensusManager) sendProposal(proposal types.Proposal) {
	// self-processing
	cm.onProposal(&proposal)
	// send to others
	payload, err := encoding.MarshalBinary(proposal)
	if err != nil {
		log.Println(err)
		return
	}
	message := types.NewMessage(types.ProposalMessage, payload)
	cm.broadcaster(message)
}

func (cm *ConsensusManager) sendRoundChange(round uint64) {
	newView := types.View{
		round,
		cm.currentState.height(),
	}
	cm.changeView(newView)
	cm.sendVote(types.RoundChange)
}

func (cm *ConsensusManager) address() string {
	validatorSet := cm.validatorSet
	return validatorSet.Self().Address
}
