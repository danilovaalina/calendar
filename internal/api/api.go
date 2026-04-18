package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/danilovaalina/calendar/internal/logger"
	"github.com/danilovaalina/calendar/internal/model"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type Service interface {
	CreateEvent(ctx context.Context, event model.Event) (model.Event, error)
	UpdateEvent(ctx context.Context, event model.Event) (model.Event, error)
	DeleteEvent(ctx context.Context, event model.Event) error
	EventsForDay(ctx context.Context, userID uuid.UUID, date time.Time) ([]model.Event, error)
	EventsForWeek(ctx context.Context, userID uuid.UUID, date time.Time) ([]model.Event, error)
	EventsForMonth(ctx context.Context, userID uuid.UUID, date time.Time) ([]model.Event, error)
}

type Logger interface {
	Log(log logger.Log)
}

type API struct {
	*echo.Echo
	service Service
	logger  Logger
}

func New(svc Service, l Logger) *API {
	api := &API{
		Echo:    echo.New(),
		service: svc,
		logger:  l,
	}
	api.Validator = &CustomValidator{validator: validator.New()}

	// Middleware логирования
	api.Use(api.LoggerMiddleware())

	// Routes
	api.POST("/create_event", api.createEvent)
	api.POST("/update_event", api.updateEvent)
	api.POST("/delete_event", api.deleteEvent)
	api.GET("/events_for_day", api.eventsForDay)
	api.GET("/events_for_week", api.eventsForWeek)
	api.GET("/events_for_month", api.eventsForMonth)

	return api
}

func (a *API) LoggerMiddleware() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:  true,
		LogURI:     true,
		LogLatency: true,
		LogStatus:  true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			// Отправляем структурированные данные в канал
			a.logger.Log(logger.Log{
				Method:  v.Method,
				URI:     v.URI,
				Latency: v.Latency,
				Status:  v.Status,
			})
			return nil
		},
	})
}

type Date time.Time

// UnmarshalParam Для Form/Query параметров
func (d *Date) UnmarshalParam(src string) error {
	t, err := time.Parse("2006-01-02", src)
	*d = Date(t)
	return err
}

// UnmarshalJSON Для JSON тела запроса
func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	t, err := time.Parse("2006-01-02", s)
	*d = Date(t)
	return err
}

func (d *Date) MarshalJSON() ([]byte, error) {
	// Преобразуем Date обратно в time.Time, форматируем и оборачиваем в кавычки
	formatted := fmt.Sprintf("\"%s\"", time.Time(*d).Format("2006-01-02"))
	return []byte(formatted), nil
}

type createEventRequest struct {
	UserID    uuid.UUID `form:"user_id" json:"user_id" validate:"required"`
	Date      Date      `form:"date" json:"date" validate:"required"`
	Event     string    `form:"event" json:"event" validate:"required"`
	Channel   string    `json:"channel"`
	Recipient string    `json:"recipient"`
	RemindAt  time.Time `json:"remind_at"`
}

type response struct {
	Result interface{} `json:"result,omitempty"`
	Reason string      `json:"reason,omitempty"`
}

func (a *API) createEvent(c *echo.Context) error {
	var req createEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "invalid request format or params"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "user_id, date and text are required fields"})
	}

	event := model.Event{
		UserID: req.UserID,
		Date:   time.Time(req.Date),
		Text:   req.Event,
	}

	if !req.RemindAt.IsZero() {
		event.Notification = model.NotificationSettings{
			Channel:   req.Channel,
			Recipient: req.Recipient,
			RemindAt:  req.RemindAt,
		}
	}

	e, err := a.service.CreateEvent(c.Request().Context(), event)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, response{Reason: err.Error()})
	}

	return c.JSON(http.StatusOK, a.eventFromModel(e))
}

type updateEventRequest struct {
	ID     uuid.UUID `form:"id" json:"id" validate:"required"`
	UserID uuid.UUID `form:"user_id" json:"user_id" validate:"required"`
	Date   Date      `form:"date" json:"date" validate:"required"`
	Event  string    `form:"event" json:"event" validate:"required"`
}

func (a *API) updateEvent(c *echo.Context) error {
	var req updateEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "invalid request format or params"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "id, user_id, date and text are required fields"})
	}

	event := model.Event{
		ID:     req.ID,
		UserID: req.UserID,
		Date:   time.Time(req.Date),
		Text:   req.Event,
	}

	e, err := a.service.UpdateEvent(c.Request().Context(), event)
	if err != nil {
		if errors.Is(err, model.ErrEventNotFound) {
			return c.JSON(http.StatusServiceUnavailable, response{Reason: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, response{Reason: err.Error()})
	}

	return c.JSON(http.StatusOK, a.eventFromModel(e))
}

type deleteEventRequest struct {
	ID     uuid.UUID `form:"id" json:"id" validate:"required"`
	UserID uuid.UUID `form:"user_id" json:"user_id" validate:"required"`
}

func (a *API) deleteEvent(c *echo.Context) error {
	var req deleteEventRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "invalid request format or params"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "id and user_id are required fields"})
	}

	event := model.Event{
		ID:     req.ID,
		UserID: req.UserID,
	}

	err := a.service.DeleteEvent(c.Request().Context(), event)
	if err != nil {
		if errors.Is(err, model.ErrEventNotFound) {
			return c.JSON(http.StatusServiceUnavailable, response{Reason: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, response{Reason: err.Error()})
	}

	return c.JSON(http.StatusOK, response{Result: "deleted"})
}

type eventsRequest struct {
	UserID uuid.UUID `query:"user_id" validate:"required"`
	Date   Date      `query:"date" validate:"required"`
}

func (a *API) eventsForDay(c *echo.Context) error {
	var req eventsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "invalid request format or params"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "user_id and date are required"})
	}

	events, err := a.service.EventsForDay(c.Request().Context(), req.UserID, time.Time(req.Date))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response{Reason: err.Error()})
	}

	return c.JSON(http.StatusOK, a.eventsFromModel(events))
}

func (a *API) eventsForWeek(c *echo.Context) error {
	var req eventsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "invalid request format or params"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "user_id and date are required"})
	}

	events, err := a.service.EventsForWeek(c.Request().Context(), req.UserID, time.Time(req.Date))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response{Reason: err.Error()})
	}

	return c.JSON(http.StatusOK, a.eventsFromModel(events))
}

func (a *API) eventsForMonth(c *echo.Context) error {
	var req eventsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "invalid request format or params"})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, response{Reason: "user_id and date are required"})
	}

	events, err := a.service.EventsForMonth(c.Request().Context(), req.UserID, time.Time(req.Date))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, response{Reason: err.Error()})
	}

	return c.JSON(http.StatusOK, a.eventsFromModel(events))
}

type eventResponse struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
	Date   Date      `json:"date"`
	Text   string    `json:"text"`
}

func (a *API) eventsFromModel(events []model.Event) []eventResponse {
	r := make([]eventResponse, 0, len(events))
	for _, event := range events {
		r = append(r, a.eventFromModel(event))
	}

	return r
}

func (a *API) eventFromModel(event model.Event) eventResponse {
	return eventResponse{
		ID:     event.ID,
		UserID: event.UserID,
		Date:   Date(event.Date),
		Text:   event.Text,
	}
}
