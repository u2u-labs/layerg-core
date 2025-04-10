package server

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/u2u-labs/go-layerg-common/api"
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
		return nil, fmt.Errorf("failed to send telegram OTP: %w", err)
	}

	return &response, nil
}

type UAOtpLoginResponse struct {
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

func TelegramLogin(ctx context.Context, token string, request TelegramLoginRequest, config Config) (*UAOtpLoginResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/telegram-login"

	headers, err := GetUAAuthHeaders(config)
	if err != nil {
		return nil, err
	}
	var response UAOtpLoginResponse
	err = http.POST(ctx, endpoint, token, "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to login with telegram: %w", err)
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

func SendUAOnchainTX(ctx context.Context, token string, waitForReceipt bool, request runtime.UATransactionRequest, config Config) (*runtime.ReceiptResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/onchain/send"
	headers, err := GetUAAuthHeaders(config)
	if err != nil {
		return nil, err
	}
	var response runtime.UATransactionResponse
	err = http.POST(ctx, endpoint, token, "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to send onchain transaction: %w", err)
	}

	endpoint = baseUrl + "/onchain/user-op-receipt/" + strconv.Itoa(request.ChainID) + "/" + response.Data.UserOpHash
	fmt.Println("endpoint", endpoint)

	// Implement polling with timeout
	if waitForReceipt {
		maxAttempts := 10 // Try up to 10 times
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			time.Sleep(2 * time.Second) // Wait 2 seconds between attempts

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

	return nil, nil
}

func RefreshUAToken(ctx context.Context, token string, config Config) (*runtime.UARefreshTokenResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/refresh"
	headers, err := GetUAAuthHeaders(config)
	if err != nil {
		return nil, err
	}
	request := runtime.UARefreshTokenRequest{
		RefreshToken: token,
	}
	var response runtime.UARefreshTokenResponse
	err = http.POST(ctx, endpoint, "", "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("Failed to refresh UA token: %w", err)
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

type UALoginCallBackRequest struct {
	Code  string `json:"code"`
	Error string `json:"error,omitempty"`
	State string `json:"state"`
}

type EmailOTPVerifyRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

type EmailSendOTPRequest struct {
	Email string `json:"email"`
}

type UALoginCallbackResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		User struct {
			Type                string    `json:"type"`
			APIKey              string    `json:"apiKey"`
			EOAWallet           string    `json:"eoaWallet"`
			EncryptedPrivateKey string    `json:"encryptedPrivateKey"`
			GoogleID            string    `json:"googleId"`
			GoogleEmail         string    `json:"googleEmail"`
			GoogleFirstName     string    `json:"googleFirstName"`
			GoogleLastName      string    `json:"googleLastName"`
			GoogleAvatarURL     string    `json:"googleAvatarUrl"`
			AAWallets           []string  `json:"aaWallets"`
			DeveloperKeys       []string  `json:"developerKeys"`
			ID                  string    `json:"id"`
			CreatedAt           time.Time `json:"createdAt"`
			UpdatedAt           time.Time `json:"updatedAt"`
			TelegramId          string    `json:"telegramId"`
			TelegramUsername    string    `json:"telegramUsername"`
			TelegramFirstName   string    `json:"telegramFirstName"`
			TelegramLastName    string    `json:"telegramLastName"`
			TelegramAvatarURL   string    `json:"telegramAvatarUrl"`
			Email               string    `json:"email"`
			FacebookId          string    `json:"facebookId"`
			FacebookEmail       string    `json:"facebookEmail"`
			FacebookFirstName   string    `json:"facebookFirstName"`
			FacebookLastName    string    `json:"facebookLastName"`
			FacebookAvatarURL   string    `json:"facebookAvataUrl"`
			TwitterId           string    `json:"twitterId"`
			TwitterEmail        string    `json:"twitterEmail"`
			TwitterFirstName    string    `json:"twitterFirstName"`
			TwitterLastName     string    `json:"twitterLastName"`
			TwitterAvatarURL    string    `json:"twitterAvatarUrl"`
		} `json:"user"`
		W struct {
			AAAddress      string    `json:"aaAddress"`
			OwnerAddress   string    `json:"ownerAddress"`
			FactoryAddress string    `json:"factoryAddress"`
			UserID         string    `json:"userId"`
			ID             string    `json:"id"`
			CreatedAt      time.Time `json:"createdAt"`
			UpdatedAt      time.Time `json:"updatedAt"`
		} `json:"w"`
		RefreshToken       string `json:"refreshToken"`
		RefreshTokenExpire int64  `json:"refreshTokenExpire"`
		AccessToken        string `json:"accessToken"`
		AccessTokenExpire  int64  `json:"accessTokenExpire"`
		UserID             string `json:"userId"`
		APIKey             string `json:"apiKey"`
	} `json:"data"`
}

func GoogleLoginCallback(ctx context.Context, token string, request UALoginCallBackRequest, config Config) (*UALoginCallbackResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/google/callback"

	headers, err := GetUAAuthHeaders(config)
	if err != nil {
		return nil, err
	}

	var response UALoginCallbackResponse
	err = http.POST(ctx, endpoint, token, "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to send google login callback: %w", err)
	}

	return &response, nil
}

func NewUALoginCallBackRequest(code, errorStr, state string) *UALoginCallBackRequest {
	return &UALoginCallBackRequest{
		Code:  code,
		Error: errorStr,
		State: state,
	}
}

func TwitterLoginCallback(ctx context.Context, token string, request UALoginCallBackRequest, config Config) (*UALoginCallbackResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/twitter/callback"

	headers, err := GetUAAuthHeaders(config)
	if err != nil {
		return nil, err
	}

	var response UALoginCallbackResponse
	err = http.POST(ctx, endpoint, token, "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to send twitter login callback: %w", err)
	}

	return &response, nil
}

type UAWeb3AuthRequest struct {
	Signature string `json:"signature"`
	Signer    string `json:"signer"`
}

func EVMAuthUA(ctx context.Context, token string, request UAWeb3AuthRequest, config Config) (*UALoginCallbackResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/web3"

	headers, err := GetUAAuthHeaders(config)
	if err != nil {
		return nil, err
	}

	var response UALoginCallbackResponse
	err = http.POST(ctx, endpoint, token, "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to send twitter login callback: %w", err)
	}

	return &response, nil
}

type UASocialLoginRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

func SocialLoginUA(ctx context.Context, token string, param *api.UASocialLoginRequest, config Config) (*runtime.UADirectLoginResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/" + param.GetSource().String() + "/callback"

	fmt.Println("endpoint callback: ", endpoint)

	headers, err := GetUAAuthHeaders(config)
	if err != nil {
		return nil, err
	}

	request := UASocialLoginRequest{
		Code:  param.Code,
		State: param.State,
	}

	fmt.Println("request: ", request)

	var response runtime.UADirectLoginResponse
	err = http.POST(ctx, endpoint, token, "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to send social login callback: %w", err)
	}

	fmt.Println("response: ", response)

	return &response, nil
}

func EmailLoginUA(ctx context.Context, request EmailOTPVerifyRequest, config Config) (*UAOtpLoginResponse, error) {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/email-otp-verify"

	headers, err := GetUAAuthHeaders(config)
	if err != nil {
		return nil, err
	}

	var response UAOtpLoginResponse
	err = http.POST(ctx, endpoint, "", "", headers, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to send email login callback: %w", err)
	}

	return &response, nil
}

func EmailUASendOTP(ctx context.Context, email string, config Config) error {
	baseUrl := config.GetLayerGCoreConfig().UniversalAccountURL
	endpoint := baseUrl + "/auth/email-otp-request"

	headers, err := GetUAAuthHeaders(config)
	if err != nil {
		return err
	}

	request := EmailSendOTPRequest{
		Email: email,
	}

	var response UALoginCallbackResponse
	err = http.POST(ctx, endpoint, "", "", headers, request, &response)
	if err != nil {
		return fmt.Errorf("failed to send email otp: %w", err)
	}

	return nil
}
