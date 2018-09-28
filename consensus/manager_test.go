package consensus

import (
	"testing"
	"bft/crypto"
	"bft/types"
	"time"
	"bft/encoding"
	"fmt"
	"log"
)

type tester struct {
	managers []*ConsensusManager
}

func newTester() *tester {
	t := &tester{}
	managers := consensusManagers()
	t.managers = managers
	return t
}

func newValidator(privateKey *crypto.PrivateKey) types.Validator {
	return types.Validator{
		PublicKey: *privateKey.PublicKey(),
		Address: privateKey.PublicKey().Address(),
	}
}

func consensusManagers() []*ConsensusManager {
	validators := make([]types.Validator, 0)
	privateKeys := make([]*crypto.PrivateKey, 0)
	cms := make([]*ConsensusManager, 0)
	for i := 0; i < 4; i++ {
		privateKey, _ := crypto.NewRandomPrivateKey()
		privateKeys = append(privateKeys, privateKey)
		validator := newValidator(privateKey)
		validators = append(validators, validator)
	}
	for i := 0; i < 4; i++ {
		cm := NewConsensusManager(validators, privateKeys[i].PublicKey().Address())
		cm.SetSigner(privateKeys[i].Sign)
		view := types.View{
			1,
			2,
		}
		cm.currentState = NewConsensusState(view, cm.validatorSet)
		cm.currentState.setSate(NewRound)
		cm.validatorSet.CalculateProposer(view.Round)
		log.Println(cm.validatorSet.Proposer().Address)
		cms = append(cms, cm)
	}
	return cms
}

// return manager of the proposer and it's index
func (t *tester) managerOfProposer() (*ConsensusManager, int) {
	managers := t.managers
	for i, manager := range managers {
		if manager.isProposer() {
			return manager, i
		}
	}
	return nil, -1
}

func (t *tester) getProposer() (types.Validator, error) {
	manager, _ := t.managerOfProposer()
	if manager == nil {
		return types.Validator{}, fmt.Errorf("there's not a proposer")
	}
	return manager.validatorSet.Self(), nil
}

func (t *tester) newProposal(round, height uint64) (*types.Proposal, error) {
	manager, _ := t.managerOfProposer()
	head := manager.head()
	blockHeightId := types.BlockHeightId{ Height: head.Height() + 1 }
	proposer, err := t.getProposer()
	if err != nil {
		return nil, err
	}
	blockHeader := types.BlockHeader{
		HeightId: blockHeightId,
		PreviousId: head.Id(),
		Proposer: proposer,
		Timestamp: time.Now().UTC(),
	}
	blockHeader.HeightId.Id = blockHeader.CalculateId(encoding.MarshalBinary)
	signedBlockHeader := types.SignedBlockHeader{Header: blockHeader}
	blockId := blockHeader.Id()
	signedBlockHeader.Signature, err = manager.signer(blockId[:])
	if err != nil {
		return nil, err
	}
	signedBlock := types.Block{ SignedHeader: signedBlockHeader }
	proposal := &types.Proposal{
		View: types.View{
			round,
			height,
		},
		Block: signedBlock,
	}
	return proposal, nil
}

func TestParseAndVerifyProposal(t *testing.T) {
	tester := newTester()
	proposal, err := tester.newProposal(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := encoding.MarshalBinary(proposal)
	if err != nil {
		t.Fatal(err)
	}
	message := types.NewMessage(types.ProposalMessage, payload)
	parsedProposal, _ := message.ToProposal(encoding.UnmarshalBinary)
	if parsedProposal == nil {
		t.Fatal("can not parse proposal")
	}
	proposer := proposal.Proposer()
	managers := tester.managers
	for _, cm := range managers {
		if !cm.validatorSet.IsProposer(proposer) {
			t.Fatal("Don't accept a proposal from unknown proposer")
		}
		if err := cm.verifyProposal(proposal); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEnterPrePrepared(t *testing.T) {
	tester := newTester()
	proposal, err := tester.newProposal(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	managers := tester.managers
	for _, cm := range managers {
		cm.SetBroadcaster(broadcastNothing)
		cm.enterPrePrepared(proposal)
		currentState := cm.currentState
		currentProposal := currentState.proposal
		if currentState.stateType != PrePrepared {
			t.Fatalf("expected preprepared, got %s", currentState.stateType.String())
		}
		if currentProposal == nil {
			t.Fatal("current's state should have a proposal")
		}
		if !currentProposal.BlockHeightId().Equals(proposal.BlockHeightId()) {
			t.Fatalf("expected height-id %s, got %s", proposal.BlockHeightId().String(), currentProposal.BlockHeightId().String())
		}
	}
}

func TestEnterPrepared(t *testing.T) {
	tester := newTester()
	err := tester.enterPrepared()
	if err != nil {
		t.Fatal(err)
	}
	firstManager := tester.managers[0]
	state := firstManager.currentState.stateType
	if state != Prepared {
		t.Fatalf("expected prepared, got %s", state.String())
	}
	//for _, cm := range tester.managers {
	//	prepares := cm.currentState.prepares()
	//	t.Log(prepares.Size())
	//}
}

func TestSendLockedProposal(t *testing.T) {
	tester := newTester()
	err := tester.enterPrepared()
	if err != nil {
		t.Fatal(err)
	}
	// first manager should be locked
	firstManager := tester.managers[0]
	lastManager := tester.managers[len(tester.managers) - 1]
	if !firstManager.currentState.isLocked() {
		t.Fatal("first manager should be locked")
	}
	tester.setBroadcaster(tester.broadcast)
	for _, cm := range tester.managers {
		cm.sendRoundChange(cm.currentState.round() + 1)
	}
	for _, cm := range tester.managers {
		t.Log(cm.address())
		t.Log(cm.currentState.stateType.String())
		t.Log(cm.currentState.view)
	}
	state := firstManager.currentState.stateType
	if state != Prepared && !lastManager.isProposer() {
		t.Fatalf("%s expected prepared, got %s", firstManager.address(), state.String())
	}
}

func (t *tester) enterPrepared() error {
	proposal, err := t.newProposal(1, 2)
	if err != nil {
		return err
	}
	managers := t.managers
	t.setBroadcaster(t.broadcastPrepare)
	for _, cm := range managers {
		cm.enterPrePrepared(proposal)
	}
	return nil
}

func (t *tester) setBroadcaster(broadcastFunc BroadcastFunc) {
	managers := t.managers
	for _, cm := range managers {
		cm.SetBroadcaster(broadcastFunc)
	}
}

func broadcastNothing(message types.Message) {}

func (t *tester) broadcastPrepare(message types.Message) {
	if message.Type == types.VoteMessage {
		vote, _ := message.ToVote(encoding.UnmarshalBinary)
		if vote == nil {
			log.Println("broadcast a nil prepare message")
			return
		}
		if vote.Type != types.Prepare {
			return
		}
		for _, manager := range t.managers {
			manager.Receive(message)
		}
	}
}

func (t *tester) broadcast(message types.Message) {
	if message.Type == types.VoteMessage {
		vote, _ := message.ToVote(encoding.UnmarshalBinary)
		if vote == nil {
			log.Println("broadcast a nil round-change message")
			return
		}
		if vote.Type != types.RoundChange {
			return
		}
	} else if message.Type == types.ProposalMessage{
		proposal, _ := message.ToProposal(encoding.UnmarshalBinary)
		if proposal == nil {
			log.Println("broadcast a nil proposal")
			return
		}
	}
	for _, manager := range t.managers {
		manager.Receive(message)
	}
}