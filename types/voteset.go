package types

import (
	"sync"
	"fmt"
)

type VoteSet struct {
	mutex sync.RWMutex
	view View
	voteType VoteType
	validatorSet *ValidatorSet
	votes map[string]Vote
}

func NewVoteSet(view View, voteType VoteType, validatorSet *ValidatorSet) *VoteSet {
	return &VoteSet{
		view:view,
		validatorSet:validatorSet,
		voteType:voteType,
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
		return fmt.Errorf("voter %s sent duplicate vote", vote.Address)
	}
	voteSet.votes[vote.Address] = vote
	return nil
}

func (voteSet *VoteSet) verifyVote(vote Vote) error {
	// check vote type
	if voteSet.voteType != vote.Type {
		return fmt.Errorf("VoteSet's type: %s, vote's type: %s", voteSet.voteType.String(), vote.Type.String())
	}
	// check view
	if voteSet.view.Compare(vote.View) != 0 {
		return fmt.Errorf("VoteSet's view: %d %d, vote's view: %d %d", voteSet.view.Round, voteSet.view.Height, vote.View.Round, voteSet.view.Height)
	}
	// check whether voter is a valid validator
	index, voter := voteSet.validatorSet.GetByAddress(vote.Address)
	if index == -1 {
		return fmt.Errorf("invalid voter address: %s", vote.Address)
	}
	// check signature
	if !vote.Signature.Verify(voter.Address, vote.Hash[:]) {
		return fmt.Errorf("invalid signature from voter %s", vote.Address)
	}
	return nil
}

func (voteSet *VoteSet) ChangeView(view View) {
	voteSet.view = view
}

func (voteSet *VoteSet) Votes() map[string]Vote {
	voteSet.mutex.RLock()
	defer voteSet.mutex.RUnlock()
	return voteSet.votes
}

func (voteSet *VoteSet) Size() int {
	voteSet.mutex.RLock()
	defer voteSet.mutex.RUnlock()
	return len(voteSet.votes)
}