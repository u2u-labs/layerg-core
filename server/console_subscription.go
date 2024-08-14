package server

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/u2u-labs/go-layerg-common/api"
	"github.com/u2u-labs/layerg-core/console"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ConsoleServer) ListSubscriptions(ctx context.Context, in *console.ListSubscriptionsRequest) (*api.SubscriptionList, error) {
	if in.UserId != "" {
		_, err := uuid.FromString(in.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "expects a valid user ID filter")
		}
	}

	if in.Limit < 1 || in.Limit > 100 {
		return nil, status.Error(codes.InvalidArgument, "expects a limit value between 1 and 100")
	}

	subscriptions, err := ListSubscriptions(ctx, s.logger, s.db, in.UserId, int(in.Limit), in.Cursor)
	if err != nil {
		s.logger.Error("Failed to list subscriptions", zap.Error(err))
		return nil, status.Error(codes.Internal, "Error listing purchases.")
	}

	return subscriptions, nil
}

func (s *ConsoleServer) GetSubscription(ctx context.Context, in *console.GetSubscriptionRequest) (*api.ValidatedSubscription, error) {
	if in.GetOriginalTransactionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "original transaction id is required")
	}

	subscription, err := getSubscriptionByOriginalTransactionId(ctx, s.logger, s.db, in.GetOriginalTransactionId())
	if err != nil || subscription == nil {
		return nil, status.Error(codes.NotFound, "subscription not found")
	}

	return subscription, nil
}
