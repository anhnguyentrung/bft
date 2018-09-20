package types

import (
	"time"
	"bft/crypto"
	"log"
	"bft/encoding"
)

type MessageType uint8
const (
	HandshakeMessage MessageType = iota
	ProposalMessage
	VoteMessage
	SyncRequestMessage
)

type MessageHeader struct {
	Type 	MessageType // 1 byte
	Length  uint32  	// 4 bytes
}
type Message struct {
	Header MessageHeader
	Payload []byte
}

func NewMessage(messageType MessageType, payload []byte) Message {
	return Message{
		Header: MessageHeader{
			Type: messageType,
			Length: uint32(len(payload)),
		},
		Payload: payload,
	}
}

func (message Message) ToHandshake() *Handshake {
	handshake := Handshake{}
	err := encoding.UnmarshalBinary(message.Payload, &handshake)
	if err != nil {
		return nil
	}
	return &handshake
}

func (message Message) ToProposal() *Proposal {
	proposal := Proposal{}
	err := encoding.UnmarshalBinary(message.Payload, &proposal)
	if err != nil {
		return nil
	}
	return &proposal
}

func (message Message) ToVote() *Vote {
	vote := Vote{}
	err := encoding.UnmarshalBinary(message.Payload, &vote)
	if err != nil {
		return nil
	}
	return &vote
}

func (message Message) ToSyncRequest() *SyncRequest {
	syncRequest := SyncRequest{}
	err := encoding.UnmarshalBinary(message.Payload, &syncRequest)
	if err != nil {
		return nil
	}
	return &syncRequest
}

type SyncRequest struct {
	StartHeight uint64
	EndHeight uint64
}

type Handshake struct {
	NetworkVersion string
	ChainId Hash
	Address string
	LastHeightId BlockHeightId
	Timestamp time.Time
	Signature crypto.Signature
}

func NewHandshake(chainId Hash, address string, lastHeightId BlockHeightId, signer crypto.SignFunc) *Handshake {
	signature, err := signer(chainId[:])
	if err != nil {
		log.Println(err)
		return nil
	}
	handshake := &Handshake{
		NetworkVersion,
		chainId,
		address,
		lastHeightId,
		time.Now().UTC(),
		signature,
	}
	return handshake
}

func (handshake *Handshake) IsValid() bool {
	return !handshake.ChainId.IsEmpty() && handshake.LastHeightId.IsValid() && handshake.Signature.IsValid()
}
