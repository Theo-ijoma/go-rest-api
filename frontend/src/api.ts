export type User = {
  id: string;
  fullName: string;
  email: string;
  createdAt: string;
};

export type Todo = {
  id: string;
  title: string;
  completed: boolean;
};

export class APIError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
    this.name = "APIError";
  }
}

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || "/api").replace(/\/+$/, "");
const REQUEST_TIMEOUT_MS = 10_000;

function isUser(value: unknown): value is User {
  return (
    typeof value === "object" &&
    value !== null &&
    "id" in value &&
    typeof value.id === "string" &&
    "fullName" in value &&
    typeof value.fullName === "string" &&
    "email" in value &&
    typeof value.email === "string"
  );
}

function isTodo(value: unknown): value is Todo {
  return (
    typeof value === "object" &&
    value !== null &&
    "id" in value &&
    typeof value.id === "string" &&
    "title" in value &&
    typeof value.title === "string" &&
    "completed" in value &&
    typeof value.completed === "boolean"
  );
}

export async function readUser(response: Response): Promise<User> {
  const value: unknown = await response.json();
  if (!isUser(value)) {
    throw new APIError("The backend returned an unexpected response. Restart it from the backend folder.", 0);
  }
  return value;
}

export async function readTodos(response: Response): Promise<Todo[]> {
  const value: unknown = await response.json();
  if (!Array.isArray(value) || !value.every(isTodo)) {
    throw new APIError("The backend returned an unexpected todo list. Restart it from the backend folder.", 0);
  }
  return value;
}

export async function apiRequest(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  if (init.body) {
    headers.set("Content-Type", "application/json");
  }

  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      ...init,
      credentials: "include",
      headers,
      signal: controller.signal,
    });
  } catch (error) {
    if (controller.signal.aborted) {
      throw new APIError("The server took too long to respond. Check that the backend is running, then try again.", 0);
    }
    throw new APIError("Could not connect to the server. Check that the backend is running, then try again.", 0);
  } finally {
    window.clearTimeout(timeout);
  }

  if (!response.ok) {
    if ((response.status === 404 || response.status === 405) && path.startsWith("/auth/")) {
      throw new APIError("The backend is outdated and does not support accounts yet. Stop it, then restart it from the backend folder.", response.status);
    }
    let message = `Request failed (${response.status})`;
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) {
        message = body.error;
      }
    } catch {
      // Keep the status-based message when the server response is not JSON.
    }
    throw new APIError(message, response.status);
  }
  return response;
}
