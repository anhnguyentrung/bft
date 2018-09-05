package types

import (
	"bft/crypto"
)

type Validator struct {
	Address string
	PublicKey crypto.PublicKey
}

func (v Validator) Equals(target Validator) bool {
	return v.Address == target.Address && v.PublicKey.Equals(target.PublicKey)
}

type Validators []Validator

// sort interface
func (validators Validators) Len() int {
	return len(validators)
}

func (validators Validators) Swap(i, j int) {
	validators[i], validators[j] = validators[j], validators[i]
}

func (validators Validators) Less(i, j int) bool {
	return validators[i].Address < validators[j].Address
}
