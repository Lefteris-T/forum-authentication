package web

import (
	"forum/internal/web/middleware"
	"log"
	"net/http"
	"strings"
)

type Route struct {
	Methods []string
	Pattern string
	Handler http.Handler
}

type Handlers struct {
	Home            http.Handler
	Register        http.Handler
	Login           http.Handler
	Logout          http.Handler
	PostCreation    http.Handler
	PostDetail      http.Handler
	CommentCreate   http.Handler
	PostReaction    http.Handler
	CommentReaction http.Handler
}

func NewRouter(routes []Route) http.Handler {
	mux := http.NewServeMux()

	for _, route := range routes {
		mux.Handle(
			route.Pattern,
			methodHandler(
				route.Methods,
				route.Handler,
			),
		)
	}

	return mux
}

func methodHandler(
	allowedMethods []string,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(allowedMethods) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		for _, method := range allowedMethods {
			if r.Method == method {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set(
			"Allow",
			strings.Join(allowedMethods, ", "),
		)

		writeError(
			w,
			http.StatusMethodNotAllowed,
		)
	})
}
func writeError(
	w http.ResponseWriter,
	status int,
) {
	w.WriteHeader(status)

	_, _ = w.Write(
		[]byte(http.StatusText(status)),
	)
}

func NewForumRouter(h Handlers) http.Handler {
	return NewRouter([]Route{
		{
			Methods: []string{http.MethodGet},
			Pattern: "/",
			Handler: h.Home,
		},
		{
			Methods: []string{
				http.MethodGet,
				http.MethodPost,
			},
			Pattern: "/register",
			Handler: h.Register,
		},
		{
			Methods: []string{
				http.MethodGet,
				http.MethodPost,
			},
			Pattern: "/login",
			Handler: h.Login,
		},
		{
			Methods: []string{http.MethodPost},
			Pattern: "/logout",
			Handler: h.Logout,
		},
		{
			Methods: []string{http.MethodGet},
			Pattern: "/posts/new",
			Handler: h.PostCreation,
		},
		{

			Methods: []string{http.MethodPost},
			Pattern: "/posts",
			Handler: h.PostCreation,
		},
		{
			Methods: nil,
			Pattern: "/posts/",
			Handler: postRoutes(
				h.PostDetail,
				h.CommentCreate,
				h.PostReaction,
			),
		},
		{
			Methods: nil,
			Pattern: "/comments/",
			Handler: commentRoutes(
				h.CommentReaction,
			),
		},
	})
}

func postRoutes(
	detail http.Handler,
	commentCreate http.Handler,
	postReaction http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/posts/")
		parts := strings.Split(path, "/")

		switch {
		case len(parts) == 1 && parts[0] != "":
			methodHandler(
				[]string{http.MethodGet},
				detail,
			).ServeHTTP(w, r)

		case len(parts) == 2 && parts[0] != "" && parts[1] == "comments":
			methodHandler(
				[]string{http.MethodPost},
				commentCreate,
			).ServeHTTP(w, r)

		case len(parts) == 2 && parts[0] != "" && parts[1] == "react":
			methodHandler(
				[]string{http.MethodPost},
				postReaction,
			).ServeHTTP(w, r)

		default:
			writeError(
				w,
				http.StatusNotFound,
			)
		}
	})
}
func commentRoutes(
	commentReaction http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(
			r.URL.Path,
			"/comments/",
		)

		parts := strings.Split(path, "/")

		switch {
		case len(parts) == 2 &&
			parts[0] != "" &&
			parts[1] == "react":

			methodHandler(
				[]string{http.MethodPost},
				commentReaction,
			).ServeHTTP(w, r)

		default:
			writeError(
				w,
				http.StatusNotFound,
			)
		}
	})
}
func WithMiddleware(
	logger *log.Logger,
	next http.Handler,
) http.Handler {
	return middleware.RequestLogging(
		logger,
		middleware.Recovery(
			logger,
			next,
		),
	)
}
