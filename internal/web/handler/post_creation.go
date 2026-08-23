package handler

import (
	"net/http"
	"strconv"

	"forum/internal/model"
	"forum/internal/validation"
	"forum/internal/web/middleware"
	"forum/internal/web/view"
)

// PostCreationService is the validated post write required by the handler.
type PostCreationService interface {
	Create(
		authorID int64,
		input validation.PostInput,
	) (int64, error)
}

// CategoryReader supplies choices and validates submitted category identifiers.
type CategoryReader interface {
	All() ([]model.Category, error)
}

// PostCreationHandler renders the protected form and processes submissions.
type PostCreationHandler struct {
	service    PostCreationService
	categories CategoryReader
	renderer   *view.Renderer
}

type newPostPageData struct {
	Categories  []model.Category
	CurrentUser *model.User
}

// NewPostCreationHandler constructs post-creation HTTP behavior.
func NewPostCreationHandler(
	service PostCreationService,
	categories CategoryReader,
	renderer *view.Renderer,
) *PostCreationHandler {
	return &PostCreationHandler{
		service:    service,
		categories: categories,
		renderer:   renderer,
	}
}

// ServeHTTP requires a current user for both viewing and submitting the form.
func (h *PostCreationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/posts/new":
		h.handleGet(w, r)

	case r.Method == http.MethodPost && r.URL.Path == "/posts":
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

func (h *PostCreationHandler) handleGet(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(
			w,
			http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized,
		)
		return
	}

	categories, err := h.categories.All()
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	data := newPostPageData{
		Categories:  categories,
		CurrentUser: &user,
	}

	if err := h.renderer.Render(
		w,
		http.StatusOK,
		"new_post.html",
		data,
	); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

func (h *PostCreationHandler) handlePost(
	w http.ResponseWriter,
	r *http.Request,
) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(
			w,
			http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized,
		)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	categoryValues := r.Form["category"]

	categoryIDs := make(
		[]int64,
		0,
		len(categoryValues),
	)

	for _, value := range categoryValues {
		id, err := strconv.ParseInt(
			value,
			10,
			64,
		)
		if err != nil || id <= 0 {
			http.Error(
				w,
				http.StatusText(http.StatusBadRequest),
				http.StatusBadRequest,
			)
			return
		}

		categoryIDs = append(
			categoryIDs,
			id,
		)
	}

	input := validation.PostInput{
		Title:       r.FormValue("title"),
		Body:        r.FormValue("body"),
		CategoryIDs: categoryIDs,
	}

	postID, err := h.service.Create(
		user.ID,
		input,
	)
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/posts/"+strconv.FormatInt(postID, 10),
		http.StatusSeeOther,
	)
}
