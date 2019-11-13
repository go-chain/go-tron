// Package address provides functionality for parsing and manipulating Tron addresses.
package address

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

// All addresses are prefixed with 0x41 so that when they are encoded into base 58 they
// start with 'T'.
const prefix byte = 0x41

// Address is a public identifier for an account that exists on the Tron network.
type Address [21]byte

var Zero = Address([21]byte{})

func FromPublicKey(pub *ecdsa.PublicKey) Address {
	// TODO(271): Remove dependencies for go-ethereum.

	// Compressed ECDSA keys have a byte prefix that we need to trim off to get the coordinates.
	xy := crypto.FromECDSAPub(pub)[1:]

	h := sha3.NewLegacyKeccak256()
	if _, err := h.Write(xy); err != nil {
		panic("address: unexpected error encountered while writing key")
	}

	// All addresses start with 0x41, the last 20 bytes of the hash are the address.
	var addr Address
	addr[0] = prefix
	copy(addr[1:], h.Sum(nil)[12:])

	return addr
}

// FromBase16 parses a base 16 (hexadecimal) string into an address.
func FromBase16(str string) (Address, error) {
	bs, err := hex.DecodeString(str)
	if err != nil {
		return Zero, err
	}

	if len(bs) != 21 {
		return Zero, fmt.Errorf("address: hex string is invalid length (%d)", len(bs))
	}

	var addr Address
	copy(addr[:], bs)

	return addr, err
}

// FromBase58 parses a base 58 checked string into an address.
func FromBase58(str string) (Address, error) {
	bs, check, err := base58.CheckDecode(str)
	if err != nil {
		return Zero, err
	}

	if check != prefix {
		return Zero, fmt.Errorf("address: invalid prefix (%d)", check)
	}

	var addr Address
	addr[0] = prefix
	copy(addr[1:], bs)

	return addr, nil
}

// ToBase16 encodes the address into a base 16 string.
func (a Address) ToBase16() string {
	return hex.EncodeToString(a[:])
}

// ToBase58 encodes the address into a checked base 58 string.
func (a Address) ToBase58() string {
	return base58.CheckEncode(a[1:], prefix)
}

func (a *Address) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	var (
		addr Address
		err  error
	)

	switch len(str) {
	case 42:
		addr, err = FromBase16(str)
	case 34:
		addr, err = FromBase58(str)
	default:
		return fmt.Errorf("address: unexpected length of json string (%d)", len(str))
	}

	if err != nil {
		return err
	}

	*a = addr

	return nil
}
