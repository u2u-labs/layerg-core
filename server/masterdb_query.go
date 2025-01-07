package server

import (
	"context"
	"fmt"

	"github.com/u2u-labs/go-layerg-common/masterdb"
	"github.com/u2u-labs/layerg-core/console"
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
	err := POST(ctx, endpoint, "", request, &response)
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
		s.logger.Error("failed to create collection", zap.Error(err), zap.String("collection address", in.CollectionAddress))
	}
	return &emptypb.Empty{}, nil

}
