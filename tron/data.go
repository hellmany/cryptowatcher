package tron

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/shopspring/decimal"
	"github.com/tron-us/go-common/crypto"
)

type decodedData struct {
	Address       common.Address
	AddressString string
	Amount        *big.Int
	AmountFloat   decimal.Decimal
}

type tronTransaction struct {
	Ret []struct {
		ContractRet string `json:"contractRet"`
	} `json:"ret"`
	Signature []string `json:"signature"`
	TxId      string   `json:"txID"`
	RawData   struct {
		Contract []struct {
			Parameter struct {
				Value struct {
					Data            string `json:"data"`
					OwnerAddress    string `json:"owner_address"`
					ContractAddress string `json:"contract_address"`
				} `json:"value"`
				TypeUrl    string `json:"type_url"`
				RawDataHex string `json:"raw_data_hex"`
			} `json:"parameter"`
		} `json:"contract"`
	} `json:"raw_data"`
}

func (c *TronClient) checkTransaction(txId string) (*decodedData, error) {

	t, err := c.getTransaction(txId)
	if err != nil {
		return nil, err
	}
	d, err := c.decodeTransData(t)
	if err != nil {
		return nil, err
	}
	return d, nil
}
func (c *TronClient) decodeTransData(tx *tronTransaction) (*decodedData, error) {
	if len(tx.RawData.Contract) == 0 {
		return nil, fmt.Errorf("no contract")
	}
	if tx.RawData.Contract[0].Parameter.Value.Data == "" {
		return nil, fmt.Errorf("no data")
	}
	decodeData, err := hex.DecodeString(tx.RawData.Contract[0].Parameter.Value.Data)
	if err != nil {
		return nil, err
	}
	// a9059cbb == transfer
	if method, ok := c.ABIs[tx.RawData.Contract[0].Parameter.Value.ContractAddress].Methods["transfer"]; ok {
		params, err := method.Inputs.Unpack(decodeData[4:])
		if err != nil {
			return nil, err
		}
		t := &decodedData{}

		if len(params) == 2 {
			t.Address = params[0].(common.Address)
			t.AddressString = t.Address.String()[2:]
			t.Amount = params[1].(*big.Int)
			t.AmountFloat = decimal.NewFromBigInt(t.Amount, 0).Div(decimal.NewFromFloat(1000000))

			return t, nil
		} else {
			return nil, err
		}

	}

	return nil, fmt.Errorf("method not found")
}
func (c *TronClient) checkTransactionGrpc(txId string) (*decodedData, error) {

	t, err := c.getTransactionGrpc(txId)

	if err != nil {
		return nil, err
	}
	d, err := c.decodeTransDataGrpc(t)
	if err != nil {
		return nil, err
	}
	return d, nil
}
func (c *TronClient) decodeTransDataGrpc(tx *core.TransactionInfo) (*decodedData, error) {
	if tx.Result != 0 {
		return nil, fmt.Errorf("Transaction failed")
	}
	if (len(tx.Log) < 1 || len(tx.Log[0].Topics) < 3 || len(tx.Log[0].Topics[2]) < 24) || (len(tx.Log[0].Data) == 0) {
		return nil, fmt.Errorf("no data")
	}
	method := hex.EncodeToString(tx.Log[0].Topics[0])
	if method != "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef" {
		return nil, fmt.Errorf("method not transfer")
	}

	tO := hex.EncodeToString(tx.Log[0].Topics[2])
	if len(tx.Log[0].Topics) != 3 && len(tO) < 24 {
		return nil, fmt.Errorf("no topics")
	}

	toAddress := "41" + tO[24:]
	toAddressBase58, err := crypto.Encode58Check(&toAddress)

	if err != nil {
		return nil, err
	}

	amountHex := hex.EncodeToString(tx.Log[0].Data)
	i := new(big.Int)
	i.SetString(amountHex, 16)

	t := &decodedData{}
	t.AddressString = *toAddressBase58
	t.AmountFloat = decimal.NewFromBigInt(i, 0)

	return t, nil
}
func logData(v any) {
	//log.Println(v)
	s, err := json.Marshal(v)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(string(s))
	log.Println()
}

func (c *TronClient) getTransactionGrpc(txId string) (*core.TransactionInfo, error) {

	tx, err := c.C.GetTransactionInfoByID(txId)

	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (c *TronClient) getTransaction(txId string) (*tronTransaction, error) {

	reqStr := []byte(`{"value":"` + txId + `"}`)
	req, err := http.NewRequest("POST", c.Cfg.Solidity, bytes.NewBuffer(reqStr))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	trans := &tronTransaction{}
	err = json.NewDecoder(resp.Body).Decode(&trans)
	if err != nil {
		return nil, err
	}
	return trans, nil
}
