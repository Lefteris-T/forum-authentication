# Forum — Detailed Mandatory TDD Task Plan

## Project goal

Build the mandatory forum as a server-rendered Go application. Guests can read
posts and comments. Registered users can create content, react, and use personal
filters. SQLite persists the data and Docker runs the application.

This plan intentionally includes bcrypt password hashing and UUID session IDs in
the mandatory baseline. Although the exercise calls them bonus points, they are
small, allowed dependencies and avoid building authentication around unsafe
temporary behavior.

There are no bonus implementation tasks in this document. The architecture has
clean boundaries for later extensions, but the mandatory release must contain no
unused bonus tables, routes, roles, upload logic, or OAuth code.

## Fixed decisions

- Backend: Go with `net/http`
- Views: Go `html/template`
- Frontend: HTML and CSS only
- JavaScript: forbidden everywhere in this project
- Database: SQLite through the exercise-allowed `sqlite3` package
- Passwords: bcrypt hashes; plaintext is never stored
- Sessions: server-side records with UUID identifiers
- Session policy: one active session per user; a new login replaces the old one
- State changes: POST only
- Successful forms: Post/Redirect/Get with `303 See Other`
- Invalid or malformed form input: `400 Bad Request`
- Guest requesting a protected route: `401 Unauthorized`
- Authenticated user lacking permission: `403 Forbidden`
- Duplicate registration email/username: `409 Conflict`
- Unknown login email or wrong password: `401 Unauthorized` with the same
  `Wrong email or password` message
- Repeated identical reaction: toggle it off
- Opposite reaction: replace the existing reaction
- Development method: tests first, one small understandable commit per phase

## Architecture goal

Keep the previous layered architecture, but introduce interfaces only at useful
boundaries. Business rules must be testable without starting the full server.

```txt
forum/
├── cmd/
│   └── forum/
│       └── main.go
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   └── routes.go
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   ├── database.go
│   │   └── migrate.go
│   ├── model/
│   │   ├── user.go
│   │   ├── session.go
│   │   ├── post.go
│   │   ├── comment.go
│   │   ├── category.go
│   │   └── reaction.go
│   ├── repository/
│   │   ├── users.go
│   │   ├── sessions.go
│   │   ├── posts.go
│   │   ├── comments.go
│   │   ├── categories.go
│   │   └── reactions.go
│   ├── service/
│   │   ├── auth.go
│   │   ├── posts.go
│   │   ├── comments.go
│   │   └── reactions.go
│   ├── session/
│   │   └── manager.go
│   ├── validation/
│   │   ├── auth.go
│   │   └── content.go
│   └── web/
│       ├── handler/
│       ├── middleware/
│       └── view/
├── migrations/
│   ├── 001_auth.sql
│   ├── 002_forum_content.sql
│   └── 003_seed_categories.sql
├── templates/
├── static/
│   └── css/
├── data/
├── scripts/
├── testdata/
├── Dockerfile
├── compose.yml
├── go.mod
└── README.md
```

Request flow:

```txt
browser request
→ recovery/logging/authentication middleware
→ HTTP handler
→ input validation
→ service business rule
→ repository
→ SQLite
→ template or redirect
```

Future changes should plug into boundaries instead of changing mandatory flows:

- another login method resolves a local user and calls the same session manager;
- another post attachment type extends post creation without changing auth;
- authorization middleware can later use roles without moving rules into templates;
- new migrations extend the schema without rewriting any applied migration.

Do not implement any of those future features now.

## Mandatory database

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

Important constraints:

- normalized email and username are unique;
- password column contains only a bcrypt hash;
- session UUID is unique and `sessions.user_id` is unique;
- foreign keys are enabled on every SQLite connection;
- reaction value is only `-1` or `1`;
- one reaction exists per user and target;
- one post/category pair exists once;
- multi-row writes are transactional.

## Mandatory routes

```txt
GET  /                         list and filter posts
GET  /register                 registration form
POST /register                 create user
GET  /login                    login form
POST /login                    create/replace session
POST /logout                   delete session
GET  /posts/new                authenticated post form
POST /posts                    authenticated post creation
GET  /posts/{id}               public post detail
POST /posts/{id}/comments      authenticated comment creation
POST /posts/{id}/reaction      authenticated post reaction
POST /comments/{id}/reaction   authenticated comment reaction
GET  /static/...               CSS only
```

Filters use `/?category=<id>`, `/?filter=created`, and `/?filter=liked`.

## How to use this plan

Each phase is one small green commit. For every phase:

1. add the listed test;
2. run it and see the expected failure;
3. implement only that phase;
4. run the focused test and `go test ./...`;
5. run `gofmt`;
6. make the suggested commit.

Never commit a knowingly failing test to the main learning branch. The test and
the small implementation normally belong in the same commit.

---

# Foundation

## Phase 1 — Initialize the Go module

Create `go.mod`, the command directory, and a minimal executable.

- Test: `go test ./...` discovers the module successfully.
- Done when: `go run ./cmd/forum` compiles and starts a minimal server.
- Commit: `chore: initialize forum go module`

## Phase 2 — Add application configuration

Create a config type for address, database path, session duration, cookie name,
and secure-cookie mode.

- Tests: valid defaults; environment overrides; invalid address, duration,
  database path, or cookie name is rejected.
- Rule: configuration parsing returns errors and never calls `log.Fatal`.
- Commit: `feat: add application configuration`

## Phase 3 — Add HTTP server lifecycle

Wire configuration into an `app` package, construct `http.Server`, and support
graceful shutdown through `context.Context` with a defined timeout. OS signal
handling belongs only in `cmd/forum/main.go`.

- Tests: invalid listen address/startup failure is returned; cancellation shuts
  down an active server within the timeout; normal `http.ErrServerClosed` is not
  treated as an application failure; other server errors reach the caller.
- Rules: this phase contains no database/repository logic and does not claim to
  guarantee database integrity—that is handled by transactions and connection rules.
- Commit: `feat: add http server lifecycle`

## Phase 4 — Define domain models

Add focused types for user, session, post, comment, category, and reaction.

- Tests: reaction constants are exactly dislike `-1` and like `1`.
- Rule: models contain data, not SQL or HTTP behavior.
- Do not add roles, provider identities, reports, or images.
- Commit: `feat: define mandatory forum models`

---

# SQLite foundation

## Phase 5 — Open SQLite safely

Implement `database.Open(path)` with pinging and SQLite-appropriate connection
settings.

- Tests: temporary file opens; empty/invalid path fails.
- Rule: ensure foreign-key enforcement applies to every connection, not only one.
- Commit: `feat: open sqlite database safely`

## Phase 6 — Build the migration runner

Apply numbered SQL migrations in order and record completed versions.

- Tests: first run applies; second run is harmless; a newly added migration runs
  without rerunning older ones; failed migration rolls back and is not recorded.
- Rules: each migration is transactional, errors retain useful context, and an
  applied migration is immutable—every later schema change gets a new file.
- Commit: `feat: add transactional migration runner`

## Phase 7 — Create user and session tables

Create `001_auth.sql` containing only `users` and `sessions`.

- Tests: tables/columns exist; duplicate email/username/session-user fail.
- Include timestamps and session expiry.
- Commit: `feat: add authentication schema`

## Phase 8 — Create content and reaction tables

Create `002_forum_content.sql` containing posts, comments, categories,
post-category relations, and both reaction tables. It may reference the user
table because `001_auth.sql` is guaranteed to run first.

- Tests: foreign keys fail correctly; duplicate post/category relation fails;
  duplicate reactions and values other than `-1`/`1` fail.
- Decide and test deliberate delete behavior for each relation.
- Commit: `feat: add forum content and reaction schema`

## Phase 9 — Seed mandatory categories

Create `003_seed_categories.sql` with a short deterministic category list.

- Tests: expected order/content; migrations run in numeric order; running setup
  twice creates no duplicates; an existing database receives migration 003 once.
- Keep the list small enough to understand during the audit.
- Commit: `feat: seed forum categories`

---

# Registration

## Phase 10 — Validate registration input

Validate and normalize email, username, and password before database access.

- Tests: valid input; blank fields; whitespace; malformed email; length bounds.
- Rule: document normalization so login uses the identical form.
- Commit: `test: define registration validation rules`

## Phase 11 — Add bcrypt password handling

Wrap bcrypt behind a small password component with `Hash` and `Compare`.

- Tests: hash differs from plaintext; correct password matches; wrong one fails.
- Rule: never log or return plaintext/hash values in errors.
- Commit: `feat: add bcrypt password protection`

## Phase 12 — Implement the user repository

Add parameterized user creation and lookup by email/ID.

- Tests use real temporary SQLite: create/select/not-found/conflicts.
- Translate driver constraint errors into stable domain errors.
- Commit: `feat: add sqlite user repository`

## Phase 13 — Implement registration service

Compose validation, hashing, and repository creation.

- Tests: valid registration hashes before storage; invalid input skips repository;
  duplicate email and username remain distinguishable to the form.
- Commit: `feat: add user registration service`

## Phase 14 — Build the template renderer

Parse templates once and render status/data consistently.

- Tests: correct template/status; user HTML escaped; missing template fails safely.
- Rule: use only `html/template`; never use `template.HTML` for user content.
- Commit: `feat: add safe html template renderer`

## Phase 15 — Add registration pages

Implement `GET /register` and `POST /register` with an HTML form.

- Tests: GET 200; valid POST 303; malformed/invalid fields render the form with
  400; duplicate email or username renders it with 409; wrong method returns 405
  with `Allow`; password is never rendered back.
- Rule: a repository uniqueness conflict must still become 409, including a
  race where validation passed before another request inserted the same value.
- Commit: `feat: add registration handlers and template`

---

# Login and sessions

## Phase 16 — Validate login input

Reuse email normalization and reject empty/malformed credentials.

- Tests: valid, blank, whitespace, malformed email, oversized input. These input
  errors become 400 at the handler; they are different from incorrect credentials.
- Commit: `test: define login validation rules`

## Phase 17 — Implement the session repository

Add UUID session insert/replace, lookup, deletion, and expired cleanup.

- Tests: CRUD; expiry; second session atomically replaces first for the user.
- Rule: a failed replacement must not leave two sessions or lose a valid one.
- Commit: `feat: add sqlite session repository`

## Phase 18 — Implement the UUID session manager

Generate UUIDs and create/read/clear the browser cookie.

- Tests: unique valid UUIDs; expiry; `HttpOnly`; `SameSite=Lax`; `Path=/`;
  configurable `Secure`; missing/malformed cookie treated as guest.
- Cookie contains no user ID, email, or password.
- Commit: `feat: add uuid session manager`

## Phase 19 — Implement login and logout services

Compare bcrypt credentials and delegate all session state to the manager.

- Tests: correct login; unknown email and wrong password produce the same generic
  invalid-credentials error without revealing which credential failed; password
  comparison is still performed against a fixed valid dummy bcrypt hash when the
  email is unknown; empty input stops early; new login replaces old; logout
  deletes session.
- Rule: do not assert exact response durations in tests. Instead, inject the
  password comparer and verify that both unknown-email and wrong-password paths
  perform a bcrypt comparison.
- Commit: `feat: add login and logout services`

## Phase 20 — Add authentication and authorization middleware

Resolve the session/user once, put the current user in request context, and add a
simple `RequireAuth` boundary for protected routes.

- Tests: guest; valid session; expired/unknown session; current user available;
  guest rejected consistently from protected handler; valid user reaches it.
- Keep authentication resolution independent of future login providers.
- Do not add role logic yet; later authorization can extend this boundary.
- Commit: `feat: add authentication middleware`

## Phase 21 — Add login and logout pages

Implement login form processing and POST-only logout.

- Tests: GET login 200; success sets cookie and redirects; empty/malformed input
  renders a warning with 400; unknown email and wrong password both render the
  exact safe message `Wrong email or password` with 401; logout clears the
  cookie; GET logout is 405.
- Rule: response body and logs must not reveal whether the submitted email exists.
- Commit: `feat: add login and logout handlers`

## Phase 22 — Verify the one-session browser rule

Use two independent cookie jars against `httptest.Server`.

- Tests: guest browser stays guest; login in browser B invalidates browser A;
  refresh reflects both states correctly.
- Commit: `test: verify single active browser session`

---

# Public forum and posts

## Phase 23 — Implement category repository

Load all categories deterministically and validate selected IDs.

- Tests: ordered list; known IDs; unknown IDs; duplicate input policy.
- Commit: `feat: add category repository`

## Phase 24 — Validate post input

Require bounded non-empty title/body and at least one category.

- Tests: whitespace, missing category, one/many categories, duplicates, limits.
- Commit: `test: define post validation rules`

## Phase 25 — Implement transactional post creation

Insert a post and all selected categories in one transaction.

- Tests: one category; several; author stored; unknown category rolls back all.
- Commit: `feat: create posts transactionally`

## Phase 26 — Add the post service

Combine authentication identity, validation, and repository behavior.

- Tests: guest cannot create; invalid request stops early; valid author is used.
- Handler input must never choose another user's author ID.
- Commit: `feat: add post creation service`

## Phase 27 — Query public post lists and details

Return public list items newest-first and one post with its ordered comments.
Include authors, categories, and post/comment reaction counts.

- Tests: empty list; full view data; count accuracy; deterministic ties; no
  duplicate rows from joins; comment ordering; missing post domain error.
- Avoid an avoidable query per post.
- Commit: `feat: query forum posts and comments`

## Phase 28 — Add public home and post detail pages

Render the post list and individual post/comment view for guests and users.

- Tests: empty/non-empty 200 pages; escaped user content; navigation reflects
  auth; malformed post ID 400; missing post 404.
- HTML/CSS only: no script tag, inline event attribute, or JavaScript URL.
- Commit: `feat: add public forum pages`

## Phase 29 — Add post creation pages

Implement `GET /posts/new` and `POST /posts`.

- Tests: guest rejected; user sees categories; one/many selections work;
  empty/invalid input 400; success 303.
- Commit: `feat: add post creation handlers and template`

---

# Comments and reactions

## Phase 30 — Validate and store comments

Add validation, repository insertion, and the comment service.

- Tests: valid; empty/whitespace/oversized; unknown post; author stored.
- Commit: `feat: create validated comments`

## Phase 31 — Add comment submission handler

Implement `POST /posts/{id}/comments` and redirect to the post.

- Tests: guest rejected; valid 303; bad ID 400; missing post 404; GET 405.
- Commit: `feat: add comment submission handler`

## Phase 32 — Implement reaction persistence and services

Add atomic insert, toggle-off, and switch behavior for both target types, then
wrap it with identity, target, and value validation in a reaction service.

- Tests: every state transition; guest rejected; invalid values stop early;
  missing target; uniqueness remains; counts follow transitions.
- Keep common transition logic shared without hiding target-specific SQL.
- Commit: `feat: add forum reaction behavior`

## Phase 33 — Add post reaction handler

Implement the POST route with ordinary HTML forms.

- Tests: auth, like/dislike/toggle/switch, bad ID/value, missing post, GET 405.
- Commit: `feat: add post reaction handler`

## Phase 34 — Add comment reaction handler

Implement the equivalent comment POST route.

- Tests: auth, all transitions, public counts after refresh, bad/missing target.
- Commit: `feat: add comment reaction handler`

---

# Filters

## Phase 35 — Add category filtering

Implement public `/?category=<id>` filtering.

- Tests: exact results; multi-category post appears once; malformed ID 400;
  unknown category policy is consistent and documented.
- Commit: `feat: filter posts by category`

## Phase 36 — Add created-post filtering

Implement authenticated `/?filter=created` using only the current user ID.

- Tests: guest rejected; only own posts; full category/count view data retained.
- Commit: `feat: filter current user posts`

## Phase 37 — Add liked-post filtering

Implement authenticated `/?filter=liked` for current reaction value `1`.

- Tests: likes included; dislikes/removed likes excluded; other users isolated;
  duplicate rows impossible.
- Commit: `feat: filter current user liked posts`

---

# HTTP quality, presentation, and audit

## Phase 38 — Centralize routes, methods, and error pages

Register every route in one place, enforce its method, and apply this convention:

```txt
400  malformed or invalid request input
401  guest on protected route, or incorrect login credentials
403  authenticated user lacks permission
404  resource does not exist
405  HTTP method is not allowed
409  registration email or username already exists
500  unexpected internal failure
```

The mandatory feature set may not naturally produce a 403 because it has no
roles or ownership-based edit/delete routes. Keep the convention explicit and
test the shared error response without inventing an unnecessary feature.

- Tests: full route/method table; every 405 includes the correct `Allow` value;
  each status renders correctly; duplicate registration is always 409; both
  incorrect-login cases are indistinguishable 401 responses; internal details
  stay hidden.
- GET must never mutate SQLite.
- Commit: `feat: centralize forum routes and errors`

## Phase 39 — Add recovery and request logging

Recover request panics, log useful context, and keep the server running.

- Tests: panic becomes safe 500; following request succeeds; secrets are absent.
- Commit: `feat: add recovery and request logging`

## Phase 40 — Finish the HTML and CSS presentation

Create accessible navigation, forms, lists, responsive layout, and visible errors.

- Tests/checks: labels, keyboard submission, readable mobile layout, static CSS 200.
- Repository-wide check finds no `.js`, `<script>`, `javascript:`, or inline `on*` handlers.
- Commit: `style: complete forum html and css`

## Phase 41 — Add complete HTTP integration flows

Exercise registration through filters using `httptest.Server` and real SQLite.

- Cover guest/user permission matrices, content visibility across browsers,
  reactions, filters, redirects, and all important error statuses.
- Commit: `test: cover mandatory forum http flows`

## Phase 42 — Add SQLite audit verification

Create data through HTTP, then query tables directly.

- Verify users/posts/comments exist; password is a bcrypt hash; CREATE, INSERT,
  and SELECT queries are present and executed.
- Commit: `test: verify forum sqlite persistence`

## Phase 43 — Add Docker, Compose, and helper scripts

Create a multi-stage `Dockerfile` compatible with the chosen SQLite driver,
Compose configuration with persistent data, and simple build/run scripts.

- Verify clean build, port 8080, templates/CSS, non-root operation where
  practical, persistent `/app/data`, clean shutdown, and scripts that use
  `set -eu`, propagate failures, and do not silently duplicate containers.
- Commit: `feat: dockerize forum with run helpers`

## Phase 44 — Write the final README

Document features, architecture, schema, routes, sessions, local run, Docker,
tests, and audit notes using only commands that were actually verified.

- Commit: `docs: add forum usage and architecture guide`

## Phase 45 — Run the mandatory release audit

Run every item in `docs/audit.md`, record/fix failures, and then run:

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./...
docker build -t forum .
```

Final checks:

- [ ] registration/login and two-browser session behavior pass
- [ ] guest and user permission matrices pass
- [ ] posts, comments, reactions, and all filters pass
- [ ] SQLite CLI queries show auditor-created data
- [ ] error statuses and unexpected-failure recovery pass
- [ ] only exercise-allowed Go dependencies are used
- [ ] no JavaScript exists anywhere
- [ ] no database, secret, upload, debug, or build artifact is committed
- [ ] README commands work from a clean checkout
- [ ] Docker image and container work with persistent data

- Commit: `chore: prepare mandatory forum release`
- Tag only after the audit passes: `v1.0.0`

## Quick commands

```bash
go run ./cmd/forum
go test ./...
go test -race ./...
go vet ./...
go build ./...
gofmt -w .
sqlite3 data/forum.db
docker build -t forum .
docker run --name forum -p 8080:8080 -v forum-data:/app/data forum
```
