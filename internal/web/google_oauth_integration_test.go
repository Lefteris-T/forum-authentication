package web

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"forum/internal/database"
	"forum/internal/oauth"
	"forum/internal/repository"
	"forum/internal/service"
	sessionpkg "forum/internal/session"
	"forum/internal/web/handler"
	"forum/internal/web/middleware"
)

func TestIntegrationGoogleOAuthLoginAndLogout(t *testing.T) {
	providerServer := newFakeGoogleProvider(t)
	defer providerServer.Close()

	db, err := database.Open(filepath.Join(t.TempDir(), "forum.db"))
	if err != nil {
		t.Fatalf("database.Open(): %v", err)
	}
	defer db.Close()

	if err := database.Migrate(
		db,
		filepath.Join("..", "..", "migrations"),
	); err != nil {
		t.Fatalf("database.Migrate(): %v", err)
	}

	users := repository.NewUserRepository(db)
	oauthAccounts := repository.NewOAuthAccountRepository(db)
	sessions := repository.NewSessionRepository(db)
	sessionManager := sessionpkg.NewManager("forum_session", time.Hour, false)
	loginService := service.NewLoginService(
		users,
		service.NewPasswordManager(),
		sessions,
		time.Hour,
	)
	oauthLoginService := service.NewOAuthLoginService(oauthAccounts, users)
	oauthSuccess := handler.NewOAuthSuccessHandler(
		oauthLoginService,
		sessionManager,
		loginService,
	)

	providerConfig := oauth.ProviderConfig{
		ClientID:              "google-client-id",
		ClientSecret:          "google-client-secret",
		RedirectURL:           "http://forum.example/auth/google/callback",
		AuthorizationEndpoint: providerServer.URL + "/authorize",
		TokenEndpoint:         providerServer.URL + "/token",
		UserEndpoint:          providerServer.URL + "/userinfo",
		Client:                providerServer.Client(),
	}
	googleProvider := oauth.NewGoogleProvider(providerConfig)
	stateStore := oauth.NewOAuthStateStore()

	googleStart := oauth.NewAuthorizationHandler(
		googleProvider,
		"google",
		stateStore,
		"google_oauth_state",
		false,
	)
	googleCallback := oauth.NewCallbackHandler(
		googleProvider,
		"google",
		stateStore,
		"google_oauth_state",
		false,
		oauthSuccess.Handle,
	)

	home := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.CurrentUser(r.Context())
		if !ok {
			_, _ = w.Write([]byte("guest"))
			return
		}

		_, _ = w.Write([]byte(user.Username))
	})
	logout := handler.NewLogoutHandler(loginService, sessionManager)

	router := NewForumRouter(Handlers{
		Home:                home,
		Logout:              logout,
		GoogleOAuth:         googleStart,
		GoogleOAuthCallback: googleCallback,
	})
	authenticate := middleware.NewAuthentication(
		sessionManager,
		sessions,
		users,
	)
	forumServer := httptest.NewServer(authenticate(router))
	defer forumServer.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New(): %v", err)
	}
	browser := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	completeGoogleLogin(t, browser, forumServer.URL)
	assertGoogleAuthenticated(t, browser, forumServer.URL, "Google-User")
	assertRowCount(t, db, "users", 1)
	assertRowCount(t, db, "oauth_accounts", 1)
	assertRowCount(t, db, "sessions", 1)

	completeGoogleLogin(t, browser, forumServer.URL)
	assertGoogleAuthenticated(t, browser, forumServer.URL, "Google-User")
	assertRowCount(t, db, "users", 1)
	assertRowCount(t, db, "oauth_accounts", 1)
	assertRowCount(t, db, "sessions", 1)

	logoutResponse, err := browser.Post(forumServer.URL+"/logout", "", nil)
	if err != nil {
		t.Fatalf("POST /logout: %v", err)
	}
	defer logoutResponse.Body.Close()

	if logoutResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf(
			"logout status = %d, want %d",
			logoutResponse.StatusCode,
			http.StatusSeeOther,
		)
	}

	assertGoogleAuthenticated(t, browser, forumServer.URL, "guest")
	assertRowCount(t, db, "sessions", 0)
}

func newFakeGoogleProvider(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("token method = %q, want %q", r.Method, http.MethodPost)
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("token ParseForm(): %v", err)
			return
		}

		if r.Form.Get("code") != "authorization-code" {
			t.Errorf("token code = %q", r.Form.Get("code"))
		}

		if r.Form.Get("code_verifier") == "" {
			t.Error("token code_verifier is empty")
		}

		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "google-access-token",
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer google-access-token" {
			t.Errorf("userinfo Authorization = %q", r.Header.Get("Authorization"))
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":            "google-subject-123",
			"email":          "google@example.com",
			"email_verified": true,
			"name":           "Google User",
		})
	})

	return httptest.NewServer(mux)
}

func completeGoogleLogin(
	t *testing.T,
	browser *http.Client,
	forumURL string,
) {
	t.Helper()

	startResponse, err := browser.Get(forumURL + "/auth/google")
	if err != nil {
		t.Fatalf("GET /auth/google: %v", err)
	}
	defer startResponse.Body.Close()

	if startResponse.StatusCode != http.StatusFound {
		t.Fatalf(
			"oauth start status = %d, want %d",
			startResponse.StatusCode,
			http.StatusFound,
		)
	}

	authorizationURL, err := url.Parse(startResponse.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	state := authorizationURL.Query().Get("state")
	if state == "" {
		t.Fatal("authorization state is empty")
	}

	callbackURL := forumURL + "/auth/google/callback?code=authorization-code&state=" +
		url.QueryEscape(state)
	callbackResponse, err := browser.Get(callbackURL)
	if err != nil {
		t.Fatalf("GET Google callback: %v", err)
	}
	defer callbackResponse.Body.Close()

	if callbackResponse.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(callbackResponse.Body)
		t.Fatalf(
			"callback status = %d, want %d; body=%q",
			callbackResponse.StatusCode,
			http.StatusSeeOther,
			strings.TrimSpace(string(body)),
		)
	}

	if callbackResponse.Header.Get("Location") != "/" {
		t.Fatalf(
			"callback Location = %q, want %q",
			callbackResponse.Header.Get("Location"),
			"/",
		)
	}
}

func assertGoogleAuthenticated(
	t *testing.T,
	browser *http.Client,
	forumURL string,
	want string,
) {
	t.Helper()

	response, err := browser.Get(forumURL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read home response: %v", err)
	}

	if string(body) != want {
		t.Fatalf("home identity = %q, want %q", string(body), want)
	}
}

func assertRowCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	if count != want {
		t.Fatalf("%s count = %d, want %d", table, count, want)
	}
}
