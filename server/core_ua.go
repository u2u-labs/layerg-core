package server

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/u2u-labs/go-layerg-common/runtime"
	"github.com/u2u-labs/layerg-core/server/http"
)

type TelegramOTPRequest struct {
	TelegramID string `json:"telegramId"`
	APIKey     string `json:"apiKey"`
	Domain     string `json:"domain"`
}

type TelegramOTPResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Message string `json:"message"`
	} `json:"data"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

func SendTelegramOTP(ctx context.Context, request TelegramOTPRequest, config Config) (*TelegramOTPResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/telegram-otp-request"

	headers, err := GetUAAuthHeaders(request.Domain, config)
	if err != nil {
		return nil, err
	}

	var response TelegramOTPResponse
	err = http.POST(ctx, endpoint, "", "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset NFT: %w", err)
	}

	return &response, nil
}

type TelegramLoginResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Rs struct {
			RefreshToken       string `json:"refreshToken"`
			RefreshTokenExpire int64  `json:"refreshTokenExpire"`
			AccessToken        string `json:"accessToken"`
			AccessTokenExpire  int64  `json:"accessTokenExpire"`
			UserID             int    `json:"userId"`
		} `json:"rs"`
		AAWallet struct {
			AAAddress      string `json:"aaAddress"`
			OwnerAddress   string `json:"ownerAddress"`
			FactoryAddress string `json:"factoryAddress"`
			UserID         int    `json:"userId"`
			ChainID        int    `json:"chainId"`
			IsDeployed     bool   `json:"isDeployed"`
			CreatedAt      string `json:"createdAt"`
			UpdatedAt      string `json:"updatedAt"`
			ID             int    `json:"id"`
		} `json:"aaWalelt"`
	} `json:"data"`
	Message string `json:"message"`
}

type TelegramLoginRequest struct {
	TelegramID string `json:"telegramId"`
	APIKey     string `json:"apiKey"`
	ChainID    int    `json:"chainId"`
	Username   string `json:"username"`
	Firstname  string `json:"firstname"`
	Lastname   string `json:"lastname"`
	AvatarURL  string `json:"avatarUrl"`
	OTP        string `json:"otp"`
	Domain     string `json:"domain"`
}

func TelegramLogin(ctx context.Context, token string, request TelegramLoginRequest, config Config) (*TelegramLoginResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/telegram-login"

	headers, err := GetUAAuthHeaders(request.Domain, config)
	if err != nil {
		return nil, err
	}
	var response TelegramLoginResponse
	err = http.POST(ctx, endpoint, token, "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset NFT: %w", err)
	}

	return &response, nil
}

type OnchainTransactionRequest struct {
	To                   string `json:"to"`
	Value                string `json:"value"`
	Data                 string `json:"data"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
}

type OnchainTransactionPayload struct {
	ProjectID      string                     `json:"projectId"`
	ChainID        int                        `json:"chainId"`
	Sponsor        bool                       `json:"sponsor"`
	TransactionReq *OnchainTransactionRequest `json:"transactionReq"`
}

func SendUAOnchainTX(ctx context.Context, token string, request runtime.UATransactionRequest, config Config) (*runtime.UATransactionResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/onchain/send"

	var response runtime.UATransactionResponse
	err := http.POST(ctx, endpoint, token, "", nil, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset NFT: %w", err)
	}

	return &response, nil
}

func RefreshUAToken(ctx context.Context, token string, config Config) (*runtime.UARefreshTokenResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/refresh"

	request := runtime.UARefreshTokenRequest{
		RefreshToken: token,
	}
	var response runtime.UARefreshTokenResponse
	err := http.POST(ctx, endpoint, "", "", nil, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset NFT: %w", err)
	}

	return &response, nil
}

func GetUAAuthHeaders(domain string, config Config) (map[string]string, error) {
	timestamp := time.Now().UnixMilli()
	signature, err := CreateSignature(timestamp, domain, config.GetLayerGCoreConfig().UAPublicApiKey, config.GetLayerGCoreConfig().UAPrivateApiKey)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"x-signature": signature.Signature,
		"x-timestamp": big.NewInt(signature.Timestamp).String(),
		"origin":      signature.Domain,
		"x-api-key":   config.GetLayerGCoreConfig().UAPublicApiKey,
	}
	return headers, nil
}
