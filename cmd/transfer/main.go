package main

import (
	"github.com/go-chain/go-tron/account"
	"github.com/go-chain/go-tron/client"
	"log"
)

const (
	srcPrivKey  = "000000000000000000000000000000000000000000000000000000000000010f"
	destPrivKey = "000000000000000000000000000000000000000000000000000000000000010e"
)

func main() {
	src, err := account.FromPrivateKeyHex(srcPrivKey)
	if err != nil {
		log.Fatal("Failed to parse private key hex - ", err)
	}

	dest, err := account.FromPrivateKeyHex(destPrivKey)
	if err != nil {
		log.Fatal("Failed to parse private key hex - ", err)
	}

	cli := client.New("http://127.0.0.1:16667")

	tx, err := cli.Transfer(src, dest.Address(), 100000000000)
	if err != nil {
		log.Fatal("Failed to transfer tron balance - ", err)
	}

	log.Printf("%#v\n", tx)
}
