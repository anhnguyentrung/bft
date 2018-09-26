package consensus

import (
	"testing"
	"bft/crypto"
	"bft/types"
	"time"
	"bft/encoding"
	"fmt"
)

var managers = consensusManagers()

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
		cm.validatorSet.CalculateProposer(view.Round)
		cms = append(cms, cm)
	}
	return cms
}

// return manager of the proposer and it's index
func managerOfProposer() (*ConsensusManager, int) {
	for i, manager := range managers {
		if manager.isProposer() {
			return manager, i
		}
	}
	return nil, -1
}

func getProposer() (types.Validator, error) {
	manager, _ := managerOfProposer()
	if manager == nil {
		return types.Validator{}, fmt.Errorf("there's not a proposer")
	}
	return manager.validatorSet.Self(), nil
}

func newProposal(round, height uint64) (*types.Proposal, error) {
	manager, _ := managerOfProposer()
	head := manager.head()
	blockHeightId := types.BlockHeightId{ Height: head.Height() + 1 }
	proposer, err := getProposer()
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
	proposal, err := newProposal(1, 2)
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
	for _, cm := range managers {
		if !cm.validatorSet.IsProposer(proposer) {
			t.Fatal("Don't accept a proposal from unknown proposer")
		}
		if err := cm.verifyProposal(proposal); err != nil {
			t.Fatal(err)
		}
	}
}

func broadcast(message types.Message) {

}
