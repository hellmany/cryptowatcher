# cryptowatcher
Real-time tron/bsc blockchain parser
A module for real-time transaction analysis on the tron blockchain.

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