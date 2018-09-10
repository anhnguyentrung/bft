package types

import (
	"sync"
	"log"
	"errors"
)

var (
	errVoteTypeNotMatch = errors.New("vote type does not match")
	errViewNotMatch = errors.New("view does not match")
	errInvalidVoter = errors.New("invalid voter")
	errDuplicateVote = errors.New("duplicate vote")
	errInvalidSignature = errors.New("invalid signature")
)

type VoteSet struct {
	mutex sync.RWMutex
	view View
	voteType VoteType
	validatorSet ValidatorSet
	votes map[string]Vote
}

func NewVoteSet(view View, voteType VoteType, validatorSet ValidatorSet) *VoteSet {
	return &VoteSet{
		view:view,
		validatorSet:validatorSet,
		votes:make(map[string]Vote, 0),
	}
}

func (voteSet *VoteSet) AddVote(vote Vote, verify bool) error {
	voteSet.mutex.Lock()
	defer voteSet.mutex.Unlock()
	if verify {
		err := voteSet.verifyVote(vote)
		if err != nil {
			return err
		}
	}
	// check duplicate vote
	if _, ok := voteSet.votes[vote.Address]; ok {
		log.Printf("voter %s sent duplicate vote", vote.Address)
		return errDuplicateVote
	}
	voteSet.votes[vote.Address] = vote
	return nil
}

func (voteSet *VoteSet) verifyVote(vote Vote) error {
	// check vote type
	if voteSet.voteType != vote.Type {
		log.Printf("VoteSet's type: %s, vote's type: %s", voteSet.voteType.String(), vote.Type.String())
		return errVoteTypeNotMatch
	}
	// check view
	if voteSet.view.Compare(vote.View) != 0 {
		log.Printf("VoteSet's view: %d %d, vote's view: %d %d", voteSet.view.Round, voteSet.view.Height, vote.View.Round, voteSet.view.Height)
		return errViewNotMatch
	}
	// check whether voter is a valid validator
	index, voter := voteSet.validatorSet.GetByAddress(vote.Address)
	if index == -1 {
		log.Printf("invalid voter address: %s", vote.Address)
		return errInvalidVoter
	}
	// check signature
	if !vote.Signature.Verify(voter.PublicKey, vote.Hash[:]) {
		log.Printf("invalid signature from voter %s", vote.Address)
		return errInvalidSignature
	}
	return nil
}

func (voteSet *VoteSet) Size() int {
	voteSet.mutex.RLock()
	defer voteSet.mutex.RUnlock()
	return len(voteSet.votes)
}