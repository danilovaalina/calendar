package model

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

var (
	ErrEventNotFound   = errors.New("event not found")
	ErrInvalidDate     = errors.New("invalid date format, expected YYYY-MM-DD")
	ErrNoAccessToEvent = errors.New("no access to this event")
)

type Event struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Date         time.Time
	Text         string
	IsSent       bool
	Notification NotificationSettings
}

type NotificationSettings struct {
	Channel   string    // "telegram" или "email"
	Recipient string    // ID чата или email адрес
	RemindAt  time.Time // Время, когда Notifier должен сработать
}
