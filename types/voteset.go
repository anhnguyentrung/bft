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

func (vs *VoteSet) AddVote(vote Vote, verify bool) error {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()
	if verify {
		err := vs.verifyVote(vote)
		if err != nil {
			return err
		}
	}
	// check duplicate vote
	if _, ok := vs.votes[vote.Address]; ok {
		return fmt.Errorf("voter %s sent duplicate vote", vote.Address)
	}
	vs.votes[vote.Address] = vote
	return nil
}

func (vs *VoteSet) verifyVote(vote Vote) error {
	// check vote type
	if vs.voteType != vote.Type {
		return fmt.Errorf("VoteSet's type: %s, vote's type: %s", vs.voteType.String(), vote.Type.String())
	}
	// check view
	if vs.view.Compare(vote.View) != 0 {
		return fmt.Errorf("VoteSet's view: %d %d, vote's view: %d %d", vs.view.Round, vs.view.Height, vote.View.Round, vs.view.Height)
	}
	// check whether voter is a valid validator
	index, voter := vs.validatorSet.GetByAddress(vote.Address)
	if index == -1 {
		return fmt.Errorf("invalid voter address: %s", vote.Address)
	}
	// check signature
	if !vote.Signature.Verify(voter.Address, vote.Hash[:]) {
		return fmt.Errorf("invalid signature from voter %s", vote.Address)
	}
	return nil
}

func (vs *VoteSet) ChangeView(view View) {
	vs.view = view
}

func (vs *VoteSet) Votes() map[string]Vote {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()
	return vs.votes
}

func (vs *VoteSet) Size() int {
	vs.mutex.RLock()
	defer vs.mutex.RUnlock()
	return len(vs.votes)
}