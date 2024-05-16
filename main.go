package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hellmany/cryptowatcher/tron"
)

func main() {
	cfg := tron.Config{
		GrpcHost: "127.0.0.1",
		GrpcPort: 50051,
		ZeroMQ:   "tcp://127.0.0.1:5555",
		//Contracts: []string{"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"},
	}

	c, err := tron.Client(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.StartListener()
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(20 * time.Second)

	cData, err := c.GetContractData("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	if err != nil {
		fmt.Printf("Error getting contract data: %s\n", err)
	}
	fmt.Printf("Contract data %+v\n", cData)

	/*
		for tx := range c.Ch {
			if tx.Contract != "" && tx.Contract != "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" {
				//	fmt.Printf("Received transaction %d '%s' '%s' '%s' '%s' %s\n", tx.Type, tx.TxId, tx.Address, tx.AddressTo, tx.Amount.String(), tx.Contract)
			}
			fmt.Printf("Received transaction %d '%s' '%s' '%s' '%s' '%s'\n", tx.Type, tx.TxId, tx.Address, tx.AddressTo, tx.Amount.String(), tx.Contract)
			//fmt.Printf("Received transaction %+v\n", tx)
			//fmt.Printf("Received transaction %+v\n", tx)
		}
	*/
}
