package types

import (
	"time"
	"bft/crypto"
	"bytes"
	"crypto/sha256"
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
	Commits []Vote
}

func (blockHeader BlockHeader) Height() uint64 {
	return blockHeader.HeightId.Height
}

func (blockHeader BlockHeader) Id() Hash {
	return blockHeader.HeightId.Id
}

func (blockHeader BlockHeader) CalculateId(encoder SerializeFunc) Hash {
	buf, _ := encoder(blockHeader)
	return sha256.Sum256(buf)
}

type SignedBlockHeader struct {
	Header BlockHeader
	Signature crypto.Signature
}

type Block struct {
	SignedHeader SignedBlockHeader
}

func NewGenesisBlock(genesis Genesis, encoder SerializeFunc) *Block {
	genesisHeader := BlockHeader{}
	genesisHeader.HeightId.Height = 1
	genesisHeader.Proposer = genesis.Proposer
	genesisHeader.Timestamp = genesis.Timestamp
	id := genesisHeader.CalculateId(encoder)
	genesisHeader.HeightId.Id = id
	signedHeader := SignedBlockHeader{
		Header: genesisHeader,
	}
	return &Block{
		signedHeader,
	}
}

func (sb *Block) Header() BlockHeader{
	return sb.SignedHeader.Header
}

func (sb *Block) Height() uint64 {
	return sb.Header().Height()
}

func (sb *Block) Id() Hash {
	return sb.Header().Id()
}

func (sb *Block) Signature() crypto.Signature {
	return sb.SignedHeader.Signature
}
