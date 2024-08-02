package server

import (
	"github.com/u2u-labs/go-layerg-common/rtapi"
	"go.uber.org/zap"
)

func (p *Pipeline) ping(logger *zap.Logger, session Session, envelope *rtapi.Envelope) (bool, *rtapi.Envelope) {
	out := &rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Pong{Pong: &rtapi.Pong{}}}
	_ = session.Send(out, true)

	return true, out
}

func (p *Pipeline) pong(logger *zap.Logger, session Session, envelope *rtapi.Envelope) (bool, *rtapi.Envelope) {
	// No application-level action in response to a pong message.
	return true, nil
}
