package tron

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/crypto"
)

type Transaction struct {
	Id              string           `json:"txID"`
	Signatures      []string         `json:"signature"`
	Results         *json.RawMessage `json:"ret"`
	ConstantResults *json.RawMessage `json:"constant_result"`
	Visible         *json.RawMessage `json:"visible"`
	RawData         *json.RawMessage `json:"raw_data"`
	RawDataHex      *json.RawMessage `json:"raw_data_hex"`
	ContractAddress *json.RawMessage `json:"contract_address"`
}

func (tx *Transaction) Sign(key *ecdsa.PrivateKey) error {
	if len(tx.Signatures) == 0 {
		tx.Signatures = make([]string, 0, 1)
	}

	hash, err := hex.DecodeString(tx.Id)
	if err != nil {
		return err
	}

	sig, err := crypto.Sign(hash, key)
	if err != nil {
		return err
	}

	tx.Signatures = append(tx.Signatures, hex.EncodeToString(sig))
	return nil
}
