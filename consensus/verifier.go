package consensus

import (
	"log"
	"bft/types"
	"fmt"
)

func (cm *ConsensusManager) verifyProposal(proposal *types.Proposal) error {
	if proposal == nil {
		return fmt.Errorf("proposal should be not nil")
	}
	// Does blockchain have a head
	head := cm.head()
	if head == nil {
		return fmt.Errorf("blockchain hasn't a head")
	}
	blockHeader := proposal.Block.Header()
	// Are block's height and hash valid
	if !blockHeader.HeightId.IsValid() {
		return fmt.Errorf("block's height or hash is invalid")
	}
	// Does block proposal's previous id equal head's id
	if !blockHeader.PreviousId.Equals(head.Id()) || blockHeader.Height() != head.Height() + 1 {
		return fmt.Errorf("unlinkable block")
	}
	publicKey := proposal.Proposer().PublicKey
	// Is block signed by proposer
	blockId := proposal.BlockId()
	signature := proposal.Block.Signature()
	if !signature.Verify(publicKey.Address(), blockId[:]) {
		fmt.Errorf("block's signature is wrong")
	}
	return nil
}

func (cm *ConsensusManager) verifyPrepare(vote types.Vote) bool {
	currentState := cm.currentState
	// check prepare's round and height
	if vote.View.Compare(currentState.view) != 0 {
		log.Println("prepare's round and height are invalid")
		return false
	}
	// check current state
	if currentState.stateType == NewRound {
		log.Println("state should not be newround when receiving a prepare message")
		return false
	}
	//// is prepare from a valid validator?
	//if index, _ := cm.validatorSet.GetByAddress(vote.Address); index == -1 {
	//	log.Println("Don't accept prepare message from a unknown validator")
	//	return false
	//}
	// verify block id
	proposal := currentState.proposal
	if !vote.BlockId.Equals(currentState.proposal.BlockId()) {
		log.Printf("vote's block id %s does not match with local state's block id %s\n", vote.BlockId.String(), proposal.BlockId().String())
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
