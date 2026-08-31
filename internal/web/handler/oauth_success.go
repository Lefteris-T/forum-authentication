package handler

import (
	"errors"
	"net/http"

	"forum/internal/model"
	"forum/internal/oauth"
	"forum/internal/service"
)

type oauthLoginService interface {
	Login(oauthUser oauth.User) (model.User, error)
}

type oauthSessionManager interface {
	Create(w http.ResponseWriter) (string, error)
	Clear(w http.ResponseWriter)
}

type oauthSessionService interface {
	CreateSession(sessionID string, userID int64) error
}

type OAuthSuccessHandler struct {
	oauthLogin     oauthLoginService
	sessionManager oauthSessionManager
	sessionService oauthSessionService
}

func NewOAuthSuccessHandler(
	oauthLogin oauthLoginService,
	sessionManager oauthSessionManager,
	sessionService oauthSessionService,
) *OAuthSuccessHandler {
	return &OAuthSuccessHandler{
		oauthLogin:     oauthLogin,
		sessionManager: sessionManager,
		sessionService: sessionService,
	}
}

func (h *OAuthSuccessHandler) Handle(
	w http.ResponseWriter,
	r *http.Request,
	oauthUser oauth.User,
) {
	user, err := h.oauthLogin.Login(oauthUser)
	if err != nil {
		if errors.Is(err, service.ErrOAuthEmailConflict) {
			http.Error(
				w,
				"OAuth account conflict",
				http.StatusConflict,
			)
			return
		}

		http.Error(
			w,
			"OAuth login failed",
			http.StatusInternalServerError,
		)
		return
	}

	sessionID, err := h.sessionManager.Create(w)
	if err != nil {
		http.Error(
			w,
			"Could not create session",
			http.StatusInternalServerError,
		)
		return
	}

	err = h.sessionService.CreateSession(
		sessionID,
		user.ID,
	)
	if err != nil {
		// Cookie was already written, but DB session failed.
		// Remove the now-useless browser cookie.
		h.sessionManager.Clear(w)

		http.Error(
			w,
			"Could not create session",
			http.StatusInternalServerError,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/",
		http.StatusSeeOther,
	)
}
