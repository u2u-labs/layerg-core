package server

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/u2u-labs/layerg-core/server/http"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrSessionTokenInvalid = errors.New("session token invalid")
	ErrRefreshTokenInvalid = errors.New("refresh token invalid")
)

type RefreshUATokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}
type RefreshUATokenResponse struct {
	RefreshToken        string `json:"refresh_token"`
	AccessToken         string `json:"access_token"`
	AccessTokenExpires  string `json:"access_token_expires_at"`
	RefreshTokenExpires string `json:"refresh_token_expires_at"`
}

func SessionRefresh(ctx context.Context, logger *zap.Logger, db *sql.DB, config Config, sessionCache SessionCache, token string) (uuid.UUID, string, map[string]string, string, RefreshUATokenResponse, error) {

	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/ua/refresh-token"
	request := &RefreshUATokenRequest{
		RefreshToken: token,
	}
	var response RefreshUATokenResponse
	err := http.POST(ctx, endpoint, "", "", request, &response)
	if err != nil {
		// return uuid.Nil, "", nil, "", status.Error(codes.Aborted, "Refresh token fetch failed")
		// TODO: call refresh token api UA, if đéo valid thì đi tiếp flow dưới
		logger.Error("hello ", zap.Error(err))

		userID, _, vars, exp, tokenId, ok := parseToken([]byte(config.GetSession().RefreshEncryptionKey), token)
		if !ok {
			return uuid.Nil, "", nil, "", RefreshUATokenResponse{}, status.Error(codes.Unauthenticated, "Refresh token invalid or expired.")
		}
		if !sessionCache.IsValidRefresh(userID, exp, tokenId) {
			return uuid.Nil, "", nil, "", RefreshUATokenResponse{}, status.Error(codes.Unauthenticated, "Refresh token invalid or expired.")
		}
		query := "SELECT username, disable_time FROM users WHERE id = $1 LIMIT 1"
		var dbUsername string
		var dbDisableTime pgtype.Timestamptz
		err = db.QueryRowContext(ctx, query, userID).Scan(&dbUsername, &dbDisableTime)
		if err != nil {
			if err == sql.ErrNoRows {
				// Account not found and creation is never allowed for this type.
				return uuid.Nil, "", nil, "", RefreshUATokenResponse{}, status.Error(codes.NotFound, "User account not found.")
			}
			logger.Error("Error looking up user by ID.", zap.Error(err), zap.String("id", userID.String()))
			return uuid.Nil, "", nil, "", RefreshUATokenResponse{}, status.Error(codes.Internal, "Error finding user account.")
		}

		// Check if it's disabled.
		if dbDisableTime.Valid && dbDisableTime.Time.Unix() != 0 {
			logger.Info("User account is disabled.", zap.String("id", userID.String()))
			return uuid.Nil, "", nil, "", RefreshUATokenResponse{}, status.Error(codes.PermissionDenied, "User account banned.")
		}

		return userID, dbUsername, vars, tokenId, RefreshUATokenResponse{}, nil
	}
	userID, _, _, tokenID, valid := validateJWT(config.GetConsole().PublicKey, response.AccessToken)
	if !valid {
		return uuid.Nil, "", nil, "", RefreshUATokenResponse{}, status.Error(codes.PermissionDenied, "User account banned.")
	}
	// Look for an existing account.
	query := "SELECT username, disable_time FROM users WHERE id = $1 LIMIT 1"
	var dbUsername string
	var dbDisableTime pgtype.Timestamptz
	err = db.QueryRowContext(ctx, query, userID).Scan(&dbUsername, &dbDisableTime)
	if err != nil {
		if err == sql.ErrNoRows {
			// Account not found and creation is never allowed for this type.
			return uuid.Nil, "", nil, "", RefreshUATokenResponse{}, status.Error(codes.NotFound, "User account not found.")
		}
		logger.Error("Error looking up user by ID.", zap.Error(err), zap.String("id", userID.String()))
		return uuid.Nil, "", nil, "", RefreshUATokenResponse{}, status.Error(codes.Internal, "Error finding user account.")
	}

	// Check if it's disabled.
	if dbDisableTime.Valid && dbDisableTime.Time.Unix() != 0 {
		logger.Info("User account is disabled.", zap.String("id", userID.String()))
		return uuid.Nil, "", nil, "", RefreshUATokenResponse{}, status.Error(codes.PermissionDenied, "User account banned.")
	}
	logger.Info("new refresh: ", zap.String("token", response.RefreshToken))

	return userID, dbUsername, make(map[string]string), tokenID, response, nil // I want return that struct here

}

func SessionLogout(config Config, sessionCache SessionCache, userID uuid.UUID, token, refreshToken string) error {
	var maybeSessionExp int64
	var maybeSessionTokenId string
	if token != "" {
		var sessionUserID uuid.UUID
		var ok bool
		sessionUserID, _, _, maybeSessionExp, maybeSessionTokenId, ok = parseToken([]byte(config.GetSession().EncryptionKey), token)
		if !ok || sessionUserID != userID {
			return ErrSessionTokenInvalid
		}
	}

	var maybeRefreshExp int64
	var maybeRefreshTokenId string
	if refreshToken != "" {
		var refreshUserID uuid.UUID
		var ok bool
		refreshUserID, _, _, maybeRefreshExp, maybeRefreshTokenId, ok = parseToken([]byte(config.GetSession().RefreshEncryptionKey), refreshToken)
		if !ok || refreshUserID != userID {
			return ErrRefreshTokenInvalid
		}
	}

	if maybeSessionTokenId == "" && maybeRefreshTokenId == "" {
		sessionCache.RemoveAll(userID)
		return nil
	}

	sessionCache.Remove(userID, maybeSessionExp, maybeSessionTokenId, maybeRefreshExp, maybeRefreshTokenId)
	return nil
}
