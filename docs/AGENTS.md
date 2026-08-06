# AGENTS.md — Forum Implementation Guide

## Project

This repository is a new Go web forum. It is not the older `lem-in` project.

Build only the mandatory exercise for now. Bcrypt password hashing and UUID
sessions are intentional baseline decisions even though the exercise labels them
as bonus points. Do not implement OAuth, image uploads, moderation, or other
optional features. Keep clean boundaries so they can be added later with small
changes.

## Source of Truth

When documents disagree, follow this order:

1. `docs/exercise.md` — supplied subject; mandatory section first, bonus subjects later
2. `docs/audit.md` — mandatory auditor checks
3. `docs/PRD.md` — clarified product behavior and decisions
4. `docs/tasks.md` — implementation order and acceptance checks
5. `README.md` — commands that are known to work in the current code
6. Existing tests and code

Do not silently reinterpret an explicit exercise requirement. Record unclear
choices in the PRD before implementation.

## Mandatory Scope

- Go server using `net/http` and server-rendered templates
- SQLite persistence
- registration with email, username, and password
- login, logout, expiring cookie, and one active session per user
- public post and comment reading
- authenticated post and comment creation
- one or more categories per post
- authenticated likes/dislikes on posts and comments
- public reaction counts
- filters by category, the current user's created posts, and the current user's liked posts
- correct HTTP methods, statuses, and safe technical-error handling
- Docker image and a build/run helper script
- tests for core behavior and audit flows

## Future-Ready Boundary

Do not add unused future tables, routes, roles, or placeholder implementations.
Readiness means small, focused service/repository/session boundaries that can be
extended later without rewriting mandatory behavior.

## Architecture

Prefer a small layered design:

```txt
browser
→ middleware
→ handler
→ service/domain rule
→ repository
→ SQLite
```

Suggested repository layout:

```txt
cmd/forum/             executable entry point
internal/app/          dependency wiring and routes
internal/database/     SQLite open and schema/migrations
internal/model/        shared domain types
internal/repository/   SQL persistence
internal/service/      business rules
internal/session/      session ID and cookie behavior
internal/web/          handlers, middleware, and rendering
migrations/            SQL schema
templates/             server-rendered HTML
static/                CSS only
data/                  ignored runtime database
```

Avoid an interface for every concrete type. Introduce interfaces at tested
boundaries where substitution is useful.

## Mandatory Rules

- Guests can read posts, comments, categories, and reaction counts.
- Guests cannot create posts/comments or react.
- A registered user can create posts with one or several categories.
- Empty posts and comments are rejected.
- A user has at most one reaction per target. Switching like/dislike replaces
  the prior value; the two states never coexist.
- Created and liked filters always refer to the logged-in user.
- Logging in again replaces that user's previous active session.
- Unknown email and wrong password return the identical `401` response and
  `Wrong email or password` message; never reveal whether an email exists.
- Invalid form data is `400`, a guest on a protected route is `401`, an
  authenticated user lacking permission is `403`, and duplicate registration
  identity data is `409`.
- Session cookies expire and use `HttpOnly`, `SameSite`, and `Path=/`.
- All authorization is enforced on the server, not only by hidden UI controls.
- GET is read-only. State changes use POST.
- User input is passed to parameterized SQL and escaped by `html/template`.
- Request failures do not crash the server or expose SQL/internal errors.
- Do not add JavaScript files, script tags, inline event handlers, or JavaScript URLs.

## Database Baseline

Mandatory tables:

```txt
users
sessions
posts
comments
categories
post_categories
post_reactions
comment_reactions
```

Enable SQLite foreign keys on every connection. Use constraints for unique
email/username, one session per user, valid reaction values, and unique join
rows. Multi-statement writes use transactions.

## Working Rules

- Work in the order in `docs/tasks.md`.
- A phase is complete only when its acceptance tests pass.
- Use temporary SQLite databases in tests; do not mock behavior the audit checks.
- Run `gofmt`, `go vet ./...`, and `go test ./...` regularly.
- Keep runtime database files, secrets, uploads, and build artifacts out of Git.
- Do not claim commands in `README.md` until they have been verified.
- Keep every commit small, coherent, tested, and easy for a learner to review.
