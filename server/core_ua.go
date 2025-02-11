package server

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/u2u-labs/go-layerg-common/runtime"
	"github.com/u2u-labs/layerg-core/server/http"
)

type TelegramOTPRequest struct {
	TelegramID string `json:"telegramId"`
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

	headers, err := GetUAAuthHeaders(config)
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
			UserID             string `json:"userId"`
		} `json:"rs"`
		AAWallet struct {
			AAAddress      string `json:"aaAddress"`
			OwnerAddress   string `json:"ownerAddress"`
			FactoryAddress string `json:"factoryAddress"`
			UserID         string `json:"userId"`
			ChainID        string `json:"chainId"`
			IsDeployed     bool   `json:"isDeployed"`
			CreatedAt      string `json:"createdAt"`
			UpdatedAt      string `json:"updatedAt"`
			ID             string `json:"id"`
		} `json:"aaWalelt"`
	} `json:"data"`
	Message string `json:"message"`
}

type TelegramLoginRequest struct {
	TelegramID string `json:"telegramId"`
	ChainID    int    `json:"chainId"`
	Username   string `json:"username"`
	Firstname  string `json:"firstname"`
	Lastname   string `json:"lastname"`
	AvatarURL  string `json:"avatarUrl"`
	OTP        string `json:"otp"`
}

func TelegramLogin(ctx context.Context, token string, request TelegramLoginRequest, config Config) (*TelegramLoginResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/telegram-login"

	headers, err := GetUAAuthHeaders(config)
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

func SendUAOnchainTX(ctx context.Context, token string, request runtime.UATransactionRequest, config Config) (*runtime.ReceiptResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/onchain/send"
	headers, err := GetUAAuthHeaders(config)
	if err != nil {
		return nil, err
	}
	var response runtime.UATransactionResponse
	err = http.POST(ctx, endpoint, token, "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset NFT: %w", err)
	}

	endpoint = baseUrl + "/onchain/user-op-receipt/" + strconv.Itoa(request.ChainID) + "/" + response.Data.UserOpHash
	fmt.Println("endpoint", endpoint)

	// Implement polling with timeout
	maxAttempts := 10 // Try up to 10 times
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		time.Sleep(5 * time.Second) // Wait 5 seconds between attempts

		var receiptResponse runtime.ReceiptResponse
		err = http.GET(ctx, endpoint, token, nil, &receiptResponse)
		if err != nil {
			fmt.Printf("Error getting receipt (attempt %d/%d): %v\n", attempt, maxAttempts, err)
			continue
		}

		fmt.Printf("Receipt Response (attempt %d/%d): %+v\n", attempt, maxAttempts, receiptResponse)

		// Check if we have actual transaction data
		if receiptResponse.Data.Success && receiptResponse.Data.UserOpHash != "" {
			return &receiptResponse, nil
		}

		fmt.Printf("Transaction not ready yet (attempt %d/%d), waiting...\n", attempt, maxAttempts)
	}

	return nil, fmt.Errorf("transaction receipt not available after %d attempts", maxAttempts)
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

func GetUAAuthHeaders(config Config) (map[string]string, error) {
	timestamp := time.Now().UnixMilli()
	signature, err := CreateSignature(timestamp, config.GetLayerGCoreConfig().UADomain, config.GetLayerGCoreConfig().UAPublicApiKey, config.GetLayerGCoreConfig().UAPrivateApiKey)
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
