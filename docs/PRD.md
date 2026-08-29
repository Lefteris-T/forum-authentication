# Forum Authentication Product Requirements

## Purpose

Extend the completed Go forum with new authentication methods. Users must be
able to authenticate with GitHub or Google while the existing email/password
registration and login continue to work.

All three methods resolve to the same local user and existing forum session
mechanism. After authentication, OAuth users have the same rights as every other
registered user: they can create posts and comments, react, use personal filters,
and log out.

The existing forum behavior is the baseline and must not regress.

## Authentication Release

### Supported methods

The application provides three authentication methods:

- email and password;
- GitHub OAuth;
- Google OAuth.

GitHub and Google are mandatory. Other providers are outside this release.

### Existing password authentication

- Registration asks for email, username, and password.
- Normalized email and username are unique; conflicts return `409 Conflict`.
- Passwords are hashed with bcrypt and plaintext passwords are never stored.
- Unknown email, wrong password, and an OAuth-only account used through the
  password form all return the same `401 Unauthorized` response with
  `Wrong email or password`.
- Existing registration, login, session, and logout behavior remains supported.

### OAuth authorization flow

Both providers use the server-side authorization-code flow:

```text
browser
→ GET /auth/{provider}
→ provider authorization page
→ GET /auth/{provider}/callback
→ validate state and authorization code
→ exchange code for access token
→ fetch verified provider identity
→ resolve/create local user
→ create normal forum session
→ set forum_session cookie
→ 303 /
```

OAuth replaces password verification only. Provider access tokens are not forum
sessions and are not used for later forum requests.

### Provider identity

- GitHub accounts are identified by GitHub's stable numeric user ID.
- GitHub login requests `user:email` and requires a primary verified email from
  the authenticated user's email endpoint.
- Google accounts are identified by the stable OpenID Connect `sub` value.
- Google requests `openid email profile` and requires `email_verified=true`.
- Email and provider username are profile attributes, never provider identity
  keys.
- Missing, blank, or unverified provider email stops first-time registration.

### Local account creation and collisions

- A returning OAuth user is resolved only by `(provider, provider_user_id)`.
- A first-time OAuth login creates one local `users` row and one
  `oauth_accounts` row in the same transaction.
- OAuth-only users have a `NULL` password hash. The application never generates
  a fake password or placeholder hash.
- If the verified provider email already belongs to any local account, OAuth
  login returns `409 Conflict` with a safe message instructing the user to use
  the original sign-in method.
- Accounts are never linked automatically. Explicit account linking is outside
  this release.
- Local usernames are derived from provider display data, normalized to the
  existing 3–32 character rules, and given numeric suffixes when occupied.
- Provider profile changes must not create a second local account.

### Forum sessions

- All successful authentication methods create the existing UUID-backed
  `forum_session` cookie and server-side session row.
- A user has at most one active session. A new password or OAuth login replaces
  the previous session.
- Cookies expire and use `HttpOnly`, `SameSite=Lax`, `Path=/`, and the configured
  `Secure` setting.
- Production configuration uses HTTPS and secure cookies.
- Logout deletes the server-side session and clears the cookie for every login
  method.
- Authentication middleware continues to resolve the current local user from
  the forum session; it never contacts an OAuth provider during ordinary forum
  requests.

## OAuth Security Requirements

- Generate authorization state with at least 32 bytes from `crypto/rand`.
- Bind state to the initiating browser with a hardened short-lived cookie.
- Store only a hash of state with its provider, PKCE verifier, and expiry in a
  concurrency-safe in-memory store.
- State expires after ten minutes, is provider-specific, is consumed atomically,
  and cannot be replayed.
- Compare state in constant time and clear its cookie on every callback outcome.
- Use PKCE S256 for GitHub and Google authorization-code exchange.
- Fixed configured redirect URIs must match the registered provider callbacks.
- Provider HTTP clients use timeouts, response-size limits, status validation,
  and strict response parsing.
- Client secrets come only from runtime environment variables.
- Authorization codes, access/refresh tokens, client secrets, raw state, and
  PKCE material are never logged.
- Provider access and refresh tokens are never persisted.
- Provider failures return safe messages without response bodies or internal
  implementation details.

## Configuration

Each provider uses three environment variables:

```text
GITHUB_CLIENT_ID
GITHUB_CLIENT_SECRET
GITHUB_REDIRECT_URL

GOOGLE_CLIENT_ID
GOOGLE_CLIENT_SECRET
GOOGLE_REDIRECT_URL
```

A provider is disabled when all three values are absent. Supplying only part of
a provider configuration is a startup error. Only enabled providers appear in
the UI and expose usable authentication routes.

`.env` files and real credentials are never committed. A committed
`.env.example`, if present, contains names/placeholders only. Docker receives
secrets at runtime rather than embedding them in the image or Compose file.

## Routes and HTTP Behavior

New public routes:

```text
GET /auth/github
GET /auth/github/callback
GET /auth/google
GET /auth/google/callback
```

The existing `/register`, `/login`, `/logout`, forum content, reaction, and
filter routes remain unchanged.

OAuth outcomes:

- successful callback: `303 See Other` to `/`;
- malformed, missing, denied, expired, mismatched, or replayed callback:
  `400 Bad Request`;
- existing-email collision: `409 Conflict`;
- provider timeout or invalid upstream response: `502 Bad Gateway`;
- unexpected local persistence/session failure: safe `500 Internal Server Error`.

Normal resource reads use GET and state-changing forum forms use POST. OAuth
start and callback routes are deliberate GET exceptions because redirects are
part of the authorization protocol; they may create/consume temporary OAuth
state and establish a forum session.

## Persistence

The completed forum tables remain mandatory. Authentication adds:

```text
users.password_hash                         nullable for OAuth-only users
oauth_accounts(id, user_id, provider,
               provider_user_id, email,
               created_at)
```

Required OAuth invariants:

- unique `(provider, provider_user_id)`;
- unique `(user_id, provider)`;
- foreign key from `oauth_accounts.user_id` to `users.id` with cascade delete;
- normalized non-empty provider data;
- atomic local-user and OAuth-account creation;
- no provider tokens stored in SQLite;
- one active session per local user regardless of authentication method.

Applied migrations are immutable. Schema changes use a new numbered migration
and preserve existing forum users and content.

## Technology Constraints

- Go with `net/http`, `html/template`, and standard-library OAuth HTTP handling
- SQLite through the exercise-allowed driver
- bcrypt and UUID packages already allowed by the exercises
- server-rendered HTML and CSS only; no JavaScript
- OAuth secrets supplied through the environment
- Docker-compatible build and runtime

Do not add an OAuth dependency that is absent from the exercise's allowed
package list. Provider-specific HTTP requests use standard Go packages.

## Quality and Acceptance

The authentication release is ready only when:

- GitHub and Google first/repeat login work with real development credentials;
- OAuth users can create forum content, log out, and later see persisted content;
- password registration/login and all forum behavior still pass regression tests;
- duplicate registration and invalid credential audit cases remain correct;
- state expiry, mismatch, replay, provider denial, upstream failure, collisions,
  and transaction rollback are tested;
- `gofmt`, `go vet ./...`, `go test ./...`, `go test -race ./...`, and
  `go build ./...` pass;
- the Docker image builds and accepts provider configuration at runtime;
- all checks in `docs/audit-authentication.md` have been exercised;
- no secrets, tokens, database files, or debug artifacts are committed;
- README setup and callback instructions work from a clean checkout.

## Out of Scope

- automatic or explicit account linking
- password creation/reset for OAuth-only users
- storing provider access/refresh tokens or calling provider APIs after login
- OAuth providers other than GitHub and Google
- image upload, moderation, roles, or unrelated forum features
- JavaScript login SDKs, SPA frameworks, or a public JSON API
- HTTPS termination, rate limiting, and deployment infrastructure
