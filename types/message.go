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

func (m Message) ToHandshake(decoder DeserializeFunc) (*Handshake, error) {
	handshake := Handshake{}
	payload := make([]byte, len(m.Payload))
	copy(payload, m.Payload)
	err := decoder(payload, &handshake)
	if err != nil {
		return nil, err
	}
	return &handshake, nil
}

func (m Message) ToProposal(decoder DeserializeFunc) (*Proposal, error) {
	proposal := Proposal{}
	payload := make([]byte, len(m.Payload))
	copy(payload, m.Payload)
	err := decoder(payload, &proposal)
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

func (m Message) ToVote(decoder DeserializeFunc) (*Vote, error) {
	vote := Vote{}
	payload := make([]byte, len(m.Payload))
	copy(payload, m.Payload)
	err := decoder(payload, &vote)
	if err != nil {
		return nil, err
	}
	return &vote, nil
}

func (m Message) ToSyncRequest(decoder DeserializeFunc) *SyncRequest {
	syncRequest := SyncRequest{}
	payload := make([]byte, len(m.Payload))
	copy(payload, m.Payload)
	err := decoder(payload, &syncRequest)
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

func (hs *Handshake) Height() uint64 {
	return hs.LastHeightId.Height
}

func (hs *Handshake) IsValid() bool {
	if hs.ChainId.IsEmpty() {
		log.Println("chain id is empty")
		return false
	}
	if !hs.LastHeightId.IsValid() {
		log.Println("height, id are invalid")
		return false
	}
	if !hs.Signature.IsValid() {
		log.Println("signature is invalid")
		return false
	}
	return true
}

func (hs *Handshake) Verify() bool {
	ts := hs.Timestamp.UnixNano()
	now := time.Now().UTC().UnixNano()
	if now - ts > HandshakeTimeout * int64(time.Second) {
		log.Println("handshake timeout")
		return false
	}
	return hs.Signature.Verify(hs.Address, hs.Digest[:])
}
