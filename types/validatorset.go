package types

import (
	"sync"
	"sort"
	"log"
)

type ValidatorSet struct {
	rwMutex sync.RWMutex
	validators Validators
	self Validator
	proposer *Validator
}

func NewValidatorSet(validators Validators, address string) *ValidatorSet {
	vs := &ValidatorSet{}
	vs.validators = validators
	sort.Sort(vs.validators)
	if vs.Size() > 0 {
		vs.proposer = vs.GetByIndex(0)
	}
	i, self := vs.GetByAddress(address)
	if i == -1 {
		log.Fatal("self address is invalid")
	}
	vs.self = self
	return vs
}

func (vs *ValidatorSet) Size() int {
	return len(vs.GetValidators())
}

func (vs *ValidatorSet) GetByIndex(i uint64) *Validator {
	vs.rwMutex.RLock()
	defer vs.rwMutex.RUnlock()
	if i >= uint64(vs.Size()) {
		log.Fatal("index is out of bounds")
	}
	return &vs.validators[i]
}

func (vs *ValidatorSet) GetByAddress(address string) (int, Validator) {
	for i, v := range vs.GetValidators() {
		if v.Address == address {
			return i, v
		}
	}
	return -1, Validator{}
}

func (vs *ValidatorSet) GetValidators() Validators {
	vs.rwMutex.RLock()
	defer vs.rwMutex.RUnlock()
	return vs.validators
}

func (vs *ValidatorSet) IsProposer(validator Validator) bool {
	i, v := vs.GetByAddress(validator.Address)
	if i == -1 {
		log.Printf("wrong proposer. expected %s, got %s\n", vs.proposer.Address, validator.Address)
		return false
	}
	return v.Equals(*vs.Proposer())
}

func (vs *ValidatorSet) Proposer() *Validator {
	vs.rwMutex.RLock()
	defer vs.rwMutex.RUnlock()
	return vs.proposer
}

func (vs *ValidatorSet) setProposer(proposer *Validator) {
	vs.rwMutex.Lock()
	defer vs.rwMutex.Unlock()
	vs.proposer = proposer
}

func (vs *ValidatorSet) Self() Validator {
	vs.rwMutex.RLock()
	defer vs.rwMutex.RUnlock()
	return vs.self
}

func (vs *ValidatorSet) CalculateProposer(round uint64) {
	offset := round
	if vs.Proposer() != nil {
		i, _ := vs.GetByAddress(vs.Proposer().Address)
		if i == -1 {
			log.Fatal("current proposer is invalid")
		}
		offset = uint64(i) + round + 1
	}
	i := offset % uint64(vs.Size())
	vs.setProposer(vs.GetByIndex(i))
}