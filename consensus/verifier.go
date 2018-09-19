package consensus

import (
	"log"
	"bft/types"
)

func (cm *ConsensusManager) verifyProposal(proposal *types.Proposal) bool {
	if proposal == nil {
		return false
	}
	// Does blockchain have a head
	if cm.head == nil {
		log.Println("blockchain hasn't a head")
		return false
	}
	blockHeader := proposal.Block.Header()
	// Are block's height and hash valid
	if !blockHeader.HeightId.IsValid() {
		log.Println("block's height or hash is invalid")
		return false
	}
	// Does block proposal's previous id equal head's id
	if !blockHeader.PreviousId.Equals(cm.head.Id()) || blockHeader.Height() != cm.head.Height() {
		log.Println("unlinkable block")
		return false
	}
	publicKey := proposal.Proposer().PublicKey
	// Is block signed by proposer
	blockId := proposal.BlockId()
	signature := proposal.Block.Signature()
	if !signature.Verify(publicKey, blockId[:]) {
		log.Println("block's signature is wrong")
		return false
	}
	return true
}

func (cm *ConsensusManager) verifyPrepare(vote types.Vote) bool {
	// check prepare's round and height
	if vote.View.Compare(cm.currentState.view) != 0 {
		log.Println("prepare's round and height are invalid")
		return false
	}
	// check current state
	if cm.currentState.stateType == NewRound {
		return false
	}
	//// is prepare from a valid validator?
	//if index, _ := cm.validatorSet.GetByAddress(vote.Address); index == -1 {
	//	log.Println("Don't accept prepare message from a unknown validator")
	//	return false
	//}
	// verify block id
	if !vote.BlockId.Equals(cm.currentState.proposal.BlockId()) {
		return false
	}
	return true
}

func (cm *ConsensusManager) verifyCommit(vote types.Vote) bool {
	// check commit's round and height
	if vote.View.Compare(cm.currentState.view) != 0 {
		log.Println("commit's round and height are invalid")
		return false
	}
	// check current state
	if cm.currentState.stateType == NewRound {
		return false
	}
	//// is commit from a valid validator?
	//if index, _ := cm.validatorSet.GetByAddress(vote.Address); index == -1 {
	//	log.Println("Don't accept commit message from a unknown validator")
	//	return false
	//}
	// verify block id
	if !vote.BlockId.Equals(cm.currentState.proposal.BlockId()) {
		return false
	}
	return true
}

func (cm *ConsensusManager) verifyRoundChange(vote types.Vote) bool {
	if vote.View.Height > cm.currentState.height() {
		return false
	} else if vote.View.Compare(cm.currentState.view) < 0 {
		return false
	}
	return true
}
