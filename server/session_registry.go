package server

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/u2u-labs/go-layerg-common/api"
	"github.com/u2u-labs/go-layerg-common/rtapi"
	"github.com/u2u-labs/go-layerg-common/runtime"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SessionFormat uint8

const (
	SessionFormatJson SessionFormat = iota
	SessionFormatProtobuf
)

type Session interface {
	Logger() *zap.Logger
	ID() uuid.UUID
	UserID() uuid.UUID
	Vars() map[string]string
	ClientIP() string
	ClientPort() string
	Lang() string

	Context() context.Context

	Username() string
	SetUsername(string)

	Expiry() int64
	Consume()

	Format() SessionFormat
	Send(envelope *rtapi.Envelope, reliable bool) error
	SendBytes(payload []byte, reliable bool) error

	Close(msg string, reason runtime.PresenceReason, envelopes ...*rtapi.Envelope)
	CloseLock()
	CloseUnlock()
}

type SessionRegistry interface {
	Stop()
	Count() int
	Get(sessionID uuid.UUID) Session
	Add(session Session)
	Remove(sessionID uuid.UUID)
	Disconnect(ctx context.Context, sessionID uuid.UUID, ban bool, reason ...runtime.PresenceReason) error
	SingleSession(ctx context.Context, tracker Tracker, userID, sessionID uuid.UUID)
	Range(fn func(session Session) bool)
	SetPeer(peer Peer)
}

type LocalSessionRegistry struct {
	metrics Metrics

	sessions     *MapOf[uuid.UUID, Session]
	sessionCount *atomic.Int32
	peer         Peer
}

func NewLocalSessionRegistry(metrics Metrics) SessionRegistry {
	return &LocalSessionRegistry{
		metrics: metrics,

		sessions:     &MapOf[uuid.UUID, Session]{},
		sessionCount: atomic.NewInt32(0),
		peer:         nil,
	}
}

func (r *LocalSessionRegistry) SetPeer(peer Peer) {
	r.peer = peer
}

func (r *LocalSessionRegistry) Stop() {}

func (r *LocalSessionRegistry) Count() int {
	return int(r.sessionCount.Load())
}

func (r *LocalSessionRegistry) Get(sessionID uuid.UUID) Session {
	session, ok := r.sessions.Load(sessionID)
	if !ok {
		return nil
	}
	return session
}

func (r *LocalSessionRegistry) Add(session Session) {
	r.sessions.Store(session.ID(), session)
	count := r.sessionCount.Inc()
	r.metrics.GaugeSessions(float64(count))
}

func (r *LocalSessionRegistry) Remove(sessionID uuid.UUID) {
	r.sessions.Delete(sessionID)
	count := r.sessionCount.Dec()
	r.metrics.GaugeSessions(float64(count))
}

func (r *LocalSessionRegistry) Disconnect(ctx context.Context, sessionID uuid.UUID, ban bool, reason ...runtime.PresenceReason) error {
	session, ok := r.sessions.Load(sessionID)
	if ok {
		// No need to remove the session from the map, session.Close() will do that.
		reasonOverride := runtime.PresenceReasonDisconnect
		if len(reason) > 0 {
			reasonOverride = reason[0]
		}

		if ban {
			session.Close("server-side session disconnect", runtime.PresenceReasonDisconnect,
				&rtapi.Envelope{Message: &rtapi.Envelope_Notifications{
					Notifications: &rtapi.Notifications{
						Notifications: []*api.Notification{
							{
								Id:         uuid.Must(uuid.NewV4()).String(),
								Subject:    "banned",
								Content:    "{}",
								Code:       NotificationCodeUserBanned,
								SenderId:   "",
								CreateTime: &timestamppb.Timestamp{Seconds: time.Now().Unix()},
								Persistent: false,
							},
						},
					},
				}})
		} else {
			session.Close("server-side session disconnect", reasonOverride)
		}
	}
	return nil
}

func (r *LocalSessionRegistry) SingleSession(ctx context.Context, tracker Tracker, userID, sessionID uuid.UUID) {
	sessionIDs := tracker.ListLocalSessionIDByStream(PresenceStream{Mode: StreamModeNotifications, Subject: userID})
	for _, foundSessionID := range sessionIDs {
		if foundSessionID == sessionID {
			// Allow the current session, only disconnect any older ones.
			continue
		}
		session, ok := r.sessions.Load(foundSessionID)
		if ok {
			// No need to remove the session from the map, session.Close() will do that.
			session.Close("server-side session disconnect", runtime.PresenceReasonDisconnect,
				&rtapi.Envelope{Message: &rtapi.Envelope_Notifications{
					Notifications: &rtapi.Notifications{
						Notifications: []*api.Notification{
							{
								Id:         uuid.Must(uuid.NewV4()).String(),
								Subject:    "single_socket",
								Content:    "{}",
								Code:       NotificationCodeSingleSocket,
								SenderId:   "",
								CreateTime: &timestamppb.Timestamp{Seconds: time.Now().Unix()},
								Persistent: false,
							},
						},
					},
				}})
		}
	}
}

func (r *LocalSessionRegistry) Range(fn func(Session) bool) {
	r.sessions.Range(func(id uuid.UUID, session Session) bool {
		return fn(session)
	})
}
