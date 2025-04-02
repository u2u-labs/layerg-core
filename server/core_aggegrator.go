package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoginRequest struct {
	ApiKey   string `json:"apiKey"`
	ApiKeyID string `json:"apiKeyID"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type Claims struct {
	ID     string `json:"id"`
	Exp    int64  `json:"exp"`
	ApiKey string `json:"apiKey"`
	jwt.StandardClaims
}

// func (s *ApiServer) AuthenticateLayerGCore(ctx context.Context) error {
// }

func LoginAndCacheToken(ctx context.Context, logger *zap.Logger, config Config, activeCache ActiveTokenCache) error {
	defaultUuid := uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000")
	apiKey := config.GetLayerGCoreConfig().ApiKey
	apiKeyID := config.GetLayerGCoreConfig().ApiKeyID
	loginURL := config.GetLayerGCoreConfig().PortalURL
	reqBody, err := json.Marshal(LoginRequest{ApiKey: apiKey, ApiKeyID: apiKeyID})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", loginURL+"/api/auth/login", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)

		logger.Error("Failed to login", zap.String("responseBody", string(body)))

		return errors.New("failed to login")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return err
	}

	_, accessExp, err := decodeJWT(loginResp.AccessToken)
	if err != nil {
		return err
	}

	_, refreshExp, err := decodeJWT(loginResp.RefreshToken)
	if err != nil {
		return err
	}

	activeCache.Add(defaultUuid, 0, "", 0, "", accessExp, loginResp.AccessToken, refreshExp, loginResp.RefreshToken)

	go refreshToken(ctx, logger, config, activeCache, refreshExp)

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

func refreshToken(ctx context.Context, logger *zap.Logger, config Config, activeCache ActiveTokenCache, exp int64) {

	for {
		waitTime := time.Until(time.Unix(exp, 0))
		select {
		case <-time.After(waitTime):
			if err := LoginAndCacheToken(ctx, logger, config, activeCache); err != nil {
				fmt.Println("Failed to refresh token:", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func GetAccessToken(ctx context.Context, logger *zap.Logger, s ActiveTokenCache, config Config) (string, error) {
	defaultUuid := uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000")

	_, _, globalSessionToken, _, _, _, globalSessionExp, _ := s.GetActiveTokens(defaultUuid)

	// Determine the most recent token and its expiration
	var accessToken = globalSessionToken
	var tokenExp = globalSessionExp
	logger.Info(accessToken)
	logger.Info(strconv.Itoa(int(tokenExp)))
	// Check if the token is still valid
	if time.Now().Unix() >= tokenExp {
		// If expired, refresh the token
		if err := LoginAndCacheToken(ctx, logger, config, s); err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}

		// Re-fetch the updated token from the cache
		_, _, globalSessionToken, _, _, _, globalSessionExp, _ = s.GetActiveTokens(defaultUuid)
		accessToken = globalSessionToken
	}

	if accessToken == "" {
		return "", status.Error(codes.NotFound, "Access token not found.")
	}

	return accessToken, nil
}
