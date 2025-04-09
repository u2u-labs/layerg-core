package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/u2u-labs/go-layerg-common/runtime"
	"go.uber.org/zap"
)

func (ec *MQTTRegistry) SubscribeToEvent(subscription runtime.EventSubscription, handler runtime.EventHandler, config Config) error {
	// Construct the MQTT topic based on the subscription
	topic := fmt.Sprintf("events/%d/%s/%s",
		subscription.ChainId,
		subscription.ContractAddress,
		subscription.EventName,
	)

	// Create MQTT config for this subscription
	mqttConfig := &runtime.MQTTConfig{
		BrokerURL: config.GetMqtt().BrokerURL,
		ClientID:  config.GetMqtt().ClientID,
		QoS:       config.GetMqtt().QoS,
		Topic:     topic,
	}

	return ec.RegisterMQTTSubscription(context.Background(), *mqttConfig, func(ctx context.Context, msg runtime.MQTTMessage) error {
		// Parse the message data
		var eventData runtime.EventData
		if err := json.Unmarshal([]byte(msg.Payload), &eventData); err != nil {
			ec.logger.Error("Failed to parse event data",
				zap.Error(err),
				zap.String("payload", string(msg.Payload)))
			return err
		}

		// Log the received event
		ec.logger.Info("Received blockchain event",
			zap.String("topic", topic),
			zap.Uint64("blockNumber", eventData.BlockNumber),
			zap.String("txHash", eventData.TxHash))

		// Call the custom handler with the parsed event data
		if err := handler(ctx, eventData); err != nil {
			ec.logger.Error("Error handling event",
				zap.Error(err),
				zap.String("topic", topic))
			return err
		}

		return nil
	})
}

// GetEventByTxHash retrieves event information using the API
// chainId and contractAddress are required parameters, while txHash is optional
func (ec *MQTTRegistry) GetEventByTxHash(chainId int, contractAddress string, txHash string, config Config) (*runtime.EventResponse, error) {
	// Validate required parameters
	if contractAddress == "" {
		return nil, fmt.Errorf("contractAddress is required")
	}
	if chainId <= 0 {
		return nil, fmt.Errorf("chainId must be greater than 0")
	}

	// Construct the API URL with query parameters
	apiURL := fmt.Sprintf("%s/api/v1/events", config.GetMqtt().APIurl)
	params := url.Values{}
	params.Add("contractAddress", contractAddress)
	params.Add("chainId", fmt.Sprintf("%d", chainId))

	if txHash != "" {
		params.Add("txHash", txHash)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Implement polling with max 5 attempts and 0.5 second delay
	maxAttempts := 5
	delay := 500 * time.Millisecond

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Make the HTTP request
		resp, err := client.Get(apiURL + "?" + params.Encode())
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to make API request: %w", err)
		}

		// Check response status
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
		}

		// Parse the response
		var eventResponse runtime.EventResponse
		if err := json.NewDecoder(resp.Body).Decode(&eventResponse); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode API response: %w", err)
		}

		resp.Body.Close()

		// Check if we need to retry based on empty data and txHash parameter
		if txHash != "" && len(eventResponse.Data) == 0 {
			ec.logger.Info("Empty event data received, retrying",
				zap.Int("attempt", attempt),
				zap.Int("maxAttempts", maxAttempts),
				zap.String("txHash", txHash))

			if attempt < maxAttempts {
				time.Sleep(delay)
				continue
			}
		}

		// Success - return the event response
		ec.logger.Info("Successfully retrieved event data",
			zap.Int("attempt", attempt),
			zap.String("txHash", txHash),
			zap.Int("dataLength", len(eventResponse.Data)))

		return &eventResponse, nil
	}

	// This should never be reached, but included for completeness
	return nil, fmt.Errorf("failed to retrieve event data after %d attempts", maxAttempts)
}
