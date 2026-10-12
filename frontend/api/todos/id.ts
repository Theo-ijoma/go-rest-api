import { proxy } from "../_proxy";

export default {
  async fetch(request: Request): Promise<Response> {
    const requestURL = new URL(request.url);
    const todoId = requestURL.searchParams.get("todoId");
    if (!todoId || !/^[a-f\d]{24}$/i.test(todoId)) {
      return new Response(JSON.stringify({ error: "todo ID must be a valid MongoDB ObjectID" }), {
        status: 400,
        headers: { "content-type": "application/json", "cache-control": "no-store" },
      });
    }

    return proxy(`/todos/${todoId}`, ["GET", "PUT", "DELETE"]).fetch(request);
  },
};
