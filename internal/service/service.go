package service

import (
	"context"
	"fmt"
	"time"

	"github.com/danilovaalina/calendar/internal/model"
	"github.com/danilovaalina/calendar/internal/notifier"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type NotifierClient interface {
	Send(ctx context.Context, req notifier.CreateRequest) error
}

type Repository interface {
	CreateEvent(ctx context.Context, event model.Event) error
	UpdateEvent(ctx context.Context, event model.Event) error
	DeleteEvent(ctx context.Context, id uuid.UUID) error
	GetByUserIDAndDateRange(ctx context.Context, userID uuid.UUID, start, end time.Time) []model.Event
	GetByID(ctx context.Context, id uuid.UUID) (model.Event, error)
	ArchiveEvents(_ context.Context, now time.Time)
}

type Service struct {
	repo      Repository
	notifier  NotifierClient
	reminders chan model.Event
}

type Options struct {
	Repo       Repository
	Client     NotifierClient
	BufferSize int
}

func New(opts Options) *Service {
	return &Service{
		repo:      opts.Repo,
		notifier:  opts.Client,
		reminders: make(chan model.Event, opts.BufferSize),
	}
}

func (s *Service) RunNotifier(ctx context.Context) {
	for {
		select {
		case event := <-s.reminders:
			req := notifier.CreateRequest{
				Channel:       event.Notification.Channel,
				Recipient:     event.Notification.Recipient,
				Message:       fmt.Sprintf("Напоминание: %s", event.Text),
				ScheduledTime: event.Notification.RemindAt,
			}

			// Отправляем в Notifier асинхронно
			err := s.notifier.Send(ctx, req)
			if err != nil {
				log.Warn().Err(err).Msg("logger is shutting down...")
			}

		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) RunCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.repo.ArchiveEvents(ctx, time.Now())
			log.Info().Msg("cleanup events")
		case <-ctx.Done():
			return
		}
	}

}

func (s *Service) CreateEvent(ctx context.Context, event model.Event) (model.Event, error) {
	event.ID = uuid.New()

	err := s.repo.CreateEvent(ctx, event)
	if err != nil {
		return model.Event{}, err
	}

	if !event.Notification.RemindAt.IsZero() {
		s.reminders <- event
	}

	return event, nil
}

func (s *Service) UpdateEvent(ctx context.Context, event model.Event) (model.Event, error) {
	existing, err := s.repo.GetByID(ctx, event.ID)
	if err != nil {
		return model.Event{}, err
	}

	if existing.UserID != event.UserID {
		return model.Event{}, model.ErrNoAccessToEvent
	}

	err = s.repo.UpdateEvent(ctx, event)
	if err != nil {
		return model.Event{}, err
	}

	return event, nil
}

func (s *Service) DeleteEvent(ctx context.Context, event model.Event) error {
	existing, err := s.repo.GetByID(ctx, event.ID)
	if err != nil {
		return err
	}

	if existing.UserID != event.UserID {
		return model.ErrEventNotFound
	}

	return s.repo.DeleteEvent(ctx, existing.ID)
}

func (s *Service) EventsForDay(ctx context.Context, userID uuid.UUID, date time.Time) ([]model.Event, error) {
	// Используем один и тот же день как начало и конец диапазона
	return s.repo.GetByUserIDAndDateRange(ctx, userID, date, date), nil
}

func (s *Service) EventsForWeek(ctx context.Context, userID uuid.UUID, date time.Time) ([]model.Event, error) {
	start := getStartOfWeek(date)
	end := start.AddDate(0, 0, 6)

	return s.repo.GetByUserIDAndDateRange(ctx, userID, start, end), nil
}

func getStartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -weekday+1).Truncate(24 * time.Hour)
}

func (s *Service) EventsForMonth(ctx context.Context, userID uuid.UUID, date time.Time) ([]model.Event, error) {

	start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	return s.repo.GetByUserIDAndDateRange(ctx, userID, start, end), nil
}
