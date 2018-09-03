package types

import "bft/crypto"

type Validator struct {
	Address string
	PublicKey crypto.PublicKey
}
