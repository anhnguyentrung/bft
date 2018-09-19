package consensus

import (
	"testing"
	"bft/crypto"
	"bft/types"
	"time"
	"crypto/sha256"
	"bft/encoding"
)

var manager = consensusManager()

func newValidator(privateKey *crypto.PrivateKey) types.Validator {
	return types.Validator{
		PublicKey: *privateKey.PublicKey(),
		Address: privateKey.PublicKey().Address(),
	}
}

func consensusManager() *ConsensusManager {
	privateKey1, _ := crypto.NewRandomPrivateKey()
	validator1 := newValidator(privateKey1)
	privateKey2, _ := crypto.NewRandomPrivateKey()
	validator2 := newValidator(privateKey2)
	privateKey3, _ := crypto.NewRandomPrivateKey()
	validator3 := newValidator(privateKey3)
	privateKey4, _ := crypto.NewRandomPrivateKey()
	validator4 := newValidator(privateKey4)
	validators := []types.Validator{validator1, validator2, validator3, validator4}
	enDecoder := types.EnDecoder{
		encoding.MarshalBinary,
		encoding.UnmarshalBinary,
	}
	cm := NewConsensusManager(validators, privateKey1.PublicKey().Address())
	cm.SetEnDecoder(enDecoder)
	cm.SetSigner(privateKey1.Sign)
	view := types.View{
		0,
		1,
	}
	cm.currentState = NewConsensusState(view, cm.validatorSet)
	return cm
}

func defaultProposer() types.Validator {
	validators := manager.validatorSet.GetValidators()
	return validators[0]
}

func newProposal() (*types.Proposal, error) {
	blockHeightId := types.BlockHeightId{ Height: 1 }
	blockHeader := types.BlockHeader{
		HeightId: blockHeightId,
		Proposer: defaultProposer(),
		Timestamp: time.Now().UTC(),
	}
	buf, err := manager.enDecoder.Encode(blockHeader)
	if err != nil {
		return nil, err
	}
	blockHeader.HeightId.Id = sha256.Sum256(buf)
	signedBlockHeader := types.SignedBlockHeader{Header: blockHeader}
	blockId := blockHeader.Id()
	signedBlockHeader.Signature, err = manager.signer(blockId[:])
	if err != nil {
		return nil, err
	}
	signedBlock := types.Block{ SignedHeader: signedBlockHeader }
	proposal := &types.Proposal{
		View: types.View{
			0,
			1,
		},
		Block: signedBlock,
	}
	return proposal, nil
}

func TestReceiveProposal(t *testing.T) {
	proposal, err := newProposal()
	if err != nil {
		t.Fatal(err)
	}
	buf, err := manager.enDecoder.Encode(proposal)
	if err != nil {
		t.Fatal(err)
	}
	message := types.Message{
		Header: types.MessageHeader{
			Type: types.ProposalMessage,
			Length: uint32(len(buf)),
		},
		Payload: buf,
	}
	manager.Receive(message)
}
