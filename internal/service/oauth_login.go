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

type oauthSessionCreator interface {
	CreateSession(userID int64) (model.Session, error)
}

type OAuthLoginService struct {
	oauthAccounts oauthAccountRepository
	users         oauthUserFinder
	sessions      oauthSessionCreator
}

func NewOAuthLoginService(
	oauthAccounts oauthAccountRepository,
	users oauthUserFinder,
	sessions oauthSessionCreator,
) *OAuthLoginService {
	return &OAuthLoginService{
		oauthAccounts: oauthAccounts,
		users:         users,
		sessions:      sessions,
	}
}

func (s *OAuthLoginService) Login(
	oauthUser oauth.User,
) (model.Session, error) {
	account, err := s.oauthAccounts.FindByProviderUserID(
		oauthUser.Provider,
		oauthUser.ProviderUserID,
	)

	// Returning OAuth user.
	if err == nil {
		user, err := s.users.ByID(account.UserID)
		if err != nil {
			return model.Session{}, err
		}

		session, err := s.sessions.CreateSession(user.ID)
		if err != nil {
			return model.Session{}, fmt.Errorf(
				"create oauth session: %w",
				err,
			)
		}

		return session, nil
	}

	if !errors.Is(err, repository.ErrOAuthAccountNotFound) {
		return model.Session{}, fmt.Errorf(
			"find oauth account: %w",
			err,
		)
	}

	// First-time OAuth user.
	if oauthUser.VerifiedEmail == "" {
		return model.Session{}, ErrOAuthVerifiedEmailRequired
	}

	// Do not automatically link an OAuth identity to an existing account.
	_, err = s.users.ByEmail(oauthUser.VerifiedEmail)

	if err == nil {
		return model.Session{}, ErrOAuthEmailConflict
	}

	if !errors.Is(err, repository.ErrUserNotFound) {
		return model.Session{}, fmt.Errorf(
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
			return model.Session{}, ErrOAuthEmailConflict
		}

		return model.Session{}, fmt.Errorf(
			"create oauth user: %w",
			err,
		)
	}

	session, err := s.sessions.CreateSession(userID)
	if err != nil {
		return model.Session{}, fmt.Errorf(
			"create oauth session: %w",
			err,
		)
	}

	return session, nil
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
