package server

import (
	"context"
	"math"
	"net"
	"runtime"
	"strconv"

	"github.com/u2u-labs/layerg-core/console"
	"github.com/u2u-labs/layerg-kit/kit"
	"go.uber.org/zap"
)

type StatusHandler interface {
	GetStatus(ctx context.Context) ([]*console.StatusList_Status, error)
	GetServices(ctx context.Context) []*console.StatusList_ServiceStatus
	SetPeer(peer Peer)
}

type LocalStatusHandler struct {
	logger          *zap.Logger
	sessionRegistry SessionRegistry
	matchRegistry   MatchRegistry
	tracker         Tracker
	metrics         Metrics
	node            string
	peer            Peer
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
	status := make([]*console.StatusList_Status, 0)
	status = append(status, &console.StatusList_Status{
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
	})

	if s.peer == nil {
		return status, nil
	}

	for _, member := range s.peer.Members() {
		if member.Name() == s.node {
			continue
		}

		status = append(status, &console.StatusList_Status{
			Name:           member.Name(),
			Health:         console.StatusHealth_STATUS_HEALTH_OK,
			SessionCount:   member.SessionCount(),
			PresenceCount:  member.PresenceCount(),
			MatchCount:     member.MatchCount(),
			GoroutineCount: member.GoroutineCount(),
			AvgLatencyMs:   member.AvgLatencyMs(),
			AvgRateSec:     member.AvgRateSec(),
			AvgInputKbs:    member.AvgInputKbs(),
			AvgOutputKbs:   member.AvgOutputKbs(),
		})
	}
	return status, nil
}

func (s *LocalStatusHandler) GetServices(ctx context.Context) []*console.StatusList_ServiceStatus {
	services := make([]*console.StatusList_ServiceStatus, 0)
	if s.peer == nil {
		return services
	}

	s.peer.GetServiceRegistry().Range(func(key string, value kit.Service) bool {
		if key == kit.SERVICE_NAME {
			return true
		}

		for _, v := range value.GetClients() {
			ip, port, _ := net.SplitHostPort(v.Addr())
			portValue, _ := strconv.Atoi(port)
			services = append(services, &console.StatusList_ServiceStatus{
				Name:        v.Name(),
				Vars:        v.Metadata().Vars,
				Ip:          ip,
				Port:        uint32(portValue),
				Role:        v.Role(),
				Status:      int32(v.Status()),
				Weight:      v.Weight(),
				Balancer:    int32(v.Balancer()),
				AllowStream: v.AllowStream(),
			})
		}
		return true
	})
	return services
}

func (s *LocalStatusHandler) SetPeer(peer Peer) {
	s.peer = peer
}
