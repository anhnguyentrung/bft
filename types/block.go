package types

import (
	"time"
	"bft/crypto"
	"bytes"
	"crypto/sha256"
	"log"
	"fmt"
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

func (blockHeightId BlockHeightId) String() string {
	return fmt.Sprintf("%v-%v", blockHeightId.Height, blockHeightId.Id.String())
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

func (block *Block) Header() *BlockHeader{
	return &block.SignedHeader.Header
}

func (block *Block) Height() uint64 {
	return block.Header().Height()
}

func (block *Block) Id() Hash {
	return block.Header().Id()
}

func (block *Block) Signature() crypto.Signature {
	return block.SignedHeader.Signature
}

func (block *Block) IsValid() bool {
	if block == nil {
		log.Println("block should be not nil")
		return false
	}
	if !block.Header().HeightId.IsValid() {
		log.Println("block's height or id is invalid")
		return false
	}
	if !block.Signature().IsValid() {
		log.Println("block's signature is invalid")
		return false
	}
	return true
}
