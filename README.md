# Go REST API starter

A small REST API built with Go's standard library and MongoDB. Todo items are saved in your MongoDB database and remain there when the server restarts.

## Run it

Set `MONGODB_URI` in your local `.env` file to the connection string from MongoDB Atlas. Go does not load `.env` files on its own; this project loads it when the server starts. Never commit your real `.env` file or share its connection string.

The app reads `MONGODB_DATABASE` to choose the database; it defaults to `go_rest_api`. Todos are stored in the `todos` collection. See `.env.example` for the variable names.

```powershell
go run ./cmd/api
```

The server checks the MongoDB connection before it starts listening on `http://localhost:8080`. If the connection fails, check that your Atlas cluster is running, your IP address is allowed in Atlas Network Access, and the URI in `.env` is correct. Set the `PORT` environment variable to use another port.

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
| `GET` | `/todos/{id}` | Get one todo by MongoDB ObjectID |
| `PUT` | `/todos/{id}` | Replace a todo's title and completed status |
| `DELETE` | `/todos/{id}` | Delete a todo |

## Test with Thunder Client

Start the API with `go run ./cmd/api` and leave that terminal running. In VS Code, open Thunder Client, choose **New Request**, then set the method and URL for each request below. Click **Send** to see the response.

For requests with a JSON body, open the **Body** tab, choose **JSON**, and enter the example body. Thunder Client will send the `Content-Type: application/json` header.

| Method | URL | Body (JSON) |
| --- | --- | --- |
| `GET` | `http://localhost:8080/healthz` | No body |
| `GET` | `http://localhost:8080/todos` | No body |
| `POST` | `http://localhost:8080/todos` | `{"title":"Learn Go"}` |
| `GET` | `http://localhost:8080/todos/{id}` | No body |
| `PUT` | `http://localhost:8080/todos/{id}` | `{"title":"Learn Go","completed":true}` |
| `DELETE` | `http://localhost:8080/todos/{id}` | No body |

Try them in this order: create a todo, then copy the `id` from Thunder Client's response and replace `{id}` in the get, update, and delete URLs with that value. Get the todo again after updating to see the change. A successful delete returns **204 No Content**.

MongoDB generates a unique ObjectID for each todo. It is a 24-character hexadecimal string, not a sequential number.

## Express concepts in Go

| Express | Go |
| --- | --- |
| `app.get("/todos", handler)` | `mux.HandleFunc("GET /todos", handler)` |
| `req` | `*http.Request` |
| `res` | `http.ResponseWriter` |
| Route parameter `req.params.id` | `r.PathValue("id")` |
| `res.json(value)` | Set the JSON content type, then encode with `json.NewEncoder(w)` |

The HTTP route handlers live in `internal/todos/handler.go`. MongoDB reads and writes live in `internal/todos/store.go`. The server starts and creates the reusable database client in `cmd/api/main.go`. Comments in those files explain Go concepts such as structs, interfaces, contexts, JSON/BSON tags, and dependency injection.

The tests use a small in-memory fake repository so they can check HTTP behavior without needing a live MongoDB cluster. The running API itself always uses MongoDB.

## Test

```powershell
go test ./...
```
