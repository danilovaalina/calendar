package repository

import (
	"context"
	"sync"
	"time"

	"github.com/danilovaalina/calendar/internal/model"
	"github.com/google/uuid"
)

type Repository struct {
	mu      sync.RWMutex
	events  map[uuid.UUID]model.Event
	archive map[uuid.UUID]model.Event
}

func New() *Repository {
	return &Repository{
		events: make(map[uuid.UUID]model.Event),
	}
}

func (r *Repository) CreateEvent(_ context.Context, event model.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events[event.ID] = event
	return nil
}

func (r *Repository) UpdateEvent(_ context.Context, event model.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.events[event.ID]; !exists {
		return model.ErrEventNotFound
	}

	r.events[event.ID] = event

	return nil
}

func (r *Repository) DeleteEvent(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.events[id]; !exists {
		return model.ErrEventNotFound
	}

	delete(r.events, id)
	return nil
}

func (r *Repository) GetByUserIDAndDateRange(_ context.Context, userID uuid.UUID, start, end time.Time) []model.Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []model.Event
	for _, e := range r.events {
		if e.UserID == userID && !e.Date.Before(start) && !e.Date.After(end) {
			result = append(result, e)
		}
	}
	return result
}

func (r *Repository) GetByID(_ context.Context, id uuid.UUID) (model.Event, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.events[id]
	if !ok {
		return model.Event{}, model.ErrEventNotFound
	}
	return e, nil
}

func (r *Repository) ArchiveEvents(_ context.Context, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, event := range r.events {
		if event.Date.Before(now) {
			r.archive[id] = event
			delete(r.events, id)
		}
	}
}
