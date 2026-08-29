# Forum Authentication Optional — Task Plan

## Goal

Extend the existing Go forum with **OAuth authentication** while keeping the current email/password authentication and the existing session architecture.

Mandatory providers:

- GitHub
- Google

The main learning goal is to understand the complete OAuth flow:

```text
Browser
→ Forum
→ OAuth Provider
→ Callback to Forum
→ Exchange code for access token
→ Fetch provider user
→ Find/Create local user
→ Create normal forum session
→ Set forum_session cookie
→ Redirect to forum
```

The existing forum session system remains the final authentication mechanism inside the application.

---

## Fixed decisions

- Go only.
- Use only the packages allowed by the exercise. Implement the OAuth HTTP flow
  with the Go standard library (`net/http`, `net/url`, `encoding/json`,
  `crypto/rand`, and related standard packages); do not add an OAuth library.
- Keep the existing layered architecture.
- Keep existing email/password registration and login.
- GitHub + Google OAuth are additional authentication methods.
- Reuse the existing `forum_session` cookie/session manager.
- OAuth client secrets must come from environment variables.
- Never commit OAuth secrets.
- Identify accounts by the provider's stable user ID, never by email or
  username.
- Require a provider-verified email for first-time OAuth registration.
- Do not automatically link accounts. If a provider email already belongs to a
  local user or another provider identity, reject the OAuth login with a safe
  conflict response.
- OAuth users have no local password until an explicit password-setting feature
  exists; never generate a fake password or reusable placeholder hash.
- Create a first-time local user and its OAuth account atomically.
- Use browser-bound, short-lived, single-use `state` and PKCE S256 during OAuth
  authorization.
- Do not persist provider access or refresh tokens; they are needed only long
  enough to fetch the authenticated identity.
- Prefer small phases: understand → test → implement → green → commit.
- Start with GitHub first. Add Google only after the GitHub flow is understood and working.

This file is the implementation plan for the new authentication extension. The
existing forum remains the tested baseline even where older planning documents
describe OAuth as future scope.

---

# Phase 1 — Map the existing authentication flow

### Goal

Before adding OAuth, identify exactly where external authentication must connect to the current forum.

### Understand

Trace the current flow:

```text
POST /login
→ LoginHandler
→ LoginService
→ UserRepository
→ bcrypt
→ SessionRepository
→ SessionManager
→ forum_session cookie
```

Answer:

- Where is the session UUID created?
- Where is it stored in SQLite?
- Where is the cookie written?
- How does authentication middleware recover the current user?
- Which existing code should OAuth reuse after the provider identifies a user?

### Implementation

No production changes yet.

### Verification

You should be able to explain:

> OAuth replaces the password-verification part of login, but after identity is established the forum can reuse its normal session system.

### Commit

```text
docs: map existing authentication flow
```

---

# Phase 2 — OAuth configuration

### Goal

Add configuration required by GitHub and Google without hardcoding secrets.

### Add configuration

Examples:

```text
GITHUB_CLIENT_ID
GITHUB_CLIENT_SECRET
GITHUB_REDIRECT_URL

GOOGLE_CLIENT_ID
GOOGLE_CLIENT_SECRET
GOOGLE_REDIRECT_URL
```

### Understand

Difference between:

- Client ID
- Client Secret
- Redirect URI

The secret belongs only on the server.

### Tests

Test:

- all variables for a provider load successfully
- a provider is disabled when all of its variables are absent
- partially configured provider credentials return a startup error
- malformed or non-HTTP(S) redirect URLs are rejected
- the UI can determine which providers are enabled without seeing secrets

### Security

Add/update:

```text
.env
.env.*
```

in `.gitignore`.

Never commit real OAuth secrets.

Add a secret-free `.env.example`. Docker/Compose configuration must pass values
from the environment and must not contain real credentials.

### Commit

```text
feat: add oauth configuration
```

---

# Phase 3 — OAuth account schema

### Goal

Give the database a clean way to associate local forum users with external providers.

### Existing user compatibility

The current `users.password_hash` column is `NOT NULL`, but a first-time OAuth
user has no password. Add a new migration that safely changes it to nullable.
Do not edit an already-applied migration and do not store a fake password hash.

Update user scanning and password login so a `NULL` hash is represented safely.
An OAuth-only account attempting password login receives the same generic
invalid-credentials response as any other incorrect login.

### OAuth account table

Create a migration for something similar to:

```text
oauth_accounts
--------------
id
user_id
provider
provider_user_id
email
created_at
```

Important constraint:

```text
UNIQUE(provider, provider_user_id)
UNIQUE(user_id, provider)
```

Also require:

- `provider`, `provider_user_id`, and `email` are non-empty
- `user_id` references `users(id)` with `ON DELETE CASCADE`
- provider email is normalized before storage

### Understand

Keep two concepts separate:

```text
users
→ local forum identity

oauth_accounts
→ external identity connected to that user
```

This allows one local user to eventually have multiple authentication methods.
Account-linking itself is outside this project, so the login flow never adds an
OAuth identity to an existing user automatically.

### Tests

Migration test:

- table exists
- `users.password_hash` accepts `NULL` without weakening password accounts
- foreign key points to users
- duplicate `(provider, provider_user_id)` is rejected
- duplicate `(user_id, provider)` is rejected
- deleting a user deletes its OAuth identities
- an OAuth-only user cannot log in through the password form

### Commit

```text
feat: add oauth account migration
```

---

# Phase 4 — OAuth account repository

### Goal

Keep OAuth SQL out of handlers and services.

### Repository responsibilities

Add operations such as:

```text
FindByProviderUserID
Create
FindByUserID
CreateUserWithOAuthAccount
```

Do not put HTTP or provider API logic here.

`CreateUserWithOAuthAccount` owns one transaction that creates the local user
with a `NULL` password hash and creates its OAuth identity. If either insert
fails, neither row remains. Translate uniqueness failures into stable repository
errors so the service can distinguish provider identity, email, and username
conflicts without parsing driver messages.

### Understand

```text
Repository = how OAuth account data is stored/retrieved
Service    = what authentication behavior should happen
```

### Tests

Use real temporary SQLite tests.

Cover:

- create OAuth account
- find OAuth account
- unknown account
- both uniqueness constraints
- first-time user and OAuth account commit together
- either insert failure rolls the whole transaction back
- concurrent duplicate attempts produce one account and one stable conflict

### Commit

```text
feat: add oauth account repository
```

---

# Phase 5 — OAuth provider abstraction

### Goal

Define the information the forum actually needs from an OAuth provider.

### Suggested model

Conceptually:

```text
OAuthUser
- Provider
- ProviderUserID
- VerifiedEmail
- SuggestedUsername
```

Define a provider interface so GitHub and Google expose the same operations to
the handler/service.

Conceptually:

```text
AuthorizationURL(...)
ExchangeCode(...)
FetchUser(...)
```

Provider implementations must use an injected `http.Client` and configurable
endpoint URLs. Production uses the official endpoints; tests use local fake HTTP
servers and never contact GitHub or Google.

Every provider request must have a timeout, check the response status, limit the
response body size, and reject malformed or incomplete JSON. Public errors must
not contain response bodies, tokens, codes, client secrets, or internal details.

### Understand

The forum should not care whether the identity came from GitHub or Google after it has been normalized into an `OAuthUser`.

### Tests

Unit-test provider-independent logic.

Do not contact real GitHub/Google APIs in normal unit tests.

### Commit

```text
feat: define oauth provider contract
```

---

# Phase 6 — GitHub OAuth: authorization start

### Goal

Implement only the first half of GitHub OAuth.

### Route

```text
GET /auth/github
```

### Flow

```text
Browser
→ GET /auth/github
→ generate random state
→ store state temporarily
→ redirect to GitHub authorization URL
```

### Security concept — state

Understand why:

```text
state sent to provider
        ↓
provider returns same state
        ↓
callback verifies it
```

Use this fixed policy:

- generate at least 32 random bytes with `crypto/rand` and encode them safely
- bind the state to the initiating browser with an `HttpOnly`, `SameSite=Lax`
  cookie using the configured `Secure` setting
- use a small concurrency-safe in-memory `OAuthStateStore` for this single-process
  application; store a hash of the state together with provider, PKCE verifier,
  and expiry, never the raw state
- expire authorization-flow data after 10 minutes; an application restart may
  safely invalidate unfinished OAuth attempts
- consume state atomically so a callback cannot be replayed
- compare returned and expected state in constant time
- clear the state cookie on success and every callback failure
- keep GitHub and Google state separate so one provider cannot consume the
  other's authorization attempt

Use PKCE S256 too: generate a cryptographically random verifier, send its SHA-256
challenge on authorization, retain the verifier only for this short-lived flow,
and send it during code exchange. Never log state values or PKCE material.

### Tests

Verify:

- route returns redirect
- redirect points to expected provider authorization endpoint
- state exists
- required OAuth parameters are present
- state is browser-bound, expires, and is single-use
- PKCE uses `code_challenge_method=S256`
- a state generated for Google cannot be used for GitHub

### Commit

```text
feat: start github oauth authorization
```

---

# Phase 7 — GitHub OAuth: callback and provider API

### Goal

Complete communication with GitHub.

### Route

```text
GET /auth/github/callback
```

### Flow

```text
GitHub callback
→ verify state
→ read authorization code
→ exchange code for access token
→ call GitHub user API
→ obtain stable numeric provider user ID
→ fetch a primary verified email from /user/emails
→ normalize to OAuthUser
```

Request the minimum `user:email` scope. Do not treat the GitHub login name or
email address as the provider identity. If no primary verified email is
available, stop with a safe authentication error and do not create local data.

### Important

Do not log:

- authorization codes
- access tokens
- refresh tokens
- client secrets
- OAuth state
- PKCE verifier/challenge values

### Tests

Use fake HTTP servers/provider mocks.

Cover:

- valid callback
- wrong/missing state
- expired or replayed state
- provider-denied authorization (`error` callback parameter)
- missing code
- token exchange failure
- non-success provider status
- malformed or oversized provider response
- provider user request failure
- missing provider user ID
- missing or unverified primary email
- HTTP client timeout

### Commit

```text
feat: complete github oauth callback
```

---

# Phase 8 — OAuth login service

### Goal

Connect external identity to the forum's local user system.

### Business logic

Given an `OAuthUser`:

```text
existing oauth account?
    ↓ yes
load local user
    ↓
create forum session
```

For a first-time OAuth user:

```text
no oauth account
→ require normalized, provider-verified email
→ reject if that email already belongs to any local user
→ generate an available local username
→ transactionally create local user + oauth_accounts row
→ create forum session
```

### Fixed email and account-linking policy

Never use matching email as proof that two accounts have the same owner.

If the verified provider email already belongs to a password account or a
different OAuth identity, return `409 Conflict` with a safe message directing
the user to sign in using the original method. Do not reveal database details
and do not create or link any rows. A future explicit account-linking flow is
outside this project.

Returning OAuth users are found by `(provider, provider_user_id)`. A later email
or username change at the provider must not create a second local user or change
which local account is selected.

### Username policy

Build the local username from the provider's suggested username/name:

1. trim whitespace and replace unsupported/spacing characters with `-`;
2. fall back to `<provider>-user` if fewer than 3 usable characters remain;
3. truncate to the existing 32-character limit;
4. if occupied, reserve space and try `-2`, `-3`, and so on;
5. handle the final uniqueness race inside the transactional repository flow.

The generated username is display data only; it is never an authentication key.

### Reuse

The service should reuse the existing:

```text
SessionRepository
SessionManager
one-active-session behavior
```

where possible.

### Tests

Cover:

- existing OAuth user login
- first OAuth login
- session creation
- duplicate provider identity
- verified provider email requirement
- password-account and other-provider email collisions return `409`
- collision rejection leaves no partial user or OAuth account
- username normalization, fallback, truncation, suffixing, and uniqueness race
- provider email/username changes do not duplicate an existing OAuth user
- OAuth-only accounts fail password login with the generic credentials response

### Commit

```text
feat: add oauth login service
```

---

# Phase 9 — Connect GitHub callback to forum session

### Goal

Finish the complete GitHub login flow.

### Full flow

```text
GET /auth/github
→ GitHub
→ /auth/github/callback
→ OAuthLoginService
→ local user
→ session UUID
→ forum_session cookie
→ 303 /
```

### Understand

At this point GitHub is no longer needed for ordinary forum requests.

After login:

```text
Browser
→ forum_session cookie
→ Authentication Middleware
→ SessionRepository
→ local user
```

exactly like the original login flow.

### HTTP outcomes

- successful callback: `303 See Other` to `/`
- missing, malformed, denied, expired, mismatched, or replayed callback: `400 Bad Request`
- existing-email collision: `409 Conflict`
- provider timeout or invalid upstream response: `502 Bad Gateway`
- unexpected local persistence/session failure: safe `500 Internal Server Error`

None of these responses may expose provider payloads or internal errors.

### Integration tests

Verify:

- successful OAuth login sets forum session
- first login creates exactly one local user and OAuth account
- repeat login reuses them and replaces the prior active forum session
- authenticated page recognizes user
- invalid callback does not authenticate user
- failure after writing a cookie clears that cookie and leaves no usable session

### Commit

```text
feat: integrate github login with forum sessions
```

---

# Phase 10 — Add GitHub login UI

### Goal

Expose GitHub authentication in the existing interface.

### UI

Add:

```text
Continue with GitHub
```

to login and/or registration page.

Show the link only when GitHub is fully configured. A disabled provider must not
expose a broken login route or UI control.

Preserve existing:

```text
email + password
```

authentication.

No JavaScript is necessary.

### Tests

Verify the link targets:

```text
/auth/github
```

and existing login/register forms still work.

Also verify the link is absent when GitHub configuration is disabled.

### Commit

```text
feat: add github login interface
```

---

# Phase 11 — Google OAuth provider

### Goal

Repeat the now-understood OAuth flow for Google.

### Routes

```text
GET /auth/google
GET /auth/google/callback
```

### Identity requirements

Request only `openid email profile`. Use Google's stable `sub` claim as
`ProviderUserID`; never use email as the provider identity. Require a non-empty
email and `email_verified=true`. Missing or unverified email stops the flow
without creating local data or a forum session.

### Reuse

Do **not** duplicate GitHub business logic.

Only provider-specific behavior should differ:

```text
authorization endpoint
token endpoint
profile endpoint
provider response parsing
```

The rest should reuse:

```text
OAuthUser
OAuthLoginService
OAuthAccountRepository
existing forum sessions
```

### Tests

Mirror the GitHub provider tests:

- authorization redirect
- state validation
- state expiry and replay rejection
- PKCE S256 and verifier exchange
- code exchange
- profile retrieval
- stable `sub` mapping
- missing or unverified email rejection
- provider denial, timeout, non-success status, malformed JSON, and oversized body
- error cases

### Commit

```text
feat: add google oauth provider
```

---

# Phase 12 — Google login integration and UI

### Goal

Complete Google authentication end-to-end.

### UI

Add:

```text
Continue with Google
```

Show the link only when Google is fully configured. A disabled provider must not
expose a broken login route or UI control.

### Integration

Verify:

```text
Google
→ callback
→ local user
→ forum session
→ middleware
→ authenticated forum
```

### Regression tests

Ensure:

- GitHub still works
- password login still works
- registration still works
- logout works for OAuth-authenticated users too

### Commit

```text
feat: integrate google oauth login
```

---

# Phase 13 — OAuth security audit

### Goal

Review the new authentication flow as a security feature, not only as working code.

### Check

- OAuth `state` has at least 32 random bytes and is validated in constant time.
- State is bound to the initiating browser, expires after 10 minutes, is
  provider-specific, is atomically single-use, and is cleared on every callback.
- Authorization code flow uses PKCE S256 and short-lived verifier storage.
- Client secrets are not committed.
- Access tokens are not logged.
- Access and refresh tokens are not persisted.
- Authorization codes are not logged.
- State and PKCE material are not logged.
- Redirect URIs are fixed/configured.
- Provider HTTP clients have timeouts and response-size limits.
- Provider errors and response bodies are not exposed to users.
- Session cookie remains `HttpOnly`.
- Production session cookie uses `Secure`.
- Appropriate `SameSite` policy remains enabled.
- Existing sessions still expire.
- Logout invalidates the local forum session.
- Provider identity is based on stable provider user ID, not only username.
- GitHub and Google emails must be verified before first registration.
- Email collisions are rejected and never trigger automatic linking.
- First-time user and OAuth identity creation is transactional.
- OAuth-only users do not have fake password hashes.

### Tests

Add missing negative/security cases.

### Commit

```text
test: harden oauth authentication flow
```

---

# Phase 14 — Full HTTP integration flows

### Goal

Test OAuth as part of the whole forum rather than as isolated functions.

### Scenarios

GitHub:

```text
guest
→ start OAuth
→ callback
→ logged in
→ access protected route
→ logout
→ protected route becomes unauthorized
```

Google:

Same flow.

Also verify:

- existing password user flow
- two different browser sessions
- malformed callback
- rejected OAuth state
- expired and replayed OAuth state
- callback from the wrong provider
- provider-denied authorization
- verified-email and email-collision failures
- provider failure does not create a session
- every failed first login leaves no partial user or OAuth account
- first and repeat OAuth login obey the one-active-session rule
- OAuth users can create posts/comments/reactions and see their content after
  logout, as required by the authentication audit

### Commit

```text
test: cover oauth http flows
```

---

# Phase 15 — README and setup guide

### Goal

Make another developer able to configure the project without reading the implementation.

### Document

- What OAuth is doing in this forum
- GitHub OAuth setup
- Google OAuth setup
- required environment variables
- callback URLs
- local run
- Docker/Compose configuration if used
- secrets policy
- architecture flow
- test commands

### Commit

```text
docs: document oauth authentication
```

---

# Phase 16 — Final audit

### Code quality

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./...
docker build -t forum-authentication .
```

### Functional audit

Verify manually:

- email/password registration
- email/password login
- GitHub login
- Google login
- OAuth first login
- OAuth repeat login
- protected routes
- logout
- posts/comments/reactions/filters remain functional
- both real provider flows work with registered development credentials
- a production-like run uses HTTPS and secure cookies

### Database audit

Check:

```text
users
oauth_accounts
sessions
```

Verify OAuth-only users have `NULL` password hashes, provider IDs are stable,
failed attempts leave no orphan rows, and only one active session exists per
user.

### Security audit

Check repository history/current files for:

```text
client secrets
access tokens
.env
database files
logs containing tokens
```

Also verify `.env.example` contains names/placeholders only, Docker receives
secrets at runtime, provider tokens are absent from SQLite, and every command in
the README works from a clean checkout.

### Final commit

```text
chore: complete oauth authentication audit
```

---

# Final mental model

```text
PASSWORD LOGIN

email/password
→ bcrypt verification
→ local user
→ forum session
→ cookie


OAUTH LOGIN

Google/GitHub
→ provider authenticates user
→ callback proves successful OAuth flow
→ forum resolves local user
→ forum session
→ cookie
```

Both flows converge here:

```text
forum_session
→ authentication middleware
→ session row
→ user_id
→ current user
```

That convergence is the key architectural idea of the optional project.
