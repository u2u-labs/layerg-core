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
	err := http.POST(ctx, endpoint, "", request, &response)
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

	var response masterdb.Add721Asset
	err := http.POST(ctx, endpoint, "", request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to add ERC721 asset: %w", err)
	}

	return &response, nil
}

func AddERC1155Asset(ctx context.Context, request masterdb.Add1155Asset, logger *zap.Logger) (*masterdb.Add1155Asset, error) {
	config := NewConfig(logger)
	baseUrl := config.GetLayerGCoreConfig().MasterDB
	endpoint := baseUrl + "/asset/erc-1155"

	var response masterdb.Add1155Asset
	err := http.POST(ctx, endpoint, "", request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to add ERC1155 asset: %w", err)
	}

	return &response, nil
}

func AddERC20Asset(ctx context.Context, request masterdb.Add20Asset, logger *zap.Logger) (*masterdb.Add20Asset, error) {
	config := NewConfig(logger)
	baseUrl := config.GetLayerGCoreConfig().MasterDB
	endpoint := baseUrl + "/asset/erc-20"

	var response masterdb.Add20Asset
	err := http.POST(ctx, endpoint, "", request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to add ERC20 asset: %w", err)
	}

	return &response, nil
}
