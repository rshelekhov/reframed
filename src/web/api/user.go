package handlers

import (
	"context"
	"errors"
	c "github.com/rshelekhov/reframed/src/constants"
	"github.com/rshelekhov/reframed/src/logger"
	"github.com/rshelekhov/reframed/src/models"
	"github.com/rshelekhov/reframed/src/server/middleware/jwtoken/service"
	"github.com/rshelekhov/reframed/src/storage"
	"github.com/segmentio/ksuid"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type UserHandler struct {
	logger      logger.Interface
	tokenAuth   *service.JWTokenService
	userStorage storage.UserStorage
	listStorage storage.ListStorage
}

func NewUserHandler(
	log logger.Interface,
	tokenAuth *service.JWTokenService,
	userStorage storage.UserStorage,
	listStorage storage.ListStorage,
) *UserHandler {
	return &UserHandler{
		logger:      log,
		tokenAuth:   tokenAuth,
		userStorage: userStorage,
		listStorage: listStorage,
	}
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handlers.CreateUser"

		log := logger.LogWithRequest(h.logger, op, r)

		user := &models.User{}
		if err := DecodeAndValidateJSON(w, r, log, user); err != nil {
			return
		}

		now := time.Now().UTC()

		newUser := models.User{
			ID:        ksuid.New().String(),
			Email:     user.Email,
			Password:  user.Password,
			UpdatedAt: &now,
		}

		// Create the user
		err := h.userStorage.CreateUser(r.Context(), newUser)
		if errors.Is(err, c.ErrUserAlreadyExists) {
			handleResponseError(w, r, log, http.StatusBadRequest, c.ErrUserAlreadyExists, slog.String(c.EmailKey, user.Email))
			return
		}
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToCreateUser, err)
			return
		}

		/*
			// Create "Inbox" list
			listID := ksuid.New().String()
			now = time.Now().UTC()

			newList := models.List{
				ID:        listID,
				Title:     "Inbox",
				UserID:    newUser.ID,
				UpdatedAt: &now,
			}

			err = h.listStorage.CreateList(r.Context(), newList)
			if err != nil {
				handleInternalServerError(w, r, log, constants.ErrFailedToCreateList, err)
				return
			}
		*/

		// Register user device
		device, err := h.registerDevice(r, newUser.ID)
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToRegisterDevice, err)
			return
		}

		// Create session
		tokens, expiresAt, err := h.createSession(r.Context(), newUser.ID, device.ID, h.tokenAuth.RefreshTokenTTL)
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToCreateSession, err)
			return
		}

		additionalFields := map[string]string{c.UserIDKey: newUser.ID}
		tokenData := service.TokenData{
			AccessToken:      tokens.AccessToken,
			RefreshToken:     tokens.RefreshToken,
			Domain:           h.tokenAuth.RefreshTokenCookieDomain,
			Path:             h.tokenAuth.RefreshTokenCookiePath,
			ExpiresAt:        expiresAt,
			HttpOnly:         true,
			AdditionalFields: additionalFields,
		}

		log.Info("user and tokens created", slog.String(c.UserIDKey, newUser.ID), slog.Any(c.TokensKey, tokens))
		service.SendTokensToWeb(w, tokenData, http.StatusCreated)
	}
}

func (h *UserHandler) LoginWithPassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handlers.LoginWithPassword"

		log := logger.LogWithRequest(h.logger, op, r)

		userInput := &models.User{}

		if err := DecodeAndValidateJSON(w, r, log, userInput); err != nil {
			return
		}

		userDB, err := h.userStorage.GetUserCredentials(r.Context(), userInput)
		if errors.Is(err, c.ErrUserNotFound) {
			handleResponseError(w, r, log, http.StatusUnauthorized, c.ErrInvalidCredentials, slog.String(c.EmailKey, userInput.Email))
			return
		}
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToGetData, err)
			return
		}

		// TODO: add validate the password using bcrypt

		// Check if device exists (if not, register it)
		device, err := h.checkDevice(r, userDB.ID)
		if errors.Is(err, c.ErrUserDeviceNotFound) {
			device, err = h.registerDevice(r, userDB.ID)
			if err != nil {
				handleInternalServerError(w, r, log, c.ErrFailedToRegisterDevice, err)
				return
			}
		}
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToCheckDevice, err)
			return
		}

		// Create session
		tokens, expiresAt, err := h.createSession(r.Context(), userDB.ID, device.ID, h.tokenAuth.RefreshTokenTTL)
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToCreateSession, err)
			return
		}

		additionalFields := map[string]string{c.UserIDKey: userDB.ID}
		tokenData := service.TokenData{
			AccessToken:      tokens.AccessToken,
			RefreshToken:     tokens.RefreshToken,
			Domain:           h.tokenAuth.RefreshTokenCookieDomain,
			Path:             h.tokenAuth.RefreshTokenCookiePath,
			ExpiresAt:        expiresAt,
			HttpOnly:         true,
			AdditionalFields: additionalFields,
		}

		log.Info(
			"user logged in, tokens created",
			slog.String(c.UserIDKey, userDB.ID),
			slog.Any(c.TokensKey, tokens),
		)
		service.SendTokensToWeb(w, tokenData, http.StatusOK)
	}
}

// TODO: refactor it when we move jwtoken to grpc (use as a reference Aooth)
func (h *UserHandler) RefreshJWTTokens() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handlers.RefreshJWTTokens"

		log := logger.LogWithRequest(h.logger, op, r)

		refreshToken, err := service.FindRefreshToken(r)
		if err != nil {
			handleResponseError(w, r, log, http.StatusUnauthorized, c.ErrFailedToGetRefreshToken, err)
			return
		}

		// Get session by refresh token
		session, err := h.userStorage.GetSessionByRefreshToken(r.Context(), refreshToken)
		if errors.Is(err, c.ErrSessionNotFound) {
			handleResponseError(w, r, log, http.StatusUnauthorized, c.ErrSessionNotFound, slog.String(c.RefreshTokenKey, refreshToken))
			return
		}
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToGetData, err)
			return
		}

		// Check if refresh token is expired
		if session.ExpiresAt.Before(time.Now()) {
			handleResponseError(w, r, log, http.StatusUnauthorized, c.ErrRefreshTokenExpired, slog.String(c.UserIDKey, session.UserID))
			return
		}

		// Check if device exists (if not, register it)
		device, err := h.checkDevice(r, session.UserID)
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrUserDeviceNotFound, err)
			return
		}

		// Create new tokens
		tokens, expiresAt, err := h.createSession(r.Context(), session.UserID, device.ID, h.tokenAuth.RefreshTokenTTL)
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToCreateSession, err)
			return
		}

		tokenData := service.TokenData{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			Domain:       h.tokenAuth.RefreshTokenCookieDomain,
			Path:         h.tokenAuth.RefreshTokenCookiePath,
			ExpiresAt:    expiresAt,
			HttpOnly:     true,
		}

		log.Info("tokens created", slog.Any(c.TokensKey, tokens))
		service.SendTokensToWeb(w, tokenData, http.StatusOK)
	}
}

func (h *UserHandler) checkDevice(r *http.Request, userID string) (models.UserDevice, error) {

	device, err := h.userStorage.GetUserDevice(r.Context(), userID, r.UserAgent())
	if errors.Is(err, c.ErrUserDeviceNotFound) {
		return models.UserDevice{}, c.ErrUserDeviceNotFound
	}
	if err != nil {
		return models.UserDevice{}, err
	}

	return device, nil
}

func (h *UserHandler) registerDevice(r *http.Request, userID string) (models.UserDevice, error) {
	ip := r.RemoteAddr
	ip = strings.Split(ip, ":")[0]

	device := models.UserDevice{
		ID:         ksuid.New().String(),
		UserID:     userID,
		UserAgent:  r.UserAgent(),
		IP:         ip,
		Detached:   false,
		DetachedAt: nil,
	}

	err := h.userStorage.AddDevice(r.Context(), device)
	if err != nil {
		return models.UserDevice{}, err
	}

	return device, nil
}

// TODO: Move sessions from Postgres to Redis
func (h *UserHandler) createSession(
	ctx context.Context,
	userID, deviceID string,
	refreshTokenTTL time.Duration,
) (models.TokenResponse, time.Time, error) {
	var (
		resp models.TokenResponse
		err  error
	)

	additionalClaims := map[string]interface{}{c.ContextUserID: userID}

	resp.AccessToken, err = h.tokenAuth.NewAccessToken(additionalClaims)
	if err != nil {
		return resp, time.Time{}, err
	}

	resp.RefreshToken, err = h.tokenAuth.NewRefreshToken()
	if err != nil {
		return resp, time.Time{}, err
	}

	expiresAt := time.Now().Add(refreshTokenTTL)

	session := models.Session{
		RefreshToken: resp.RefreshToken,
		ExpiresAt:    &expiresAt,
	}

	err = h.userStorage.SaveSession(ctx, userID, deviceID, session)
	if err != nil {
		return resp, time.Time{}, err
	}

	return resp, expiresAt, nil
}

// Logout
// TODO: add logout
func (h *UserHandler) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handlers.Logout"

		log := logger.LogWithRequest(h.logger, op, r)

		_, claims, err := service.GetTokenFromContext(r.Context())
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToGetAccessToken, err)
			return
		}
		userID := claims[c.ContextUserID].(string)

		device, err := h.checkDevice(r, userID)
		if errors.Is(err, c.ErrUserDeviceNotFound) {
			handleResponseError(w, r, log, http.StatusUnauthorized, c.ErrUserDeviceNotFound)
			return
		}
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToCheckDevice, err)
			return
		}

		err = h.userStorage.RemoveSession(r.Context(), userID, device.ID)
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToRemoveSession, err)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "refreshToken",
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
			MaxAge:   -1,
		})

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("Logged out successfully"))
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToWriteResponse, err)
			return
		}
	}
}

// GetUser get a user by ID
func (h *UserHandler) GetUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handlers.GetUser"

		log := logger.LogWithRequest(h.logger, op, r)

		_, claims, err := service.GetTokenFromContext(r.Context())
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToGetAccessToken, err)
			return
		}
		userID := claims[c.ContextUserID].(string)

		user, err := h.userStorage.GetUser(r.Context(), userID)
		switch {
		case errors.Is(err, c.ErrUserNotFound):
			handleResponseError(w, r, log, http.StatusNotFound, c.ErrUserNotFound, slog.String(c.UserIDKey, userID))
			return
		case err != nil:
			handleInternalServerError(w, r, log, c.ErrFailedToGetData, err)
			return
		default:
			handleResponseSuccess(w, r, log, "user received", user, slog.String(c.UserIDKey, userID))
		}
	}
}

// GetUsers get a list of users
func (h *UserHandler) GetUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handlers.GetUsers"

		log := logger.LogWithRequest(h.logger, op, r)

		pagination := ParseLimitAndAfterID(r)

		users, err := h.userStorage.GetUsers(r.Context(), pagination)
		switch {
		case errors.Is(err, c.ErrNoUsersFound):
			handleResponseError(w, r, log, http.StatusNotFound, c.ErrNoUsersFound)
			return
		case err != nil:
			handleInternalServerError(w, r, log, c.ErrFailedToGetUsers, err)
			return
		default:
			handleResponseSuccess(w, r, log, "users found", users,
				slog.Int(c.CountKey, len(users)),
			)
		}
	}
}

// UpdateUser updates a user by ID
func (h *UserHandler) UpdateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handlers.UpdateUser"

		log := logger.LogWithRequest(h.logger, op, r)
		user := &models.UpdateUser{}

		_, claims, err := service.GetTokenFromContext(r.Context())
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToGetAccessToken, err)
			return
		}
		userID := claims[c.ContextUserID].(string)

		if err = DecodeAndValidateJSON(w, r, log, user); err != nil {
			return
		}

		updatedUser := models.User{
			ID:       userID,
			Email:    user.Email,
			Password: user.Password,
		}

		err = h.userStorage.UpdateUser(r.Context(), updatedUser)
		switch {
		case errors.Is(err, c.ErrUserNotFound):
			handleResponseError(w, r, log, http.StatusNotFound, c.ErrUserNotFound, slog.String(c.UserIDKey, userID))
			return
		case errors.Is(err, c.ErrEmailAlreadyTaken):
			handleResponseError(w, r, log, http.StatusBadRequest, c.ErrEmailAlreadyTaken, slog.String(c.EmailKey, user.Email))
			return
		case errors.Is(err, c.ErrNoChangesDetected):
			handleResponseError(w, r, log, http.StatusBadRequest, c.ErrNoChangesDetected, slog.String(c.UserIDKey, userID))
			return
		case errors.Is(err, c.ErrNoPasswordChangesDetected):
			handleResponseError(w, r, log, http.StatusBadRequest, c.ErrNoPasswordChangesDetected, slog.String(c.UserIDKey, userID))
			return
		case err != nil:
			handleInternalServerError(w, r, log, c.ErrFailedToUpdateUser, slog.String(c.UserIDKey, userID))
			return
		default:
			handleResponseSuccess(w, r, log, "user updated", models.User{ID: userID}, slog.String(c.UserIDKey, userID))
		}
	}
}

// DeleteUser deletes a user by ID
func (h *UserHandler) DeleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "user.handlers.DeleteUser"

		log := logger.LogWithRequest(h.logger, op, r)

		_, claims, err := service.GetTokenFromContext(r.Context())
		if err != nil {
			handleInternalServerError(w, r, log, c.ErrFailedToGetAccessToken, err)
			return
		}
		userID := claims[c.ContextUserID].(string)

		err = h.userStorage.DeleteUser(r.Context(), userID)
		switch {
		case errors.Is(err, c.ErrUserNotFound):
			handleResponseError(w, r, log, http.StatusNotFound, c.ErrUserNotFound, slog.String(c.UserIDKey, userID))
			return
		case err != nil:
			handleInternalServerError(w, r, log, c.ErrFailedToDeleteUser, err)
			return
		default:
			handleResponseSuccess(w, r, log, "user deleted", models.User{ID: userID}, slog.String(c.UserIDKey, userID))
		}
	}
}
