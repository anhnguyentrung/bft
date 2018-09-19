package types

import (
	"time"
	"bft/crypto"
	"bytes"
)

type BlockHeightId struct {
	Height uint64
	Id Hash
}

func (blockHeightId BlockHeightId) IsValid() bool {
	if blockHeightId.Height == 0 {
		return false
	}
	emptyHash := Hash{}
	return !blockHeightId.Id.Equals(emptyHash)
}

func (blockHeightId BlockHeightId) Equals(target BlockHeightId) bool {
	return blockHeightId.Height == target.Height && bytes.Equal(blockHeightId.Id[:], target.Id[:])
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
	return blockHeader.HeightId.Id
}

type SignedBlockHeader struct {
	Header BlockHeader
	Signature crypto.Signature
}

type Block struct {
	SignedHeader SignedBlockHeader
	Commits []Vote
}

func (sb *Block) Header() BlockHeader{
	return sb.SignedHeader.Header
}

func (sb *Block) Signature() crypto.Signature {
	return sb.SignedHeader.Signature
}
