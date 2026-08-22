package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"forum/internal/config"
	"forum/internal/database"
	"forum/internal/repository"
	"forum/internal/service"
	sessionpkg "forum/internal/session"
	"forum/internal/web"
	"forum/internal/web/handler"
	"forum/internal/web/middleware"
	"forum/internal/web/view"
)

const shutdownTimeout = 5 * time.Second

func Run(ctx context.Context, cfg config.Config) error {
	appHandler, cleanup, err := buildHandler(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	server := &http.Server{
		Addr:    cfg.Address,
		Handler: appHandler,
	}

	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- normalizeServerError(
			server.ListenAndServe(),
		)
	}()

	select {
	case err := <-serverErrors:
		return err

	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf(
				"shutdown server: %w",
				err,
			)
		}

		err := <-serverErrors
		return err
	}
}

func buildHandler(
	cfg config.Config,
) (http.Handler, func(), error) {

	databaseDir := filepath.Dir(cfg.DatabasePath)

	if err := os.MkdirAll(databaseDir, 0755); err != nil {
		return nil, nil, fmt.Errorf(
			"create database directory: %w",
			err,
		)
	}
	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"open database: %w",
			err,
		)
	}

	cleanup := func() {
		db.Close()
	}

	if err := database.Migrate(
		db,
		resolveProjectPath("migrations"),
	); err != nil {
		cleanup()

		return nil, nil, fmt.Errorf(
			"migrate database: %w",
			err,
		)
	}

	users := repository.NewUserRepository(db)
	sessions := repository.NewSessionRepository(db)
	categories := repository.NewCategoryRepository(db)
	posts := repository.NewPostRepository(db)
	comments := repository.NewCommentRepository(db)
	reactions := repository.NewReactionRepository(db)

	passwords := service.NewPasswordManager()

	authService := service.NewAuthService(
		users,
		passwords,
	)

	loginService := service.NewLoginService(
		users,
		passwords,
		sessions,
		cfg.SessionDuration,
	)

	postService := service.NewPostService(
		posts,
	)

	commentService := service.NewCommentService(
		comments,
	)

	reactionService := service.NewReactionService(
		reactions,
	)

	sessionManager := sessionpkg.NewManager(
		cfg.CookieName,
		cfg.SessionDuration,
		cfg.SecureCookie,
	)

	renderer, err := view.NewRenderer(
		resolveProjectPath("templates"),
	)
	if err != nil {
		cleanup()

		return nil, nil, fmt.Errorf(
			"create renderer: %w",
			err,
		)
	}

	registerHandler := handler.NewRegisterHandler(
		authService,
		renderer,
	)

	loginHandler := handler.NewLoginHandler(
		loginService,
		sessionManager,
		renderer,
	)

	logoutHandler := handler.NewLogoutHandler(
		loginService,
		sessionManager,
	)

	homeHandler := handler.NewHomeHandler(
		posts,
		renderer,
	)

	postDetailHandler := handler.NewPostDetailHandler(
		posts,
		renderer,
	)

	postCreationHandler := handler.NewPostCreationHandler(
		postService,
		categories,
		renderer,
	)

	commentHandler := handler.NewCommentSubmissionHandler(
		commentService,
	)

	postReactionHandler := handler.NewPostReactionHandler(
		reactionService,
	)

	commentReactionHandler := handler.NewCommentReactionHandler(
		reactionService,
		comments,
	)

	router := web.NewForumRouter(
		web.Handlers{
			Home:            homeHandler,
			Register:        registerHandler,
			Login:           loginHandler,
			Logout:          logoutHandler,
			PostCreation:    postCreationHandler,
			PostDetail:      postDetailHandler,
			CommentCreate:   commentHandler,
			PostReaction:    postReactionHandler,
			CommentReaction: commentReactionHandler,
			Static: http.FileServer(
				http.Dir(resolveProjectPath("static")),
			),
		},
	)

	authenticate := middleware.NewAuthentication(
		sessionManager,
		sessions,
		users,
	)

	appHandler := authenticate(router)

	logger := log.New(
		os.Stdout,
		"",
		log.LstdFlags,
	)

	appHandler = web.WithMiddleware(
		logger,
		appHandler,
	)

	return appHandler, cleanup, nil
}

func normalizeServerError(err error) error {
	if errors.Is(
		err,
		http.ErrServerClosed,
	) {
		return nil
	}

	return err
}

func resolveProjectPath(name string) string {
	if _, err := os.Stat(name); err == nil {
		return name
	}

	return filepath.Join("..", "..", name)
}
