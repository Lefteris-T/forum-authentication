package service

import (
	"errors"
	"testing"
	"time"

	"forum/internal/model"
	"forum/internal/oauth"
	"forum/internal/repository"
)

type fakeOAuthAccountRepository struct {
	account model.OAuthAccount
	err     error
}

func (f *fakeOAuthAccountRepository) FindByProviderUserID(
	provider string,
	providerUserID string,
) (model.OAuthAccount, error) {
	return f.account, f.err
}
func (f *fakeOAuthAccountRepository) CreateUserWithOAuthAccount(
	email string,
	username string,
	provider string,
	providerUserID string,
) (int64, error) {
	return 0, errors.New("unexpected CreateUserWithOAuthAccount call")
}

type fakeOAuthUserRepository struct {
	user model.User
	err  error
}

func (f *fakeOAuthUserRepository) ByID(
	id int64,
) (model.User, error) {
	return f.user, f.err
}

func (f *fakeOAuthUserRepository) ByEmail(
	email string,
) (model.User, error) {
	return f.user, f.err
}

type fakeOAuthSessionCreator struct {
	userID int64
	called bool
}

func (f *fakeOAuthSessionCreator) CreateSession(
	userID int64,
) (model.Session, error) {
	f.called = true
	f.userID = userID

	return model.Session{
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil
}

func TestOAuthLoginExistingUser(t *testing.T) {
	oauthAccounts := &fakeOAuthAccountRepository{
		account: model.OAuthAccount{
			ID:             1,
			UserID:         42,
			Provider:       "github",
			ProviderUserID: "123456",
			Email:          "oauth@example.com",
		},
	}

	users := &fakeOAuthUserRepository{
		user: model.User{
			ID:       42,
			Email:    "oauth@example.com",
			Username: "oauth-user",
		},
	}

	sessions := &fakeOAuthSessionCreator{}

	service := NewOAuthLoginService(
		oauthAccounts,
		users,
		sessions,
	)

	session, err := service.Login(oauth.User{
		Provider:          "github",
		ProviderUserID:    "123456",
		VerifiedEmail:     "oauth@example.com",
		SuggestedUsername: "oauth-user",
	})
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}

	if !sessions.called {
		t.Fatal("CreateSession() was not called")
	}

	if sessions.userID != 42 {
		t.Fatalf(
			"CreateSession() userID = %d, want %d",
			sessions.userID,
			42,
		)
	}

	if session.UserID != 42 {
		t.Fatalf(
			"session.UserID = %d, want %d",
			session.UserID,
			42,
		)
	}
}

func TestOAuthLoginPropagatesUnknownLocalUser(t *testing.T) {
	oauthAccounts := &fakeOAuthAccountRepository{
		account: model.OAuthAccount{
			UserID:         42,
			Provider:       "github",
			ProviderUserID: "123456",
		},
	}

	users := &fakeOAuthUserRepository{
		err: repository.ErrUserNotFound,
	}

	sessions := &fakeOAuthSessionCreator{}

	service := NewOAuthLoginService(
		oauthAccounts,
		users,
		sessions,
	)

	_, err := service.Login(oauth.User{
		Provider:       "github",
		ProviderUserID: "123456",
		VerifiedEmail:  "oauth@example.com",
	})

	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Fatalf(
			"Login() error = %v, want %v",
			err,
			repository.ErrUserNotFound,
		)
	}
}

type fakeOAuthAccountRepositoryWithCreate struct {
	account model.OAuthAccount
	findErr error

	createdEmail          string
	createdUsername       string
	createdProvider       string
	createdProviderUserID string
	createdUserID         int64
}

func (f *fakeOAuthAccountRepositoryWithCreate) FindByProviderUserID(
	provider string,
	providerUserID string,
) (model.OAuthAccount, error) {
	return f.account, f.findErr
}

func (f *fakeOAuthAccountRepositoryWithCreate) CreateUserWithOAuthAccount(
	email string,
	username string,
	provider string,
	providerUserID string,
) (int64, error) {
	f.createdEmail = email
	f.createdUsername = username
	f.createdProvider = provider
	f.createdProviderUserID = providerUserID

	if f.createdUserID == 0 {
		f.createdUserID = 42
	}

	return f.createdUserID, nil
}
func TestOAuthLoginCreatesFirstTimeUser(t *testing.T) {
	oauthAccounts := &fakeOAuthAccountRepositoryWithCreate{
		findErr:       repository.ErrOAuthAccountNotFound,
		createdUserID: 42,
	}

	users := &fakeOAuthUserRepository{
		err: repository.ErrUserNotFound,
	}

	sessions := &fakeOAuthSessionCreator{}

	service := NewOAuthLoginService(
		oauthAccounts,
		users,
		sessions,
	)

	session, err := service.Login(oauth.User{
		Provider:          "github",
		ProviderUserID:    "123456",
		VerifiedEmail:     "oauth@example.com",
		SuggestedUsername: "octocat",
	})
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}

	if oauthAccounts.createdEmail != "oauth@example.com" {
		t.Errorf(
			"created email = %q, want %q",
			oauthAccounts.createdEmail,
			"oauth@example.com",
		)
	}

	if oauthAccounts.createdUsername != "octocat" {
		t.Errorf(
			"created username = %q, want %q",
			oauthAccounts.createdUsername,
			"octocat",
		)
	}

	if oauthAccounts.createdProvider != "github" {
		t.Errorf(
			"created provider = %q, want %q",
			oauthAccounts.createdProvider,
			"github",
		)
	}

	if oauthAccounts.createdProviderUserID != "123456" {
		t.Errorf(
			"created provider user id = %q, want %q",
			oauthAccounts.createdProviderUserID,
			"123456",
		)
	}

	if !sessions.called {
		t.Fatal("CreateSession() was not called")
	}

	if sessions.userID != 42 {
		t.Fatalf(
			"CreateSession() userID = %d, want %d",
			sessions.userID,
			42,
		)
	}

	if session.UserID != 42 {
		t.Fatalf(
			"session.UserID = %d, want %d",
			session.UserID,
			42,
		)
	}
}
func TestOAuthLoginRejectsExistingEmail(t *testing.T) {
	oauthAccounts := &fakeOAuthAccountRepositoryWithCreate{
		findErr: repository.ErrOAuthAccountNotFound,
	}

	users := &fakeOAuthUserRepository{
		user: model.User{
			ID:           7,
			Email:        "oauth@example.com",
			Username:     "existing-user",
			PasswordHash: "existing-hash",
		},
		err: nil,
	}

	sessions := &fakeOAuthSessionCreator{}

	service := NewOAuthLoginService(
		oauthAccounts,
		users,
		sessions,
	)

	_, err := service.Login(oauth.User{
		Provider:          "github",
		ProviderUserID:    "123456",
		VerifiedEmail:     "oauth@example.com",
		SuggestedUsername: "octocat",
	})

	if !errors.Is(err, ErrOAuthEmailConflict) {
		t.Fatalf(
			"Login() error = %v, want %v",
			err,
			ErrOAuthEmailConflict,
		)
	}

	if sessions.called {
		t.Fatal("CreateSession() was called for email conflict")
	}
}
func TestOAuthLoginRejectsMissingVerifiedEmail(t *testing.T) {
	oauthAccounts := &fakeOAuthAccountRepositoryWithCreate{
		findErr: repository.ErrOAuthAccountNotFound,
	}

	users := &fakeOAuthUserRepository{
		err: repository.ErrUserNotFound,
	}

	sessions := &fakeOAuthSessionCreator{}

	service := NewOAuthLoginService(
		oauthAccounts,
		users,
		sessions,
	)

	_, err := service.Login(oauth.User{
		Provider:          "github",
		ProviderUserID:    "123456",
		VerifiedEmail:     "",
		SuggestedUsername: "octocat",
	})

	if !errors.Is(err, ErrOAuthVerifiedEmailRequired) {
		t.Fatalf(
			"Login() error = %v, want %v",
			err,
			ErrOAuthVerifiedEmailRequired,
		)
	}

	if sessions.called {
		t.Fatal("CreateSession() was called without verified email")
	}
}
func TestNormalizeOAuthUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		provider string
		want     string
	}{
		{
			name:     "keeps simple username",
			input:    "octocat",
			provider: "github",
			want:     "octocat",
		},
		{
			name:     "replaces spaces",
			input:    "John Doe",
			provider: "github",
			want:     "John-Doe",
		},
		{
			name:     "falls back when unusable",
			input:    "@@@",
			provider: "github",
			want:     "github-user",
		},
		{
			name:     "truncates to 32 characters",
			input:    "abcdefghijklmnopqrstuvwxyz1234567890",
			provider: "github",
			want:     "abcdefghijklmnopqrstuvwxyz123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeOAuthUsername(
				tt.input,
				tt.provider,
			)

			if got != tt.want {
				t.Fatalf(
					"normalizeOAuthUsername() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}
func TestOAuthUsernameCandidateAddsSuffix(t *testing.T) {
	tests := []struct {
		base string
		n    int
		want string
	}{
		{
			base: "octocat",
			n:    1,
			want: "octocat",
		},
		{
			base: "octocat",
			n:    2,
			want: "octocat-2",
		},
		{
			base: "abcdefghijklmnopqrstuvwxyz123456",
			n:    2,
			want: "abcdefghijklmnopqrstuvwxyz1234-2",
		},
	}

	for _, tt := range tests {
		got := oauthUsernameCandidate(
			tt.base,
			tt.n,
		)

		if got != tt.want {
			t.Fatalf(
				"oauthUsernameCandidate(%q, %d) = %q, want %q",
				tt.base,
				tt.n,
				got,
				tt.want,
			)
		}
	}
}

type fakeOAuthAccountRepositoryWithUsernameConflicts struct {
	findErr  error
	attempts []string
}

func (f *fakeOAuthAccountRepositoryWithUsernameConflicts) FindByProviderUserID(
	provider string,
	providerUserID string,
) (model.OAuthAccount, error) {
	return model.OAuthAccount{}, f.findErr
}

func (f *fakeOAuthAccountRepositoryWithUsernameConflicts) CreateUserWithOAuthAccount(
	email string,
	username string,
	provider string,
	providerUserID string,
) (int64, error) {
	f.attempts = append(f.attempts, username)

	switch username {
	case "octocat", "octocat-2":
		return 0, repository.ErrUsernameExists
	default:
		return 42, nil
	}
}
func TestOAuthLoginRetriesUsernameOnConflict(t *testing.T) {
	oauthAccounts := &fakeOAuthAccountRepositoryWithUsernameConflicts{
		findErr: repository.ErrOAuthAccountNotFound,
	}

	users := &fakeOAuthUserRepository{
		err: repository.ErrUserNotFound,
	}

	sessions := &fakeOAuthSessionCreator{}

	service := NewOAuthLoginService(
		oauthAccounts,
		users,
		sessions,
	)

	_, err := service.Login(oauth.User{
		Provider:          "github",
		ProviderUserID:    "123456",
		VerifiedEmail:     "oauth@example.com",
		SuggestedUsername: "octocat",
	})
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}

	want := []string{
		"octocat",
		"octocat-2",
		"octocat-3",
	}

	if len(oauthAccounts.attempts) != len(want) {
		t.Fatalf(
			"attempt count = %d, want %d",
			len(oauthAccounts.attempts),
			len(want),
		)
	}

	for i := range want {
		if oauthAccounts.attempts[i] != want[i] {
			t.Fatalf(
				"attempt[%d] = %q, want %q",
				i,
				oauthAccounts.attempts[i],
				want[i],
			)
		}
	}

	if !sessions.called {
		t.Fatal("CreateSession() was not called")
	}

	if sessions.userID != 42 {
		t.Fatalf(
			"CreateSession() userID = %d, want %d",
			sessions.userID,
			42,
		)
	}
}
