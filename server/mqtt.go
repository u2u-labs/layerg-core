package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/u2u-labs/go-layerg-common/runtime"
	"go.uber.org/zap"
)

type MQTTSubscription struct {
	Config  runtime.MQTTConfig
	Handler runtime.MQTTHandler
}

type MQTTRegistry struct {
	sync.RWMutex
	logger        *zap.Logger
	client        mqtt.Client
	subscriptions map[string]*MQTTSubscription
	done          chan struct{}
}

// NewMQTTRegistry creates a new MQTT registry
func NewMQTTRegistry(logger *zap.Logger) (*MQTTRegistry, error) {
	return &MQTTRegistry{
		logger:        logger,
		subscriptions: make(map[string]*MQTTSubscription),
		done:          make(chan struct{}),
	}, nil
}

// RegisterSubscription registers a new MQTT topic subscription with its handler
func (m *MQTTRegistry) RegisterMQTTSubscription(ctx context.Context, config runtime.MQTTConfig, handler runtime.MQTTHandler) error {
	m.Lock()
	defer m.Unlock()

	if _, exists := m.subscriptions[config.Topic]; exists {
		return fmt.Errorf("MQTT topic already registered: %s", config.Topic)
	}

	// If this is the first subscription, create a default client
	if len(m.subscriptions) == 0 {
		opts := mqtt.NewClientOptions().
			AddBroker(config.BrokerURL).
			SetClientID(config.ClientID).
			SetCleanSession(true).
			SetAutoReconnect(true).
			SetConnectTimeout(30 * time.Second).
			SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
				m.logger.Info("Reconnecting to MQTT broker...")
			}).
			SetConnectionLostHandler(func(client mqtt.Client, err error) {
				m.logger.Error("MQTT connection lost",
					zap.String("broker", config.BrokerURL),
					zap.String("topic", config.Topic),
					zap.Error(err))
			}).
			SetOnConnectHandler(func(client mqtt.Client) {
				m.logger.Info("MQTT connected to broker",
					zap.String("broker", config.BrokerURL),
					zap.String("topic", config.Topic))
			})

		m.client = mqtt.NewClient(opts)
		if token := m.client.Connect(); token.Wait() && token.Error() != nil {
			return fmt.Errorf("failed to connect to MQTT broker: %v", token.Error())
		}

		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-m.done:
					return
				case <-ticker.C:
					if !m.client.IsConnected() {
						m.logger.Info("Reconnecting to MQTT broker...")
						if token := m.client.Connect(); token.Wait() && token.Error() != nil {
							m.logger.Error("Failed to reconnect to MQTT broker", zap.Error(token.Error()))
						}
					}
				}
			}
		}()
	}

	// Subscribe to both specific topic and wildcard for debugging
	topics := []string{
		config.Topic,
		"events/#", // Subscribe to all events for debugging
	}

	for _, topic := range topics {
		token := m.client.Subscribe(topic, config.QoS, func(client mqtt.Client, msg mqtt.Message) {
			m.logger.Info("Received MQTT message",
				zap.String("topic", msg.Topic()),
				zap.ByteString("payload", msg.Payload()))

			message := runtime.MQTTMessage{
				Topic:   msg.Topic(),
				Payload: msg.Payload(),
			}

			if err := handler(ctx, message); err != nil {
				m.logger.Error("Error handling MQTT message",
					zap.String("topic", msg.Topic()),
					zap.Error(err))
			} else {
				m.logger.Info("Successfully handled MQTT message",
					zap.String("topic", msg.Topic()))
			}
		})

		if token.Wait() && token.Error() != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %v", topic, token.Error())
		}

		m.subscriptions[topic] = &MQTTSubscription{
			Config:  config,
			Handler: handler,
		}

		m.logger.Info("Registered new MQTT subscription",
			zap.String("topic", topic),
			zap.String("broker", config.BrokerURL),
			zap.Int("qos", int(config.QoS)))
	}

	return nil
}

// UnregisterSubscription removes an MQTT topic subscription
func (m *MQTTRegistry) UnregisterSubscription(ctx context.Context, topic string) error {
	m.Lock()
	defer m.Unlock()

	if _, exists := m.subscriptions[topic]; exists {
		token := m.client.Unsubscribe(topic)
		if token.Wait() && token.Error() != nil {
			return fmt.Errorf("failed to unsubscribe from topic %s: %v", topic, token.Error())
		}

		delete(m.subscriptions, topic)
		m.logger.Info("Unregistered MQTT subscription", zap.String("topic", topic))
		return nil
	}

	return fmt.Errorf("MQTT topic not found: %s", topic)
}

// PublishMessage publishes a message to an MQTT topic
// func (m *MQTTRegistry) PublishMessage(ctx context.Context, topic string, payload interface{}, qos byte) error {
// 	var message []byte
// 	var err error

// 	switch p := payload.(type) {
// 	case []byte:
// 		message = p
// 	case string:
// 		message = []byte(p)
// 	default:
// 		message, err = json.Marshal(payload)
// 		if err != nil {
// 			return fmt.Errorf("failed to marshal payload: %v", err)
// 		}
// 	}

// 	token := m.client.Publish(topic, qos, false, message)
// 	if token.Wait() && token.Error() != nil {
// 		return fmt.Errorf("failed to publish message: %v", token.Error())
// 	}

// 	return nil
// }

// Stop disconnects from the MQTT broker and cleans up resources
func (m *MQTTRegistry) Stop() {
	m.Lock()
	defer m.Unlock()

	// Signal background process to stop
	close(m.done)

	// Unsubscribe from all topics
	for topic := range m.subscriptions {
		token := m.client.Unsubscribe(topic)
		if token.Wait() && token.Error() != nil {
			m.logger.Error("Failed to unsubscribe from topic",
				zap.String("topic", topic),
				zap.Error(token.Error()))
		}
	}

	// Disconnect from the broker
	if m.client != nil && m.client.IsConnected() {
		m.client.Disconnect(250) // 250ms timeout
	}
	m.subscriptions = make(map[string]*MQTTSubscription)
	m.logger.Info("MQTT registry stopped")
}
