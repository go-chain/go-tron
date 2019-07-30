package tron

import (
	"crypto/ecdsa"
	"github.com/go-chain/go-tron/address"
)

// Signer is an interface for implementations of objects that
// append signatures to signable objects.
type Signer interface {
	Sign(signable Signable) error
}

// Signable is an interface for implementations of signable objects.
// Implementations may choose if it accepts one or many signatures.
type Signable interface {
	Sign(key *ecdsa.PrivateKey) error
}

type AddressableSigner interface {
	Address() address.Address
	Signer
}