package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"forum/internal/model"
	"forum/internal/oauth"
	"forum/internal/repository"
)

var (
	ErrOAuthEmailConflict         = errors.New("oauth email already belongs to an existing account")
	ErrOAuthVerifiedEmailRequired = errors.New("verified oauth email required")
)

type oauthAccountRepository interface {
	FindByProviderUserID(
		provider string,
		providerUserID string,
	) (model.OAuthAccount, error)

	CreateUserWithOAuthAccount(
		email string,
		username string,
		provider string,
		providerUserID string,
	) (int64, error)
}

type oauthUserFinder interface {
	ByID(id int64) (model.User, error)
	ByEmail(email string) (model.User, error)
}

type OAuthLoginService struct {
	oauthAccounts oauthAccountRepository
	users         oauthUserFinder
}

func NewOAuthLoginService(
	oauthAccounts oauthAccountRepository,
	users oauthUserFinder,
) *OAuthLoginService {
	return &OAuthLoginService{
		oauthAccounts: oauthAccounts,
		users:         users,
	}
}

func (s *OAuthLoginService) Login(
	oauthUser oauth.User,
) (model.User, error) {
	account, err := s.oauthAccounts.FindByProviderUserID(
		oauthUser.Provider,
		oauthUser.ProviderUserID,
	)

	// Returning OAuth user.
	if err == nil {
		user, err := s.users.ByID(account.UserID)
		if err != nil {
			return model.User{}, fmt.Errorf(
				"find oauth user: %w",
				err,
			)
		}

		return user, nil
	}

	if !errors.Is(err, repository.ErrOAuthAccountNotFound) {
		return model.User{}, fmt.Errorf(
			"find oauth account: %w",
			err,
		)
	}

	// First-time OAuth user.
	if oauthUser.VerifiedEmail == "" {
		return model.User{}, ErrOAuthVerifiedEmailRequired
	}

	// Do not automatically link an OAuth identity
	// to an existing local account.
	_, err = s.users.ByEmail(oauthUser.VerifiedEmail)

	if err == nil {
		return model.User{}, ErrOAuthEmailConflict
	}

	if !errors.Is(err, repository.ErrUserNotFound) {
		return model.User{}, fmt.Errorf(
			"check oauth email: %w",
			err,
		)
	}

	baseUsername := normalizeOAuthUsername(
		oauthUser.SuggestedUsername,
		oauthUser.Provider,
	)

	var userID int64

	for attempt := 1; ; attempt++ {
		username := oauthUsernameCandidate(
			baseUsername,
			attempt,
		)

		userID, err = s.oauthAccounts.CreateUserWithOAuthAccount(
			oauthUser.VerifiedEmail,
			username,
			oauthUser.Provider,
			oauthUser.ProviderUserID,
		)
		if err == nil {
			break
		}

		if errors.Is(err, repository.ErrUsernameExists) {
			continue
		}

		if errors.Is(err, repository.ErrEmailExists) {
			return model.User{}, ErrOAuthEmailConflict
		}

		return model.User{}, fmt.Errorf(
			"create oauth user: %w",
			err,
		)
	}

	user, err := s.users.ByID(userID)
	if err != nil {
		return model.User{}, fmt.Errorf(
			"find created oauth user: %w",
			err,
		)
	}

	return user, nil
}

const maxUsernameLength = 32

func normalizeOAuthUsername(
	input string,
	provider string,
) string {
	input = strings.TrimSpace(input)

	var b strings.Builder

	lastDash := false

	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
			lastDash = false
			continue
		}

		if unicode.IsSpace(r) {
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	username := strings.Trim(b.String(), "-")

	if len(username) < 3 {
		username = provider + "-user"
	}

	if len(username) > maxUsernameLength {
		username = username[:maxUsernameLength]
	}

	return username
}

func oauthUsernameCandidate(
	base string,
	n int,
) string {
	if n <= 1 {
		return base
	}

	suffix := "-" + strconv.Itoa(n)

	maxBaseLength := maxUsernameLength - len(suffix)

	if len(base) > maxBaseLength {
		base = base[:maxBaseLength]
	}

	return base + suffix
}
