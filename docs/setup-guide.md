# Forum setup and architecture guide

This guide explains how to run the forum, enable real GitHub and Google login,
and understand the project flow without reading the Go implementation.

## 1. Project overview

The forum supports:

- email/password registration and login;
- GitHub OAuth login;
- Google OAuth login;
- posts, comments, categories, reactions, and user filters;
- server-side sessions with one active session per user;
- SQLite persistence and server-rendered HTML/CSS.

All login methods create the same local forum session. OAuth is used only to
identify the user: provider access tokens are discarded after identity lookup
and are never stored in SQLite.

## 2. Requirements

For a local run:

- Go 1.25 or a version compatible with `go.mod`;
- a browser;
- GitHub or Google accounts only when testing real OAuth.

For containers, install Docker with Docker Compose support. No separate SQLite
server, frontend build, or JavaScript runtime is required.

## 3. Run without OAuth

From the project root:

```bash
go run ./cmd/forum
```

Open `http://localhost:8080`. Password registration and login work while OAuth
providers are disabled.

The application creates `data/forum.db`, runs migrations, and seeds categories
automatically on first startup.

## 4. Environment configuration

Create a private environment file:

```bash
cp .env.example .env
chmod 600 .env
```

Supported variables:

| Variable | Default / purpose |
| --- | --- |
| `FORUM_ADDRESS` | `:8080` HTTP listen address |
| `FORUM_DATABASE_PATH` | `data/forum.db` SQLite file |
| `FORUM_SESSION_DURATION` | `24h` |
| `FORUM_COOKIE_NAME` | `forum_session` |
| `FORUM_SECURE_COOKIE` | `false`; use `true` with production HTTPS |
| `GITHUB_CLIENT_ID` | GitHub OAuth App client ID |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth App client secret |
| `GITHUB_REDIRECT_URL` | Exact GitHub callback URL |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret |
| `GOOGLE_REDIRECT_URL` | Exact Google callback URL |

Each provider uses an all-or-nothing configuration:

- all three provider values set: provider enabled;
- all three values empty: provider disabled;
- partial values: startup fails with a configuration error.

You can enable GitHub, Google, both, or neither. Only enabled providers appear
on the login and registration pages.

Load `.env` and run the forum:

```bash
set -a
source .env
set +a
go run ./cmd/forum
```

The application does not load `.env` automatically. `set -a` exports the values
loaded by `source`; `set +a` restores normal shell behavior.

## 5. GitHub OAuth setup

Follow GitHub's official
[OAuth App guide](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app):

1. Open GitHub **Settings → Developer settings → OAuth Apps**.
2. Create a **New OAuth App**.
3. Set the homepage URL to `http://localhost:8080`.
4. Set the authorization callback URL exactly to:

   ```text
   http://localhost:8080/auth/github/callback
   ```

5. Generate a client secret and add the values to `.env`:

   ```dotenv
   GITHUB_CLIENT_ID=your_github_client_id
   GITHUB_CLIENT_SECRET=your_github_client_secret
   GITHUB_REDIRECT_URL=http://localhost:8080/auth/github/callback
   ```

The forum requests `user:email` and requires the GitHub account to have a
primary verified email. Do not enable callback wildcards.

## 6. Google OAuth setup

Follow Google's official
[OAuth web-server guide](https://developers.google.com/identity/protocols/oauth2/web-server):

1. Open the [Google Cloud Console](https://console.cloud.google.com/).
2. Create or select a Cloud project.
3. Configure **Google Auth Platform** branding and audience.
4. If the app is in Testing mode, add the Google accounts that may log in as
   test users.
5. Under **Clients**, create an OAuth client of type **Web application**.
6. Add this exact authorized redirect URI:

   ```text
   http://localhost:8080/auth/google/callback
   ```

7. Add the generated values to `.env`:

   ```dotenv
   GOOGLE_CLIENT_ID=your_google_client_id
   GOOGLE_CLIENT_SECRET=your_google_client_secret
   GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback
   ```

The forum requests `openid email profile`, identifies the account by Google's
stable `sub`, and requires `email_verified=true`.

Callback URLs must match exactly. `localhost` and `127.0.0.1` are different,
as are different ports, schemes, paths, and trailing slashes.

## 7. Verify real login

After loading `.env` and starting the application:

1. Open `http://localhost:8080/login`.
2. Choose GitHub or Google and approve the provider consent screen.
3. Confirm that the callback returns you to the forum as logged in.
4. Open `/posts/new` and create a post, comment, and reaction.
5. Log out and confirm protected routes are unauthorized while the content is
   still publicly visible.
6. Log in again with the same provider and confirm the same forum identity is
   reused.

On first OAuth login, the local user and provider identity are created in one
transaction. Returning users are found by `(provider, provider_user_id)`, not by
email or display name. If the provider email already belongs to a local account,
the forum returns `409 Conflict` and does not automatically link accounts.

## 8. Docker and Compose

Run without OAuth:

```bash
docker compose up --build
```

Run with OAuth values from `.env`:

```bash
docker compose --env-file .env up --build
```

Stop without deleting data:

```bash
docker compose down
```

Compose passes provider variables at runtime and stores SQLite data in the
`forum-data` volume. Avoid `docker compose down -v` unless you intentionally
want to delete the database.

For a direct container run:

```bash
docker build -t forum .
docker run --rm \
  --name forum \
  --env-file .env \
  -p 8080:8080 \
  -v forum-data:/app/data \
  forum
```

The callback remains `localhost:8080` because that is the address used by the
browser, even though the application runs inside a container.

## 9. Architecture flow

### Normal forum request

```text
Browser
  -> recovery / logging / authentication middleware
  -> router and HTTP handler
  -> validation
  -> service business rules
  -> repository
  -> SQLite
  -> HTML response or redirect
```

The layers have clear roles:

- configuration validates runtime settings before startup;
- middleware handles recovery, logging, and current-user lookup;
- handlers own HTTP input, status codes, redirects, and rendering;
- services enforce authentication and forum rules;
- repositories own SQL and transactions;
- provider adapters translate GitHub and Google into one OAuth identity shape.

### OAuth request

```text
Browser
  -> GET /auth/{provider}
  -> state and PKCE challenge created
  -> provider login/consent
  -> GET /auth/{provider}/callback
  -> state, browser cookie, provider, and expiry validated
  -> code exchanged and verified identity fetched
  -> local user/OAuth account found or created
  -> existing forum session created
  -> 303 redirect to /
```

OAuth state is random, stored in memory as a hash, bound to the initiating
browser by an `HttpOnly`, `SameSite=Lax` cookie, and expires after ten minutes.
It is consumed once to prevent replay. Restarting the application invalidates
OAuth attempts that are still in progress, but does not affect stored users or
forum content.

### Identity and session model

```text
GitHub identity -> numeric GitHub user ID
Google identity -> Google sub
Both identities -> local users row -> forum session -> forum permissions
```

OAuth-only users have no password hash. Password login returns the same failure
for unknown users, wrong passwords, and OAuth-only accounts, avoiding account
type disclosure.

Every login method uses the same UUID session cookie and SQLite `sessions`
table. A new login replaces the previous active session for that user. Logout
removes the server-side session and clears the browser cookie.

### Main persistence areas

| Area | Tables |
| --- | --- |
| Authentication | `users`, `oauth_accounts`, `sessions` |
| Forum content | `posts`, `comments`, `categories`, `post_categories` |
| Reactions | `post_reactions`, `comment_reactions` |

Migrations run automatically at startup. Database constraints enforce unique
emails, usernames, provider identities, sessions, and reactions.

## 10. Production and secrets

For production:

- use HTTPS and set `FORUM_SECURE_COOKIE=true`;
- register exact HTTPS callback URLs for the production domain;
- use separate development and production provider credentials;
- inject secrets through the deployment platform's secret manager;
- place SQLite on durable storage and maintain backups;
- use one application instance for this SQLite-based design.

Never commit `.env` or put real secrets in `.env.example`, Compose, Dockerfiles,
source code, documentation, or logs. Provider codes, state, PKCE values, tokens,
secrets, and cookies must not be logged. If a secret is exposed, rotate it at the
provider immediately; deleting it from a later commit is not enough.

## 11. Tests

Automated tests use fake provider servers and temporary real SQLite databases,
so real OAuth credentials are not required.

```bash
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

Real GitHub and Google login remains a manual check because tests should not
depend on external accounts or committed secrets.

## 12. Common problems

### Incomplete OAuth configuration

Set all three variables for the provider or clear all three to disable it. A
callback URL left set while ID and secret are empty is still partial setup.

### Provider button is missing

Confirm the provider triplet was exported in the same shell running the forum,
or passed through Compose. Restart the application after changing `.env`.

### GitHub cannot find a verified email

Verify that the GitHub account has a primary verified email. A public profile
email alone is not sufficient unless it is also primary and verified.

### Google reports `redirect_uri_mismatch`

Compare the Google authorized URI and `GOOGLE_REDIRECT_URL` character for
character, including scheme, hostname, port, path, and trailing slash.

### Google blocks the user

For an external app in Testing mode, add the email under test users and confirm
the client type is **Web application**.

### OAuth returns a conflict

The verified provider email already belongs to another local account. Automatic
account linking is intentionally disabled.

### Login works at the provider but fails in the forum

Check callback configuration, credentials, outbound network access, system
time, and whether the application restarted during the flow. Logs expose only a
safe error category and should never be changed to print secrets.

### Local login loops with secure cookies

Use `FORUM_SECURE_COOKIE=false` with plain `http://localhost`. Secure cookies
are returned only over HTTPS.
