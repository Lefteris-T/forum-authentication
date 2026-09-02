package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"forum/internal/oauth"
	"forum/internal/repository"
)

type integrationFakeOAuthProvider struct {
	name        string
	user        oauth.User
	exchangeErr error
	fetchErr    error
}

func (p *integrationFakeOAuthProvider) AuthorizationURL(
	state string,
	challenge string,
) (string, error) {
	values := url.Values{
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}

	return "https://" + p.name + ".example/authorize?" + values.Encode(), nil
}

func (p *integrationFakeOAuthProvider) ExchangeCode(
	ctx context.Context,
	code string,
	verifier string,
) (string, error) {
	if p.exchangeErr != nil {
		return "", p.exchangeErr
	}

	if code == "" || verifier == "" {
		return "", errors.New("missing code or verifier")
	}

	return p.name + "-access-token", nil
}

func (p *integrationFakeOAuthProvider) FetchUser(
	ctx context.Context,
	accessToken string,
) (oauth.User, error) {
	if p.fetchErr != nil {
		return oauth.User{}, p.fetchErr
	}

	if accessToken != p.name+"-access-token" {
		return oauth.User{}, errors.New("unexpected access token")
	}

	return p.user, nil
}

func TestIntegrationOAuthProvidersAuthenticateProtectedRoutesAndLogout(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		user     oauth.User
	}{
		{
			name:     "github",
			provider: "github",
			user: oauth.User{
				Provider:          "github",
				ProviderUserID:    "123456",
				VerifiedEmail:     "github@example.com",
				SuggestedUsername: "octocat",
			},
		},
		{
			name:     "google",
			provider: "google",
			user: oauth.User{
				Provider:          "google",
				ProviderUserID:    "google-subject",
				VerifiedEmail:     "google@example.com",
				SuggestedUsername: "Google User",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &integrationFakeOAuthProvider{
				name: tt.provider,
				user: tt.user,
			}
			env := newOAuthIntegrationEnv(t, tt.provider, provider)
			defer env.Server.Close()
			browser := newIntegrationBrowser(t)

			assertHTTPStatus(
				t,
				browser,
				http.MethodGet,
				env.Server.URL+"/posts/new",
				http.StatusUnauthorized,
			)

			completeOAuthLogin(t, browser, env.Server.URL, tt.provider)

			assertHTTPStatus(
				t,
				browser,
				http.MethodGet,
				env.Server.URL+"/posts/new",
				http.StatusOK,
			)
			assertRowCount(t, env.DB, "users", 1)
			assertRowCount(t, env.DB, "oauth_accounts", 1)
			assertRowCount(t, env.DB, "sessions", 1)

			logoutOAuthBrowser(t, browser, env.Server.URL)

			assertHTTPStatus(
				t,
				browser,
				http.MethodGet,
				env.Server.URL+"/posts/new",
				http.StatusUnauthorized,
			)
			assertRowCount(t, env.DB, "sessions", 0)
		})
	}
}

func TestIntegrationOAuthCallbackFailuresLeaveNoPartialAuthentication(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*integrationFakeOAuthProvider)
		callback   func(*testing.T, *http.Client, *integrationEnv) *http.Response
		wantStatus int
	}{
		{
			name: "malformed callback",
			callback: func(t *testing.T, browser *http.Client, env *integrationEnv) *http.Response {
				return requestOAuthCallback(t, browser, env.Server.URL, "google", "", "")
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "mismatched state",
			callback: func(t *testing.T, browser *http.Client, env *integrationEnv) *http.Response {
				_ = startOAuth(t, browser, env.Server.URL, "google")
				return requestOAuthCallback(
					t,
					browser,
					env.Server.URL,
					"google",
					"different-state",
					"authorization-code",
				)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "expired state",
			callback: func(t *testing.T, browser *http.Client, env *integrationEnv) *http.Response {
				env.OAuthStateStore.Save(
					"expired-state",
					"google",
					"pkce-verifier",
					time.Now().Add(-time.Minute),
				)
				return requestOAuthCallbackWithCookie(
					t,
					browser,
					env.Server.URL,
					"google",
					"expired-state",
					"authorization-code",
				)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "provider denial",
			callback: func(t *testing.T, browser *http.Client, env *integrationEnv) *http.Response {
				state := startOAuth(t, browser, env.Server.URL, "google")
				request, err := http.NewRequest(
					http.MethodGet,
					env.Server.URL+"/auth/google/callback?error=access_denied&state="+
						url.QueryEscape(state),
					nil,
				)
				if err != nil {
					t.Fatalf("create denial callback request: %v", err)
				}

				response, err := browser.Do(request)
				if err != nil {
					t.Fatalf("provider denial callback: %v", err)
				}
				return response
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "token exchange failure",
			configure: func(provider *integrationFakeOAuthProvider) {
				provider.exchangeErr = errors.New("provider unavailable")
			},
			callback:   successfulShapeOAuthCallback,
			wantStatus: http.StatusBadGateway,
		},
		{
			name: "profile failure",
			configure: func(provider *integrationFakeOAuthProvider) {
				provider.fetchErr = errors.New("invalid provider profile")
			},
			callback:   successfulShapeOAuthCallback,
			wantStatus: http.StatusBadGateway,
		},
		{
			name: "unverified email",
			configure: func(provider *integrationFakeOAuthProvider) {
				provider.user.VerifiedEmail = ""
			},
			callback:   successfulShapeOAuthCallback,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &integrationFakeOAuthProvider{
				name: "google",
				user: oauth.User{
					Provider:          "google",
					ProviderUserID:    "google-subject",
					VerifiedEmail:     "google@example.com",
					SuggestedUsername: "Google User",
				},
			}
			if tt.configure != nil {
				tt.configure(provider)
			}

			env := newOAuthIntegrationEnv(t, "google", provider)
			defer env.Server.Close()
			browser := newIntegrationBrowser(t)

			response := tt.callback(t, browser, env)
			defer response.Body.Close()

			if response.StatusCode != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					response.StatusCode,
					tt.wantStatus,
				)
			}

			assertRowCount(t, env.DB, "users", 0)
			assertRowCount(t, env.DB, "oauth_accounts", 0)
			assertRowCount(t, env.DB, "sessions", 0)
		})
	}
}

func TestIntegrationOAuthEmailCollisionDoesNotLinkOrAuthenticate(t *testing.T) {
	provider := &integrationFakeOAuthProvider{
		name: "google",
		user: oauth.User{
			Provider:          "google",
			ProviderUserID:    "google-subject",
			VerifiedEmail:     "existing@example.com",
			SuggestedUsername: "Google User",
		},
	}
	env := newOAuthIntegrationEnv(t, "google", provider)
	defer env.Server.Close()

	users := repository.NewUserRepository(env.DB)
	if _, err := users.Create(
		"existing@example.com",
		"password-user",
		"bcrypt-hash",
	); err != nil {
		t.Fatalf("create existing password user: %v", err)
	}

	browser := newIntegrationBrowser(t)
	state := startOAuth(t, browser, env.Server.URL, "google")
	response := requestOAuthCallback(
		t,
		browser,
		env.Server.URL,
		"google",
		state,
		"authorization-code",
	)
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		t.Fatalf(
			"status = %d, want %d",
			response.StatusCode,
			http.StatusConflict,
		)
	}

	assertRowCount(t, env.DB, "users", 1)
	assertRowCount(t, env.DB, "oauth_accounts", 0)
	assertRowCount(t, env.DB, "sessions", 0)
}

func TestIntegrationWrongProviderCallbackCannotConsumeValidState(t *testing.T) {
	github := &integrationFakeOAuthProvider{
		name: "github",
		user: oauth.User{
			Provider:          "github",
			ProviderUserID:    "123456",
			VerifiedEmail:     "github@example.com",
			SuggestedUsername: "octocat",
		},
	}
	google := &integrationFakeOAuthProvider{
		name: "google",
		user: oauth.User{
			Provider:          "google",
			ProviderUserID:    "google-subject",
			VerifiedEmail:     "google@example.com",
			SuggestedUsername: "Google User",
		},
	}
	env := newIntegrationEnvWithOAuth(t, integrationOAuthProviders{
		GitHub: github,
		Google: google,
	})
	defer env.Server.Close()
	browser := newIntegrationBrowser(t)

	githubState := startOAuth(t, browser, env.Server.URL, "github")
	wrongResponse := requestOAuthCallbackWithCookie(
		t,
		browser,
		env.Server.URL,
		"google",
		githubState,
		"authorization-code",
	)
	wrongResponse.Body.Close()

	if wrongResponse.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"wrong-provider status = %d, want %d",
			wrongResponse.StatusCode,
			http.StatusBadRequest,
		)
	}

	validResponse := requestOAuthCallback(
		t,
		browser,
		env.Server.URL,
		"github",
		githubState,
		"authorization-code",
	)
	defer validResponse.Body.Close()

	if validResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf(
			"valid GitHub callback status = %d, want %d",
			validResponse.StatusCode,
			http.StatusSeeOther,
		)
	}
}

func TestIntegrationOAuthCallbackStateCannotBeReplayed(t *testing.T) {
	provider := &integrationFakeOAuthProvider{
		name: "google",
		user: oauth.User{
			Provider:          "google",
			ProviderUserID:    "google-subject",
			VerifiedEmail:     "google@example.com",
			SuggestedUsername: "Google User",
		},
	}
	env := newOAuthIntegrationEnv(t, "google", provider)
	defer env.Server.Close()
	browser := newIntegrationBrowser(t)

	state := startOAuth(t, browser, env.Server.URL, "google")
	first := requestOAuthCallback(
		t,
		browser,
		env.Server.URL,
		"google",
		state,
		"authorization-code",
	)
	first.Body.Close()
	if first.StatusCode != http.StatusSeeOther {
		t.Fatalf("first callback status = %d", first.StatusCode)
	}

	replay := requestOAuthCallbackWithCookie(
		t,
		browser,
		env.Server.URL,
		"google",
		state,
		"authorization-code",
	)
	defer replay.Body.Close()

	if replay.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"replay status = %d, want %d",
			replay.StatusCode,
			http.StatusBadRequest,
		)
	}
}

func TestIntegrationRepeatOAuthLoginEnforcesOneActiveBrowserSession(t *testing.T) {
	provider := &integrationFakeOAuthProvider{
		name: "google",
		user: oauth.User{
			Provider:          "google",
			ProviderUserID:    "google-subject",
			VerifiedEmail:     "google@example.com",
			SuggestedUsername: "Google User",
		},
	}
	env := newOAuthIntegrationEnv(t, "google", provider)
	defer env.Server.Close()
	browserA := newIntegrationBrowser(t)
	browserB := newIntegrationBrowser(t)

	completeOAuthLogin(t, browserA, env.Server.URL, "google")
	assertHTTPStatus(
		t,
		browserA,
		http.MethodGet,
		env.Server.URL+"/posts/new",
		http.StatusOK,
	)

	completeOAuthLogin(t, browserB, env.Server.URL, "google")

	assertHTTPStatus(
		t,
		browserA,
		http.MethodGet,
		env.Server.URL+"/posts/new",
		http.StatusUnauthorized,
	)
	assertHTTPStatus(
		t,
		browserB,
		http.MethodGet,
		env.Server.URL+"/posts/new",
		http.StatusOK,
	)
	assertRowCount(t, env.DB, "users", 1)
	assertRowCount(t, env.DB, "oauth_accounts", 1)
	assertRowCount(t, env.DB, "sessions", 1)
}

func TestIntegrationOAuthUserCreatesForumContentVisibleAfterLogout(t *testing.T) {
	provider := &integrationFakeOAuthProvider{
		name: "google",
		user: oauth.User{
			Provider:          "google",
			ProviderUserID:    "google-subject",
			VerifiedEmail:     "google@example.com",
			SuggestedUsername: "Google User",
		},
	}
	env := newOAuthIntegrationEnv(t, "google", provider)
	defer env.Server.Close()
	browser := newIntegrationBrowser(t)
	completeOAuthLogin(t, browser, env.Server.URL, "google")

	postLocation := createPost(t, env.Server, browser, "OAuth persisted post")
	postID := strings.TrimPrefix(postLocation, "/posts/")

	commentResponse, err := browser.PostForm(
		env.Server.URL+"/posts/"+postID+"/comments",
		url.Values{"body": {"OAuth persisted comment"}},
	)
	if err != nil {
		t.Fatalf("create OAuth comment: %v", err)
	}
	commentResponse.Body.Close()
	if commentResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf("comment status = %d", commentResponse.StatusCode)
	}

	postReactionResponse, err := browser.PostForm(
		env.Server.URL+"/posts/"+postID+"/react",
		url.Values{"value": {"1"}},
	)
	if err != nil {
		t.Fatalf("react to OAuth post: %v", err)
	}
	postReactionResponse.Body.Close()
	if postReactionResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf("post reaction status = %d", postReactionResponse.StatusCode)
	}

	var commentID int64
	if err := env.DB.QueryRow(
		"SELECT id FROM comments WHERE post_id = ?",
		postID,
	).Scan(&commentID); err != nil {
		t.Fatalf("find OAuth comment: %v", err)
	}

	commentReactionResponse, err := browser.PostForm(
		env.Server.URL+"/comments/"+strconv.FormatInt(commentID, 10)+"/react",
		url.Values{"value": {"1"}},
	)
	if err != nil {
		t.Fatalf("react to OAuth comment: %v", err)
	}
	commentReactionResponse.Body.Close()
	if commentReactionResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf("comment reaction status = %d", commentReactionResponse.StatusCode)
	}

	assertRowCount(t, env.DB, "posts", 1)
	assertRowCount(t, env.DB, "comments", 1)
	assertRowCount(t, env.DB, "post_reactions", 1)
	assertRowCount(t, env.DB, "comment_reactions", 1)
	logoutOAuthBrowser(t, browser, env.Server.URL)

	response, err := browser.Get(env.Server.URL + postLocation)
	if err != nil {
		t.Fatalf("read OAuth content after logout: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read OAuth content body: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("post detail status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	for _, want := range []string{"OAuth persisted post", "OAuth persisted comment"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("content %q is not visible after logout", want)
		}
	}
}

func newOAuthIntegrationEnv(
	t *testing.T,
	providerName string,
	provider oauth.Provider,
) *integrationEnv {
	t.Helper()

	providers := integrationOAuthProviders{}
	if providerName == "github" {
		providers.GitHub = provider
	} else {
		providers.Google = provider
	}

	return newIntegrationEnvWithOAuth(t, providers)
}

func startOAuth(
	t *testing.T,
	browser *http.Client,
	forumURL string,
	provider string,
) string {
	t.Helper()

	response, err := browser.Get(forumURL + "/auth/" + provider)
	if err != nil {
		t.Fatalf("start %s OAuth: %v", provider, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusFound {
		t.Fatalf(
			"%s start status = %d, want %d",
			provider,
			response.StatusCode,
			http.StatusFound,
		)
	}

	location, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse %s authorization URL: %v", provider, err)
	}
	state := location.Query().Get("state")
	if state == "" {
		t.Fatalf("%s authorization state is empty", provider)
	}

	return state
}

func completeOAuthLogin(
	t *testing.T,
	browser *http.Client,
	forumURL string,
	provider string,
) {
	t.Helper()

	state := startOAuth(t, browser, forumURL, provider)
	response := requestOAuthCallback(
		t,
		browser,
		forumURL,
		provider,
		state,
		"authorization-code",
	)
	defer response.Body.Close()

	if response.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf(
			"%s callback status = %d, want %d; body=%q",
			provider,
			response.StatusCode,
			http.StatusSeeOther,
			strings.TrimSpace(string(body)),
		)
	}
}

func successfulShapeOAuthCallback(
	t *testing.T,
	browser *http.Client,
	env *integrationEnv,
) *http.Response {
	t.Helper()

	state := startOAuth(t, browser, env.Server.URL, "google")
	return requestOAuthCallback(
		t,
		browser,
		env.Server.URL,
		"google",
		state,
		"authorization-code",
	)
}

func requestOAuthCallback(
	t *testing.T,
	browser *http.Client,
	forumURL string,
	provider string,
	state string,
	code string,
) *http.Response {
	t.Helper()

	values := url.Values{}
	if state != "" {
		values.Set("state", state)
	}
	if code != "" {
		values.Set("code", code)
	}

	response, err := browser.Get(
		forumURL + "/auth/" + provider + "/callback?" + values.Encode(),
	)
	if err != nil {
		t.Fatalf("%s OAuth callback: %v", provider, err)
	}

	return response
}

func requestOAuthCallbackWithCookie(
	t *testing.T,
	browser *http.Client,
	forumURL string,
	provider string,
	state string,
	code string,
) *http.Response {
	t.Helper()

	values := url.Values{"state": {state}, "code": {code}}
	request, err := http.NewRequest(
		http.MethodGet,
		forumURL+"/auth/"+provider+"/callback?"+values.Encode(),
		nil,
	)
	if err != nil {
		t.Fatalf("create %s callback request: %v", provider, err)
	}
	request.AddCookie(&http.Cookie{
		Name:  provider + "_oauth_state",
		Value: state,
	})

	response, err := browser.Do(request)
	if err != nil {
		t.Fatalf("%s callback with cookie: %v", provider, err)
	}

	return response
}

func logoutOAuthBrowser(
	t *testing.T,
	browser *http.Client,
	forumURL string,
) {
	t.Helper()

	response, err := browser.Post(forumURL+"/logout", "", nil)
	if err != nil {
		t.Fatalf("OAuth logout: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf(
			"logout status = %d, want %d",
			response.StatusCode,
			http.StatusSeeOther,
		)
	}
}

func assertHTTPStatus(
	t *testing.T,
	client *http.Client,
	method string,
	requestURL string,
	want int,
) {
	t.Helper()

	request, err := http.NewRequest(method, requestURL, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, requestURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode != want {
		t.Fatalf(
			"%s %s status = %d, want %d",
			method,
			requestURL,
			response.StatusCode,
			want,
		)
	}
}
