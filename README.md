# cryptowatcher
Real-time tron/bsc blockchain parser

A module for real-time transaction analysis on the tron/bsc blockchains.

You get a stream with transaction data such as sender, receiver, contract and amount.

You can also query the contract data .

# For Tron
The module connects to the blockchain via gRPC and clears the queue via ZeroMQ, i.e. zeromq must be enabled on the node.
https://developers.tron.network/docs/use-java-trons-built-in-message-queue-for-event-subscription
and Grpc
https://tronprotocol.github.io/documentation-en/api/rpc/

Config structure
```
type Config struct {
	GrpcHost   string
	GrpcPort   int
	GrpcAPIKEY string
	Solidity   string
	ZeroMQ     string
	Contracts  []string
}
```
If Contracts are specified, only transactions for these contracts will be analyzed.

Contract information
```
func (c *TronClient) GetContractData(contract string) (*Contract, error) 

type Contract struct {
	Decimals *big.Int
	Symbol string
	Name string
}
```

example:

```
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
		for _ = range c.Ch {
					fmt.Printf("Received transaction %d '%s' '%s' '%s' '%s' '%s'\n", tx.Type, tx.TxId, tx.Address, tx.AddressTo, tx.Amount.String(), tx.Contract)
		
			}
```

# For Bsc
Config structure

```
type Config struct {
	BscHost   string
	BscPort   int
	BscWsPort int
	BscWsPath string
}
```

Contract information
```
func (c *BscClient) GetContractData(contract string) (*Contract, error) 

type Contract struct {
	Decimals uint8
	Symbol string
	Name string
}
```

Example in main.go


Reading from channel
```
	for tx := range cBsc.Ch {
		fmt.Printf("Received transaction %d '%s' '%s' '%s' '%s' '%s'\n", tx.Type, tx.TxId, tx.Address, tx.AddressTo, tx.Amount.String(), tx.Contract)
	}
```

Example:
```
Received transaction 2 '0x24f2dff84b0e1ecff758ac604430d570cf69af30d31e8b1956e5154d68944c89' '0x214E596200B99c9c0e8a92b92FABd86535eA1eE9' '0xC36750db3bAf87D9B92FD7f70A75Cf0cCbd03712' '1001581293' 0x1236a887ef31B4d32E1F0a2b5e4531F52CeC7E75
```

and 

```
cData, err := cBsc.GetContractData("0x55d398326f99059ff775485246999027b3197955")
	if err != nil {
		fmt.Printf("Error getting contract data: %s\n", err)
	}
	fmt.Printf("Contract data %+v\n", cData)
```
Response
```
Contract data &{Decimals:18 Symbol:USDT Name:Tether USD}
```

Example

```
	cfgBsc := bsc.Config{
		BscHost:   "127.0.0.1",
		BscPort:   8545,
		BscWsPort: 8546,
		BscWsPath: "/",
	
	}

	cBsc, err := bsc.Client(&cfgBsc)
	if err != nil {
		log.Fatal(err)
	}

	cBsc.StartListener()

	
	
	cData, err := cBsc.GetContractData("0x55d398326f99059ff775485246999027b3197955")
	if err != nil {
		fmt.Printf("Error getting contract data: %s\n", err)
	}
	fmt.Printf("Contract data %+v\n", cData)

	for _ = range cBsc.Ch {

					fmt.Printf("Received transaction %d '%s' '%s' '%s' '%s' '%s'\n", tx.Type, tx.TxId, tx.Address, tx.AddressTo, tx.Amount.String(), tx.Contract)
		
	}
```
Made for self-usage, but PR welcomes.
yk@rbmedia.io
