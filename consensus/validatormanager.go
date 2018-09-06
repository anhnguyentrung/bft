package consensus

import (
	"bft/types"
	"sync"
	"sort"
	"log"
)

type ValidatorManager struct {
	rwMutex sync.RWMutex
	validators types.Validators
	self types.Validator
	proposer types.Validator
}

func NewValidatorManager(validators types.Validators, address string) *ValidatorManager {
	vm := &ValidatorManager{}
	vm.validators = validators
	sort.Sort(vm.validators)
	if vm.size() > 0 {
		vm.proposer = vm.getByIndex(0)
	}
	index, self := vm.getByAddress(address)
	if index == -1 {
		log.Fatal("self address is invalid")
	}
	vm.self = self
	return vm
}

func (vm *ValidatorManager) size() int {
	vm.rwMutex.RLock()
	defer vm.rwMutex.RUnlock()
	return len(vm.validators)
}

func (vm *ValidatorManager) getByIndex(index uint64) types.Validator {
	vm.rwMutex.RLock()
	defer vm.rwMutex.RUnlock()
	if index >= uint64(vm.size()) {
		log.Fatal("index is out of bounds")
	}
	return vm.validators[index]
}

func (vm *ValidatorManager) getByAddress(address string) (int, types.Validator) {
	for index, v := range vm.getValidators() {
		if v.Address == address {
			return index, v
		}
	}
	return -1, types.Validator{}
}

func (vm *ValidatorManager) getValidators() types.Validators {
	vm.rwMutex.RLock()
	defer vm.rwMutex.RUnlock()
	return vm.validators
}

func (vm *ValidatorManager) isProposer(validator types.Validator) bool {
	i, v := vm.getByAddress(validator.Address)
	if i == -1 {
		return false
	}
	return v.Equals(vm.proposer)
}

func (vm *ValidatorManager) calculateProposer(round uint64) {
	index := round % uint64(vm.size())
	vm.proposer = vm.getByIndex(index)
}