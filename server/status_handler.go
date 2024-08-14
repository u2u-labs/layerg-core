package server

import (
	"context"
	"math"
	"runtime"

	"github.com/u2u-labs/layerg-core/console"
	"go.uber.org/zap"
)

type StatusHandler interface {
	GetStatus(ctx context.Context) ([]*console.StatusList_Status, error)
}

type LocalStatusHandler struct {
	logger          *zap.Logger
	sessionRegistry SessionRegistry
	matchRegistry   MatchRegistry
	tracker         Tracker
	metrics         Metrics
	node            string
}

func NewLocalStatusHandler(logger *zap.Logger, sessionRegistry SessionRegistry, matchRegistry MatchRegistry, tracker Tracker, metrics Metrics, node string) StatusHandler {
	return &LocalStatusHandler{
		logger:          logger,
		sessionRegistry: sessionRegistry,
		matchRegistry:   matchRegistry,
		tracker:         tracker,
		metrics:         metrics,
		node:            node,
	}
}

func (s *LocalStatusHandler) GetStatus(ctx context.Context) ([]*console.StatusList_Status, error) {
	return []*console.StatusList_Status{
		{
			Name:           s.node,
			Health:         console.StatusHealth_STATUS_HEALTH_OK,
			SessionCount:   int32(s.sessionRegistry.Count()),
			PresenceCount:  int32(s.tracker.Count()),
			MatchCount:     int32(s.matchRegistry.Count()),
			GoroutineCount: int32(runtime.NumGoroutine()),
			AvgLatencyMs:   math.Floor(s.metrics.SnapshotLatencyMs()*100) / 100,
			AvgRateSec:     math.Floor(s.metrics.SnapshotRateSec()*100) / 100,
			AvgInputKbs:    math.Floor(s.metrics.SnapshotRecvKbSec()*100) / 100,
			AvgOutputKbs:   math.Floor(s.metrics.SnapshotSentKbSec()*100) / 100,
		},
	}, nil
}
