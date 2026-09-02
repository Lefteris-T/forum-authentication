package oauth

import (
	"crypto/subtle"
	"net/http"
)

type OAuthSuccessHandler func(
	w http.ResponseWriter,
	r *http.Request,
	user User,
)

type CallbackHandler struct {
	provider     Provider
	providerName string
	store        *OAuthStateStore
	cookieName   string
	secure       bool
	onSuccess    OAuthSuccessHandler
}

func NewCallbackHandler(
	provider Provider,
	providerName string,
	store *OAuthStateStore,
	cookieName string,
	secure bool,
	onSuccess OAuthSuccessHandler,
) http.Handler {
	return &CallbackHandler{
		provider:     provider,
		providerName: providerName,
		store:        store,
		cookieName:   cookieName,
		secure:       secure,
		onSuccess:    onSuccess,
	}
}

func NewGitHubCallbackHandler(
	provider Provider,
	store *OAuthStateStore,
	cookieName string,
	secure bool,
	onSuccess OAuthSuccessHandler,
) http.Handler {
	return NewCallbackHandler(
		provider,
		"github",
		store,
		cookieName,
		secure,
		onSuccess,
	)
}

func (h *CallbackHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	ClearStateCookie(
		w,
		h.cookieName,
		h.secure,
	)

	if providerError := r.URL.Query().Get("error"); providerError != "" {
		http.Error(
			w,
			"Invalid OAuth callback",
			http.StatusBadRequest,
		)
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(
			w,
			"Invalid OAuth callback",
			http.StatusBadRequest,
		)
		return
	}

	cookie, err := r.Cookie(h.cookieName)
	if err != nil {
		http.Error(
			w,
			"Invalid OAuth callback",
			http.StatusBadRequest,
		)
		return
	}

	if !constantTimeEqual(state, cookie.Value) {
		http.Error(
			w,
			"Invalid OAuth callback",
			http.StatusBadRequest,
		)
		return
	}

	flow, err := h.store.Consume(
		state,
		h.providerName,
	)
	if err != nil {
		http.Error(
			w,
			"Invalid OAuth callback",
			http.StatusBadRequest,
		)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(
			w,
			"Invalid OAuth callback",
			http.StatusBadRequest,
		)
		return
	}

	accessToken, err := h.provider.ExchangeCode(
		r.Context(),
		code,
		flow.Verifier,
	)
	if err != nil {
		http.Error(
			w,
			"OAuth provider error",
			http.StatusBadGateway,
		)
		return
	}

	user, err := h.provider.FetchUser(
		r.Context(),
		accessToken,
	)
	if err != nil {
		http.Error(
			w,
			"OAuth provider error",
			http.StatusBadGateway,
		)
		return
	}

	h.onSuccess(
		w,
		r,
		user,
	)
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(a),
		[]byte(b),
	) == 1
}
