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
func (vs Validators) Len() int {
	return len(vs)
}

func (vs Validators) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs Validators) Less(i, j int) bool {
	return vs[i].Address < vs[j].Address
}
