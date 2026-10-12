# Daymark

A personal task list with a Go API, MongoDB storage, and a React landing page. Every account has its own private todo list.

## Start the app

1. Copy `backend/.env.example` to `backend/.env` and set `MONGODB_URI` to your MongoDB Atlas connection string. Never commit or share your real `.env`.
2. Start the API:

   ```powershell
   cd backend
   go run ./cmd/api
   ```

   The API checks its MongoDB connection before listening on `http://localhost:8080`.
3. In a second terminal, start the React app:

   ```powershell
   cd frontend
   npm.cmd install
   npm.cmd run dev
   ```

4. Open the Vite URL shown in the terminal. Create an account with your full name, email, and a password of at least 8 characters.

The app stores accounts in MongoDB's `users` collection, hashes passwords with bcrypt, and stores revocable, expiring sessions in `sessions`. Sessions use HttpOnly cookies. Each todo stores its owner's user ID, so users cannot list or modify one another's tasks. Existing todos without an owner remain in MongoDB and are not shown to new accounts.

The frontend's Vite development proxy forwards `/api` requests to `http://localhost:8080`. Set `FRONTEND_ORIGIN` in `backend/.env` if the frontend is served from a different origin. The API accepts browser mutations only from this configured origin. In production, the frontend's Vercel proxy keeps browser requests same-origin while the API can remain a separate HTTPS deployment. Set `SESSION_COOKIE_SECURE=true` for production.

## Deploy frontend and API as separate Vercel projects

This repository has Vercel configuration for two separate projects. The frontend includes a small same-origin `/api` proxy. It forwards requests to the independently deployed Go API and passes the HttpOnly session cookie through the frontend domain. This avoids relying on third-party cookies between separate `vercel.app` domains.

### 1. Deploy the backend

1. Import this Git repository as a new Vercel project named, for example, `daymark-api`.
2. Set **Root Directory** to `backend`. Leave the Go project build settings at their defaults.
3. Add these Production environment variables in **Settings → Environment Variables**:

   | Name | Value |
   | --- | --- |
   | `MONGODB_URI` | MongoDB Atlas connection string |
   | `MONGODB_DATABASE` | `go_rest_api` or your database name |
   | `FRONTEND_ORIGIN` | The exact frontend production origin, such as `https://daymark-web.vercel.app` |
   | `SESSION_COOKIE_SECURE` | `true` |

4. Deploy the project. Check `https://<backend-project>.vercel.app/healthz`; it should return `{"status":"ok"}`.

### 2. Deploy the frontend

1. Import the same Git repository as a second Vercel project named, for example, `daymark-web`.
2. Set **Root Directory** to `frontend`. Vercel builds the Vite app to `dist` and deploys the `/api` proxy function.
3. Add this Production environment variable:

   | Name | Value |
   | --- | --- |
   | `BACKEND_URL` | Backend origin only, such as `https://daymark-api.vercel.app` |

   Do not set `VITE_API_BASE_URL` to the backend deployment. The browser must call the frontend's `/api` proxy so its session cookie remains first-party.

4. Deploy the frontend, then copy its exact production origin into the backend's `FRONTEND_ORIGIN` variable and redeploy the backend if it changed.
5. Open the frontend deployment and test account registration, sign-in, todo changes, refresh, and sign-out.

Set environment variables for **Production** and **Preview** deliberately. Preview frontend URLs have different origins; add the exact preview origin to the backend's allowed origin before testing authenticated preview deployments. Keep MongoDB credentials only in Vercel environment variables, never in Vite `VITE_*` variables. Configure Atlas network access for Vercel according to your Atlas plan, and avoid broad IP access where your plan offers private networking or stable egress.

The backend uses path-based Vercel Go functions under `backend/api/`; each endpoint directory has an `index.go` handler, and `backend/vercel.json` rewrites public API URLs to those functions, including `/healthz` to `/api/healthz`. Todo item requests rewrite to a fixed function and pass the ID as a query parameter. The handlers share a cached API instance. The `go.mod` file is at the backend project root, as required by the Vercel Go runtime. The frontend uses explicit Vercel functions under `frontend/api/` for each API route. Those functions share `frontend/api/_proxy.ts`, read `BACKEND_URL` at runtime, and forward requests to the backend deployment.

If `/api/auth/login` returns `405 Method Not Allowed` with `index.html` in the response headers, the frontend deployment is serving the SPA fallback instead of its API function. Confirm the frontend project's Root Directory is `frontend`, then redeploy it. If the frontend proxy returns `405` from the backend, redeploy the backend project from the current repository revision and verify `POST /auth/login` on the backend deployment.

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/` | Show an API welcome message |
| `GET` | `/healthz` | Check that the API is running |
| `POST` | `/auth/register` | Create an account and start a session |
| `POST` | `/auth/login` | Sign in and start a session |
| `POST` | `/auth/logout` | Revoke the current session |
| `GET` | `/auth/me` | Get the signed-in account |
| `GET` | `/todos` | List the signed-in user's todos |
| `POST` | `/todos` | Create a todo for the signed-in user |
| `GET` | `/todos/{id}` | Get one of the signed-in user's todos |
| `PUT` | `/todos/{id}` | Update one of the signed-in user's todos |
| `DELETE` | `/todos/{id}` | Delete one of the signed-in user's todos |

## Express concepts in Go

| Express | Go |
| --- | --- |
| `app.get("/todos", handler)` | `mux.HandleFunc("GET /todos", handler)` |
| `req` | `*http.Request` |
| `res` | `http.ResponseWriter` |
| Route parameter `req.params.id` | `r.PathValue("id")` |
| `res.json(value)` | Set the JSON content type, then encode with `json.NewEncoder(w)` |

The tests use in-memory repositories and account services, so they run without a live MongoDB cluster.

## Test

```powershell
cd backend
go test ./...
```
