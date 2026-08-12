package handler

import (
	"errors"
	"fmt"
	"net/http"

	"forum/internal/repository"
	"forum/internal/validation"
	"forum/internal/web/view"
)

type RegistrationService interface {
	Register(input validation.RegistrationInput) (int64, error)
}

type RegisterHandler struct {
	service  RegistrationService
	renderer *view.Renderer
}

type registerPageData struct {
	Email    string
	Username string
	Error    string
}

func NewRegisterHandler(
	service RegistrationService,
	renderer *view.Renderer,
) *RegisterHandler {
	return &RegisterHandler{
		service:  service,
		renderer: renderer,
	}
}

func (h *RegisterHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w)

	case http.MethodPost:
		h.handlePost(w, r)

	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(
			w,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed,
		)
	}
}

func (h *RegisterHandler) handleGet(w http.ResponseWriter) {
	if err := h.renderer.Render(
		w,
		http.StatusOK,
		"register.html",
		registerPageData{},
	); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

func (h *RegisterHandler) handlePost(
	w http.ResponseWriter,
	r *http.Request,
) {
	if err := r.ParseForm(); err != nil {
		h.renderForm(
			w,
			http.StatusBadRequest,
			registerPageData{
				Error: "Invalid form",
			},
		)
		return
	}

	input := validation.RegistrationInput{
		Email:    r.FormValue("email"),
		Username: r.FormValue("username"),
		Password: r.FormValue("password"),
	}

	_, err := h.service.Register(input)
	if err == nil {
		http.Redirect(
			w,
			r,
			"/login",
			http.StatusSeeOther,
		)
		return
	}

	data := registerPageData{
		Email:    input.Email,
		Username: input.Username,
		Error:    err.Error(),
	}

	switch {
	case errors.Is(err, repository.ErrEmailExists),
		errors.Is(err, repository.ErrUsernameExists):

		h.renderForm(
			w,
			http.StatusConflict,
			data,
		)

	default:
		h.renderForm(
			w,
			http.StatusBadRequest,
			data,
		)
	}
}

func (h *RegisterHandler) renderForm(
	w http.ResponseWriter,
	status int,
	data registerPageData,
) {
	if err := h.renderer.Render(
		w,
		status,
		"register.html",
		data,
	); err != nil {
		fmt.Println("render register form:", err)

		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}
