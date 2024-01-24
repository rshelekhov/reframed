package handlers

import (
	"errors"
	"fmt"
	"github.com/rshelekhov/reframed/internal/logger"
	"github.com/rshelekhov/reframed/internal/models"
	"github.com/rshelekhov/reframed/internal/storage"
	"github.com/segmentio/ksuid"
	"log/slog"
	"net/http"
	"time"
)

type ListHandler struct {
	Storage storage.ListStorage
	Logger  logger.Interface
}

func (h *ListHandler) CreateList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "list.handlers.CreateList"

		log := logger.LogWithRequest(h.Logger, op, r)

		list := &models.List{}

		// Decode the request body
		err := DecodeJSON(w, r, log, list)
		if err != nil {
			return
		}

		id := ksuid.New().String()
		now := time.Now().UTC()

		newList := models.List{
			ID:        id,
			Title:     list.Title,
			UserID:    list.UserID,
			UpdatedAt: &now,
		}

		err = h.Storage.CreateList(r.Context(), newList)
		if err != nil {
			log.Error("failed to create list", logger.Err(err))
			responseError(w, r, http.StatusInternalServerError, "failed to create list")
			return
		}

		log.Info("list created", slog.Any("list_id", id))
		responseSuccess(w, r, http.StatusCreated, "list created", models.List{ID: id})
	}
}

func (h *ListHandler) GetListByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "list.handlers.GetListByID"
	}
}

func (h *ListHandler) GetLists() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "list.handlers.GetLists"

		log := logger.LogWithRequest(h.Logger, op, r)

		pagination, err := ParseLimitAndOffset(r)
		if err != nil {
			log.Error(ErrFailedToParsePagination.Error(), logger.Err(err))
			responseError(w, r, http.StatusBadRequest, ErrFailedToParsePagination.Error())
			return
		}

		// TODO: implement JWT auth
		userID, statusCode, err := GetID(r, log)
		if err != nil {
			responseError(w, r, statusCode, err.Error())
			return
		}

		lists, err := h.Storage.GetLists(r.Context(), userID, pagination)
		if errors.Is(err, storage.ErrNoListsFound) {
			log.Error(fmt.Sprintf("%v", storage.ErrNoListsFound))
			responseError(w, r, http.StatusNotFound, fmt.Sprintf("%v", storage.ErrNoListsFound))
			return
		}
		if err != nil {
			log.Error("failed to get lists", logger.Err(err))
			responseError(w, r, http.StatusInternalServerError, "failed to get lists")
			return
		}

		log.Info(
			"users found",
			slog.Int("count", len(lists)),
			slog.Int("limit", pagination.Limit),
			slog.Int("offset", pagination.Offset),
		)
		responseSuccess(w, r, http.StatusOK, "lists found", lists)
	}
}

func (h *ListHandler) UpdateList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "list.handlers.UpdateList"
	}
}

func (h *ListHandler) DeleteList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "list.handlers.DeleteList"
	}
}
