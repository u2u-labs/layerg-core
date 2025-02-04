package server

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type ContractCallParams struct {
	Sender          string
	ContractAddress string
	Abi             string
	Method          string
	Params          []interface{}
	Value           string
}

type TransactionRequest struct {
	From  common.Address
	To    common.Address
	Data  []byte
	Value *big.Int
}

func BuildContractCallRequest(p ContractCallParams) (*TransactionRequest, error) {
	if p.Value == "" {
		p.Value = "0"
	}
	parsedAbi, err := abi.JSON(strings.NewReader(p.Abi))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	data, err := parsedAbi.Pack(p.Method, p.Params...)
	if err != nil {
		return nil, fmt.Errorf("failed to pack data: %w", err)
	}
	value := new(big.Int)
	if _, ok := value.SetString(p.Value, 10); !ok {
		return nil, fmt.Errorf("invalid value: %s", p.Value)
	}
	tx := &TransactionRequest{
		From:  common.HexToAddress(p.Sender),
		To:    common.HexToAddress(p.ContractAddress),
		Data:  data,
		Value: value,
	}
	return tx, nil
}
