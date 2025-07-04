package server

import (
	"context"
	"strconv"

	"github.com/u2u-labs/go-layerg-common/rtapi"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (p *Pipeline) any(logger *zap.Logger, session Session, envelope *rtapi.Envelope) (bool, *rtapi.Envelope) {
	req := envelope.GetAnyRequest()
	if req.Name == "" {
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_BAD_INPUT),
			Message: "Name must be set",
		}}}, true)
		return false, nil
	}

	req.Context = make(map[string]string)
	req.Context["client_ip"] = session.ClientIP()
	req.Context["client_port"] = session.ClientPort()
	req.Context["client_cid"] = envelope.Cid
	req.Context["userId"] = session.UserID().String()
	req.Context["username"] = session.Username()
	req.Context["expiry"] = strconv.FormatInt(session.Expiry(), 10)
	for k, v := range session.Vars() {
		req.Context["vars_"+k] = v
	}

	peer, ok := p.runtime.GetPeer()
	if !ok {
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(codes.Unavailable),
			Message: "Service Unavailable",
		}}}, true)
		return false, nil
	}

	if err := peer.SendMS(context.Background(), req); err != nil {
		code, ok := status.FromError(err)
		if !ok {
			code = status.New(codes.Internal, err.Error())
		}
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(code.Code()),
			Message: code.Message(),
		}}}, true)
		return false, nil
	}
	return true, nil
}
