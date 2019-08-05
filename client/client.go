// Package client provides functionality for interacting with the Tron node RESTful APIs.
package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chain/go-tron"
	"github.com/go-chain/go-tron/abi"
	"github.com/go-chain/go-tron/account"
	"github.com/go-chain/go-tron/address"
	"io/ioutil"
	"net/http"
	"time"
)

type Client struct {
	// Host is the host of the full node API.
	// TODO(271): Potentially look at bundling this with a more generic network config.
	host string

	// Throttle is the amount of time to wait between querying the state of a transaction.
	throttle time.Duration
}

// New creates a new client for the provided host.
func New(host string) *Client {
	return &Client{
		host:     host,
		throttle: 3 * time.Second,
	}
}

type Getaccount struct {
	Address             string `json:"address"`
	Balance             int64  `json:"balance"`
	AssetV2             []V2   `json:"assetV2"`
	FreeAssetNetUsageV2 []V2   `json:"free_asset_net_usageV2"`
}

type V2 struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

//getaccount

func (c *Client) GetAccount(addr string) (Getaccount, error) {

	add, err := address.FromBase58(addr)
	if err != nil {
		return Getaccount{}, err
	}

	var request = struct {
		Address string `json:"address"`
	}{
		Address: add.ToBase16(),
	}

	var acc Getaccount
	if err := c.post("/wallet/getaccount", &request, &acc); err != nil {
		return Getaccount{}, err
	}
	return acc, nil

}

// GetBlockByHeight returns the block at the specified height.
func (c *Client) GetBlockByHeight(n uint64) (*tron.Block, error) {
	var request = struct {
		Num uint64 `json:"num"`
	}{
		Num: n,
	}

	var block tron.Block
	if err := c.post("wallet/getblockbynum", &request, &block); err != nil {
		return nil, err
	}

	if block.Id == "" {
		return nil, nil
	}

	return &block, nil
}

// GetBlockById returns the block for the specified id.
func (c *Client) GetBlockById(id string) (*tron.Block, error) {
	var request = struct {
		Value string `json:"value"`
	}{
		Value: id,
	}

	var block tron.Block
	if err := c.post("wallet/getblockbyid", &request, &block); err != nil {
		return nil, err
	}

	if block.Id == "" {
		return nil, nil
	}

	return &block, nil
}

// GetBlockRange returns the blocks within a height range, end exclusive.
func (c *Client) GetBlockRange(start, end uint64) ([]tron.Block, error) {
	var request = struct {
		Start uint64 `json:"startNum"`
		End   uint64 `json:"endNum"`
	}{
		Start: start,
		End:   end,
	}

	var response = struct{ Blocks []tron.Block `json:"block"` }{}
	if err := c.post("wallet/getblockbylimitnext", &request, &response); err != nil {
		return nil, err
	}

	return response.Blocks, nil
}

// GetLatestBlocks returns the last n blocks synced to the node.
func (c *Client) GetLatestBlocks(n int) ([]tron.Block, error) {
	var request = struct {
		Num int `json:"num"`
	}{
		Num: n,
	}

	var response = struct{ Blocks []tron.Block `json:"block"` }{}
	if err := c.post("wallet/getblockbylatestnum", &request, &response); err != nil {
		return nil, err
	}

	return response.Blocks, nil
}

// GetLatestBlock returns the latest block synced to the node.
func (c *Client) GetLatestBlock() (tron.Block, error) {
	var request = struct{}{}

	var block tron.Block
	if err := c.post("wallet/getnowblock", &request, &block); err != nil {
		return tron.Block{}, err
	}

	// TODO(271): This shouldn't ever return this, right? :npc:
	if block.Id == "" {
		return tron.Block{}, errors.New("client: not expecting latest block to be nil")
	}

	return block, nil
}

// TransactionInfo contains information about a processed transaction.
type TransactionInfo struct {
	Id              string             `json:"id"`
	Fee             uint64             `json:"fee"`
	BlockNumber     uint64             `json:"blockNumber"`
	BlockTimestamp  uint64             `json:"blockTimestamp"`
	ContractResult  []string           `json:"contractResult"`
	ContractAddress address.Address    `json:"contract_address"`
	Receipt         TransactionReceipt `json:"receipt"`
	Log             *json.RawMessage   `json:"log"`
}

func (t TransactionInfo) Error() error {
	switch t.Receipt.Result {
	case TxResultSuccess:
		return nil
	default:
		return errors.New("TODO")
	}
}

// TransactionResult is an enumeration which described what happened when
// processing a transaction. This is not to be confused with the result
// of executing a contract which will be an ABI encoded payload.
type TransactionResult string

const (
	TxResultSuccess             TransactionResult = "SUCCESS"
	TxResultRevert              TransactionResult = "REVERT"
	TxResultBadJump             TransactionResult = "BAD_JUST_DESTINATION"
	TxResultOutOfMemory         TransactionResult = "OUT_OF_MEMORY"
	TxResultPrecompiledContract TransactionResult = "PRECOMPILED_CONTRACT"
	TxResultStackTooSmall       TransactionResult = "STACK_TOO_SMALL"
	TxResultStackTooLarge       TransactionResult = "STACK_TOO_LARGE"
	TxResultIllegalOp           TransactionResult = "ILLEGAL_OPERATION"
	TxResultStackOverflow       TransactionResult = "STACK_OVERFLOW"
	TxResultOutOfEnergy         TransactionResult = "OUT_OF_ENERGY"
	TxOutOfTime                 TransactionResult = "OUT_OF_TIME"
	TxResultJVMStackOverflow    TransactionResult = "JVM_STACK_OVER_FLOW"
	TxResultUnknown             TransactionResult = "UNKNOWN"
	TxResultTransferFailed      TransactionResult = "TRANSFER_FAILED"
)

type TransactionReceipt struct {
	EnergyFee        uint64            `json:"energy_fee"`
	EnergyUsageTotal uint64            `json:"energy_usage_total"`
	NetFee           uint64            `json:"net_fee"`
	NetUsage         uint64            `json:"net_usage"`
	Result           TransactionResult `json:"result"`
}

// Transfer transfers a balance of Tron from a source account to a destination address.
func (c *Client) Transfer(src account.Account, dest address.Address, amount uint64) (tron.Transaction, error) {
	var request = struct {
		Owner  string `json:"owner_address"`
		To     string `json:"to_address"`
		Amount uint64 `json:"amount"`
	}{
		Owner:  src.Address().ToBase16(),
		To:     dest.ToBase16(),
		Amount: amount,
	}

	var tx tron.Transaction
	if err := c.post("wallet/createtransaction", &request, &tx); err != nil {
		return tron.Transaction{}, err
	}

	if err := src.Sign(&tx); err != nil {
		return tron.Transaction{}, err
	}

	//if err := c.BroadcastTransaction(&tx); err != nil {
	//	return "", err
	//}

	//return c.await(tx.Id)
	return tx, nil

}

//TransferAsset trc10
func (c *Client) TransferAsset(src account.Account, dest address.Address, assetName string, amount uint64) (tron.Transaction, error) {
	var request = struct {
		Owner  string `json:"owner_address"`
		To     string `json:"to_address"`
		Amount uint64 `json:"amount"`
		Asset  string `json:"asset_name"`
	}{
		Owner:  src.Address().ToBase16(),
		To:     dest.ToBase16(),
		Amount: amount,
		Asset:  assetName,
	}
	var tx tron.Transaction
	if err := c.post("wallet/transferasset", &request, &tx); err != nil {
		return tron.Transaction{}, err
	}

	if err := src.Sign(&tx); err != nil {
		return tron.Transaction{}, err
	}

	//if err := c.BroadcastTransaction(&tx); err != nil {
	//	return "", err
	//}

	//return c.await(tx.Id)
	return tx, nil

}

// TransactionInfoById returns the information about a processed transaction. If the transaction
// does not exist or has not yet been processed then the returned information will be nil even
// though an error will not be returned.
func (c *Client) TransactionInfoById(id string) (*TransactionInfo, error) {
	var request = struct {
		Value string `json:"value"`
	}{
		Value: id,
	}

	var info TransactionInfo
	if err := c.post("wallet/gettransactioninfobyid", &request, &info); err != nil {
		return nil, err
	}

	// Transactions that exist will always have an identifier returned.
	if info.Id == "" {
		return nil, nil
	}

	return &info, nil
}

// TransactionById returns the transaction for the provided id.
func (c *Client) TransactionById(id string) (*tron.Transaction, error) {
	var request = struct {
		Value string `json:"value"`
	}{
		Value: id,
	}

	var info tron.Transaction
	if err := c.post("wallet/gettransactionbyid", &request, &info); err != nil {
		return nil, err
	}

	// Transactions that exist will always have an identifier returned.
	if info.Id == "" {
		return nil, nil
	}

	return &info, nil
}

type DeployContractInput struct {
	ABI               abi.ABI
	Arguments         []interface{}
	Bytecode          []byte
	Name              string
	FeeLimit          uint64
	CallValue         uint64
	Owner             address.Address
	OriginEnergyLimit uint64
}

// DeployContract deploys a contract. The owner of the deployed contract will be the
// account that this function was called with.
func (c *Client) DeployContract(acc account.Account, input DeployContractInput) (*TransactionInfo, error) {
	// TODO(271): ABI encoding.
	request := struct {
		ABI               string `json:"abi"`
		Bytecode          string `json:"bytecode"`
		Name              string `json:"name"`
		FeeLimit          uint64 `json:"fee_limit"`
		CallValue         uint64 `json:"call_value"`
		OwnerAddress      string `json:"owner_address"`
		OriginEnergyLimit uint64 `json:"origin_energy_limit"`
		Parameter         string `json:"parameter"`
	}{
		ABI:               "[]",
		Bytecode:          hex.EncodeToString(input.Bytecode),
		Name:              input.Name,
		FeeLimit:          input.FeeLimit,
		CallValue:         input.CallValue,
		OwnerAddress:      acc.Address().ToBase16(),
		OriginEnergyLimit: input.OriginEnergyLimit,
		Parameter:         hex.EncodeToString(input.ABI.Constructor.Encode(input.Arguments...)),
	}

	var tx tron.Transaction
	if err := c.post("wallet/deploycontract", &request, &tx); err != nil {
		return nil, err
	}

	if err := acc.Sign(&tx); err != nil {
		return nil, err
	}

	if err := c.BroadcastTransaction(&tx); err != nil {
		return nil, err
	}

	return c.await(tx.Id)
}

type CallContractInput struct {
	Address   address.Address
	Function  abi.Function
	Arguments []interface{}
	FeeLimit  uint64
	CallValue uint64
	Result    interface{}
}

// CallContract calls a function of a contract. If the function is immutable (either 'pure' or 'view') then
// the constant function is triggered and the returned encoded ABI value is unmarshaled to
// CallContractInput.Result. Immutable functions will return nil transaction info because there is no
// transaction that is committed to the blockchain. Mutable function calls will be broadcasted and
// the function will wait until the call has been completed. The returned ABI value is also unmarshaled
// to CallContractInput.Result. Mutable functions will return transaction info if they are successfully
// processed.
func (c *Client) CallContract(acc account.Account, input CallContractInput) (tron.Transaction, error) {
	request := struct {
		ContractAddress  string `json:"contract_address"`
		FunctionSelector string `json:"function_selector"`
		Parameter        string `json:"parameter"`
		FeeLimit         uint64 `json:"fee_limit"`
		CallValue        uint64 `json:"call_value"`
		OwnerAddress     string `json:"owner_address"`
	}{
		ContractAddress:  input.Address.ToBase16(),
		FunctionSelector: input.Function.Signature(),
		Parameter:        hex.EncodeToString(input.Function.Encode(input.Arguments...)),
		FeeLimit:         input.FeeLimit,
		CallValue:        input.CallValue,
		OwnerAddress:     acc.Address().ToBase16(),
	}

	var endpoint string
	switch {
	case input.Function.Immutable():
		endpoint = "wallet/triggerconstantcontract"
	default:
		endpoint = "wallet/triggersmartcontract"
	}

	if !input.Function.Payable() {
		if input.CallValue > 0 {
			return tron.Transaction{}, errors.New("client: cannot send tron to non-payable function")
		}
	}

	response := struct {
		Result      []string         `json:"constant_result"`
		Transaction tron.Transaction `json:"transaction"`
	}{}
	if err := c.post(endpoint, &request, &response); err != nil {
		return tron.Transaction{}, err
	}

	switch {
	case input.Function.Immutable():
		if len(response.Result) < 1 {
			return tron.Transaction{}, nil
		}

		bs, err := hex.DecodeString(response.Result[0])
		if err != nil {
			return tron.Transaction{}, err
		}

		if err := abi.Unmarshal(bs, input.Function, input.Result); err != nil {
			return tron.Transaction{}, err
		}

		return tron.Transaction{}, nil
	default:
	}

	tx := response.Transaction

	if err := acc.Sign(&tx); err != nil {
		return tron.Transaction{}, err
	}

	//if err := c.BroadcastTransaction(&tx); err != nil {
	//	return "", err
	//}

	return tx, nil

	//info, err := c.await(tx.Id)
	//if err != nil {
	//	return nil, err
	//}
	//
	//if len(info.ContractResult) < 1 {
	//	return info, nil
	//}
	//
	//// TODO(271): Probably can be simplified with above code as well.
	//bs, err := hex.DecodeString(info.ContractResult[0])
	//if err != nil {
	//	return nil, err
	//}
	//
	//if err := abi.Unmarshal(bs, input.Function, input.Result); err != nil {
	//	return nil, err
	//}
	//
	//return info, nil
}

func (c *Client) TriggerSmartContract(acc account.Account, input CallContractInput) ([]string, error) {
	request := struct {
		ContractAddress  string `json:"contract_address"`
		FunctionSelector string `json:"function_selector"`
		Parameter        string `json:"parameter"`
		FeeLimit         uint64 `json:"fee_limit"`
		CallValue        uint64 `json:"call_value"`
		OwnerAddress     string `json:"owner_address"`
	}{
		ContractAddress:  input.Address.ToBase16(),
		FunctionSelector: input.Function.Signature(),
		Parameter:        hex.EncodeToString(input.Function.Encode(input.Arguments...)),
		FeeLimit:         input.FeeLimit,
		CallValue:        input.CallValue,
		OwnerAddress:     input.Address.ToBase16(),
	}

	var endpoint string
	switch {
	case input.Function.Immutable():
		endpoint = "wallet/triggerconstantcontract"
	default:
		endpoint = "wallet/triggersmartcontract"
	}

	if !input.Function.Payable() {
		if input.CallValue > 0 {
			return nil, errors.New("client: cannot send tron to non-payable function")
		}
	}

	response := struct {
		Result      []string          `json:"constant_result"`
		Transaction *tron.Transaction `json:"transaction"`
	}{}
	if err := c.post(endpoint, &request, &response); err != nil {
		return nil, err
	}

	if len(response.Result) < 1 {
		return nil, errors.New("response result length err")
	}

	return response.Result, nil

}

// BroadcastTransaction broadcasts a signed transaction to the network.
func (c *Client) BroadcastTransaction(tx *tron.Transaction) error {
	// TODO(271): Add in additional pieces for errors.
	var response = struct {
		Result bool `json:"result"`
	}{}

	if err := c.post("wallet/broadcasttransaction", &tx, &response); err != nil {
		return err
	}

	if !response.Result {
		return errors.New("client: failed to broadcast transaction")
	}

	return nil
}

// post marshals a request to json and then posts it to an endpoint of the full node server,
// then once the response is received it unmarshals it into the response.
func (c *Client) post(endpoint string, request interface{}, response interface{}) error {
	bs, err := json.Marshal(request)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.getFullNodeURL(endpoint), bytes.NewReader(bs))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("client: unexpected status code (%d)", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.NewDecoder(bytes.NewReader(data)).Decode(response); err != nil {
		return err
	}

	return nil
}

// await waits for a transaction to complete processing. The number of requests
// that are made per unit of time is controlled by the throttle config, and
// the timeout will dictate how long this function will await before giving up.
// TODO(271): Allow this to be public?
func (c *Client) await(id string) (*TransactionInfo, error) {
	for {
		info, err := c.TransactionInfoById(id)
		if err != nil {
			return nil, err
		}

		if info == nil {
			time.Sleep(c.throttle)
			continue
		}

		return info, nil
	}
}

// getFullNodeURL returns the URL to a service endpoint.
func (c *Client) getFullNodeURL(endpoint string) string {
	return fmt.Sprintf("%s/%s", c.host, endpoint)
}
