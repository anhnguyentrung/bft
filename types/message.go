package types

import (
	"time"
	"bft/crypto"
	"log"
	"crypto/sha256"
)

type MessageType uint8
const (
	HandshakeMessage MessageType = iota
	ProposalMessage
	VoteMessage
	SyncRequestMessage
)

type Message struct {
	Type 	MessageType
	Payload []byte
}

func NewMessage(messageType MessageType, payload []byte) Message {
	return Message{
		Type: messageType,
		Payload: payload,
	}
}

func (message Message) ToHandshake(decoder DeserializeFunc) *Handshake {
	handshake := Handshake{}
	err := decoder(message.Payload, &handshake)
	if err != nil {
		log.Println(err)
		return nil
	}
	return &handshake
}

func (message Message) ToProposal(decoder DeserializeFunc) *Proposal {
	proposal := Proposal{}
	err := decoder(message.Payload, &proposal)
	if err != nil {
		return nil
	}
	return &proposal
}

func (message Message) ToVote(decoder DeserializeFunc) *Vote {
	vote := Vote{}
	err := decoder(message.Payload, &vote)
	if err != nil {
		return nil
	}
	return &vote
}

func (message Message) ToSyncRequest(decoder DeserializeFunc) *SyncRequest {
	syncRequest := SyncRequest{}
	err := decoder(message.Payload, &syncRequest)
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
	Digest Hash
	Signature crypto.Signature
}

func NewHandshake(chainId Hash, address string, lastHeightId BlockHeightId, signer crypto.SignFunc, encoder SerializeFunc) *Handshake {
	handshake := Handshake{
		NetworkVersion: NetworkVersion,
		ChainId: chainId,
		Address: address,
		LastHeightId: lastHeightId,
		Timestamp: time.Now().UTC(),
	}
	buf, err := encoder(handshake)
	if err != nil {
		log.Println(err)
		return nil
	}
	handshake.Digest = sha256.Sum256(buf)
	signature, err := signer(handshake.Digest[:])
	if err != nil {
		log.Println(err)
		return nil
	}
	handshake.Signature = signature
	return &handshake
}

func (handshake *Handshake) Height() uint64 {
	return handshake.LastHeightId.Height
}

func (handshake *Handshake) IsValid() bool {
	if handshake.ChainId.IsEmpty() {
		log.Println("chain id is empty")
		return false
	}
	if !handshake.LastHeightId.IsValid() {
		log.Println("height, id are invalid")
		return false
	}
	if !handshake.Signature.IsValid() {
		log.Println("signature is invalid")
		return false
	}
	return true
}

func (handshake *Handshake) Verify() bool {
	ts := handshake.Timestamp.UnixNano()
	now := time.Now().UTC().UnixNano()
	if now - ts > HandshakeTimeout * int64(time.Second) {
		log.Println("handshake timeout")
		return false
	}
	return handshake.Signature.Verify(handshake.Address, handshake.Digest[:])
}
