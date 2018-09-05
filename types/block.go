package types

import (
	"time"
	"bft/crypto"
	"strconv"
	"encoding/hex"
)

type BlockHeightId struct {
	Height uint64
	Id Hash
}

func (blockHeightId *BlockHeightId) String() string {
	h := strconv.FormatUint(blockHeightId.Height, 10)
	id := hex.EncodeToString(blockHeightId.Id[:])
	return h + id
}

type BlockHeader struct {
	HeightId BlockHeightId
	PreviousId Hash
	Proposer Validator
	Timestamp time.Time
}

type Block struct {
	Header BlockHeader
}

type SignedBlock struct {
	Block Block
	Signature crypto.Signature
}

func (sb *SignedBlock) Header() BlockHeader{
	return sb.Block.Header
}
