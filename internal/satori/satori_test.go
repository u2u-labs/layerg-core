package satori

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/u2u-labs/go-layerg-common/runtime"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestSatoriClient_EventsPublish(t *testing.T) {
	t.SkipNow()

	identityID := uuid.Must(uuid.NewV4()).String()

	logger := NewConsoleLogger(os.Stdout, true)
	client := NewSatoriClient(logger, "<URL>", "<API KEY NAME>", "<API KEY>", "<SIGNING KEY>")

	ctx, ctxCancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancelFn()

	if err := client.Authenticate(ctx, identityID); err != nil {
		t.Fatalf("error in client.Authenticate: %+v", err)
	}

	evt := &runtime.Event{
		Name: "gameStarted",
		// Id:   "optionalEventId",
		Metadata: map[string]string{
			"someKey": "someValue",
		},
		Value:     "someValue",
		Timestamp: time.Now().Unix(),
	}

	if err := client.EventsPublish(ctx, identityID, []*runtime.Event{evt}); err != nil {
		t.Fatalf("error in client.EventsPublish: %+v", err)
	}
}

func NewConsoleLogger(output *os.File, verbose bool) *zap.Logger {
	consoleEncoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})

	core := zapcore.NewCore(consoleEncoder, output, &loggerEnabler{})
	options := []zap.Option{zap.AddStacktrace(zap.ErrorLevel)}

	return zap.New(core, options...)
}

type loggerEnabler struct{}

func (l *loggerEnabler) Enabled(level zapcore.Level) bool {
	return true
}
