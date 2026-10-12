type RuntimeEnvironment = typeof globalThis & {
  process?: { env?: Record<string, string | undefined> };
};

const METHODS = new Set(["GET", "POST", "PUT", "DELETE", "OPTIONS"]);

export function jsonError(status: number, message: string): Response {
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

export function proxy(apiPath: string, allowedMethods: readonly string[]) {
  return {
    async fetch(request: Request): Promise<Response> {
      const origin = backendOrigin();
      if (!origin) {
        return jsonError(500, "Set BACKEND_URL to the HTTPS origin of the backend Vercel deployment.");
      }
      if (!METHODS.has(request.method) || !allowedMethods.includes(request.method)) {
        return jsonError(405, "Method not allowed");
      }

      const requestURL = new URL(request.url);
      const upstreamURL = new URL(apiPath, origin);
      upstreamURL.search = requestURL.search;

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
}
