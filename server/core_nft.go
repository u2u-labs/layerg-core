package server

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/u2u-labs/go-layerg-common/runtime"
	"github.com/u2u-labs/layerg-core/server/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Struct for the POST request response
type CreateAssetNFTResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AvatarUrl   string `json:"avatarUrl"`
	ProjectId   string `json:"projectId"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// Struct to represent media information
type Media struct {
	ID      string `json:"id"`
	S3Url   string `json:"S3Url"`
	IPFSUrl string `json:"IPFSUrl"`
	AssetId string `json:"AssetId"`
}

// Struct to represent individual metadata attributes
type Attribute struct {
	Value     interface{} `json:"value"`
	TraitType string      `json:"trait_type"`
}

// Struct to represent metadata details
type MetadataDetails struct {
	Creator    string      `json:"creator"`
	Attributes []Attribute `json:"attributes"`
}

// Struct to represent metadata with asset information
type Metadata struct {
	ID       string          `json:"id"`
	Metadata MetadataDetails `json:"metadata"`
	IPFSUrl  string          `json:"IPFSUrl"`
	AssetId  string          `json:"AssetId"`
}

// Struct to represent individual NFT data item
type NFTData struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	TokenId           string   `json:"tokenId"`
	Description       string   `json:"description"`
	CollectionAddress string   `json:"collectionAddress"`
	Media             Media    `json:"media"`
	Metadata          Metadata `json:"metadata"`
	OffChainBalance   string   `json:"offChainBalance"`
	OnChainBalance    string   `json:"onChainBalance"`
	OwnerAddress      string   `json:"ownerAddress"`
	Type              string   `json:"type"`
}

// Struct to represent pagination information
type Paging struct {
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
	HasNext bool `json:"hasNext"`
}

// Main struct to represent the NFT response
type NFTResponse struct {
	Data   []NFTData `json:"data"`
	Paging Paging    `json:"paging"`
}

type NFTQueryParams struct {
	CollectionAddress string   `json:"collectionAddress" required:"true"`
	Mode              string   `json:"mode"`
	Page              string   `json:"page"`
	Limit             string   `json:"limit"`
	OwnerAddress      string   `json:"ownerAddress"`
	TokenIds          []string `json:"tokenIds"`
}

func buildQueryParams(v interface{}) (map[string]string, error) {
	params := make(map[string]string)

	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Struct {
		return nil, errors.New("buildQueryParams: expected a struct")
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		tag := field.Tag.Get("json")
		required := field.Tag.Get("required") == "true"
		value := val.Field(i).Interface()

		// Handle required fields
		if required && (value == "" || (reflect.TypeOf(value).Kind() == reflect.Slice && reflect.ValueOf(value).Len() == 0)) {
			return nil, fmt.Errorf("field %s is required", field.Name)
		}

		// Add non-empty fields to params
		switch v := value.(type) {
		case string:
			if v != "" {
				params[tag] = v
			}
		case []string:
			for _, id := range v {
				// Add each item in the slice as a separate entry with the same key
				params[tag+"[]"] = id
			}
		}
	}

	return params, nil
}

func GetNFTs(ctx context.Context, params runtime.NFTQueryParams, config Config) (*runtime.NFTResponse, error) {
	// Build query parameters from struct
	if params.CollectionAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "CollectionAddress is required.")
	}
	queryParams, err := buildQueryParams(params)
	if err != nil {
		return nil, err
	}
	baseUrl := config.GetLayerGCoreConfig().URL
	endpoint := baseUrl + "/api/nft"

	// Execute GET request and unmarshal response
	var response runtime.NFTResponse
	err = http.GET(ctx, endpoint, "", queryParams, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get NFT details: %w", err)
	}

	return &response, nil
}

func GetCollectionAsset(ctx context.Context, params runtime.CollectionAssetQueryParams, token string, config Config) (*runtime.CollectionAssetResponse, error) {
	// Build query parameters from struct
	if params.CollectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "CollectionId is required.")
	}
	queryParams, err := buildQueryParams(params)
	if err != nil {
		return nil, err
	}
	baseUrl := config.GetLayerGCoreConfig().URL
	endpoint := baseUrl + "/api/assets"

	// Execute GET request and unmarshal response
	var response runtime.CollectionAssetResponse
	err = http.GET(ctx, endpoint, token, queryParams, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get NFT details: %w", err)
	}
	fmt.Print(response)
	return &response, nil
}

// Struct for the POST request body
type CreateAssetNFTRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	TokenId      string `json:"tokenId"`
	CollectionId string `json:"collectionId"`
	Quantity     string `json:"quantity,omitempty"`
	Media        struct {
		S3Url string `json:"S3Url"`
	} `json:"media"`
	Metadata struct {
		Metadata struct {
			Creator    string      `json:"creator"`
			Attributes []Attribute `json:"attributes"`
		} `json:"metadata"`
	} `json:"metadata"`
}

func CreateAssetNFT(ctx context.Context, token string, request CreateAssetNFTRequest, config Config) (*CreateAssetNFTResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().URL
	endpoint := baseUrl + "/api/asset-nft/create"

	var response CreateAssetNFTResponse
	err := http.POST(ctx, endpoint, token, "", request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset NFT: %w", err)
	}

	return &response, nil
}

// Common struct for NFT transaction details
type NFTTransactionDetails struct {
	NFTId   string `json:"nftId"`
	Amount  string `json:"amount"`
	UAToken string `json:"UAToken"`
}

// Struct for the Transfer NFT request
type TransferNFTRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
	NFTTransactionDetails
}

// Struct for the Mint NFT request
type MintNFTRequest struct {
	Recipient string `json:"recipient"`
	NFTTransactionDetails
}

func TransferNFT(ctx context.Context, token string, request TransferNFTRequest, config Config) error {
	baseUrl := config.GetLayerGCoreConfig().URL

	endpoint := baseUrl + "/api/transaction/transfer-nft"

	err := http.POST(ctx, endpoint, token, "", request, nil)
	if err != nil {
		return fmt.Errorf("failed to transfer NFT: %w", err)
	}

	return nil
}

func MintNFT(ctx context.Context, token string, request MintNFTRequest, config Config) error {
	baseUrl := config.GetLayerGCoreConfig().URL
	endpoint := baseUrl + "/api/transaction/mint-nft"

	err := http.POST(ctx, endpoint, token, "", request, nil)
	if err != nil {
		return fmt.Errorf("failed to mint NFT: %w", err)
	}

	return nil
}
