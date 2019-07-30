# Go-Tron

A Tron API implementation written in Go

## Notice

This API is currently in alpha and it is not recommended it be used in production. __Use this at your own risk.__ When the API is stablized this notice will be removed and an issue will be filed indicating that it is safe to use. This will be done when test coverage hits a healthy percentage and the API has matured a bit.

## Examples

### Transfer

```go
package main

import (
	"github.com/0x10f/go-tron/account"
	"github.com/0x10f/go-tron/client"
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

	tx, err := cli.Transfer(src, dest.Address(), 1000000 /* in sun */)
	if err != nil {
		log.Fatal("Failed to transfer tron balance - ", err)
	}

	log.Printf("%#v\n", tx)
}
```