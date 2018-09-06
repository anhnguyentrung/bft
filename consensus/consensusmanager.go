package consensus

import (
	"bft/types"
	"sync"
	"log"
	"crypto/sha256"
	"bft/crypto"
)

type ConsensusManager struct {
	mutex sync.Mutex
	state *ConsensusState
	validatorManager *ValidatorManager
	head *types.BlockHeader
	enDecoder types.EnDecoder
	signer types.Signfunc
}

func NewConsensusManager(enDecoder types.EnDecoder, signer types.Signfunc, validators types.Validators, address string) *ConsensusManager {
	cm := &ConsensusManager{}
	cm.state = NewConsensusState()
	cm.validatorManager = NewValidatorManager(validators, address)
	cm.enDecoder = enDecoder
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

func (cm *ConsensusManager) Receive(message types.Message) {
	messageType := message.Header.Type
	switch messageType {
	case types.VoteMessage:
		vote := types.Vote{}
		err := cm.enDecoder.Decoder(message.Payload, &vote)
		if err != nil {
			log.Fatal(err)
		}
		cm.onVote(vote)
	case types.ProposalMessage:
		proposal := types.Proposal{}
		err := cm.enDecoder.Decoder(message.Payload, &proposal)
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
	if result := proposal.View.Compare(cm.state.view); result != 0 {
		// if proposal is an existing block, broadcast commit
		if result < 0 {
			//TODO: handle the existing block
		}
	}
	proposer := proposal.ProposalBlock.Header().Proposer
	// Is proposal from valid proposer
	if !cm.validatorManager.isProposer(proposer) {
		log.Println("Don't accept a proposal from invalid proposer")
		return
	}
	// check proposal
	if !cm.verifyProposal(proposal) {
		return
	}
	//TODO: handle the future block
	if cm.state.stateType == NewRound {
		cm.state.setProposal(proposal)
		cm.state.setSate(PrePrepared)
	}
}

func (cm *ConsensusManager) verifyProposal(proposal types.Proposal) bool {
	// Does blockchain have a head
	if cm.head == nil {
		log.Println("blockchain hasn't a head")
		return false
	}
	blockHeader := proposal.ProposalBlock.Header()
	// Are block's height and hash valid
	if !blockHeader.HeightId.IsValid() {
		log.Println("block's height or hash is invalid")
		return false
	}
	// Does block proposal's previous id equal head's id
	if !blockHeader.PreviousId.Equals(cm.head.Id()) {
		log.Println("unlinkable block")
		return false
	}
	publicKey := proposal.ProposalBlock.Header().Proposer.PublicKey
	// Is block signed by proposer
	blockHeaderBuf, err := cm.enDecoder.Encoder(blockHeader)
	if err != nil {
		log.Println(err)
		return false
	}
	blockHash := sha256.Sum256(blockHeaderBuf)
	signature := proposal.ProposalBlock.Signature()
	if !signature.Verify(publicKey, blockHash[:]) {
		log.Println("block's signature is wrong")
		return false
	}
	return true
}

func (cm *ConsensusManager) onPrepare(vote types.Vote) {

}

func (cm *ConsensusManager) onCommit(vote types.Vote) {

}

func (cm *ConsensusManager) onRoundChange(vote types.Vote) {

}

func (cm *ConsensusManager) broadCast(voteType types.VoteType) {
	voter := cm.validatorManager.self
	view := cm.state.proposal.View
	blockId := cm.state.proposal.ProposalBlock.Header().Id()
	vote := types.Vote {
		voter,
		voteType,
		view,
		blockId,
		crypto.Signature{},
	}
	buf, err := cm.enDecoder.Encoder(vote)
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
	//TODO: broadcast vote
}
