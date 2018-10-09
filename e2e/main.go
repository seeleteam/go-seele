/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// API which complete the interface can be executed
type API interface {
	Handle()
}

var (
	// Accounts is default accounts from config
	Accounts = make(map[string]map[string]Account)
	filePath = "./config/keystore.json"
)

// Account is the public and private Key
type Account struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

// Handlers is the functions that will be executed
var Handlers []API

// main is the entrance
func main() {
	run()
}

func init() {
	buff, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic("read file err")
	}

	if err = json.Unmarshal(buff, &Accounts); err != nil {
		fmt.Println("Failed to unmarshal, err:", err)
		panic("Failed to unmarshal")
	}

}

// run execute every handle one by one by using gorutine
func run() {
	for _, obj := range Handlers {
		go obj.Handle()
	}
}
