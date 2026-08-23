package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/model"
	"forum/internal/repository"
	"forum/internal/web/middleware"
	"forum/internal/web/view"
)

// PostReader provides the public and current-user-specific forum read models.
type PostReader interface {
	List() ([]repository.PostListItem, error)

	ListByCategory(
		categoryID int64,
	) ([]repository.PostListItem, error)

	ListByAuthor(
		authorID int64,
	) ([]repository.PostListItem, error)

	ListLikedByUser(
		userID int64,
	) ([]repository.PostListItem, error)

	Detail(id int64) (repository.PostDetail, error)
}

// HomeHandler lists posts and applies category, created, or liked filters.
type HomeHandler struct {
	posts    PostReader
	renderer *view.Renderer
}

type homePageData struct {
	Posts       []repository.PostListItem
	CurrentUser *model.User
}

// NewHomeHandler constructs the forum listing endpoint.
func NewHomeHandler(
	posts PostReader,
	renderer *view.Renderer,
) *HomeHandler {
	return &HomeHandler{
		posts:    posts,
		renderer: renderer,
	}
}

// ServeHTTP keeps category filtering public while requiring authentication for
// the created and liked filters.
func (h *HomeHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")

		http.Error(
			w,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed,
		)
		return
	}

	categoryValue := r.URL.Query().Get("category")
	filterValue := r.URL.Query().Get("filter")

	var posts []repository.PostListItem
	var err error

	switch {
	case filterValue == "created":
		user, ok := middleware.CurrentUser(r.Context())
		if !ok {
			http.Error(
				w,
				http.StatusText(http.StatusUnauthorized),
				http.StatusUnauthorized,
			)
			return
		}

		posts, err = h.posts.ListByAuthor(user.ID)

	case filterValue == "liked":
		user, ok := middleware.CurrentUser(r.Context())
		if !ok {
			http.Error(
				w,
				http.StatusText(http.StatusUnauthorized),
				http.StatusUnauthorized,
			)
			return
		}

		posts, err = h.posts.ListLikedByUser(user.ID)

	case categoryValue != "":
		categoryID, parseErr := strconv.ParseInt(
			categoryValue,
			10,
			64,
		)
		if parseErr != nil || categoryID <= 0 {
			http.Error(
				w,
				http.StatusText(http.StatusBadRequest),
				http.StatusBadRequest,
			)
			return
		}

		posts, err = h.posts.ListByCategory(categoryID)

	default:
		posts, err = h.posts.List()
	}

	if errors.Is(err, repository.ErrCategoryNotFound) {
		http.Error(
			w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound,
		)
		return
	}

	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	data := homePageData{
		Posts: posts,
	}

	if user, ok := middleware.CurrentUser(r.Context()); ok {
		data.CurrentUser = &user
	}

	if err := h.renderer.Render(
		w,
		http.StatusOK,
		"home.html",
		data,
	); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

// PostDetailHandler renders one public post with comments and reaction counts.
type PostDetailHandler struct {
	posts    PostReader
	renderer *view.Renderer
}

type postDetailPageData struct {
	Post        repository.PostDetail
	CurrentUser *model.User
}

// NewPostDetailHandler constructs the post-detail endpoint.
func NewPostDetailHandler(
	posts PostReader,
	renderer *view.Renderer,
) *PostDetailHandler {
	return &PostDetailHandler{
		posts:    posts,
		renderer: renderer,
	}
}

// ServeHTTP validates the resource identifier and maps missing posts to 404.
func (h *PostDetailHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")

		http.Error(
			w,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed,
		)
		return
	}

	id, err := parsePostID(r.URL.Path)
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusBadRequest),
			http.StatusBadRequest,
		)
		return
	}

	post, err := h.posts.Detail(id)
	if errors.Is(err, repository.ErrPostNotFound) {
		http.Error(
			w,
			http.StatusText(http.StatusNotFound),
			http.StatusNotFound,
		)
		return
	}
	if err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	data := postDetailPageData{
		Post: post,
	}

	if user, ok := middleware.CurrentUser(r.Context()); ok {
		data.CurrentUser = &user
	}

	if err := h.renderer.Render(
		w,
		http.StatusOK,
		"post.html",
		data,
	); err != nil {
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

func parsePostID(path string) (int64, error) {
	const prefix = "/posts/"

	if !strings.HasPrefix(path, prefix) {
		return 0, strconv.ErrSyntax
	}

	rawID := strings.TrimPrefix(
		path,
		prefix,
	)

	if rawID == "" || strings.Contains(rawID, "/") {
		return 0, strconv.ErrSyntax
	}

	id, err := strconv.ParseInt(
		rawID,
		10,
		64,
	)
	if err != nil || id <= 0 {
		return 0, strconv.ErrSyntax
	}

	return id, nil
}
