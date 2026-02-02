package model

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

var (
	ErrEventNotFound   = errors.New("event not found")
	ErrNoAccessToEvent = errors.New("no access to this event")
)

type Event struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Date   time.Time
	Text   string
}
