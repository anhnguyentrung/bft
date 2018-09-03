package types

type MessageType uint8
const (
	Handshake MessageType = iota
	Vote
)

type MessageHeader struct {
	Type 	MessageType // 1 byte
	Length  uint32  	// 4 bytes
}
type Message struct {
	Header MessageHeader
	Payload []byte
}
