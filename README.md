# Forum

A full-stack web forum written in **Go** using the standard `net/http` package, **SQLite**, server-side HTML templates, and plain CSS.

The project uses a layered architecture and test-driven development, with emphasis on authentication, sessions, persistence, HTTP correctness, and clean separation between handlers, services, repositories, and the database.

## Features

- User registration
- Secure password hashing with bcrypt
- Login and logout
- UUID-based sessions
- One active session per user
- Public post listing and post detail pages
- Authenticated post creation
- Categories and category filtering
- Comments
- Like / dislike reactions
- Reaction toggle and switch behavior
- Filter posts created by the current user
- Filter posts liked by the current user
- Centralized HTTP routing and method handling
- Consistent HTTP error statuses
- Panic recovery middleware
- Request logging
- SQLite persistence
- Responsive HTML/CSS interface
- Docker and Docker Compose support
- Persistent Docker volume for SQLite data
- Helper scripts for build, run, and stop

No JavaScript is required by the application.

---

## Tech Stack

- **Go**
- **net/http**
- **html/template**
- **SQLite**
- **bcrypt**
- **UUID sessions**
- **HTML**
- **CSS**
- **Docker**
- **Docker Compose**

---

## Architecture

```text
Browser Request
      |
      v
Middleware
Recovery / Logging / Authentication
      |
      v
HTTP Handler
      |
      v
Validation
      |
      v
Service
Business Rules
      |
      v
Repository
      |
      v
SQLite
      |
      v
Template / Redirect / HTTP Response
```

Project structure:

```text
forum/
├── cmd/
│   └── forum/
│       └── main.go
├── internal/
│   ├── app/
│   ├── config/
│   ├── database/
│   ├── model/
│   ├── repository/
│   ├── service/
│   ├── session/
│   ├── validation/
│   └── web/
│       ├── handler/
│       ├── middleware/
│       └── view/
├── migrations/
├── templates/
├── static/
├── scripts/
├── data/
├── Dockerfile
├── compose.yml
├── go.mod
└── README.md
```

---

## Database

The application uses SQLite.

Main tables:

```text
users
sessions
posts
comments
categories
post_categories
post_reactions
comment_reactions
```

Important rules:

- normalized email and username are unique
- password values are stored only as bcrypt hashes
- UUID session identifiers
- one active session per user
- foreign keys enabled
- reaction values limited to `-1` and `1`
- one reaction per user per target
- unique post/category relations
- transactional multi-row writes

Migrations live in:

```text
migrations/
```

and are applied automatically when the application starts.

The default local database path is:

```text
data/forum.db
```

Runtime database files are ignored by Git.

---

## Filters

Public category filter:

```text
/?category=<id>
```

Authenticated filters:

```text
/?filter=created
/?filter=liked
```

---

## Routes

### Public

```text
GET  /
GET  /register
POST /register
GET  /login
POST /login
GET  /posts/{id}
GET  /static/*
```

### Authenticated

```text
POST /logout
GET  /posts/new
POST /posts
POST /posts/{id}/comments
POST /posts/{id}/react
POST /comments/{id}/react
```

---

## HTTP Behaviour

```text
200  successful page response
303  successful form submission redirect
400  malformed or invalid request
401  authentication required / invalid credentials
403  authenticated user lacks permission
404  resource or route not found
405  method not allowed
409  duplicate registration conflict
500  unexpected internal error
```

Successful state-changing form submissions use `303 See Other`.

GET requests do not mutate SQLite.

---

## Sessions

Authentication uses UUID-based session cookies.

Default configuration:

```text
Cookie name: forum_session
Duration:    24h
```

Creating a new session replaces the previous active session for the same user.

Session state is stored in SQLite.

---

## Configuration

Environment variables:

```text
FORUM_ADDRESS
FORUM_DATABASE_PATH
FORUM_SESSION_DURATION
FORUM_COOKIE_NAME
FORUM_SECURE_COOKIE
```

Defaults:

```text
FORUM_ADDRESS=:8080
FORUM_DATABASE_PATH=data/forum.db
FORUM_SESSION_DURATION=24h
FORUM_COOKIE_NAME=forum_session
FORUM_SECURE_COOKIE=false
```

---

## Run Locally

From the project root:

```bash
go run ./cmd/forum
```

Open:

```text
http://localhost:8080
```

The application creates the local SQLite data directory when needed.

---

## Tests

Run the full suite:

```bash
go test ./...
```

Format:

```bash
gofmt -w .
```

Static analysis:

```bash
go vet ./...
```

Build:

```bash
go build ./...
```

The test suite covers configuration, server lifecycle, migrations, repositories, validation, authentication, sessions, posts, comments, reactions, filters, routing, middleware, template rendering, HTTP integration flows, and real SQLite persistence.

HTTP integration tests use `httptest.Server` with a real temporary SQLite database.

---

## Docker

Build the image:

```bash
docker build -t forum .
```

Run directly:

```bash
docker run --rm \
  --name forum \
  -p 8080:8080 \
  -v forum-data:/app/data \
  forum
```

The named volume:

```text
forum-data
```

stores the SQLite database outside the container so users, posts, comments, and reactions survive container recreation.

---

## Docker Compose

Start:

```bash
docker compose up --build
```

Stop:

```bash
docker compose down
```

The Compose configuration uses the same persistent `forum-data` volume.

Avoid:

```bash
docker compose down -v
```

unless you intentionally want to delete the persistent database volume.

---

## Helper Scripts

```text
scripts/build.sh
scripts/run.sh
scripts/stop.sh
```

Build:

```bash
./scripts/build.sh
```

Run:

```bash
./scripts/run.sh
```

Stop:

```bash
./scripts/stop.sh
```

The scripts use `set -eu` so failures are not silently ignored.

---

## Logging

Requests are logged with useful HTTP information such as:

```text
method
path
status
```

The logging middleware avoids exposing request bodies, passwords, cookies, or other sensitive values.

Unexpected panics are recovered and converted into safe `500 Internal Server Error` responses so the server can continue handling later requests.

---

## Security

Implemented security-related decisions include:

- bcrypt password hashing
- parameterized SQL queries
- server-side sessions
- UUID session IDs
- unique active session per user
- foreign key enforcement
- protected authenticated routes
- centralized method enforcement
- safe internal error responses
- automatic HTML escaping through `html/template`
- no JavaScript dependency
- runtime databases and environment files excluded from Git

For HTTPS deployments:

```text
FORUM_SECURE_COOKIE=true
```

---

## UI

The frontend uses server-rendered HTML and CSS only.

It includes:

- responsive navigation
- post cards
- category badges
- login and registration forms
- post creation form
- comment cards
- like / dislike controls
- developer-themed background artwork
- responsive mobile layout

Static assets are served from:

```text
/static/
```

---

## Useful SQLite Commands

```bash
sqlite3 data/forum.db
```

Inside SQLite:

```sql
.tables
SELECT * FROM users;
SELECT * FROM posts;
SELECT * FROM comments;
```

Exit:

```sql
.quit
```

---

## Development Workflow

```text
write test
→ observe expected failure
→ implement the smallest change
→ run focused tests
→ run full test suite
→ format
→ commit
```

---

## Final Verification

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./...
docker build -t forum .
```

Also verify manually that:

- registration and login work
- session replacement behaves correctly
- guests cannot access protected actions
- posts and comments persist
- reactions toggle and switch correctly
- category, created, and liked filters work
- unknown routes return `404`
- invalid methods return `405`
- SQLite data survives container recreation
- no JavaScript exists in the repository
- no database, secret, log, or build artifact is committed

---

## Project Goal

The goal is not only to build a working forum, but to practice the structure and behaviour of a real web application:

- HTTP request lifecycle
- authentication
- session management
- database design
- SQL migrations
- transactional persistence
- middleware
- routing
- server-side rendering
- testing
- containerization
- application configuration

It provides a solid foundation for further work in backend engineering, DevOps, and application security.


set -a
source .env
set +a
go run ./cmd/forum