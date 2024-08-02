package server

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/u2u-labs/go-layerg-common/api"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *ApiServer) GetMatchmakerStats(ctx context.Context, in *emptypb.Empty) (*api.MatchmakerStats, error) {
	userID := ctx.Value(ctxUserIDKey{}).(uuid.UUID)

	// Before hook.
	if fn := s.runtime.BeforeGetMatchmakerStats(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			err, code := fn(ctx, s.logger, userID.String(), ctx.Value(ctxUsernameKey{}).(string), ctx.Value(ctxVarsKey{}).(map[string]string), ctx.Value(ctxExpiryKey{}).(int64), clientIP, clientPort)
			if err != nil {
				return status.Error(code, err.Error())
			}
			// Empty input never overridden.
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	stats := s.matchmaker.GetStats()

	// After hook.
	if fn := s.runtime.AfterGetMatchmakerStats(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, userID.String(), ctx.Value(ctxUsernameKey{}).(string), ctx.Value(ctxVarsKey{}).(map[string]string), ctx.Value(ctxExpiryKey{}).(int64), clientIP, clientPort, stats)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}

	return stats, nil
}
