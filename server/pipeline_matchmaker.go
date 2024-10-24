package server

import (
	"github.com/u2u-labs/go-layerg-common/rtapi"
	"github.com/u2u-labs/go-layerg-common/runtime"
	"go.uber.org/zap"
)

func (p *Pipeline) matchmakerAdd(logger *zap.Logger, session Session, envelope *rtapi.Envelope) (bool, *rtapi.Envelope) {
	incoming := envelope.GetMatchmakerAdd()

	// Minimum count.
	minCount := int(incoming.MinCount)
	if minCount < 2 {
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_BAD_INPUT),
			Message: "Invalid minimum count, must be >= 2",
		}}}, true)
		return false, nil
	}

	// Maximum count, must be at least minimum count.
	maxCount := int(incoming.MaxCount)
	if maxCount < minCount {
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_BAD_INPUT),
			Message: "Invalid maximum count, must be >= minimum count",
		}}}, true)
		return false, nil
	}

	// Count multiple if supplied, otherwise defaults to 1.
	countMultiple := 1
	if incoming.CountMultiple != nil {
		countMultiple = int(incoming.CountMultiple.GetValue())
		if countMultiple < 1 {
			_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
				Code:    int32(rtapi.Error_BAD_INPUT),
				Message: "Invalid count multiple, must be >= 1",
			}}}, true)
			return false, nil
		}
		if minCount%countMultiple != 0 {
			_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
				Code:    int32(rtapi.Error_BAD_INPUT),
				Message: "Invalid count multiple for minimum count, must divide",
			}}}, true)
			return false, nil
		}
		if maxCount%countMultiple != 0 {
			_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
				Code:    int32(rtapi.Error_BAD_INPUT),
				Message: "Invalid count multiple for maximum count, must divide",
			}}}, true)
			return false, nil
		}
	}

	query := incoming.Query
	if query == "" {
		query = "*"
	}

	presences := []*MatchmakerPresence{{
		UserId:    session.UserID().String(),
		SessionId: session.ID().String(),
		Username:  session.Username(),
		Node:      p.node,
		SessionID: session.ID(),
	}}

	// Run matchmaker add.
	ticket, _, err := p.matchmaker.Add(session.Context(), presences, session.ID().String(), "", query, minCount, maxCount, countMultiple, incoming.StringProperties, incoming.NumericProperties)
	if err != nil {
		logger.Error("Error adding to matchmaker", zap.Error(err))
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_RUNTIME_EXCEPTION),
			Message: "Error adding to matchmaker",
		}}}, true)
		return false, nil
	}

	peer := p.runtime.GetPeer()
	if peer != nil {
		peer.MatchmakerAdd(matchmakerExtract2pb(&MatchmakerExtract{
			Presences:         presences,
			SessionID:         session.ID().String(),
			PartyId:           "",
			Query:             query,
			MinCount:          minCount,
			MaxCount:          maxCount,
			CountMultiple:     countMultiple,
			StringProperties:  incoming.StringProperties,
			NumericProperties: incoming.NumericProperties,
		}))
	}

	// Return the ticket.
	out := &rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_MatchmakerTicket{MatchmakerTicket: &rtapi.MatchmakerTicket{
		Ticket: ticket,
	}}}
	_ = session.Send(out, true)

	return true, out
}

func (p *Pipeline) matchmakerRemove(logger *zap.Logger, session Session, envelope *rtapi.Envelope) (bool, *rtapi.Envelope) {
	incoming := envelope.GetMatchmakerRemove()

	// Ticket is required.
	if incoming.Ticket == "" {
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_BAD_INPUT),
			Message: "Invalid matchmaker ticket",
		}}}, true)
		return false, nil
	}

	// Run matchmaker remove.
	if err := p.matchmaker.RemoveSession(session.ID().String(), incoming.Ticket); err != nil {
		if err == runtime.ErrMatchmakerTicketNotFound {
			_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
				Code:    int32(rtapi.Error_BAD_INPUT),
				Message: "Matchmaker ticket not found",
			}}}, true)
			return false, nil
		}

		logger.Error("Error removing matchmaker ticket", zap.Error(err))
		_ = session.Send(&rtapi.Envelope{Cid: envelope.Cid, Message: &rtapi.Envelope_Error{Error: &rtapi.Error{
			Code:    int32(rtapi.Error_RUNTIME_EXCEPTION),
			Message: "Error removing matchmaker ticket",
		}}}, true)
		return false, nil
	}

	peer := p.runtime.GetPeer()
	if peer != nil {
		peer.MatchmakerRemoveSession(session.ID().String(), incoming.Ticket)
	}

	out := &rtapi.Envelope{Cid: envelope.Cid}
	_ = session.Send(out, true)

	return true, out
}
