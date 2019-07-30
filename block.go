package tron

type Block struct {
	Id           string        `json:"blockId"`
	BlockHeader  BlockHeader   `json:"block_header"`
	Transactions []Transaction `json:"transactions"`
}

type BlockHeader struct {
	RawData struct {
		Number              uint64 `json:"number"`
		TransactionTrieRoot string `json:"txTrieRoot"`
		WitnessAddress      string `json:"witness_address"`
		ParentHash          string `json:"parentHash"`
		Version             uint64 `json:"version"`
		Timestamp           uint64 `json:"timestamp"`
	} `json:"raw_data"`
	WitnessSignature string `json:"witness_signature"`
}
