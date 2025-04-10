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

func (s *ApiServer) SessionRefresh(ctx context.Context, in *api.SessionRefreshRequest) (*api.Session, error) {
	// Before hook.
	if fn := s.runtime.BeforeSessionRefresh(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, "", "", nil, 0, clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)))
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

	if in.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "Refresh token is required.")
	}

	userID, username, vars, tokenId, err := SessionRefresh(ctx, s.logger, s.db, s.config, s.sessionCache, in.Token)
	if err != nil {
		s.logger.Error("Error refreshing session.", zap.Error(err))
		return nil, err
	}

	uaToken, _ := s.tokenPairCache.Get(userID.String())

	refreshResponse, err := RefreshUAToken(ctx, uaToken.RefreshToken, s.config)
	if err != nil {
		s.logger.Error("Error refreshing UA token.", zap.Error(err))
		return nil, err
	}
	s.tokenPairCache.Update(userID.String(), refreshResponse.Data.AccessToken, refreshResponse.Data.RefreshToken, refreshResponse.Data.AccessTokenExpire, refreshResponse.Data.RefreshTokenExpire)

	// Use updated vars if they are provided, otherwise use existing ones from refresh token.
	useVars := in.Vars
	if useVars == nil {
		useVars = vars
	}
	userIDStr := userID.String()

	//newTokenId := uuid.Must(uuid.NewV4()).String()
	//token, tokenExp := generateToken(s.config, newTokenId, userIDStr, username, useVars)
	//refreshToken, refreshTokenExp := generateRefreshToken(s.config, newTokenId, userIDStr, username, useVars)
	//s.sessionCache.Remove(userID, tokenExp, "", refreshTokenExp, tokenId)
	//s.sessionCache.Add(userID, tokenExp, newTokenId, refreshTokenExp, newTokenId)
	//session := &api.Session{Created: false, Token: token, RefreshToken: refreshToken}

	// token, tokenExp := generateToken(s.config, tokenId, userIDStr, username, useVars)
	// tokenExp, err := time.Parse(time.RFC3339Nano, uaResponse.AccessTokenExpires)
	// if err != nil {
	// 	return nil, err
	// }
	// refreshExp, err := time.Parse(time.RFC3339Nano, uaResponse.RefreshTokenExpires)
	// if err != nil {
	// 	return nil, err
	// }

	// refreshToken, refreshTokenExp := generateRefreshToken(s.config, tokenId, userIDStr, username, useVars)
	// s.sessionCache.Add(userID, tokenExp.Unix(), tokenId, refreshExp.Unix(), tokenId)
	// session := &api.Session{Created: false, Token: uaResponse.AccessToken, RefreshToken: uaResponse.RefreshToken}

	token, tokenExp := generateToken(s.config, tokenId, userIDStr, username, useVars)
	refreshToken, refreshTokenExp := generateRefreshToken(s.config, tokenId, userIDStr, username, useVars)
	s.sessionCache.Add(userID, tokenExp, tokenId, refreshTokenExp, tokenId)
	session := &api.Session{Created: false, Token: token, RefreshToken: refreshToken}

	// After hook.
	if fn := s.runtime.AfterSessionRefresh(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, userIDStr, username, useVars, tokenExp, clientIP, clientPort, session, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}
	return session, nil
}

func (s *ApiServer) SessionLogout(ctx context.Context, in *api.SessionLogoutRequest) (*emptypb.Empty, error) {
	userID := ctx.Value(ctxUserIDKey{}).(uuid.UUID)

	// Before hook.
	if fn := s.runtime.BeforeSessionLogout(); fn != nil {
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

	if err := SessionLogout(s.config, s.sessionCache, userID, in.Token, in.RefreshToken); err != nil {
		if err == ErrSessionTokenInvalid {
			return nil, status.Error(codes.InvalidArgument, "Session token invalid.")
		}
		if err == ErrRefreshTokenInvalid {
			return nil, status.Error(codes.InvalidArgument, "Refresh token invalid.")
		}
		s.logger.Error("Error processing session logout.", zap.Error(err))
		return nil, status.Error(codes.Internal, "Error processing session logout.")
	}

	s.tokenPairCache.Remove(userID.String())

	// After hook.
	if fn := s.runtime.AfterSessionLogout(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, userID.String(), ctx.Value(ctxUsernameKey{}).(string), ctx.Value(ctxVarsKey{}).(map[string]string), ctx.Value(ctxExpiryKey{}).(int64), clientIP, clientPort, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}

	return &emptypb.Empty{}, nil
}
