# AGENTS.md — Forum Authentication Implementation Guide

## Project

This repository contains a completed Go web forum that is now being extended by
the forum-authentication exercise.

Preserve the existing email/password flow and all forum behavior. Add exactly
the mandatory GitHub and Google OAuth methods. Do not add other providers,
account linking, password reset, image uploads, moderation, roles, or unrelated
optional features.

The existing bcrypt password handling, UUID sessions, layered architecture,
SQLite persistence, server-rendered templates, and Docker setup remain the
baseline. OAuth must converge on the same local users and forum sessions.

## Source of Truth

When documents disagree, follow this order:

1. `docs/exercise-authentication.md` — supplied authentication subject
2. `docs/audit-authentication.md` — authentication auditor checks
3. `docs/PRD.md` — clarified behavior and fixed security decisions
4. `docs/tasks.md` — implementation order and phase acceptance checks
5. `README.md` — commands verified against the current implementation
6. Existing tests and code — completed forum baseline and regression behavior

Do not silently reinterpret an exercise requirement. Record a genuinely unclear
product decision in the PRD and tasks before implementing it.

## Mandatory Scope

- retain email/password registration, login, and logout
- add GitHub OAuth authorization-code login
- add Google OAuth authorization-code login
- require stable provider user IDs and verified provider emails
- find or transactionally create a local forum user after provider authentication
- reuse the existing UUID session, cookie, middleware, expiry, and one-session rule
- give OAuth users all normal registered-user forum permissions
- preserve posts, comments, categories, reactions, filters, and public reading
- provide safe status codes and provider-independent user-facing errors
- load provider credentials from environment variables only
- test provider adapters with fake HTTP servers and audit real provider flows manually
- keep Docker/runtime configuration free of embedded secrets

## Architecture

Keep the existing layered flow and add a provider boundary:

```text
browser
→ OAuth start/callback handler
→ OAuth provider adapter
→ OAuth login service
→ user/OAuth-account repositories
→ existing session service and manager
→ forum_session cookie
→ authentication middleware
→ local forum user
```

Provider adapters own authorization URLs, code exchange, provider HTTP calls,
and response normalization. They do not create local users or sessions.

The OAuth login service owns identity resolution, collision policy, username
selection, and session orchestration. Repositories own SQL and transactional
local-user/OAuth-account creation. Handlers own HTTP input, cookies, redirects,
and safe error mapping.

Use focused interfaces only at tested boundaries such as provider HTTP behavior,
authorization-state storage, repositories, and session/cookie management. Do not
introduce an interface for every concrete type.

## Authentication Rules

- Password accounts continue to use normalized email plus bcrypt verification.
- Unknown email, wrong password, and password login for an OAuth-only account
  return the identical `401` response and `Wrong email or password` message.
- GitHub identity uses its stable numeric user ID and a primary verified email.
- Google identity uses stable `sub` and requires `email_verified=true`.
- Never identify a provider account by username or email.
- Returning OAuth users are selected by `(provider, provider_user_id)`.
- First OAuth login creates the local user and OAuth account in one transaction.
- OAuth-only users have `NULL` password hashes; never manufacture a password.
- If the provider email already belongs to any local account, return a safe
  `409 Conflict`. Never auto-link identities.
- Normalize generated usernames to the existing 3–32 character rules and use
  numeric suffixes for uniqueness.
- Provider profile changes never create a duplicate local account.
- Every login method creates/replaces the same server-side forum session.
- OAuth-authenticated users have exactly the same forum rights as password users.

## OAuth Flow Security

- Generate at least 32 random state bytes with `crypto/rand`.
- Bind state to the initiating browser with an `HttpOnly`, `SameSite=Lax`,
  short-lived cookie using the configured `Secure` setting.
- Use a concurrency-safe in-memory state store containing only a state hash,
  provider, PKCE verifier, and ten-minute expiry.
- Consume state atomically, compare it in constant time, reject replay or provider
  mismatch, and clear the browser cookie on every callback outcome.
- Use PKCE S256 for both providers.
- Use fixed validated redirect URIs from configuration.
- Give provider HTTP clients timeouts and response-body size limits; validate
  response status and required JSON fields before use.
- Never log authorization codes, tokens, secrets, raw state, PKCE values, or
  provider response bodies.
- Never store access or refresh tokens. Fetch identity and discard them.
- Provider failures must not create a local user, OAuth identity, or forum session.
- Production runs use HTTPS and secure session/state cookies.

## HTTP Rules

- `GET /auth/github` and `GET /auth/google` start provider authorization.
- Provider callbacks use `GET /auth/{provider}/callback`.
- OAuth GET routes are protocol-specific exceptions to the forum's ordinary
  read-only GET rule: they may create/consume temporary state and establish a
  forum session.
- Success redirects to `/` with `303 See Other`.
- Invalid/denied callback input is `400`, email collision is `409`, provider
  failure is `502`, and unexpected local failure is a safe `500`.
- Existing forum method/status behavior remains unchanged.
- All authorization is enforced server-side, never only through UI visibility.
- User input uses parameterized SQL and is escaped by `html/template`.
- Do not add JavaScript files, script tags, inline event handlers, or JavaScript URLs.

## Configuration Rules

Provider variables are:

```text
GITHUB_CLIENT_ID
GITHUB_CLIENT_SECRET
GITHUB_REDIRECT_URL
GOOGLE_CLIENT_ID
GOOGLE_CLIENT_SECRET
GOOGLE_REDIRECT_URL
```

All three absent disables that provider. Partial configuration is a startup
error. Only enabled providers appear in the login UI. `.env` files remain
ignored; examples contain placeholders only; Docker receives secrets at runtime.

## Database Rules

Keep every existing forum table and add `oauth_accounts`. A new migration makes
`users.password_hash` nullable without modifying an applied migration.

Enforce:

- unique normalized user email and username;
- unique `(provider, provider_user_id)`;
- unique `(user_id, provider)`;
- cascading OAuth-account foreign key to users;
- atomic first-time user and OAuth-account creation;
- one active session per local user;
- no provider tokens in SQLite.

Enable SQLite foreign keys on every connection. Use real transactions for
multi-statement writes and real temporary SQLite databases in repository tests.

## Working Rules

- Work in the order in `docs/tasks.md`; a phase is complete only when its tests pass.
- Use only exercise-allowed packages. Implement provider OAuth requests with the
  Go standard library rather than adding an unlisted OAuth dependency.
- Fake only external provider HTTP endpoints in automated tests; do not mock
  SQLite behavior that the audit examines.
- Run focused tests plus `gofmt`, `go vet ./...`, `go test ./...`,
  `go test -race ./...`, and `go build ./...` regularly.
- Keep existing forum tests green throughout the extension.
- Keep secrets, tokens, `.env` files, runtime databases, logs, and build artifacts
  out of Git.
- Do not claim README or Docker commands until verified.
- Keep commits small, coherent, tested, and easy for a learner to review.
