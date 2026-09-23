import type { FastifyInstance } from "fastify";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { buildApp } from "../app.js";
import type { Config } from "../config.js";
import {
  NotFoundError,
  type CreateParams,
  type Record,
  type ReplaceParams,
  type Store,
  type UpdateParams,
} from "./record.js";

/**
 * A direct port of backend/internal/record/handler_test.go. Same cases, same
 * expected status codes — the two backends are only interchangeable if these
 * agree.
 *
 * app.inject() drives the full HTTP stack without binding a socket, which is
 * the Fastify equivalent of httptest.NewServer.
 */

/** In-memory Store. Because routes depend on the interface, no database. */
class FakeStore implements Store {
  private rows = new Map<number, Record>();
  private nextId = 1;
  /** When set, every method rejects with it. */
  failWith: Error | null = null;

  private check(): void {
    if (this.failWith) throw this.failWith;
  }

  async list(): Promise<Record[]> {
    this.check();
    return [...this.rows.values()];
  }

  async get(id: number): Promise<Record> {
    this.check();
    const row = this.rows.get(id);
    if (!row) throw new NotFoundError();
    return row;
  }

  async create(params: CreateParams): Promise<Record> {
    this.check();
    const now = new Date().toISOString();
    const row: Record = { id: this.nextId++, ...params, createdAt: now, updatedAt: now };
    this.rows.set(row.id, row);
    return row;
  }

  /** Mirrors the SQL: every mutable field is overwritten unconditionally. */
  async replace(id: number, params: ReplaceParams): Promise<Record> {
    this.check();
    const existing = this.rows.get(id);
    if (!existing) throw new NotFoundError();

    const replaced: Record = {
      id: existing.id,
      name: params.name,
      description: params.description,
      active: params.active,
      createdAt: existing.createdAt,
      updatedAt: new Date().toISOString(),
    };
    this.rows.set(id, replaced);
    return replaced;
  }

  /** Mirrors the COALESCE: only supplied fields change. */
  async update(id: number, params: UpdateParams): Promise<Record> {
    this.check();
    const existing = this.rows.get(id);
    if (!existing) throw new NotFoundError();

    const updated: Record = {
      ...existing,
      ...(params.name !== undefined ? { name: params.name } : {}),
      ...(params.description !== undefined ? { description: params.description } : {}),
      ...(params.active !== undefined ? { active: params.active } : {}),
      updatedAt: new Date().toISOString(),
    };
    this.rows.set(id, updated);
    return updated;
  }

  async remove(id: number): Promise<void> {
    this.check();
    if (!this.rows.delete(id)) throw new NotFoundError();
  }
}

const config: Config = {
  port: 0,
  databaseUrl: "postgres://unused",
  allowedOrigins: ["http://localhost:5173"],
  shutdownTimeoutMs: 1000,
};

const BASE = "/api/v1/records";

let app: FastifyInstance;
let store: FakeStore;

beforeEach(async () => {
  store = new FakeStore();
  app = await buildApp({ config, store });
});

afterEach(async () => {
  await app.close();
});

/** Inserts a fully-populated row so replace/update tests have non-zero fields. */
async function seed(): Promise<Record> {
  return store.create({ name: "seeded", description: "seeded description", active: true });
}

describe("CREATE", () => {
  it("stores every field, trims the name and returns 201 with a Location header", async () => {
    const res = await app.inject({
      method: "POST",
      url: BASE,
      payload: { name: "  alpha  ", description: "first", active: true },
    });

    expect(res.statusCode).toBe(201);

    const body = res.json();
    expect(body.name).toBe("alpha");
    expect(body.active).toBe(true);
    expect(body.id).toBeGreaterThan(0);
    expect(res.headers.location).toBe(`${BASE}/1`);
  });

  it.each([
    { label: "an empty name", payload: { name: "" }, expected: 422 },
    { label: "a whitespace-only name", payload: { name: "   " }, expected: 422 },
    { label: "a missing name", payload: { description: "orphan" }, expected: 422 },
    { label: "an unknown field", payload: { name: "x", colour: "red" }, expected: 400 },
    { label: "a wrong-typed field", payload: { name: 42 }, expected: 400 },
  ])("rejects $label with $expected", async ({ payload, expected }) => {
    const res = await app.inject({ method: "POST", url: BASE, payload });
    expect(res.statusCode).toBe(expected);
  });

  it("rejects malformed JSON with 400", async () => {
    const res = await app.inject({
      method: "POST",
      url: BASE,
      headers: { "content-type": "application/json" },
      payload: '{"name":',
    });
    expect(res.statusCode).toBe(400);
  });
});

describe("READ", () => {
  it("returns an empty array, never null, so the client needs no null check", async () => {
    const res = await app.inject({ method: "GET", url: BASE });

    expect(res.statusCode).toBe(200);
    expect(res.body.trim()).toBe("[]");
  });

  it("returns 404 for a row that does not exist", async () => {
    const res = await app.inject({ method: "GET", url: `${BASE}/999` });
    expect(res.statusCode).toBe(404);
  });

  it.each(["abc", "0", "-1"])("returns 400 for the invalid id %s", async (id) => {
    const res = await app.inject({ method: "GET", url: `${BASE}/${id}` });
    expect(res.statusCode).toBe(400);
    expect(res.json().error).toBe("id must be a positive integer");
  });
});

// This pair is the point of having both verbs. Same URL, same partial body,
// different outcome.
describe("UPDATE: PUT vs PATCH", () => {
  it("PUT resets every field the body omits", async () => {
    await seed();

    const res = await app.inject({ method: "PUT", url: `${BASE}/1`, payload: { name: "replaced" } });

    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.name).toBe("replaced");
    expect(body.description).toBe("");
    expect(body.active).toBe(false);
  });

  it("PATCH leaves every field the body omits untouched", async () => {
    await seed();

    // Byte-for-byte the same payload as the PUT case above.
    const res = await app.inject({ method: "PATCH", url: `${BASE}/1`, payload: { name: "patched" } });

    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.name).toBe("patched");
    expect(body.description).toBe("seeded description");
    expect(body.active).toBe(true);
  });

  it("PUT is idempotent — the same body twice leaves the same row", async () => {
    await seed();
    const payload = { name: "fixed", description: "fixed description", active: true };

    const first = (await app.inject({ method: "PUT", url: `${BASE}/1`, payload })).json();
    const second = (await app.inject({ method: "PUT", url: `${BASE}/1`, payload })).json();

    expect({ ...first, updatedAt: null }).toEqual({ ...second, updatedAt: null });
  });

  it("PUT requires a name, because a partial replacement is a contradiction", async () => {
    await seed();

    const res = await app.inject({
      method: "PUT",
      url: `${BASE}/1`,
      payload: { description: "no name here" },
    });

    expect(res.statusCode).toBe(422);
    expect(res.json().details.name).toBe("name is required");
  });

  // Distinguishing "absent" from "false" is the entire reason PATCH exists
  // alongside PUT, and the reason Go uses pointers here.
  it("PATCH can set active to false as a deliberate value", async () => {
    await seed();

    const res = await app.inject({ method: "PATCH", url: `${BASE}/1`, payload: { active: false } });

    expect(res.statusCode).toBe(200);
    expect(res.json().active).toBe(false);
    expect(res.json().name).toBe("seeded");
  });

  it("PATCH rejects an empty body — there is no change to apply", async () => {
    await seed();

    const res = await app.inject({ method: "PATCH", url: `${BASE}/1`, payload: {} });

    expect(res.statusCode).toBe(422);
  });

  it.each(["PUT", "PATCH"] as const)("%s returns 404 for a missing row", async (method) => {
    const res = await app.inject({ method, url: `${BASE}/999`, payload: { name: "x" } });
    expect(res.statusCode).toBe(404);
  });
});

describe("DELETE", () => {
  it("returns 204 and the row is genuinely gone afterwards", async () => {
    await seed();

    const del = await app.inject({ method: "DELETE", url: `${BASE}/1` });
    expect(del.statusCode).toBe(204);
    expect(del.body).toBe("");

    const get = await app.inject({ method: "GET", url: `${BASE}/1` });
    expect(get.statusCode).toBe(404);
  });

  it("reports 404 rather than pretending to succeed on an already-deleted row", async () => {
    const res = await app.inject({ method: "DELETE", url: `${BASE}/999` });
    expect(res.statusCode).toBe(404);
  });
});

describe("errors", () => {
  it("surfaces a store failure as 500 without leaking the underlying error", async () => {
    store.failWith = new Error("connection refused on 127.0.0.1");

    const res = await app.inject({ method: "GET", url: BASE });

    expect(res.statusCode).toBe(500);
    expect(res.body).not.toContain("connection refused");
    expect(res.json().error).toBe("internal server error");
  });

  it("returns 404 with our error shape for an unknown endpoint", async () => {
    const res = await app.inject({ method: "GET", url: "/api/v1/nope" });

    expect(res.statusCode).toBe(404);
    expect(res.json().error).toBe("endpoint not found");
  });
});

describe("CORS", () => {
  it("answers the preflight for every verb the API serves, including PUT", async () => {
    for (const method of ["POST", "PUT", "PATCH", "DELETE"]) {
      const res = await app.inject({
        method: "OPTIONS",
        url: `${BASE}/1`,
        headers: {
          origin: "http://localhost:5173",
          "access-control-request-method": method,
        },
      });

      expect(res.statusCode).toBeLessThan(300);
      expect(res.headers["access-control-allow-methods"]).toContain(method);
      expect(res.headers["access-control-allow-origin"]).toBe("http://localhost:5173");
    }
  });

  it("sends no allow-origin header to an origin that is not allowlisted", async () => {
    const res = await app.inject({
      method: "GET",
      url: BASE,
      headers: { origin: "http://evil.example" },
    });

    // The browser is what blocks it — the server simply declines to consent.
    expect(res.headers["access-control-allow-origin"]).toBeUndefined();
  });

  it("exposes Location so cross-origin JS can actually read it", async () => {
    const res = await app.inject({
      method: "POST",
      url: BASE,
      headers: { origin: "http://localhost:5173" },
      payload: { name: "alpha" },
    });

    expect(res.headers["access-control-expose-headers"]).toContain("Location");
  });
});

describe("health", () => {
  it("reports liveness without touching the database", async () => {
    const res = await app.inject({ method: "GET", url: "/healthz" });

    expect(res.statusCode).toBe(200);
    expect(res.json()).toEqual({ status: "ok" });
  });
});
