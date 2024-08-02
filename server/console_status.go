package server

import (
	"context"
	"time"

	"github.com/u2u-labs/layerg-core/console"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *ConsoleServer) GetStatus(ctx context.Context, in *emptypb.Empty) (*console.StatusList, error) {
	nodes, err := s.statusHandler.GetStatus(ctx)
	if err != nil {
		s.logger.Error("Error getting status.", zap.Error(err))
		return nil, status.Error(codes.Internal, "An error occurred while getting status.")
	}

	return &console.StatusList{
		Nodes:     nodes,
		Timestamp: &timestamppb.Timestamp{Seconds: time.Now().UTC().Unix()},
	}, nil
}
