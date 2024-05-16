package main

import (
	"fmt"
	"log"

	"github.com/hellmany/cryptowatcher/bsc"
)

func main() {
	/*
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
	*/
	cfgBsc := bsc.Config{
		BscHost:   "127.0.0.1",
		BscPort:   8545,
		BscWsPort: 8546,
		BscWsPath: "/",

		//Contracts: []string{"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"},
	}

	cBsc, err := bsc.Client(&cfgBsc)
	if err != nil {
		log.Fatal(err)
	}

	cBsc.StartListener()

	/*	time.Sleep(20 * time.Second)

		cData, err := c.GetContractData("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
		if err != nil {
			fmt.Printf("Error getting contract data: %s\n", err)
		}
		fmt.Printf("Contract data %+v\n", cData)
	*/
	cData, err := cBsc.GetContractData("0x55d398326f99059ff775485246999027b3197955")
	if err != nil {
		fmt.Printf("Error getting contract data: %s\n", err)
	}
	fmt.Printf("Contract data %+v\n", cData)

	for _ = range cBsc.Ch {
		//if tx.Contract != "" && tx.Contract != "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" {
		//	fmt.Printf("Received transaction %d '%s' '%s' '%s' '%s' %s\n", tx.Type, tx.TxId, tx.Address, tx.AddressTo, tx.Amount.String(), tx.Contract)
		//}
		//			fmt.Printf("Received transaction %d '%s' '%s' '%s' '%s' '%s'\n", tx.Type, tx.TxId, tx.Address, tx.AddressTo, tx.Amount.String(), tx.Contract)
		//fmt.Printf("Received transaction %+v\n", tx)
		//fmt.Printf("Received transaction %+v\n", tx)
	}

}
