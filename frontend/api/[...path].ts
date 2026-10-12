type RuntimeEnvironment = typeof globalThis & {
  process?: { env?: Record<string, string | undefined> };
};

const API_METHODS = new Set(["GET", "POST", "PUT", "DELETE", "OPTIONS"]);
const API_PATHS = [
  /^\/auth\/(register|login|logout|me)$/,
  /^\/todos$/,
  /^\/todos\/[a-f\d]{24}$/i,
  /^\/healthz$/,
];

function jsonError(status: number, message: string): Response {
  return new Response(JSON.stringify({ error: message }), {
    status,
    headers: { "content-type": "application/json", "cache-control": "no-store" },
  });
}

function backendOrigin(): string | null {
  const value = (globalThis as RuntimeEnvironment).process?.env?.BACKEND_URL;
  if (!value) {
    return null;
  }
  try {
    const origin = new URL(value);
    if (
      origin.protocol !== "https:" ||
      origin.username ||
      origin.password ||
      origin.pathname !== "/" ||
      origin.search ||
      origin.hash
    ) {
      return null;
    }
    return origin.origin;
  } catch {
    return null;
  }
}

export default {
  async fetch(request: Request): Promise<Response> {
    const origin = backendOrigin();
    if (!origin) {
      return jsonError(500, "Set BACKEND_URL to the HTTPS URL of the backend Vercel deployment.");
    }

    const url = new URL(request.url);
    const apiPath = url.pathname.replace(/^\/api(?=\/|$)/, "") || "/";
    if (!API_PATHS.some((pattern) => pattern.test(apiPath))) {
      return jsonError(404, "API route not found");
    }
    if (!API_METHODS.has(request.method)) {
      return jsonError(405, "Method not allowed");
    }

    const upstreamURL = new URL("/api", origin);
    upstreamURL.searchParams.set("path", apiPath);

    const headers = new Headers();
    for (const name of ["accept", "content-type", "cookie", "origin"]) {
      const value = request.headers.get(name);
      if (value) {
        headers.set(name, value);
      }
    }

    try {
      const upstream = await fetch(upstreamURL, {
        method: request.method,
        headers,
        body: request.method === "GET" || request.method === "HEAD" || request.method === "OPTIONS"
          ? undefined
          : await request.arrayBuffer(),
        redirect: "manual",
        cache: "no-store",
      });
      const responseHeaders = new Headers({ "cache-control": "no-store" });
      const contentType = upstream.headers.get("content-type");
      const setCookie = upstream.headers.get("set-cookie");
      if (contentType) {
        responseHeaders.set("content-type", contentType);
      }
      if (setCookie) {
        responseHeaders.set("set-cookie", setCookie);
      }
      const body = upstream.status === 204 || upstream.status === 304 ? null : await upstream.arrayBuffer();
      return new Response(body, { status: upstream.status, headers: responseHeaders });
    } catch (error) {
      console.error("Backend proxy request failed", error);
      return jsonError(502, "The backend could not be reached. Check its Vercel deployment and logs.");
    }
  },
};
