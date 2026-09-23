import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api, RESOURCE_PATH } from "./client";

/**
 * The client is the only place that knows about HTTP, so it is the only place
 * that needs to be tested against raw responses. fetch is stubbed rather than
 * the module mocked, so these exercise the real parsing path.
 */

const BASE = "http://localhost:8080";

function respondWith(status: number, body: string, ok?: boolean): Response {
  return {
    status,
    ok: ok ?? (status >= 200 && status < 300),
    text: async () => body,
  } as Response;
}

let fetchMock: ReturnType<typeof vi.fn>;

/** mock.calls[i] is possibly-undefined under noUncheckedIndexedAccess. */
function callAt(i: number): [string, RequestInit] {
  const call = fetchMock.mock.calls[i];
  if (!call) throw new Error(`no fetch call at index ${i}`);
  return call as [string, RequestInit];
}

beforeEach(() => {
  fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("request shape", () => {
  it("sends a JSON content-type and a serialised body for POST", async () => {
    fetchMock.mockResolvedValue(respondWith(201, '{"id":1}'));

    await api.create({ name: "alpha", description: "d", active: true });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = callAt(0);
    expect(url).toBe(`${BASE}${RESOURCE_PATH}`);
    expect(init.method).toBe("POST");
    expect(init.headers).toEqual({ "Content-Type": "application/json" });
    expect(JSON.parse(init.body as string)).toEqual({ name: "alpha", description: "d", active: true });
  });

  it("omits the body and content-type entirely for GET and DELETE", async () => {
    fetchMock.mockResolvedValue(respondWith(200, "[]"));
    await api.list();

    let [, init] = callAt(0);
    expect(init.body).toBeUndefined();
    expect(init.headers).toBeUndefined();

    fetchMock.mockResolvedValue(respondWith(204, ""));
    await api.remove("3");

    [, init] = callAt(1);
    expect(init.method).toBe("DELETE");
    expect(init.body).toBeUndefined();
  });

  it("puts the id in the path for single-resource verbs", async () => {
    fetchMock.mockResolvedValue(respondWith(200, "{}"));

    await api.get("7");
    expect(callAt(0)[0]).toBe(`${BASE}${RESOURCE_PATH}/7`);

    await api.patch("7", { active: false });
    expect(callAt(1)[0]).toBe(`${BASE}${RESOURCE_PATH}/7`);
    expect(callAt(1)[1].method).toBe("PATCH");

    await api.replace("7", { name: "n", description: "", active: false });
    expect(callAt(2)[1].method).toBe("PUT");
  });
});

describe("response handling", () => {
  it("parses a JSON body and reports ok", async () => {
    fetchMock.mockResolvedValue(respondWith(200, '{"id":1,"name":"alpha"}'));

    const result = await api.get("1");

    expect(result.status).toBe(200);
    expect(result.ok).toBe(true);
    expect(result.body).toEqual({ id: 1, name: "alpha" });
    expect(result.networkError).toBeUndefined();
  });

  it("returns a null body for 204 rather than failing to parse", async () => {
    fetchMock.mockResolvedValue(respondWith(204, ""));

    const result = await api.remove("1");

    expect(result.status).toBe(204);
    expect(result.ok).toBe(true);
    expect(result.body).toBeNull();
  });

  it("surfaces a 422 with its per-field details intact", async () => {
    fetchMock.mockResolvedValue(
      respondWith(422, '{"error":"validation failed","details":{"name":"name is required"}}'),
    );

    const result = await api.create({ name: "", description: "", active: false });

    expect(result.status).toBe(422);
    expect(result.ok).toBe(false);
    expect(result.body).toEqual({
      error: "validation failed",
      details: { name: "name is required" },
    });
  });

  it("keeps a non-JSON body as text instead of throwing", async () => {
    fetchMock.mockResolvedValue(respondWith(500, "upstream exploded"));

    const result = await api.list();

    expect(result.status).toBe(500);
    expect(result.body).toBe("upstream exploded");
  });

  it("reports a transport failure as status 0 with a CORS hint", async () => {
    fetchMock.mockRejectedValue(new TypeError("Failed to fetch"));

    const result = await api.list();

    expect(result.status).toBe(0);
    expect(result.ok).toBe(false);
    expect(result.body).toBeNull();
    // The two causes are indistinguishable from JS, so the message must name both.
    expect(result.networkError).toMatch(/CORS_ALLOWED_ORIGINS/);
    expect(result.networkError).toContain(BASE);
  });

  it("records a duration for every call", async () => {
    fetchMock.mockResolvedValue(respondWith(200, "[]"));

    const result = await api.list();

    expect(result.durationMs).toBeGreaterThanOrEqual(0);
    expect(Number.isFinite(result.durationMs)).toBe(true);
  });
});
