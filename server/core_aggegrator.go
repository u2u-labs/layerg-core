package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/u2u-labs/go-layerg-common/runtime"
	"github.com/u2u-labs/layerg-core/server/http"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoginRequest struct {
	ApiKey   string `json:"apiKey"`
	ApiKeyID string `json:"apiKeyID"`
}

type LoginResponse struct {
	AccessToken        string `json:"accessToken"`
	AccessTokenExpire  int64  `json:"accessTokenExpire"`
	RefreshToken       string `json:"refreshToken"`
	RefreshTokenExpire int64  `json:"refreshTokenExpire"`
	ApiKeyID           string `json:"apiKeyID"`
	ProjectId          string `json:"projectId"`
	Type               string `json:"type"`
}

type Claims struct {
	ID     string `json:"id"`
	Exp    int64  `json:"exp"`
	ApiKey string `json:"apiKey"`
	jwt.StandardClaims
}

func LoginAndCacheToken(ctx context.Context, config Config, activeCache *TokenPairCache) error {
	apiKey := config.GetLayerGCoreConfig().HubApiKey
	apiKeyID := config.GetLayerGCoreConfig().HubApiKeyID
	loginURL := config.GetLayerGCoreConfig().HubURL

	// Create the login request
	loginReq := LoginRequest{
		ApiKey:   apiKey,
		ApiKeyID: apiKeyID,
	}

	var loginResponse LoginResponse
	err := http.POST(ctx, loginURL+"/auth/login", "", "", nil, loginReq, &loginResponse)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	if loginResponse.AccessToken == "" {
		return errors.New("failed to login: empty access token received")
	}

	activeCache.Add("00000000-0000-0000-0000-000000000000", loginResponse.AccessToken, loginResponse.RefreshToken, loginResponse.AccessTokenExpire, loginResponse.RefreshTokenExpire)
	return nil
}

func decodeJWT(tokenStr string) (*Claims, int64, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, &Claims{})
	if err != nil {
		return nil, 0, err
	}

	if claims, ok := token.Claims.(*Claims); ok {
		return claims, claims.Exp, nil
	}

	return nil, 0, errors.New("invalid token")
}

func GetAccessToken(ctx context.Context, tokenPairCache *TokenPairCache, config Config) (string, error) {
	defaultUuid := "00000000-0000-0000-0000-000000000000"

	tokenPair, exists := tokenPairCache.Get(defaultUuid)
	if !exists {
		// If no token exists, login to get a new one
		if err := LoginAndCacheToken(ctx, config, tokenPairCache); err != nil {
			return "", fmt.Errorf("failed to login: %w", err)
		}
		tokenPair, _ = tokenPairCache.Get(defaultUuid)
	}

	// Check if the token is still valid
	if time.Now().Unix() >= tokenPair.AccessExp/1000 {
		// If expired, refresh the token
		if err := LoginAndCacheToken(ctx, config, tokenPairCache); err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}
		tokenPair, _ = tokenPairCache.Get(defaultUuid)
	}

	if tokenPair.AccessToken == "" {
		return "", status.Error(codes.NotFound, "Access token not found.")
	}

	return tokenPair.AccessToken, nil
}

func AssetCreate(ctx context.Context, tokenPairCache *TokenPairCache, config Config, args runtime.CreateNFTArgs) error {
	token, err := GetAccessToken(ctx, tokenPairCache, config)
	if err != nil {
		return err
	}

	endpoint := config.GetLayerGCoreConfig().HubURL + "/assets/create"

	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	var response interface{}
	err = http.POST(ctx, endpoint, token, "", nil, args, &response)
	if err != nil {
		return fmt.Errorf("failed to create asset: %w", err)
	}

	return nil
}
