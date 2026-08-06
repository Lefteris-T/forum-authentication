# Forum Product Requirements

## Purpose

Build a server-rendered web forum in Go. Everyone can browse discussions.
Registered users can publish posts, comment, react, and use personal filters.
SQLite stores all application data and Docker runs the finished application.

Only the mandatory forum is planned now. Its boundaries should support future
extensions without adding unused optional code or schema.

## Mandatory Release (`v1.0`)

### Visitors and authentication

- Registration asks for email, username, and password.
- Email and username are unique; conflicts produce a clear user-facing error.
- Login validates email/password. An unknown email and a wrong password both
  return `401 Unauthorized` with `Wrong email or password`; the response never
  reveals which credential failed.
- Empty or malformed input produces a warning and an appropriate HTTP status.
- Login creates a server-side session referenced by an expiring cookie.
- A user may have only one active session. A new login invalidates the old one.
- Logout invalidates the server-side session and clears the cookie.
- A request without a valid, unexpired session is a guest request.

Passwords are hashed with bcrypt and sessions use UUID identifiers from the
start. These are intentional project decisions even though the supplied exercise
labels them as bonus points.

### Communication

- Guests and users can list posts and open post details.
- Post details show comments.
- Only authenticated users can create posts and comments.
- A post has a non-empty title/body and one or more categories.
- A comment has non-empty content and belongs to an existing post.
- Newly created content is persisted and visible to other browsers after refresh.

### Reactions

- Only authenticated users can like or dislike posts and comments.
- Counts are visible to everyone.
- Each user has zero or one reaction per post/comment.
- Repeating a reaction may toggle it off; choosing the opposite reaction switches it.

### Filters

- Category filtering is public.
- “My posts” returns only posts created by the current user.
- “Liked posts” returns only posts currently liked by the current user.
- Personal filters require authentication.

### HTTP and error behavior

- Read routes use GET; state changes use POST.
- Unsupported methods return `405` with an `Allow` header.
- Malformed or invalid request data returns `400`.
- A guest requesting a protected route receives `401 Unauthorized`.
- An authenticated user without permission receives `403 Forbidden`. The
  mandatory feature set may not naturally exercise this because it has no roles.
- Missing resources return `404`.
- Duplicate registration email/username renders the form with `409 Conflict`.
- Unexpected failures return a safe `500` page and are logged server-side.
- A panic in one request must not terminate the server.
- Successful form submissions use Post/Redirect/Get (`303 See Other`).

### Persistence

SQLite is the only database. The schema contains at least:

```txt
users(id, email, username, password, created_at)
sessions(id, user_id, expires_at)
posts(id, user_id, title, content, created_at)
comments(id, post_id, user_id, content, created_at)
categories(id, name)
post_categories(post_id, category_id)
post_reactions(user_id, post_id, value)
comment_reactions(user_id, comment_id, value)
```

Required invariants:

- unique normalized email and username;
- one active session row per user;
- foreign keys enabled;
- reaction value restricted to `-1` or `1`;
- one reaction per user/target;
- no duplicate post/category association;
- post plus categories is created atomically.

The implementation must visibly contain and execute CREATE, INSERT, and SELECT
queries. Auditor-created users, posts, and comments must be queryable with the
`sqlite3` command-line client.

### Technology constraints

- Go
- `net/http` and `html/template`
- SQLite using the exercise-allowed `sqlite3` package
- server-rendered HTML and CSS only
- no JavaScript files, script tags, inline event handlers, or JavaScript URLs
- only packages allowed by the exercise
- Docker

The selected SQLite driver and Docker image must be compatible. If using a CGO
driver such as `github.com/mattn/go-sqlite3`, both build and runtime stages must
include the required toolchain/runtime libraries.

## Routes for `v1.0`

Exact paths may change, but the behavior must remain consistent:

```txt
GET  /                         list/filter posts
GET  /register                 registration form
POST /register                 create account
GET  /login                    login form
POST /login                    create/replace session
POST /logout                   end session
GET  /posts/new                post form (authenticated)
POST /posts                    create post (authenticated)
GET  /posts/{id}               post and comments
POST /posts/{id}/comments      create comment (authenticated)
POST /posts/{id}/reaction      react to post (authenticated)
POST /comments/{id}/reaction   react to comment (authenticated)
```

Filters may use `/?category=<id>`, `/?filter=created`, and
`/?filter=liked`.

## Quality and acceptance

The mandatory release is ready only when:

- `gofmt` produces no changes;
- `go vet ./...`, `go test ./...`, and `go build ./...` pass;
- all checks in `docs/audit.md` have been exercised;
- a Docker image builds and its container serves the forum;
- SQLite data persists at the documented path/volume;
- there are no committed secrets, database files, or debug artifacts;
- the README matches commands verified on the current code.

## Out of scope for the mandatory release

- OAuth or other third-party login
- image upload
- moderator/admin workflows
- HTTPS termination, rate limiting, or deployment infrastructure
- SPA/frontend frameworks
- public JSON API
