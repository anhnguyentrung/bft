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
		vote, err := message.ToVote(encoding.UnmarshalBinary)
		if err != nil {
			log.Println(err)
			return
		}
		cm.onVote(vote)
	case types.ProposalMessage:
		proposal, err := message.ToProposal(encoding.UnmarshalBinary)
		if err != nil {
			log.Println(err)
			return
		}
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
	sender := proposal.Sender
	// Is proposal from valid proposer
	if !cm.validatorSet.IsProposer(sender) {
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
	cs := cm.currentState
	if cs.stateType == NewRound {
		if cs.isLocked() {
			if cs.proposal.BlockHeightId().Equals(cs.lockedHeightId) {
				cs.setSate(Prepared)
				cm.sendVote(types.Commit)
			} else {
				// should go to next round
				cm.sendRoundChange(cm.currentState.round() + 1)
			}
		} else {
			cs.setProposal(proposal)
			cs.setSate(PrePrepared)
			cm.sendVote(types.Prepare)
		}
	}
}

func (cm *ConsensusManager) onPrepare(vote types.Vote) {
	cs := cm.currentState
	if err := cm.verifyVote(vote); err != nil {
		return
	}
	cs.applyVote(vote)
	proposalHeightId := cs.proposalHeightId()
	// if the validator have a locked block, she should broadcast COMMIT on the locked block and enter prepared
	// or the validator received +2/3 prepare
	if (cs.isLocked() && proposalHeightId.Equals(cs.lockedHeightId)) || cm.canEnterPrepared() {
		cm.enterPrepared()
	}
}

func (cm *ConsensusManager) enterPrepared() {
	cs := cm.currentState
	// lock proposal block
	cs.lock()
	cs.setSate(Prepared)
	cm.sendVote(types.Commit)
}

// check whether the validator received +2/3 prepare
func (cm *ConsensusManager) canEnterPrepared() bool {
	cs := cm.currentState
	if cs.stateType >= Prepared {
		log.Printf("current state %s is greater than prepared", cs.stateType.String())
		return false
	}
	if cs.prepares().Size() < 2*cm.f + 1 {
		return false
	}
	return true
}

func (cm *ConsensusManager) onCommit(vote types.Vote) {
	cs := cm.currentState
	if err := cm.verifyVote(vote); err != nil {
		return
	}
	cs.applyVote(vote)
	if cm.canEnterCommitted() {
		cm.enterCommitted()
	}
}

// check whether the validator received +2/3 commit
func (cm *ConsensusManager) canEnterCommitted() bool {
	cs := cm.currentState
	if cs.stateType >= Committed {
		log.Printf("current state %s is greater than prepared", cs.stateType.String())
		return false
	}
	if cs.commits().Size() < 2*cm.f + 1 {
		return false
	}
	return true
}

func (cm *ConsensusManager) enterCommitted() {
	log.Println("enter committed")
	cs := cm.currentState
	// lock proposal block
	cs.lock()
	cs.setSate(Committed)
	proposal := cs.proposal
	if proposal != nil {
		//TODO: Commit proposal block
		commits := make([]types.Vote, 0)
		for _, vote := range cs.commits().Votes() {
			commits = append(commits, vote)
		}
		if err := cm.commitBlock(&proposal.Block, commits); err != nil {
			cs.unLock()
			cm.sendRoundChange(cs.round() + 1)
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
	cs := cm.currentState
	voteCount := cs.roundChanges[round].Size()
	if cs.stateType == RoundChange && voteCount == cm.f + 1 {
		if cs.round() < round {
			return true
		}
	}
	return false
}

func (cm *ConsensusManager) shouldStartNewRound(round uint64) bool {
	cs := cm.currentState
	stateType := cs.stateType
	currentRound := cs.round()
	voteCount := cs.roundChanges[round].Size()
	if voteCount == 2*cm.f + 1 && (stateType == RoundChange || currentRound < round) {
		return true
	}
	return false
}

func (cm *ConsensusManager) isProposer() bool {
	vs := cm.validatorSet
	if vs == nil || vs.Size() == 0 {
		log.Println("validator set should not be nil or empty")
		return false
	}
	self := vs.Self()
	return vs.IsProposer(self)
}

func (cm *ConsensusManager) startNewRound(round uint64) {
	cs := cm.currentState
	vs := cm.validatorSet
	head := cm.head()
	if head == nil {
		log.Fatal("blockchain must have a head")
	}
	newView := types.View{
		0,
		head.Height() + 1,
	}
	if cs == nil {
		log.Println("initial round")
		cs = NewConsensusState(newView, cm.validatorSet)
	} else if head.Height() >= cs.height() {
		log.Println("catch up latest proposal")
		log.Println(head.Height())
		cs = NewConsensusState(newView, cm.validatorSet)
	} else if head.Height() == cs.height() - 1 {
		if round == 0 {
			return
		}
		if round < cs.round() {
			log.Println("new round should be greater than current round")
			return
		}
		newView.Round = round
	} else {
		log.Println("new height should be greater than current height")
	}
	// delete all old votes
	for k, _ := range cs.roundChanges {
		delete(cs.roundChanges, k)
	}
	cs.updateView(newView)
	vs.CalculateProposer(round)
	cs.setSate(NewRound)
	if newView.Round != 0 && cm.isProposer() {
		if cs.isLocked() {
			if cs.proposal != nil {
				cm.sendProposal(*cs.proposal)
			}
		} else {
			//if cs.pendingProposal != nil {
			//	cm.sendProposal(*cs.pendingProposal)
			//}
		}
	}
	cm.newRoundChangeTimer()
}

func (cm *ConsensusManager) changeView(v types.View) {
	cs := cm.currentState
	cs.setSate(RoundChange)
	cs.updateView(v)
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
	cs := cm.currentState
	if cs.stateType != RoundChange {
		threshold := cm.f + 1
		maxRound := cs.getMaxRound(threshold)
		if maxRound != math.MaxUint64 && maxRound > cs.round() {
			cm.sendRoundChange(maxRound)
			return
		}
	}
	head := cm.head()
	if head != nil && head.Height() >= cs.height() {
		cm.startNewRound(0)
	} else {
		cm.sendRoundChange(cs.round() + 1)
	}
}

func (cm *ConsensusManager) sendVote(voteType types.VoteType) {
	cs := cm.currentState
	vs := cm.validatorSet
	voter := vs.Self()
	view := cs.view
	blockId := types.Hash{}
	if voteType != types.RoundChange {
		blockId = cs.proposal.BlockId()
	}
	vote := types.Vote {
		types.Hash{},
		voter.Address,
		voteType,
		view,
		blockId,
		crypto.Signature{},
	}
	b, err := encoding.MarshalBinary(vote)
	if err != nil {
		log.Println(err)
		return
	}
	hash := sha256.Sum256(b)
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
	cs := cm.currentState
	vs := cm.validatorSet
	proposal.View = cs.view
	proposal.Sender = vs.Self()
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
