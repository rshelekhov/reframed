package user

import (
	"database/sql"
	"errors"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	resp "github.com/rshelekhov/remedi/internal/lib/api/response"
	"github.com/rshelekhov/remedi/internal/lib/logger/sl"
	"io"
	"log/slog"
	"net/http"
)

type userHandler struct {
	service Service
	logger  *slog.Logger
}

func RegisterHandlers(r *chi.Mux, log *slog.Logger, db *sql.DB) {
	userService := NewService(NewStorage(db))
	NewHandler(r, log, userService)
}

func NewHandler(r *chi.Mux, log *slog.Logger, srv Service) {
	h := &userHandler{
		service: srv,
		logger:  log,
	}

	r.Get("/users", h.ListUsers())
	r.Post("/users", h.CreateUser())
	r.Get("/users/{id}", h.ReadUser())
	r.Put("/users/{id}", h.UpdateUser())
	r.Delete("/users/{id}", h.DeleteUser())
}

func (h *userHandler) ListUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handler.ListUsers"
	}
}

func (h *userHandler) CreateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handler.CreateUser"

		log := h.logger.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var user CreateUser

		err := render.DecodeJSON(r.Body, &user)
		if errors.Is(err, io.EOF) {
			log.Error("request body is empty")

			render.JSON(w, r, resp.Error("request body is empty"))

			return
		}
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.JSON(w, r, resp.Error("failed to decode request body"))

			return
		}

		log.Info("request body decoded", slog.Any("user", user))

		id, err := h.service.CreateUser(user)
		if err != nil {
			log.Error("failed to create user", sl.Err(err))

			render.JSON(w, r, resp.Error("failed to create user"))

			return
		}

		log.Info("User created", slog.Any("user_id", id))

		render.JSON(w, r, resp.Success("User created", id))
	}
}

func (h *userHandler) ReadUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handler.ReadUser"
	}
}

func (h *userHandler) UpdateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handler.UpdateUser"
	}
}

func (h *userHandler) DeleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handler.DeleteUser"
	}
}
