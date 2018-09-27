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
		cms = append(cms, cm)
	}
	return cms
}

// return manager of the proposer and it's index
func (tester *tester) managerOfProposer() (*ConsensusManager, int) {
	managers := tester.managers
	for i, manager := range managers {
		if manager.isProposer() {
			return manager, i
		}
	}
	return nil, -1
}

func (tester *tester) getProposer() (types.Validator, error) {
	manager, _ := tester.managerOfProposer()
	if manager == nil {
		return types.Validator{}, fmt.Errorf("there's not a proposer")
	}
	return manager.validatorSet.Self(), nil
}

func (tester *tester) newProposal(round, height uint64) (*types.Proposal, error) {
	manager, _ := tester.managerOfProposer()
	head := manager.head()
	blockHeightId := types.BlockHeightId{ Height: head.Height() + 1 }
	proposer, err := tester.getProposer()
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
	parsedProposal := message.ToProposal(encoding.UnmarshalBinary)
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
}

func TestEnterPreparedWhenLocking(t *testing.T) {
	tester := newTester()
	err := tester.enterPrepared()
	if err != nil {
		t.Fatal(err)
	}
	// first manager should be locked
	firstManager := tester.managers[0]
	if !firstManager.currentState.isLocked() {
		t.Fatal("first manager should be locked")
	}


}

func (tester *tester) enterPrepared() error {
	proposal, err := tester.newProposal(1, 2)
	if err != nil {
		return err
	}
	managers := tester.managers
	tester.setBroadcaster(tester.broadcastPrepare)
	for _, cm := range managers {
		cm.enterPrePrepared(proposal)
	}
	return nil
}

func (tester *tester) setBroadcaster(broadcastFunc BroadcastFunc) {
	managers := tester.managers
	for _, cm := range managers {
		cm.SetBroadcaster(broadcastFunc)
	}
}

func broadcastNothing(message types.Message) {
}

func (tester *tester) broadcastPrepare(message types.Message) {
	if message.Type == types.VoteMessage {
		vote := message.ToVote(encoding.UnmarshalBinary)
		if vote == nil {
			log.Println("broadcast a nil prepare message")
			return
		}
		if vote.Type != types.Prepare {
			log.Println("it's not a prepare message")
			return
		}
		for _, manager := range tester.managers {
			manager.Receive(message)
		}
	}
}

func (tester *tester) broadcastRoundChange(message types.Message) {
	if message.Type == types.VoteMessage {
		vote := message.ToVote(encoding.UnmarshalBinary)
		if vote == nil {
			log.Println("broadcast a nil round-change message")
			return
		}
		if vote.Type != types.RoundChange {
			log.Println("it's not a round-change message")
			return
		}
		for _, manager := range tester.managers {
			if manager.address() != vote.Address {
				manager.Receive(message)
			}
		}
	}
}