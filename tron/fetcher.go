package tron

import (
	"context"
	"encoding/json"
	"math/big"
	"slices"
	"sync"
	"time"

	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	tron_client "github.com/fbsobreira/gotron-sdk/pkg/client"
	"google.golang.org/grpc"

	"github.com/go-zeromq/zmq4"
	zmq "github.com/go-zeromq/zmq4"
	"github.com/shopspring/decimal"
)

var DEBUG = false

type Message struct {
	Timestamp                   int64           `json:"timestamp"`
	TriggerName                 string          `json:"triggerName"`
	TransactionId               string          `json:"transactionId"`
	BlockHash                   string          `json:"blockHash"`
	BlockNumber                 int64           `json:"blockNumber"`
	EnergyUsage                 decimal.Decimal `json:"energyUsage"`
	EnergyFee                   decimal.Decimal `json:"energyFee"`
	OriginEnergyUsage           decimal.Decimal `json:"originEnergyUsage"`
	EnergyUsageTotal            decimal.Decimal `json:"energyUsageTotal"`
	NetUsage                    decimal.Decimal `json:"netUsage"`
	NetFee                      decimal.Decimal `json:"netFee"`
	Result                      string          `json:"result"`
	ContractAddress             string          `json:"contractAddress"`
	ContractType                string          `json:"contractType"`
	FeeLimit                    decimal.Decimal `json:"feeLimit"`
	ContractCallValue           decimal.Decimal `json:"contractCallValue"`
	ContractResult              string          `json:"contractResult"`
	FromAddress                 string          `json:"fromAddress"`
	ToAddress                   string          `json:"toAddress"`
	AssetName                   string          `json:"assetName"`
	AssetAmount                 decimal.Decimal `json:"assetAmount"`
	LatestSolidifiedBlockNumber int64           `json:"latestSolidifiedBlockNumber"`
	Data                        string          `json:"data"`
	TransactionIndex            int64           `json:"transactionIndex"`
	CumulativeEnergyUsed        decimal.Decimal `json:"cumulativeEnergyUsed"`
	PreCumulativeLogCount       int64           `json:"preCumulativeLogCount"`
	EnergyUnitPrice             decimal.Decimal `json:"energyUnitPrice"`
}

type Config struct {
	GrpcHost   string
	GrpcPort   int
	GrpcAPIKEY string
	Solidity   string
	ZeroMQ     string
	Contracts  []string
}

type Contract struct {
	Decimals *big.Int
	Symbol   string
	Name     string
}

type TronClient struct {
	C         *tron_client.GrpcClient
	Cfg       *Config
	ABIs      map[string]abi.ABI
	ChStatus  bool
	Ch        chan Transaction
	Mu        *sync.RWMutex
	Contracts map[string]Contract
	Ctx       context.Context
	CancelCtx context.CancelFunc
}

type Transaction struct {
	Type         int
	Raw          Message
	TxId         string
	Contract     string
	Address      string
	AddressTo    string
	Amount       decimal.Decimal
	ContractData *Contract
}

func Client(cfg *Config) (*TronClient, error) {

	DEBUG = false
	if os.Getenv("DEBUG") == "true" {
		DEBUG = true
	}
	TronClient := TronClient{}
	TronClient.Cfg = cfg

	TronClient.C = tron_client.NewGrpcClient(cfg.GrpcHost + ":" + strconv.Itoa(cfg.GrpcPort))
	if cfg.GrpcAPIKEY != "" {
		TronClient.C.SetAPIKey(cfg.GrpcAPIKEY)
	}
	err := TronClient.C.Start(grpc.WithInsecure())

	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	TronClient.Ctx = ctx
	TronClient.CancelCtx = cancel

	TronClient.Contracts = make(map[string]Contract)

	return &TronClient, nil
}

func (c *TronClient) GetContractData(contract string) (*Contract, error) {

	c.Mu.RLock()
	if cData, ok := c.Contracts[contract]; ok {
		c.Mu.RUnlock()
		return &cData, nil
	} else {
		c.Mu.RUnlock()

		dec, err := c.C.TRC20GetDecimals(contract)
		if err != nil {
			return nil, err
		}
		symbol, err := c.C.TRC20GetSymbol(contract)
		if err != nil {
			return nil, err
		}
		name, err := c.C.TRC20GetName(contract)
		if err != nil {
			return nil, err
		}

		c.Mu.Lock()
		c.Contracts[contract] = Contract{Decimals: dec, Symbol: symbol, Name: name}
		c.Mu.Unlock()

		return &Contract{Decimals: dec, Symbol: symbol, Name: name}, nil
	}
}

func (c *TronClient) StopListener() {
	c.ChStatus = false
	close(c.Ch)
}
func shirnk(c *TronClient, size int) {
	for {
		select {
		case <-c.Ctx.Done():
			//fmt.Printf("Context done initWs\n")
			return

		default:

			if len(c.Ch) > size {
				for i := 0; i < len(c.Ch)-size; i++ {
					<-c.Ch
				}
			}
		}
	}

}
func (c *TronClient) StartListener() (chan Transaction, error) {

	zctx := context.Background()
	s := zmq.NewSub(zctx, zmq.WithDialerRetry(time.Second))

	err := s.Dial(c.Cfg.ZeroMQ)
	if err != nil {
		return nil, err
	}

	c.Ch = make(chan Transaction, 1024)
	err = s.SetOption(zmq4.OptionSubscribe, "transaction")

	if err != nil {
		return nil, err
	}
	c.ChStatus = true
	c.Mu = &sync.RWMutex{}
	go shirnk(c, 1024)
	go func(c *TronClient, s zmq.Socket) {

		defer func() {
			if recover() != nil {
				c.ChStatus = false
				return
			}
		}()

		for {
			select {
			case <-c.Ctx.Done():
				//fmt.Printf("Context done initWs\n")
				return

			default:
				if !c.ChStatus {
					return
				}
				msgC, err := s.Recv()
				if err != nil {
					continue
				}
				msg := string(msgC.Frames[1])

				if msg == "transactionTrigger" {
					continue
				}

				go func(msg string) {
					defer func() {
						if err := recover(); err != nil {
							c.ChStatus = false
							return
						}
					}()

					t := Transaction{}
					contents := Message{}
					if err := json.Unmarshal([]byte(msg), &contents); err != nil {
						return
					}
					t.Type = 1
					t.Raw = contents
					t.TxId = contents.TransactionId
					t.Contract = contents.ContractAddress

					if contents.ContractAddress == "" && contents.FromAddress != "" && contents.ToAddress != "" && contents.AssetAmount.Cmp(decimal.NewFromFloat(0)) != 0 {

						t.Address = contents.FromAddress
						t.AddressTo = contents.ToAddress
						t.Amount = contents.AssetAmount
						if !c.ChStatus {
							return
						}

						c.Ch <- t
						//return

					} else {

						c.Mu.RLock()
						if len(c.Cfg.Contracts) == 0 || slices.Contains(c.Cfg.Contracts, contents.ContractAddress) {
							c.Mu.RUnlock()

							d, err := c.checkTransactionGrpc(contents.TransactionId)
							if err != nil {
								//log.Printf("Error checking transaction %s", err)
								return
							}

							if d == nil {
								return

							} else {
								t.Type = 2
								t.Address = contents.FromAddress
								t.AddressTo = d.AddressString
								t.Amount = d.AmountFloat

								if !c.ChStatus {
									return
								}
								c.Ch <- t
							}

							return
						}
						c.Mu.RUnlock()
						//return
					}

				}(msg)
			}
		}
	}(c, s)
	return nil, nil
}
