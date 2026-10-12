import { jsonError, proxy } from "../_proxy.js";

export default {
  async fetch(request: Request): Promise<Response> {
    const requestURL = new URL(request.url);
    const todoId = requestURL.searchParams.get("todoId");
    if (!todoId || !/^[a-f\d]{24}$/i.test(todoId)) {
      return jsonError(400, "todo ID must be a valid MongoDB ObjectID");
    }

    return proxy(`/todos/${todoId}`, ["GET", "PUT", "DELETE"]).fetch(request);
  },
};
