package server

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/u2u-labs/go-layerg-common/api"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *ApiServer) ListNotifications(ctx context.Context, in *api.ListNotificationsRequest) (*api.NotificationList, error) {
	userID := ctx.Value(ctxUserIDKey{}).(uuid.UUID)

	// Before hook.
	if fn := s.runtime.BeforeListNotifications(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, userID.String(), ctx.Value(ctxUsernameKey{}).(string), ctx.Value(ctxVarsKey{}).(map[string]string), ctx.Value(ctxExpiryKey{}).(int64), clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)), zap.String("uid", userID.String()))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	limit := 1
	if in.GetLimit() != nil {
		if in.GetLimit().Value < 1 || in.GetLimit().Value > 100 {
			return nil, status.Error(codes.InvalidArgument, "Invalid limit - limit must be between 1 and 100.")
		}
		limit = int(in.GetLimit().Value)
	}

	notificationList, err := NotificationList(ctx, s.logger, s.db, userID, limit, in.CacheableCursor, true)
	if err != nil {
		return nil, status.Error(codes.Internal, "Error retrieving notifications.")
	}

	// After hook.
	if fn := s.runtime.AfterListNotifications(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, userID.String(), ctx.Value(ctxUsernameKey{}).(string), ctx.Value(ctxVarsKey{}).(map[string]string), ctx.Value(ctxExpiryKey{}).(int64), clientIP, clientPort, notificationList, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}

	return notificationList, nil
}

func (s *ApiServer) DeleteNotifications(ctx context.Context, in *api.DeleteNotificationsRequest) (*emptypb.Empty, error) {
	userID := ctx.Value(ctxUserIDKey{}).(uuid.UUID)

	// Before hook.
	if fn := s.runtime.BeforeDeleteNotifications(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, ctx.Value(ctxUserIDKey{}).(uuid.UUID).String(), ctx.Value(ctxUsernameKey{}).(string), ctx.Value(ctxVarsKey{}).(map[string]string), ctx.Value(ctxExpiryKey{}).(int64), clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)), zap.String("uid", userID.String()))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if len(in.GetIds()) == 0 {
		return &emptypb.Empty{}, nil
	}

	if err := NotificationDelete(ctx, s.logger, s.db, userID, in.GetIds()); err != nil {
		return nil, status.Error(codes.Internal, "Error while deleting notifications.")
	}

	// After hook.
	if fn := s.runtime.AfterDeleteNotifications(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, ctx.Value(ctxUserIDKey{}).(uuid.UUID).String(), ctx.Value(ctxUsernameKey{}).(string), ctx.Value(ctxVarsKey{}).(map[string]string), ctx.Value(ctxExpiryKey{}).(int64), clientIP, clientPort, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}

	return &emptypb.Empty{}, nil
}
