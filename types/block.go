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

func (blockHeightId BlockHeightId) String() string {
	h := strconv.FormatUint(blockHeightId.Height, 10)
	id := hex.EncodeToString(blockHeightId.Id[:])
	return h + id
}

func (blockHeightId BlockHeightId) IsValid() bool {
	if blockHeightId.Height == 0 {
		return false
	}
	emptyHash := Hash{}
	return !blockHeightId.Id.Equals(emptyHash)
}

type BlockHeader struct {
	HeightId BlockHeightId
	PreviousId Hash
	Proposer Validator
	Timestamp time.Time
}

func (blockHeader BlockHeader) Height() uint64 {
	return blockHeader.Height()
}

func (blockHeader BlockHeader) Id() Hash {
	return blockHeader.Id()
}

type SignedBlockHeader struct {
	Header BlockHeader
	Signature crypto.Signature
}

type SignedBlock struct {
	SignedHeader SignedBlockHeader
}

func (sb *SignedBlock) Header() BlockHeader{
	return sb.SignedHeader.Header
}

func (sb *SignedBlock) Signature() crypto.Signature {
	return sb.SignedHeader.Signature
}
