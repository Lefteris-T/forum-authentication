package handler

import "net/http"

type LogoutHandler struct {
	service  LoginService
	sessions SessionManager
}

func NewLogoutHandler(
	service LoginService,
	sessions SessionManager,
) *LogoutHandler {
	return &LogoutHandler{
		service:  service,
		sessions: sessions,
	}
}

func (h *LogoutHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")

		http.Error(
			w,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed,
		)
		return
	}

	sessionID, ok := h.sessions.Read(r)
	if ok {
		if err := h.service.Logout(sessionID); err != nil {
			http.Error(
				w,
				http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError,
			)
			return
		}
	}

	h.sessions.Clear(w)

	http.Redirect(
		w,
		r,
		"/",
		http.StatusSeeOther,
	)
}
