package types

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

type SyncRequest struct {
	StartHeight uint64
	EndHeight uint64
}
