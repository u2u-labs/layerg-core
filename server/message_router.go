package server

import (
	"github.com/u2u-labs/go-layerg-common/rtapi"
	"github.com/u2u-labs/layerg-kit/pb"
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
}

type LocalMessageRouter struct {
	protojsonMarshaler *protojson.MarshalOptions
	sessionRegistry    SessionRegistry
	tracker            Tracker
	peer               Peer
}

func NewLocalMessageRouter(sessionRegistry SessionRegistry, tracker Tracker, protojsonMarshaler *protojson.MarshalOptions) MessageRouter {
	return &LocalMessageRouter{
		protojsonMarshaler: protojsonMarshaler,
		sessionRegistry:    sessionRegistry,
		tracker:            tracker,
		peer:               nil,
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
	for _, presenceID := range presenceIDs {
		// group
		if presenceID.Node != r.peer.Local().Name() {
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
	if r.peer == nil {
		return
	}

	for remoteNode, sessions := range toRemote {
		endpoint, ok := r.peer.Member(remoteNode)
		if !ok {
			continue
		}

		m := &pb.Request{
			Payload: &pb.Request_Out{Out: &pb.ResponseWriter{
				Recipient: sessions,
				Payload:   &pb.ResponseWriter_Envelope{Envelope: envelope},
			}},
		}

		if err := r.peer.Send(endpoint, m, reliable); err != nil {
			logger.Error("Failed to route message", zap.String("node", remoteNode), zap.Error(err))
		}
	}
}

func (r *LocalMessageRouter) SendToStream(logger *zap.Logger, stream PresenceStream, envelope *rtapi.Envelope, reliable bool) {
	presenceIDs := r.tracker.ListLocalPresenceIDByStream(stream)
	r.SendToPresenceIDs(logger, presenceIDs, envelope, reliable)

	if r.peer == nil {
		return
	}

	recipient := &pb.Recipienter{
		Action:  pb.Recipienter_STREAM,
		Payload: &pb.Recipienter_Stream{Stream: presenceStream2PB(stream)},
	}

	r.peer.Broadcast(&pb.Request{
		Payload: &pb.Request_Out{
			Out: &pb.ResponseWriter{
				Recipient: []*pb.Recipienter{recipient},
				Payload:   &pb.ResponseWriter_Envelope{Envelope: envelope},
			},
		},
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

	if r.peer == nil {
		return
	}

	r.peer.Broadcast(&pb.Request{
		Payload: &pb.Request_Out{
			Out: &pb.ResponseWriter{
				Recipient: make([]*pb.Recipienter, 0),
				Payload:   &pb.ResponseWriter_Envelope{Envelope: envelope},
			},
		},
	}, reliable)
}

func (r *LocalMessageRouter) SetPeer(peer Peer) {
	r.peer = peer
}
