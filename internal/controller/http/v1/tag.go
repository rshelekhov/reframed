package v1

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/rshelekhov/reframed/internal/domain"
	"github.com/rshelekhov/reframed/pkg/constants/key"
	"github.com/rshelekhov/reframed/pkg/httpserver/middleware/jwtoken"
	"github.com/rshelekhov/reframed/pkg/logger"
	"log/slog"
	"net/http"
)

type tagController struct {
	logger  logger.Interface
	jwt     *jwtoken.TokenService
	usecase domain.TagUsecase
}

func NewTagRoutes(
	r *chi.Mux,
	log logger.Interface,
	jwt *jwtoken.TokenService,
	usecase domain.TagUsecase,
) {
	c := &tagController{
		logger:  log,
		jwt:     jwt,
		usecase: usecase,
	}

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(jwtoken.Verifier(jwt))
		r.Use(jwtoken.Authenticator())

		r.Get("/user/tags", c.GetTagsByUserID())
	})
}

func (c *tagController) GetTagsByUserID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "tag.controller.GetTagsByUserID"

		ctx := r.Context()
		log := logger.LogWithRequest(c.logger, op, r)
		userID := jwtoken.GetUserID(ctx).(string)

		tagsResp, err := c.usecase.GetTagsByUserID(ctx, userID)
		switch {
		case errors.Is(err, domain.ErrNoTagsFound):
			handleResponseError(w, r, log, http.StatusNotFound, domain.ErrNoTagsFound)
			return
		case err != nil:
			handleInternalServerError(w, r, log, domain.ErrFailedToGetData, err)
			return
		default:
			handleResponseSuccess(w, r, log, "tags found", tagsResp, slog.Int(key.Count, len(tagsResp)))
		}
	}
}
