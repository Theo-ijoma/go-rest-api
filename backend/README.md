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

The frontend's Vite development proxy forwards `/api` requests to `http://localhost:8080`. Set `FRONTEND_ORIGIN` in `backend/.env` if the frontend is served from a different origin. The API accepts browser mutations only from this configured origin. For production, serve the frontend and API on the same site over HTTPS, configure the exact frontend origin, and set `SESSION_COOKIE_SECURE=true`. Set `VITE_API_BASE_URL` in `frontend/.env.local` if the API is hosted elsewhere.

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
