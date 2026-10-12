# Daymark codebase guide

Daymark is a private, per-account task manager. The React frontend serves the landing page, account forms, and task dashboard. A Go HTTP API handles authentication and task operations, with MongoDB as the persistent store.

## Project layout

```text
backend/
  cmd/api/main.go             Go server entry point for local and Vercel deployments
  internal/app/               MongoDB setup and composition of HTTP handlers
  internal/auth/              Account, password, session, cookie, and CORS logic
  internal/todos/             Todo HTTP handlers and MongoDB repository
  go.mod, go.sum              Go module and dependency lock data
  vercel.json                 Selects Vercel's Go server runtime
  README.md                   Setup and deployment instructions
frontend/
  src/App.tsx                 Session restoration and top-level screen state
  src/api.ts                  Browser API client, response validation, and errors
  src/components/              Landing, account, dashboard, and reveal components
  api/                         Vercel server-side proxy functions
  vite.config.ts               Local `/api` proxy to the Go server
  vercel.json                  Static SPA deployment and API route rewrites
```

## Frontend

### Screens and state

`src/App.tsx` owns the main application state:

- On startup, it requests `GET /api/auth/me` to restore the session.
- An authenticated user is shown the dashboard and their tasks.
- An unauthenticated user sees the landing page and can open sign-in or registration.
- `/login` and `/register` are distinct browser routes; `/dashboard` is the authenticated route and `/` is the landing page.
- Navigation uses the History API, so direct links, refreshes, and browser back/forward work without a client-side router dependency.
- Expired sessions return the user to the landing page. Task-loading failures remain visible in the dashboard and can be retried.

UI is split between `src/components/LandingPage.tsx`, `AccountForm.tsx`, and `Dashboard.tsx`. `Reveal.tsx` applies the scroll-into-view animation used by the landing page. The styles are organized alongside the corresponding views in `src/`.

### Browser API client

`src/api.ts` is the only API client used by the React views. It:

- Sends requests to the same-origin `/api` path with credentials enabled.
- Applies a request timeout and turns network/HTTP failures into `APIError`.
- Checks the shape of account and todo responses before the UI consumes them.

The API base is intentionally fixed at `/api`. Do not set a browser-side backend URL: using the frontend proxy keeps the session cookie first-party across the separately deployed Vercel projects.

### Vercel API proxy

`frontend/api/` contains explicit Vercel functions for account and todo routes. The small route files select an upstream path and permitted HTTP methods; `api/_proxy.ts` validates `BACKEND_URL`, forwards the method, body, origin, and session cookie, and returns the backend response. Login/registration `Set-Cookie` headers are relayed to the browser.

Todo IDs are passed through the fixed `api/todos/id.ts` function using the `todoId` query parameter. The rewrite is necessary because the function route uses a normal TypeScript filename. The `.js` import suffixes in these function files are intentional: Vercel runs the emitted functions as Node ESM, which needs an explicit file extension.

For local development, `vite.config.ts` rewrites `/api/...` to `http://localhost:8080/...`. In production, Vercel's functions at `/api/...` provide the same browser-facing API paths.

## Backend

### Server startup and dependency wiring

`backend/cmd/api/main.go` loads a local `.env` file when present, creates the application handler, and starts an HTTP server on `PORT` (default `8080`). Vercel runs this Go server using the `go` framework preset and provides its port.

`internal/app/app.go` validates configuration, connects and pings MongoDB, creates the account and todo stores, ensures indexes, and composes the handlers. The auth handler registers account routes; the todo handler receives the auth middleware and repository through its constructor. Keeping setup here allows local and deployed servers to share the same routes.

Do not add backend `/api` rewrites or a competing `backend/api` function. The Go server registers and handles the public paths directly: `/healthz`, `/auth/...`, and `/todos/...`.

### Authentication and sessions

Registration validates full name, email, and password, normalizes email, hashes the password with bcrypt, and stores the user in MongoDB's `users` collection. A unique email index prevents duplicate accounts. Password hashes are not returned to the browser.

Successful registration and sign-in create a random session token. MongoDB stores only its SHA-256 digest in `sessions`, with a seven-day expiration and TTL index. The raw token is sent in an HttpOnly cookie named `daymark_session`. The cookie is Secure in Vercel by default; `SESSION_COOKIE_SECURE` can explicitly set the behavior.

`GET /auth/me` validates the cookie and returns the account. Protected todo routes use the same session middleware. Signing out deletes the stored session and expires the cookie. `FRONTEND_ORIGIN` configures the allowed browser origin for API CORS.

### Todos and ownership

`internal/todos/handler.go` validates HTTP input, requires an authenticated account for todo operations, and formats JSON responses. `internal/todos/store.go` implements the repository using MongoDB.

Todo documents store the owner's MongoDB ID in `userId`. Every list, lookup, update, and delete filters by both todo ID and owner ID, so one account cannot access another account's tasks. An index on `userId` supports list queries. Legacy todos without an owner remain in the database and are not exposed to accounts.

## HTTP API

All paths below are exposed directly by the backend. Account and todo routes are also exposed through the frontend's same-origin `/api` proxy; the health route is backend-only.

| Method | Path | Authentication | Purpose |
| --- | --- | --- | --- |
| `GET` | `/healthz` | No | Lightweight health response: `{"status":"ok"}` |
| `POST` | `/auth/register` | No | Create an account and session; body: `fullName`, `email`, `password` |
| `POST` | `/auth/login` | No | Start a session; body: `email`, `password` |
| `POST` | `/auth/logout` | Session cookie | Revoke the current session |
| `GET` | `/auth/me` | Session cookie | Return the current account |
| `GET` | `/todos` | Session cookie | List the current account's tasks |
| `POST` | `/todos` | Session cookie | Create a task; body: `title` |
| `GET` | `/todos/{id}` | Session cookie | Read one owned task |
| `PUT` | `/todos/{id}` | Session cookie | Replace task fields; body: `title`, `completed` |
| `DELETE` | `/todos/{id}` | Session cookie | Delete one owned task |

Errors are JSON objects with an `error` field. Common responses include `400` for invalid input, `401` for an absent or invalid session, `404` for a missing or non-owned todo, and `500` for server/database errors.

## Configuration

### Backend

Copy `backend/.env.example` to `backend/.env` for local use. In Vercel, configure values under the backend project (Root Directory `backend`):

| Variable | Purpose |
| --- | --- |
| `MONGODB_URI` | MongoDB connection string; required |
| `MONGODB_DATABASE` | Database name; defaults to `go_rest_api` |
| `FRONTEND_ORIGIN` | Exact browser origin allowed by CORS; defaults to `http://localhost:5173` |
| `SESSION_COOKIE_SECURE` | Optional `true`/`false`; otherwise Secure defaults on in Vercel |
| `PORT` | HTTP port; defaults to `8080` locally |

### Frontend

The Vercel frontend project uses Root Directory `frontend` and needs:

| Variable | Purpose |
| --- | --- |
| `BACKEND_URL` | HTTPS origin of the backend, without a path, query, or credentials |

Set `FRONTEND_ORIGIN` on the backend to the exact production frontend origin. Set the matching origins deliberately for Preview deployments too.

## Local development and checks

Start the backend in one terminal:

```powershell
cd backend
go run ./cmd/api
```

Start the frontend in another:

```powershell
cd frontend
npm.cmd install
npm.cmd run dev
```

The Vite server proxies `/api` requests to the local backend. The backend needs a reachable MongoDB instance.

Run backend checks from `backend/`:

```powershell
go test ./...
go vet ./...
go build ./...
```

Run the frontend production/type-check build from `frontend/`:

```powershell
npm.cmd run build
```
