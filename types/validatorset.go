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
	index, self := vs.GetByAddress(address)
	if index == -1 {
		log.Fatal("self address is invalid")
	}
	vs.self = self
	return vs
}

func (vs *ValidatorSet) Size() int {
	vs.rwMutex.RLock()
	defer vs.rwMutex.RUnlock()
	return len(vs.validators)
}

func (vs *ValidatorSet) GetByIndex(index uint64) *Validator {
	vs.rwMutex.RLock()
	defer vs.rwMutex.RUnlock()
	if index >= uint64(vs.Size()) {
		log.Fatal("index is out of bounds")
	}
	return &vs.validators[index]
}

func (vs *ValidatorSet) GetByAddress(address string) (int, Validator) {
	for index, v := range vs.GetValidators() {
		if v.Address == address {
			return index, v
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
	return v.Equals(*vs.proposer)
}

func (vs *ValidatorSet) Self() Validator {
	vs.rwMutex.RLock()
	defer vs.rwMutex.RUnlock()
	return vs.self
}

func (vs *ValidatorSet) CalculateProposer(round uint64) {
	offset := round
	if vs.proposer != nil {
		pIndex, _ := vs.GetByAddress(vs.proposer.Address)
		if pIndex == -1 {
			log.Fatal("current proposer is invalid")
		}
		offset = uint64(pIndex) + round + 1
	}
	newPIndex := offset % uint64(vs.Size())
	vs.proposer = vs.GetByIndex(newPIndex)
}