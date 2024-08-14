package server

import (
	"strings"

	"github.com/u2u-labs/go-layerg-common/api"
	"github.com/u2u-labs/go-layerg-common/rtapi"
	"go.uber.org/zap"
)

func (p *Pipeline) rpc(logger *zap.Logger, session Session, envelope *rtapi.Envelope) (bool, *rtapi.Envelope) {
	rpcMessage := envelope.GetRpc()
	if rpcMessage.Id == "" {
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_BAD_INPUT),
			Message: "RPC ID must be set",
		}}}, true)
		return false, nil
	}

	id := strings.ToLower(rpcMessage.Id)

	fn := p.runtime.Rpc(id)
	if fn == nil {
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_RUNTIME_FUNCTION_NOT_FOUND),
			Message: "RPC function not found",
		}}}, true)
		return false, nil
	}

	result, fnErr, _ := fn(session.Context(), nil, nil, session.UserID().String(), session.Username(), session.Vars(), session.Expiry(), session.ID().String(), session.ClientIP(), session.ClientPort(), session.Lang(), rpcMessage.Payload)
	if fnErr != nil {
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_RUNTIME_FUNCTION_EXCEPTION),
			Message: fnErr.Error(),
		}}}, true)
		return false, nil
	}

	out := &rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Rpc{Rpc: &api.Rpc{
		Id:      rpcMessage.Id,
		Payload: result,
	}}}
	_ = session.Send(out, true)

	return true, out
}
