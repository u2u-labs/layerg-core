package server

import (
	"testing"

	"github.com/gofrs/uuid/v5"
)

func TestMatchPresenceList(t *testing.T) {
	list := NewMatchPresenceList()

	p1 := &MatchPresence{
		Node:      "nakama",
		UserID:    uuid.Must(uuid.NewV4()),
		SessionID: uuid.Must(uuid.NewV4()),
		Username:  "user1",
		Reason:    0,
	}
	p2 := &MatchPresence{
		Node:      "nakama",
		UserID:    uuid.Must(uuid.NewV4()),
		SessionID: uuid.Must(uuid.NewV4()),
		Username:  "user2",
		Reason:    0,
	}
	p3 := &MatchPresence{
		Node:      "nakama",
		UserID:    uuid.Must(uuid.NewV4()),
		SessionID: uuid.Must(uuid.NewV4()),
		Username:  "user3",
		Reason:    0,
	}

	list.Join([]*MatchPresence{p1, p2, p3})

	list.Leave([]*MatchPresence{p2})
	if list.Size() != 2 {
		t.Fatalf("list size error: %+v", list.ListPresences())
	}

	list.Leave([]*MatchPresence{p1})
	if list.Size() != 1 {
		t.Fatalf("list size error: %+v", list.ListPresences())
	}

	list.Leave([]*MatchPresence{p3})
	if list.Size() != 0 {
		t.Fatalf("list size error: %+v", list.ListPresences())
	}
}
