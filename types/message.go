package types

import (
	"time"
	"bft/crypto"
	"log"
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

func NewHanshake(chainId Hash, address string, lastHeightId BlockHeightId, signer crypto.SignFunc) *Handshake {
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
