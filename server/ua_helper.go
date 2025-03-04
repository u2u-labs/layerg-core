package server

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/u2u-labs/go-layerg-common/runtime"
)

func BuildContractCallRequest(p runtime.ContractCallParams) (*runtime.TransactionRequest, error) {
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

	hexData := "0x" + hex.EncodeToString(data)

	tx := &runtime.TransactionRequest{
		From:  ptr(common.HexToAddress(p.Sender).Hex()),
		To:    ptr(common.HexToAddress(p.ContractAddress).Hex()),
		Data:  ptr(hexData),
		Value: ptr(value.String()),
	}

	return tx, nil
}

// ptr is a helper function to return a pointer to a string
func ptr[T any](v T) *T {
	return &v
}

func decodeURIToken(encoded string) (string, error) {
	uriDecoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return "", fmt.Errorf("URI decode failed: %v", err)
	}

	return uriDecoded, nil
}
