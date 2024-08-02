package server

import "github.com/gofrs/uuid/v5"

type testStreamManager struct{}

func (t testStreamManager) UserJoin(stream PresenceStream, userID, sessionID uuid.UUID, hidden, persistence bool, status string) (bool, bool, error) {
	return true, true, nil
}

func (t testStreamManager) UserUpdate(stream PresenceStream, userID, sessionID uuid.UUID, hidden, persistence bool, status string) (bool, error) {
	return true, nil
}

func (t testStreamManager) UserLeave(stream PresenceStream, userID, sessionID uuid.UUID) error {
	return nil
}
