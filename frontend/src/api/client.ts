/**
 * Typed client for the Go API.
 *
 * This is a console, so nothing throws: every call resolves to an ApiResult
 * carrying the status code, the parsed body and how long it took. Failures are
 * data to be displayed, not exceptions to be caught.
 */

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export const RESOURCE_PATH = "/api/v1/records";

export interface Record {
  id: number;
  name: string;
  description: string;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

/** Mirrors httpx.ErrorBody on the Go side. */
export interface ApiErrorBody {
  error: string;
  details?: { [field: string]: string };
}

export interface ApiResult<T = unknown> {
  /** HTTP status, or 0 when the request never completed (network / CORS). */
  status: number;
  ok: boolean;
  durationMs: number;
  body: T | ApiErrorBody | null;
  /** Set only for transport-level failures. */
  networkError?: string;
}

export type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

async function send<T>(
  method: HttpMethod,
  path: string,
  body?: unknown,
): Promise<ApiResult<T>> {
  const started = performance.now();

  let response: Response;
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      method,
      headers: body === undefined ? undefined : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    // fetch only rejects on transport failures. Locally that means the Go
    // server is down, or CORS blocked the response before JS could read it —
    // the browser console will say which.
    return {
      status: 0,
      ok: false,
      durationMs: Math.round(performance.now() - started),
      body: null,
      networkError: `Could not reach ${BASE_URL}. Is the API running, and is this origin in CORS_ALLOWED_ORIGINS?`,
    };
  }

  const durationMs = Math.round(performance.now() - started);
  const text = await response.text();

  let parsed: unknown = null;
  if (text) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = text; // show whatever came back rather than swallowing it
    }
  }

  return {
    status: response.status,
    ok: response.ok,
    durationMs,
    body: parsed as T | ApiErrorBody | null,
  };
}

/** Request bodies, mirroring the Go param structs. */
export interface CreateBody {
  name: string;
  description: string;
  active: boolean;
}

/** PUT sends every field — that is what makes it a replacement. */
export type ReplaceBody = CreateBody;

/** PATCH sends only the fields you choose to include. */
export interface PatchBody {
  name?: string;
  description?: string;
  active?: boolean;
}

export const api = {
  list: () => send<Record[]>("GET", RESOURCE_PATH),
  get: (id: string) => send<Record>("GET", `${RESOURCE_PATH}/${id}`),
  create: (body: CreateBody) => send<Record>("POST", RESOURCE_PATH, body),
  replace: (id: string, body: ReplaceBody) => send<Record>("PUT", `${RESOURCE_PATH}/${id}`, body),
  patch: (id: string, body: PatchBody) => send<Record>("PATCH", `${RESOURCE_PATH}/${id}`, body),
  remove: (id: string) => send<never>("DELETE", `${RESOURCE_PATH}/${id}`),
};
