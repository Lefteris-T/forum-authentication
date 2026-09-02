package service

import (
	"errors"
	"forum/internal/model"
	"forum/internal/oauth"
	"forum/internal/repository"
	"testing"
)

type fakeOAuthAccountRepository struct {
	account        model.OAuthAccount
	err            error
	provider       string
	providerUserID string
}

func (f *fakeOAuthAccountRepository) FindByProviderUserID(
	provider string,
	providerUserID string,
) (model.OAuthAccount, error) {
	f.provider = provider
	f.providerUserID = providerUserID
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
	user       model.User
	byEmailErr error
	byIDErr    error
}

func (f *fakeOAuthUserRepository) ByID(
	id int64,
) (model.User, error) {
	return f.user, f.byIDErr
}

func (f *fakeOAuthUserRepository) ByEmail(
	email string,
) (model.User, error) {
	return f.user, f.byEmailErr
}

type fakeOAuthAccountRepositoryWithCreate struct {
	account               model.OAuthAccount
	findErr               error
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
func TestOAuthLoginReturnsExistingLocalUser(t *testing.T) {
	oauthAccounts := &fakeOAuthAccountRepository{
		account: model.OAuthAccount{
			UserID:         42,
			Provider:       "github",
			ProviderUserID: "123456",
		},
	}

	users := &fakeOAuthUserRepository{
		user: model.User{
			ID:       42,
			Email:    "oauth@example.com",
			Username: "octocat",
		},
	}

	service := NewOAuthLoginService(
		oauthAccounts,
		users,
	)

	got, err := service.Login(oauth.User{
		Provider:       "github",
		ProviderUserID: "123456",
	})
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}

	if got.ID != 42 {
		t.Fatalf("user ID = %d, want 42", got.ID)
	}

	if got.Email != "oauth@example.com" {
		t.Fatalf(
			"user email = %q, want %q",
			got.Email,
			"oauth@example.com",
		)
	}

	if oauthAccounts.provider != "github" {
		t.Fatalf(
			"lookup provider = %q, want %q",
			oauthAccounts.provider,
			"github",
		)
	}

	if oauthAccounts.providerUserID != "123456" {
		t.Fatalf(
			"lookup provider user ID = %q, want %q",
			oauthAccounts.providerUserID,
			"123456",
		)
	}
}
func TestOAuthLoginCreatesFirstTimeUser(t *testing.T) {
	oauthAccounts := &fakeOAuthAccountRepositoryWithCreate{
		findErr:       repository.ErrOAuthAccountNotFound,
		createdUserID: 42,
	}

	users := &fakeOAuthUserRepository{
		user: model.User{
			ID:       42,
			Email:    "oauth@example.com",
			Username: "octocat",
		},
		byEmailErr: repository.ErrUserNotFound,
	}

	service := NewOAuthLoginService(
		oauthAccounts,
		users,
	)

	got, err := service.Login(oauth.User{
		Provider:          "github",
		ProviderUserID:    "123456",
		VerifiedEmail:     "oauth@example.com",
		SuggestedUsername: "octocat",
	})
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}

	if oauthAccounts.createdEmail != "oauth@example.com" {
		t.Fatalf(
			"created email = %q, want %q",
			oauthAccounts.createdEmail,
			"oauth@example.com",
		)
	}

	if oauthAccounts.createdUsername != "octocat" {
		t.Fatalf(
			"created username = %q, want %q",
			oauthAccounts.createdUsername,
			"octocat",
		)
	}

	if got.ID != 42 {
		t.Fatalf("user ID = %d, want 42", got.ID)
	}
}

func TestOAuthLoginRequiresVerifiedEmailForFirstTimeUser(t *testing.T) {
	oauthAccounts := &fakeOAuthAccountRepositoryWithCreate{
		findErr: repository.ErrOAuthAccountNotFound,
	}
	users := &fakeOAuthUserRepository{
		byEmailErr: repository.ErrUserNotFound,
	}
	oauthLogin := NewOAuthLoginService(oauthAccounts, users)

	_, err := oauthLogin.Login(oauth.User{
		Provider:          "google",
		ProviderUserID:    "google-subject",
		SuggestedUsername: "Google User",
	})

	if !errors.Is(err, ErrOAuthVerifiedEmailRequired) {
		t.Fatalf(
			"Login() error = %v, want %v",
			err,
			ErrOAuthVerifiedEmailRequired,
		)
	}

	if oauthAccounts.createdProviderUserID != "" {
		t.Fatal("OAuth user was created without a verified email")
	}
}

func TestOAuthLoginRejectsExistingEmailWithoutLinking(t *testing.T) {
	oauthAccounts := &fakeOAuthAccountRepositoryWithCreate{
		findErr: repository.ErrOAuthAccountNotFound,
	}
	users := &fakeOAuthUserRepository{
		user: model.User{
			ID:    7,
			Email: "existing@example.com",
		},
	}
	oauthLogin := NewOAuthLoginService(oauthAccounts, users)

	_, err := oauthLogin.Login(oauth.User{
		Provider:          "google",
		ProviderUserID:    "google-subject",
		VerifiedEmail:     "existing@example.com",
		SuggestedUsername: "Google User",
	})

	if !errors.Is(err, ErrOAuthEmailConflict) {
		t.Fatalf("Login() error = %v, want %v", err, ErrOAuthEmailConflict)
	}

	if oauthAccounts.createdProviderUserID != "" {
		t.Fatal("existing email was automatically linked to Google")
	}
}
