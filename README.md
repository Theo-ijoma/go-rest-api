# Go REST API starter

A small REST API built with Go's standard library. It keeps Todo items in memory, so they reset whenever the server restarts.

## Run it

```powershell
go run ./cmd/api
```

The server listens on `http://localhost:8080`. Set the `PORT` environment variable to use another port:

```powershell
$env:PORT = "3000"
go run ./cmd/api
```

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/` | Show a welcome message and a few useful endpoints |
| `GET` | `/healthz` | Check that the API is running |
| `GET` | `/todos` | List all todos |
| `POST` | `/todos` | Create a todo |
| `GET` | `/todos/{id}` | Get one todo |
| `PUT` | `/todos/{id}` | Replace a todo's title and completed status |
| `DELETE` | `/todos/{id}` | Delete a todo |

Create a todo:

```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/todos `
  -ContentType "application/json" -Body '{"title":"Learn Go"}'
```

List todos:

```powershell
Invoke-RestMethod http://localhost:8080/todos
```

Update todo `1`:

```powershell
Invoke-RestMethod -Method Put -Uri http://localhost:8080/todos/1 `
  -ContentType "application/json" -Body '{"title":"Learn Go","completed":true}'
```

## Express concepts in Go

| Express | Go |
| --- | --- |
| `app.get("/todos", handler)` | `mux.HandleFunc("GET /todos", handler)` |
| `req` | `*http.Request` |
| `res` | `http.ResponseWriter` |
| Route parameter `req.params.id` | `r.PathValue("id")` |
| `res.json(value)` | Set the JSON content type, then encode with `json.NewEncoder(w)` |

The route handlers live in `internal/todos/handler.go`, the in-memory data store is in `internal/todos/store.go`, and the server starts in `cmd/api/main.go`.

## Test

```powershell
go test ./...
```
