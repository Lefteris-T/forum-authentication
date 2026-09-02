package oauth

import (
	"net/http"
	"time"
)

type authorizationURLProvider interface {
	AuthorizationURL(state, challenge string) (string, error)
}

type AuthorizationHandler struct {
	provider     authorizationURLProvider
	providerName string
	store        *OAuthStateStore
	cookieName   string
	secure       bool
}

func NewAuthorizationHandler(
	provider authorizationURLProvider,
	providerName string,
	store *OAuthStateStore,
	cookieName string,
	secure bool,
) http.Handler {
	return &AuthorizationHandler{
		provider:     provider,
		providerName: providerName,
		store:        store,
		cookieName:   cookieName,
		secure:       secure,
	}
}

func NewGitHubAuthorizationHandler(
	cfg ProviderConfig,
	store *OAuthStateStore,
	cookieName string,
	secure bool,
) http.Handler {
	return NewAuthorizationHandler(
		NewGitHubProvider(cfg),
		"github",
		store,
		cookieName,
		secure,
	)
}

func (h *AuthorizationHandler) ServeHTTP(
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
		h.providerName,
		verifier,
		time.Now().Add(10*time.Minute),
	)

	WriteStateCookie(
		w,
		h.cookieName,
		state,
		h.secure,
	)

	redirectURL, err := h.provider.AuthorizationURL(state, challenge)
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
