package oauth

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type GitHubAuthorizationHandler struct {
	cfg        ProviderConfig
	store      *OAuthStateStore
	cookieName string
	secure     bool
}

func NewGitHubAuthorizationHandler(
	cfg ProviderConfig,
	store *OAuthStateStore,
	cookieName string,
	secure bool,
) http.Handler {
	return &GitHubAuthorizationHandler{
		cfg:        cfg,
		store:      store,
		cookieName: cookieName,
		secure:     secure,
	}
}

func (h *GitHubAuthorizationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	state, err := GenerateState()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.store.Save(
		state,
		"github",
		verifier,
		time.Now().Add(10*time.Minute),
	)

	WriteStateCookie(
		w,
		h.cookieName,
		state,
		h.secure,
	)

	redirectURL, err := h.authorizationURL(state, challenge)
	if err != nil {
		ClearStateCookie(
			w,
			h.cookieName,
			h.secure,
		)

		http.Error(
			w,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}

	http.Redirect(
		w,
		r,
		redirectURL,
		http.StatusFound,
	)
}

func (h *GitHubAuthorizationHandler) authorizationURL(
	state string,
	challenge string,
) (string, error) {
	u, err := url.Parse(h.cfg.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf(
			"parse github authorization endpoint: %w",
			err,
		)
	}

	query := u.Query()

	query.Set("client_id", h.cfg.ClientID)
	query.Set("redirect_uri", h.cfg.RedirectURL)
	query.Set("scope", "user:email")
	query.Set("state", state)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")

	u.RawQuery = query.Encode()

	return u.String(), nil
}
