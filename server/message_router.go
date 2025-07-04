package server

import (
	"github.com/u2u-labs/go-layerg-common/rtapi"
	"github.com/u2u-labs/layerg-kit/pb"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Deferred message expected to be batched with other deferred messages.
// All deferred messages in a batch are expected to be for the same stream/mode and share a logger context.
type DeferredMessage struct {
	PresenceIDs []*PresenceID
	Envelope    *rtapi.Envelope
	Reliable    bool
}

// MessageRouter is responsible for sending a message to a list of presences or to an entire stream.
type MessageRouter interface {
	SendToPresenceIDs(*zap.Logger, []*PresenceID, *rtapi.Envelope, bool)
	SendToStream(*zap.Logger, PresenceStream, *rtapi.Envelope, bool)
	SendDeferred(*zap.Logger, []*DeferredMessage)
	SendToAll(*zap.Logger, *rtapi.Envelope, bool)
	SetPeer(peer Peer)
	GetPeer() (Peer, bool)
}

type LocalMessageRouter struct {
	protojsonMarshaler *protojson.MarshalOptions
	sessionRegistry    SessionRegistry
	tracker            Tracker
	peer               *atomic.Value
}

func NewLocalMessageRouter(sessionRegistry SessionRegistry, tracker Tracker, protojsonMarshaler *protojson.MarshalOptions) MessageRouter {
	return &LocalMessageRouter{
		protojsonMarshaler: protojsonMarshaler,
		sessionRegistry:    sessionRegistry,
		tracker:            tracker,
		peer:               &atomic.Value{},
	}
}

func (r *LocalMessageRouter) SendToPresenceIDs(logger *zap.Logger, presenceIDs []*PresenceID, envelope *rtapi.Envelope, reliable bool) {
	if len(presenceIDs) == 0 {
		return
	}

	// Prepare payload variables but do not initialize until we hit a session that needs them to avoid unnecessary work.
	var payloadProtobuf []byte
	var payloadJSON []byte

	toRemote := make(map[string][]*pb.Recipienter)
	peer, peerOk := r.GetPeer()
	for _, presenceID := range presenceIDs {
		// group
		if peerOk && presenceID.Node != peer.Local().Name() {
			if _, ok := toRemote[presenceID.Node]; !ok {
				toRemote[presenceID.Node] = make([]*pb.Recipienter, 0)
			}

			toRemote[presenceID.Node] = append(toRemote[presenceID.Node], &pb.Recipienter{
				Action:  pb.Recipienter_SESSIONID,
				Payload: &pb.Recipienter_Token{Token: presenceID.SessionID.String()},
			})
			continue
		}

		session := r.sessionRegistry.Get(presenceID.SessionID)
		if session == nil {
			logger.Debug("No session to route to", zap.String("sid", presenceID.SessionID.String()), zap.Any("envelope", envelope))
			continue
		}

		var err error
		switch session.Format() {
		case SessionFormatProtobuf:
			if payloadProtobuf == nil {
				// Marshal the payload now that we know this format is needed.
				payloadProtobuf, err = proto.Marshal(envelope)
				if err != nil {
					logger.Error("Could not marshal message", zap.Error(err))
					return
				}
			}
			err = session.SendBytes(payloadProtobuf, reliable)
		case SessionFormatJson:
			fallthrough
		default:
			if payloadJSON == nil {
				// Marshal the payload now that we know this format is needed.
				if buf, err := r.protojsonMarshaler.Marshal(envelope); err == nil {
					payloadJSON = buf
				} else {
					logger.Error("Could not marshal message", zap.Error(err))
					return
				}
			}
			err = session.SendBytes(payloadJSON, reliable)
		}
		if err != nil {
			logger.Error("Failed to route message", zap.String("sid", presenceID.SessionID.String()), zap.Error(err))
		}
	}

	// send to remote
	if !peerOk {
		return
	}

	for remoteNode, sessions := range toRemote {
		endpoint, ok := peer.Member(remoteNode)
		if !ok {
			continue
		}

		m := &pb.Peer_Envelope{
			Cid:       "",
			Recipient: sessions,
			Payload:   &pb.Peer_Envelope_NkEnvelope{NkEnvelope: envelope},
		}
		if err := peer.Send(endpoint, m, reliable); err != nil {
			logger.Error("Failed to route message", zap.String("node", remoteNode), zap.Error(err))
		}
	}
}

func (r *LocalMessageRouter) SendToStream(logger *zap.Logger, stream PresenceStream, envelope *rtapi.Envelope, reliable bool) {
	presenceIDs := r.tracker.ListLocalPresenceIDByStream(stream)
	r.SendToPresenceIDs(logger, presenceIDs, envelope, reliable)

	peer, ok := r.GetPeer()
	if !ok {
		return
	}

	recipient := &pb.Recipienter{
		Action:  pb.Recipienter_STREAM,
		Payload: &pb.Recipienter_Stream{Stream: presenceStream2PB(stream)},
	}

	peer.Broadcast(&pb.Peer_Envelope{
		Recipient: []*pb.Recipienter{recipient},
		Payload:   &pb.Peer_Envelope_NkEnvelope{NkEnvelope: envelope},
	}, reliable)
}

func (r *LocalMessageRouter) SendDeferred(logger *zap.Logger, messages []*DeferredMessage) {
	for _, message := range messages {
		r.SendToPresenceIDs(logger, message.PresenceIDs, message.Envelope, message.Reliable)
	}
}

func (r *LocalMessageRouter) SendToAll(logger *zap.Logger, envelope *rtapi.Envelope, reliable bool) {
	// Prepare payload variables but do not initialize until we hit a session that needs them to avoid unnecessary work.
	var payloadProtobuf []byte
	var payloadJSON []byte

	r.sessionRegistry.Range(func(session Session) bool {
		var err error
		switch session.Format() {
		case SessionFormatProtobuf:
			if payloadProtobuf == nil {
				// Marshal the payload now that we know this format is needed.
				payloadProtobuf, err = proto.Marshal(envelope)
				if err != nil {
					logger.Error("Could not marshal message", zap.Error(err))
					return false
				}
			}
			err = session.SendBytes(payloadProtobuf, reliable)
		case SessionFormatJson:
			fallthrough
		default:
			if payloadJSON == nil {
				// Marshal the payload now that we know this format is needed.
				if buf, err := r.protojsonMarshaler.Marshal(envelope); err == nil {
					payloadJSON = buf
				} else {
					logger.Error("Could not marshal message", zap.Error(err))
					return false
				}
			}
			err = session.SendBytes(payloadJSON, reliable)
		}
		if err != nil {
			logger.Error("Failed to route message", zap.String("sid", session.ID().String()), zap.Error(err))
		}
		return true
	})

	peer, ok := r.GetPeer()
	if !ok {
		return
	}

	peer.Broadcast(&pb.Peer_Envelope{
		Recipient: make([]*pb.Recipienter, 0),
		Payload:   &pb.Peer_Envelope_NkEnvelope{NkEnvelope: envelope},
	}, reliable)
}

func (r *LocalMessageRouter) SetPeer(peer Peer) {
	r.peer.Store(peer)
}

func (r *LocalMessageRouter) GetPeer() (Peer, bool) {
	peer := r.peer.Load()
	if peer == nil {
		return nil, false
	}

	return peer.(Peer), true
}
