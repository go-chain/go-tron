// Package account provides functionality for managing Tron network accounts.
package account

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"github.com/go-chain/go-tron"
	"github.com/go-chain/go-tron/address"
	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/crypto"
)

type Account interface {
	Address() address.Address
	tron.Signer
}

// LocalAccount is a private key address pair.
// TODO(271): Add more functionality to this.
type LocalAccount struct {
	addr address.Address
	priv *ecdsa.PrivateKey
}

func NewLocalAccount() *LocalAccount {

	acc := new(LocalAccount)

	privateKey, err := ecdsa.GenerateKey(btcec.S256(), rand.Reader)

	if err != nil {
		panic(err)
	}

	pub := ecdsa.PublicKey{Curve: btcec.S256(), X: privateKey.X, Y: privateKey.Y}

	acc.addr = address.FromPublicKey(&pub)
	acc.priv = privateKey

	return acc
}

func (a *LocalAccount) PrivateKey() string {
	return fmt.Sprintf("%x",a.priv.D.Bytes())
}


// FromPrivateKeyHex derives an account from a hexadecimal private key string.
func FromPrivateKeyHex(hex string) (*LocalAccount, error) {
	priv, err := crypto.HexToECDSA(hex)
	if err != nil {
		return nil, err
	}

	return &LocalAccount{
		addr: address.FromPublicKey(&priv.PublicKey),
		priv: priv,
	}, nil
}

// Address returns the address of the account.
func (a *LocalAccount) Address() address.Address {
	return a.addr
}

// Sign signs a signable object with the account's private key.
func (a *LocalAccount) Sign(signable tron.Signable) error {
	return signable.Sign(a.priv)
}
