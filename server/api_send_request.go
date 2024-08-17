package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/u2u-labs/go-layerg-common/api"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthRequest represents the request body for the authentication API.
type AuthRequest struct {
	Email    string            `json:"email"`
	Password string            `json:"password"`
	Vars     map[string]string `json:"vars"`
}

// AuthSuccessResponse represents the successful response from the authentication API.
type AuthSuccessResponse struct {
	Created      bool   `json:"created"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthErrorResponse represents the error response from the authentication API.
type AuthErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details []struct {
		Type string `json:"@type"`
	} `json:"details"`
}

// APIRequest represents the structure for making an API request.
type APIRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    interface{}
}

// APIResponse represents the structure for the API response.
type APIResponse struct {
	StatusCode int
	Body       []byte
}

// MakeAPIRequest is a general-purpose function for making HTTP requests.
func MakeAPIRequest(req APIRequest) (*APIResponse, error) {
	// Marshal the request body to JSON if it's not nil
	var jsonBody []byte
	var err error
	if req.Body != nil {
		jsonBody, err = json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
	}

	// Create the HTTP request
	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Return the API response
	return &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
	}, nil
}

// AuthenticateUser wraps the general API call for email authentication
func AuthenticateUser(email, password string, vars map[string]string) (string, string, error) {
	authReq := AuthRequest{
		Email:    email,
		Password: password,
		Vars:     vars,
	}

	apiReq := APIRequest{
		URL:    "http://localhost:8350/v2/account/authenticate/email",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Basic ZGVmYXVsdGtleTo=",
		},
		Body: authReq,
	}

	apiResp, err := MakeAPIRequest(apiReq)
	if err != nil {
		return "", "", fmt.Errorf("error making API request: %v", err)
	}

	if apiResp.StatusCode >= 200 && apiResp.StatusCode < 300 {
		var successResp AuthSuccessResponse

		if err := json.Unmarshal(apiResp.Body, &successResp); err != nil {
			return "", "", fmt.Errorf("failed to unmarshal success response: %v", err)
		}
		fmt.Print(successResp)
		return successResp.Token, successResp.RefreshToken, nil
	} else {
		var errorResp AuthErrorResponse
		if err := json.Unmarshal(apiResp.Body, &errorResp); err != nil {
			return "", "", fmt.Errorf("failed to unmarshal error response: %v", err)
		}
		return "", "", fmt.Errorf("authentication failed: %s (code: %d)", errorResp.Message, errorResp.Code)
	}
}

func parseTimestamp(s string) (*timestamppb.Timestamp, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return timestamppb.New(t), nil
}

func GetAccountRequest(authorization string) (*api.Account, error) {

	apiReq := APIRequest{
		URL:    "http://localhost:8350/v2/account",
		Method: "GET",
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": authorization,
		},
	}

	apiResp, err := MakeAPIRequest(apiReq)
	if err != nil {
		return nil, fmt.Errorf("error making API request: %v", err)
	}

	if apiResp.StatusCode >= 200 && apiResp.StatusCode < 300 {
		var respData struct {
			User struct {
				Id                    string `json:"id"`
				Username              string `json:"username"`
				DisplayName           string `json:"displayName"`
				AvatarUrl             string `json:"avatarUrl"`
				LangTag               string `json:"langTag"`
				Location              string `json:"location"`
				Timezone              string `json:"timezone"`
				Metadata              string `json:"metadata"`
				FacebookId            string `json:"facebookId"`
				GoogleId              string `json:"googleId"`
				GamecenterId          string `json:"gamecenterId"`
				SteamId               string `json:"steamId"`
				Online                bool   `json:"online"`
				EdgeCount             int32  `json:"edgeCount"`
				CreateTime            string `json:"create_time"`
				UpdateTime            string `json:"update_time"`
				FacebookInstantGameId string `json:"facebookInstantGameId"`
				AppleId               string `json:"appleId"`
			} `json:"user"`
			Wallet      string `json:"wallet"`
			Email       string `json:"email"`
			CustomId    string `json:"customId"`
			VerifyTime  string `json:"verify_time"`
			DisableTime string `json:"disable_time"`
		}
		// var successResp api.Account
		if err := json.Unmarshal(apiResp.Body, &respData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal success response: %v", err)
		}
		// fmt.Print(respData)
		// Parse timestamps with optional handling
		var createTime, updateTime, verifyTime, disableTime *timestamppb.Timestamp
		if respData.User.CreateTime != "" {
			createTime, err = parseTimestamp(respData.User.CreateTime)
			if err != nil {
				return nil, fmt.Errorf("failed to parse create time: %v", err)
			}
		}
		if respData.User.UpdateTime != "" {
			updateTime, err = parseTimestamp(respData.User.UpdateTime)
			if err != nil {
				return nil, fmt.Errorf("failed to parse update time: %v", err)
			}
		}
		if respData.VerifyTime != "" {
			verifyTime, err = parseTimestamp(respData.VerifyTime)
			if err != nil {
				return nil, fmt.Errorf("failed to parse verify time: %v", err)
			}
		}
		if respData.DisableTime != "" {
			disableTime, err = parseTimestamp(respData.DisableTime)
			if err != nil {
				return nil, fmt.Errorf("failed to parse disable time: %v", err)
			}
		}
		account := &api.Account{
			User: &api.User{
				Id:                    respData.User.Id,
				Username:              respData.User.Username,
				DisplayName:           respData.User.DisplayName,
				AvatarUrl:             respData.User.AvatarUrl,
				LangTag:               respData.User.LangTag,
				Location:              respData.User.Location,
				Timezone:              respData.User.Timezone,
				Metadata:              respData.User.Metadata,
				AppleId:               respData.User.AppleId,
				FacebookId:            respData.User.FacebookId,
				FacebookInstantGameId: respData.User.FacebookInstantGameId,
				GoogleId:              respData.User.GoogleId,
				GamecenterId:          respData.User.GamecenterId,
				SteamId:               respData.User.SteamId,
				EdgeCount:             respData.User.EdgeCount,
				CreateTime:            createTime,
				UpdateTime:            updateTime,
				Online:                respData.User.Online,
			},
			Wallet:      respData.Wallet,
			Email:       respData.Email,
			CustomId:    respData.CustomId,
			VerifyTime:  verifyTime,
			DisableTime: disableTime,
		}
		fmt.Print(account)
		return account, nil
	} else {
		var errorResp AuthErrorResponse
		if err := json.Unmarshal(apiResp.Body, &errorResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal error response: %v", err)
		}
		return nil, fmt.Errorf("authentication failed: %s (code: %d)", errorResp.Message, errorResp.Code)
	}
}
