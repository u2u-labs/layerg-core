package utils

import (
	"context"
	"math/big"
	"strings"

	"github.com/u2u-labs/layerg-core/server/crawler/utils/models"
	"github.com/unicornultrafoundation/go-u2u/common"
	"github.com/unicornultrafoundation/go-u2u/ethclient"
	"github.com/unicornultrafoundation/go-u2u/rpc"
)

// AddressSet holds unique Ethereum addresses using a map
type AddressSet struct {
	addresses map[common.Address]struct{}
}

// NewAddressSet initializes a new AddressSet
func NewAddressSet() *AddressSet {
	return &AddressSet{
		addresses: make(map[common.Address]struct{}),
	}
}

// AddAddress adds a new address to the set if it doesn't already exist
func (a *AddressSet) AddAddress(addr common.Address) {
	if addr == (common.Address{}) {
		return
	}

	if _, exists := a.addresses[addr]; exists {
		return
	}
	a.addresses[addr] = struct{}{} // Use an empty struct for memory efficiency
}

// GetAddresses returns the list of unique addresses
func (a *AddressSet) GetAddresses() []common.Address {
	addressList := make([]common.Address, 0, len(a.addresses))
	for addr := range a.addresses {
		addressList = append(addressList, addr)
	}
	return addressList
}

// Reset clears all addresses in the AddressSet
func (a *AddressSet) Reset() {
	a.addresses = make(map[common.Address]struct{}) // Reinitialize the map
}

// FOR ERC 721

// TokenIdSet holds unique token IDs using a map of values
type TokenIdSet struct {
	tokenIds map[string]struct{}
}

// NewTokenIdSet initializes a new TokenIdSet
func NewTokenIdSet() *TokenIdSet {
	return &TokenIdSet{
		tokenIds: make(map[string]struct{}),
	}
}

// AddTokenId adds a new token ID to the set if it doesn't already exist
func (t *TokenIdSet) AddTokenId(tokenId *big.Int) {
	if tokenId == nil || tokenId.Cmp(big.NewInt(0)) == 0 {
		return
	}

	// Use string representation for the token ID
	tokenStr := tokenId.String()

	if _, exists := t.tokenIds[tokenStr]; exists {
		return
	}
	t.tokenIds[tokenStr] = struct{}{} // Use an empty struct for memory efficiency
}

// GetTokenIds returns the list of unique token IDs
func (t *TokenIdSet) GetTokenIds() []*big.Int {
	tokenIdList := make([]*big.Int, 0, len(t.tokenIds))
	for tokenStr := range t.tokenIds {
		id := new(big.Int)
		id.SetString(tokenStr, 10) // Convert string back to big.Int
		tokenIdList = append(tokenIdList, id)
	}
	return tokenIdList
}

// Reset clears all token IDs in the TokenIdSet
func (t *TokenIdSet) Reset() {
	t.tokenIds = make(map[string]struct{}) // Reinitialize the map
}

// ///
// FOR ERC 1155
// ///

// TokenIdContractAddressSet holds unique token ID and contract address pairs using a map of values
type TokenIdContractAddressSet struct {
	tokenIds map[string]struct{}
}

// NewTokenIdContractAddressSet initializes a new TokenIdContractAddressSet
func NewTokenIdContractAddressSet() *TokenIdContractAddressSet {
	return &TokenIdContractAddressSet{
		tokenIds: make(map[string]struct{}),
	}
}

// AddTokenId adds a new token ID and contract address pair to the set if it doesn't already exist
func (t *TokenIdContractAddressSet) AddTokenIdContractAddress(tokenId *big.Int, contractAddress string) {
	if tokenId == nil || tokenId.Cmp(big.NewInt(0)) == 0 || contractAddress == "" {
		return
	}

	// Create a unique key combining token ID and contract address
	tokenStr := tokenId.String() + ":" + contractAddress

	if _, exists := t.tokenIds[tokenStr]; exists {
		return
	}
	t.tokenIds[tokenStr] = struct{}{} // Use an empty struct for memory efficiency
}

// GetTokenIds returns the list of unique token ID and contract address pairs
func (t *TokenIdContractAddressSet) GetTokenIdContractAddressses() []struct {
	TokenId         *big.Int
	ContractAddress string
} {
	tokenIdList := make([]struct {
		TokenId         *big.Int
		ContractAddress string
	}, 0, len(t.tokenIds))

	for tokenStr := range t.tokenIds {
		// Split the unique key back into token ID and contract address
		parts := strings.Split(tokenStr, ":")
		if len(parts) != 2 {
			continue // Skip malformed entries
		}

		id := new(big.Int)
		id.SetString(parts[0], 10) // Convert string back to big.Int
		tokenIdList = append(tokenIdList, struct {
			TokenId         *big.Int
			ContractAddress string
		}{
			TokenId:         id,
			ContractAddress: parts[1],
		})
	}
	return tokenIdList
}

// Reset clears all token ID and contract address pairs in the TokenIdContractAddressSet
func (t *TokenIdContractAddressSet) Reset() {
	t.tokenIds = make(map[string]struct{}) // Reinitialize the map
}

func GetLastestBlockFromChainUrl(url string) (uint64, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	latest, err := client.BlockNumber(context.Background())
	if err != nil {
		return 0, err
	}

	return latest, nil
}

func InitChainClient(chain models.Chain) (*ethclient.Client, error) {
	return ethclient.Dial(chain.RpcUrl)
}

func InitNewRPCClient(url string) (*rpc.Client, error) {
	return rpc.Dial(url)
}
