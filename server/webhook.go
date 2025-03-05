package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/u2u-labs/go-layerg-common/runtime"
	"go.uber.org/zap"
)

// type WebhookHandler func(ctx context.Context, payload map[string]interface{}) (interface{}, error)

// type WebhookConfig struct {
// 	Path        string                 `json:"path"`
// 	Method      string                 `json:"method"`
// 	Description string                 `json:"description"`
// 	Params      map[string]interface{} `json:"params"`
// }

type Webhook struct {
	Config  runtime.WebhookConfig
	Handler runtime.WebhookHandler
}

type WebhookRegistry struct {
	sync.RWMutex
	logger   *zap.Logger
	webhooks map[string]*Webhook
}

func NewWebhookRegistry(logger *zap.Logger) *WebhookRegistry {
	return &WebhookRegistry{
		logger:   logger,
		webhooks: make(map[string]*Webhook),
	}
}

func (w *WebhookRegistry) RegisterWebhook(ctx context.Context, config runtime.WebhookConfig, handler runtime.WebhookHandler) error {
	w.Lock()
	defer w.Unlock()

	if _, exists := w.webhooks[config.Path]; exists {
		return fmt.Errorf("webhook path already registered: %s", config.Path)
	}

	w.webhooks[config.Path] = &Webhook{
		Config:  config,
		Handler: handler,
	}

	w.logger.Info("Registered new webhook", zap.String("path", config.Path), zap.String("method", config.Method))
	return nil
}

func (w *WebhookRegistry) HandleWebhook(router *mux.Router) {
	w.RLock()
	defer w.RUnlock()

	for _, webhook := range w.webhooks {
		webhook := webhook // Create new variable to avoid closure issues
		router.HandleFunc(webhook.Config.Path, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != webhook.Config.Method {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "Invalid request payload", http.StatusBadRequest)
				return
			}

			result, err := webhook.Handler(r.Context(), payload)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		}).Methods(webhook.Config.Method)
	}
}
