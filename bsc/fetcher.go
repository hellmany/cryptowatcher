package bsc

import (

	/*
		ethereum_watcher "github.com/HydroProtocol/ethereum-watcher"
		"github.com/HydroProtocol/ethereum-watcher/plugin"
		"github.com/shopspring/decimal"
		"github.com/sirupsen/logrus"
	*/

	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

var DEBUG = false

const (
	NOTIFY_TYPE_NONE = iota
	NOTIFY_TYPE_TX
	NOTIFY_TYPE_ADMIN
)

type SubscriptionResponse struct {
	Id     int    `json:"id"`
	Result string `json:"result"`
}
type SubscriptionMessage struct {
	JsonRpc string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}
type BlockHeader struct {
	ParentHash string `json:"parentHash"`
	Difficulty string `json:"difficulty"`
	Number     string `json:"number"`
	GasLimit   string `json:"gasLimit"`
	GasUsed    string `json:"gasUsed"`
	Timestamp  string `json:"timestamp"`
	Hash       string `json:"hash"`
}

type Params struct {
	Subscription string      `json:"subscription"`
	Result       interface{} `json:"result"`
}

type ResponseMessage struct {
	JsonRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  Params `json:"params"`
}

const (
	TYPE_BLOCK_HASH = iota
	TYPE_TXN_HASH
)

type ObjMessage struct {
	Type   int
	Hash   string
	Number *big.Int
}
type NotifyMessage struct {
	MessageType     int
	AddressFrom     string
	AddressTo       string
	Amount          *big.Int
	ContractAddress string
	IsPending       bool
	TxHash          string
}

type Config struct {
	BscHost   string
	BscPort   int
	BscWsPort int
	BscWsPath string

	// Contracts []string
}

func weiToEther(wei *big.Int) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(params.Ether))
}

func (c *BscClient) initWs() error {
	var MessageId int
	MessageId = 1

	//subHashHeads, err := SendMessage(c, MessageId, "newHeads")
	_, err := SendMessage(c.WSC, MessageId, "newHeads")
	if err != nil {

		return err
	}

	MessageId += 1

	subHashTransactions, err := SendMessage(c.WSC, MessageId, "newPendingTransactions")
	if err != nil {

		return err
	}

	for {
		var response ResponseMessage
		select {
		case <-c.Ctx.Done():
			//fmt.Printf("Context done initWs\n")
			return nil

		default:

			_, message, err := c.WSC.ReadMessage()
			if err != nil {

				return err
			}

			go func(message []byte) error {
				//	fmt.Printf("Message: %s\n", message)
				err = json.Unmarshal(message, &response)
				if err != nil {
					return fmt.Errorf("Could not decode message/parse json: %v, message %s", err, message)
				}
				if response.Params.Result == nil {

					return nil
				}

				if response.Params.Subscription == subHashTransactions {
					txHash := response.Params.Result.(string)

					c.ChObj <- ObjMessage{TYPE_TXN_HASH, txHash, nil}
				} else {
					var Header BlockHeader
					response.Params.Result = &Header

					err = json.Unmarshal(message, &response)
					if err != nil {
						return fmt.Errorf("Could not decode message/parse json: %v", err)
					}

					bgInt, err := hexutil.DecodeBig(Header.Number)
					if err != nil {
						return fmt.Errorf("Could not decode block number: %v", err)
					}

					c.ChObj <- ObjMessage{TYPE_BLOCK_HASH, Header.Hash, bgInt}
				}
				return nil
			}(message)
		}
	}
}

func (c *BscClient) StartWs() error {
	u := url.URL{Scheme: "ws", Host: c.Cfg.BscHost + ":" + strconv.Itoa(c.Cfg.BscWsPort), Path: c.Cfg.BscWsPath}

	for {
		select {
		case <-c.Ctx.Done():
			//fmt.Printf("Context done StartWs\n")
			return nil

		default:

			var err error
			c.WSC, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				return err
			}
			defer c.WSC.Close()

			c.initWs()
			time.Sleep(5 * time.Second)
		}
	}

}

func (c *BscClient) Listener() {
	defer func() {
		if err := recover(); err != nil {
			c.ChStatus = false
			return
		}
	}()

	for message := range c.ChObj {
		select {
		case <-c.Ctx.Done():
			//fmt.Printf("Context done Listener\n")
			return

		default:
			switch message.Type {
			case TYPE_BLOCK_HASH:

				go func() {
					defer func() {
						if err := recover(); err != nil {
							c.ChStatus = false
							return
						}
					}()

					_, txns, err := ReadBlock(c.C, message.Hash, nil)
					if err != nil {
						//continue
						return
					}

					for _, txn := range txns {
						if txn.Amount == nil || txn.Amount == big.NewInt(0) {
							continue
						}
						amount := decimal.NewFromBigInt(txn.Amount, 0)
						if amount.IsZero() {
							return
						}
						t := Transaction{}
						t.Type = 1
						if txn.ContractAddress != "" {
							t.Type = 2
						}
						t.Raw = txn
						t.TxId = txn.TxHash
						t.Address = txn.AddressFrom
						t.AddressTo = txn.AddressTo
						t.Amount = amount
						t.Contract = txn.ContractAddress
						t.IsPending = txn.IsPending
						//fmt.Println("Block: Sending to channel Transaction", txn)
						c.Ch <- t
					}
				}()

			case TYPE_TXN_HASH:

				go func() {
					txn, err := c.ReadTransaction(message.Hash)
					if err != nil || txn.Amount == nil || txn.Amount == big.NewInt(0) {
						return
					}
					//fmt.Println("Trans: Sending to channel Transaction", txn)
					amount := decimal.NewFromBigInt(txn.Amount, 0)
					if amount.IsZero() {
						return
					}
					t := Transaction{}
					t.Type = 1
					if txn.ContractAddress != "" {
						t.Type = 2
					}
					t.Raw = txn
					t.TxId = txn.TxHash
					t.Address = txn.AddressFrom
					t.AddressTo = txn.AddressTo
					t.Amount = amount
					t.Contract = txn.ContractAddress
					t.IsPending = txn.IsPending
					//t.ContractData, _ = c.GetContractData(txn.ContractAddress)
					//fmt.Println("Transaction: Sending to channel Transaction", t)
					c.Ch <- t
				}()

			default:

			}
		}

	}
}

type Transaction struct {
	Type         int
	Raw          NotifyMessage
	TxId         string
	Contract     string
	Address      string
	AddressTo    string
	Amount       decimal.Decimal
	IsPending    bool
	ContractData *Contract
}

type Contract struct {
	Decimals uint8
	Symbol   string
	Name     string
}
type BscClient struct {
	C        *ethclient.Client
	WSC      *websocket.Conn
	Cfg      *Config
	ABIs     map[string]abi.ABI
	ChStatus bool
	Ch       chan Transaction
	ChObj    chan ObjMessage
	//	ChNotify  chan NotifyMessage
	Mu        *sync.RWMutex
	Contracts map[string]Contract
	Ctx       context.Context
	CancelCtx context.CancelFunc
}

func shirnk(c *BscClient, size int) {
	for {
		select {
		case <-c.Ctx.Done():
			//fmt.Printf("Context done shirnk\n")
			return

		default:
			if len(c.Ch) > size {
				for i := 0; i < len(c.Ch)-size; i++ {
					<-c.Ch
				}

			}
			if len(c.ChObj) > size {
				for i := 0; i < len(c.ChObj)-size; i++ {
					<-c.ChObj
				}
			}

		}
	}

}
func (c *BscClient) GetContractData(contract string) (*Contract, error) {

	c.Mu.RLock()
	if cData, ok := c.Contracts[contract]; ok {
		c.Mu.RUnlock()
		return &cData, nil
	} else {
		c.Mu.RUnlock()

		dec, err := c.GetERC20Decimals(contract)
		if err != nil {
			return nil, err
		}
		symbol, err := c.GetERC20Symbol(contract)
		if err != nil {
			return nil, err
		}
		name, err := c.GetERC20Name(contract)
		if err != nil {
			return nil, err
		}

		c.Mu.Lock()
		c.Contracts[contract] = Contract{Decimals: dec, Symbol: symbol, Name: name}
		c.Mu.Unlock()

		return &Contract{Decimals: dec, Symbol: symbol, Name: name}, nil
	}
}
func Client(cfg *Config) (*BscClient, error) {

	if cfg.BscWsPath == "" {
		cfg.BscWsPath = "/"
	}
	BscClient := BscClient{}
	BscClient.Cfg = cfg
	var err error
	BscClient.C, err = ethclient.Dial("http://" + cfg.BscHost + ":" + strconv.Itoa(cfg.BscPort) + "/")
	if err != nil {
		return nil, err
	}

	BscClient.Mu = &sync.RWMutex{}
	BscClient.Contracts = make(map[string]Contract)
	ctx, cancel := context.WithCancel(context.Background())
	BscClient.Ctx = ctx
	BscClient.CancelCtx = cancel

	BscClient.Ch = make(chan Transaction, 1024)
	BscClient.ChObj = make(chan ObjMessage, 1024)
	//BscClient.ChNotify = make(chan NotifyMessage, 1024)
	go shirnk(&BscClient, 1000)

	return &BscClient, nil
}

func (c *BscClient) StopListener() {
	//fmt.Println("StopListener")
	c.ChStatus = false
	c.CancelCtx()
	time.Sleep(1 * time.Second)
	shirnk(c, 0)
	//close(c.Ch)
	//close(c.ChObj)

}
func (c *BscClient) StartListener() {

	go c.StartWs()
	go c.Listener()
	//go c.Notifier()

}
func GetSubscriptionMessage(messageId int, subscription string) ([]byte, error) {
	params := make([]interface{}, 0)
	params = append(params, subscription)

	m := SubscriptionMessage{
		"2.0",
		messageId,
		"eth_subscribe",
		params,
	}

	b, err := json.Marshal(m)
	if err != nil {
		return []byte{}, err
	}

	return b, nil
}

func SendMessage(c *websocket.Conn, messageId int, subscription string) (string, error) {
	var resp SubscriptionResponse

	w, err := c.NextWriter(websocket.TextMessage)
	if err != nil {
		return "", err
	}

	messageTo, err := GetSubscriptionMessage(messageId, subscription)

	if err != nil {
		return "", err
	}

	w.Write(messageTo)
	w.Write([]byte{'\n'})
	w.Close()

	_, message, err := c.ReadMessage()
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(message, &resp)
	if err != nil {

		return "", err
	}

	return resp.Result, err
}

/*
func a() {

	api := "http://162.252.20.228:8545/"
	w := ethereum_watcher.NewHttpBasedEthWatcher(context.Background(), api)

	// we use TxReceiptPlugin here
	w.RegisterTxReceiptPlugin(plugin.NewERC20TransferPlugin(
		func(token, from, to string, amount decimal.Decimal, isRemove bool) {
			//if to == "0x328623887ecc8af6315e77404a9603e9daa83869" {
			logrus.Infof("New ERC20 Transfer >> token(%s), %s -> %s, amount: %s, isRemoved: %t",
				token, from, to, amount, isRemove)
			//}

		},
	))

	w.RunTillExit()

}

*/
