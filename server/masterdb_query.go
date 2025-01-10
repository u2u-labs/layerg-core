package server

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/u2u-labs/go-layerg-common/masterdb"
	"github.com/u2u-labs/layerg-core/console"
	crawlerQuery "github.com/u2u-labs/layerg-core/server/crawler/crawler_query"
	"github.com/u2u-labs/layerg-core/server/crawler/utils"
	"github.com/u2u-labs/layerg-core/server/crawler/utils/models"
	"github.com/u2u-labs/layerg-core/server/http"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CollectionRequest struct {
	CollectionAddress string                  `json:"collectionAddress" required:"true"`
	Type              masterdb.CollectionType `json:"type" required:"true"`
}

func ProduceSignature(requestParamFrom any, config Config) (string, error) {
	// Convert requestParamFrom to bytes
	dataBytes, err := masterdb.ConvertToBytes(requestParamFrom)
	if err != nil {
		return "", fmt.Errorf("failed to convert request to bytes: %v", err)
	}

	pvk := config.GetLayerGCoreConfig().MasterPvk
	// Sign the data using the private key
	signature, err := masterdb.SignECDSA(pvk, dataBytes)
	if err != nil {
		return "", fmt.Errorf("failed to produce signature: %v", err)
	}

	return signature, nil
}
func CreateCollection(ctx context.Context, request CollectionRequest, config Config) (*masterdb.CollectionResponse, error) {
	if request.CollectionAddress == "" {
		return nil, fmt.Errorf("collectionAddress is required")
	}
	if request.Type == "" {
		return nil, fmt.Errorf("type is required")
	}

	baseUrl := config.GetLayerGCoreConfig().MasterDB
	endpoint := baseUrl + "/chain/1/collection"

	var response masterdb.CollectionResponse
	err := http.POST(ctx, endpoint, "", "", request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	return &response, nil
}

func (s *ConsoleServer) AddNFTCollection(ctx context.Context, in *console.AddNFTCollectionRequest) (*emptypb.Empty, error) {

	request := CollectionRequest{
		CollectionAddress: in.CollectionAddress,
		Type:              masterdb.CollectionType(in.Type),
	}

	_, err := CreateCollection(ctx, request, s.config)
	if err != nil {
		s.logger.Error("failed to create collection masterdb", zap.Error(err), zap.String("collection address", in.CollectionAddress))
	}

	chain, err := crawlerQuery.GetChainById(ctx, s.db, int32(in.ChainId))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chain rpc")
	}

	latestBlockNumber, err := utils.GetLastestBlockFromChainUrl(chain.RpcUrl)
	s.logger.Info("Latest block: ", zap.Int("block", int(latestBlockNumber)))
	if err != nil {
		s.logger.Error("failed to fetch latest block from", zap.Error(err), zap.String("chain", chain.Name))
		return nil, fmt.Errorf("failed to fetch latest block")
	}

	assetID := strconv.Itoa(int(in.ChainId)) + ":" + in.CollectionAddress
	newAssetParam := models.AddNewAssetParams{
		ID:                assetID,
		ChainID:           int32(in.ChainId),
		CollectionAddress: in.CollectionAddress,
		Type:              models.AssetType(in.Type),
		InitialBlock: sql.NullInt64{
			Valid: in.InitialBlock > 0,
			Int64: int64(latestBlockNumber),
		},
		DecimalData: sql.NullInt16{
			Valid: true,
			Int16: 0,
		},
	}
	newAsset, err := crawlerQuery.AddNewAsset(ctx, s.db, newAssetParam)
	if err != nil {
		s.logger.Error("failed to create collection locally-2", zap.Error(err), zap.String("collection address", in.CollectionAddress))
	}

	backfillParams := models.AddBackfillCrawlerParams{
		ChainID:           int32(in.ChainId),
		CollectionAddress: in.CollectionAddress,
		CurrentBlock:      int64(in.InitialBlock - 100),
	}
	err = crawlerQuery.AddBackfillCrawler(ctx, s.db, backfillParams)
	if err != nil {
		s.logger.Error("failed to create collection locally-1", zap.Error(err), zap.String("collection address", in.CollectionAddress))
	}
	models.ContractType[int32(in.ChainId)][in.CollectionAddress] = newAsset

	return &emptypb.Empty{}, nil

}

func AddERC721Asset(ctx context.Context, request masterdb.Add721Asset, logger *zap.Logger) (*masterdb.Add721Asset, error) {
	config := NewConfig(logger)
	if request.Asset721.CollectionId == "" {
		return nil, fmt.Errorf("collectionId is required")
	}
	if request.Asset721.TokenId == "" {
		return nil, fmt.Errorf("tokenId is required")
	}
	if request.History.From == "" || request.History.To == "" {
		return nil, fmt.Errorf("from and to addresses are required")
	}

	baseUrl := config.GetLayerGCoreConfig().MasterDB
	endpoint := baseUrl + "/asset/erc-721"
	signature, err := ProduceSignature(request, config)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request")
	}
	var response masterdb.Add721Asset
	err = http.POST(ctx, endpoint, "", signature, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to add ERC721 asset: %w", err)
	}

	return &response, nil
}

func AddERC1155Asset(ctx context.Context, request masterdb.Add1155Asset, logger *zap.Logger) (*masterdb.Add1155Asset, error) {
	config := NewConfig(logger)
	baseUrl := config.GetLayerGCoreConfig().MasterDB
	endpoint := baseUrl + "/asset/erc-1155"
	signature, err := ProduceSignature(request, config)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request")
	}
	var response masterdb.Add1155Asset
	err = http.POST(ctx, endpoint, "", signature, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to add ERC1155 asset: %w", err)
	}

	return &response, nil
}

func AddERC20Asset(ctx context.Context, request masterdb.Add20Asset, logger *zap.Logger) (*masterdb.Add20Asset, error) {
	config := NewConfig(logger)
	baseUrl := config.GetLayerGCoreConfig().MasterDB
	endpoint := baseUrl + "/asset/erc-20"
	signature, err := ProduceSignature(request, config)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request")
	}
	var response masterdb.Add20Asset
	err = http.POST(ctx, endpoint, "", signature, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to add ERC20 asset: %w", err)
	}

	return &response, nil
}

func SubmitERC721BatchRequest(ctx context.Context, batchRequest masterdb.Add721AssetBatch, logger *zap.Logger) (*masterdb.Add721AssetBatch, error) {
	config := NewConfig(logger)
	baseUrl := config.GetLayerGCoreConfig().MasterDB
	logger.Info("base url: ", zap.String("url", baseUrl))
	endpoint := baseUrl + "/asset/erc-721-batch"
	signature, err := ProduceSignature(batchRequest, config)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request")
	}
	var response masterdb.Add721AssetBatch
	err = http.POST(ctx, endpoint, "", signature, batchRequest, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to submit ERC721 batch: %w", err)
	}

	return &response, nil
}

func SubmitERC1155BatchRequest(ctx context.Context, batchRequest masterdb.Add1155AssetBatch, logger *zap.Logger) (*masterdb.Add1155AssetBatch, error) {
	config := NewConfig(logger)
	baseUrl := config.GetLayerGCoreConfig().MasterDB
	endpoint := baseUrl + "/asset/erc-1155-batch"
	signature, err := ProduceSignature(batchRequest, config)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request")
	}
	var response masterdb.Add1155AssetBatch
	err = http.POST(ctx, endpoint, "", signature, batchRequest, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to submit ERC1155 batch: %w", err)
	}

	return &response, nil
}

func SubmitERC20BatchRequest(ctx context.Context, batchRequest masterdb.Add20AssetBatch, logger *zap.Logger) (*masterdb.Add20AssetBatch, error) {
	config := NewConfig(logger)
	baseUrl := config.GetLayerGCoreConfig().MasterDB
	endpoint := baseUrl + "/asset/erc-20-batch"
	signature, err := ProduceSignature(batchRequest, config)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request")
	}
	var response masterdb.Add20AssetBatch
	err = http.POST(ctx, endpoint, "", signature, batchRequest, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to submit ERC20 batch: %w", err)
	}

	return &response, nil
}
