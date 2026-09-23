import { afterAll, beforeEach, describe, expect, it } from "vitest";

import { connect, type Pool } from "../db.js";
import { NotFoundError, type Record } from "./record.js";
import { PostgresStore } from "./store.js";

/**
 * A direct port of backend/internal/record/store_integration_test.go.
 *
 *   make test-integration-ts
 *
 * The route tests use an in-memory fake, which proves the HTTP layer but says
 * nothing about whether the SQL is right. These run the actual queries, so a
 * COALESCE that silently stops coalescing is caught here rather than in
 * production.
 */

const databaseUrl = process.env["DATABASE_URL"];

// Skipped rather than failed when there is no database, so `npm test` stays
// runnable on a laptop with nothing installed.
const describeIntegration = databaseUrl ? describe : describe.skip;

let pool: Pool;
let store: PostgresStore;

afterAll(async () => {
  if (pool) await pool.end();
});

describeIntegration("PostgresStore", () => {
  beforeEach(async () => {
    pool ??= await connect(databaseUrl!);
    store = new PostgresStore(pool);

    // Each test starts from a known-empty table. RESTART IDENTITY keeps ids
    // predictable so failures are readable.
    await pool.query("TRUNCATE records RESTART IDENTITY");
  });

  async function seed(): Promise<Record> {
    return store.create({ name: "seeded", description: "seeded description", active: true });
  }

  it("writes every column and returns the generated id and timestamps", async () => {
    const got = await store.create({ name: "alpha", description: "d", active: true });

    expect(got.id).toBeGreaterThan(0);
    expect(got.name).toBe("alpha");
    expect(got.description).toBe("d");
    expect(got.active).toBe(true);
    expect(Date.parse(got.createdAt)).not.toBeNaN();
    expect(Date.parse(got.updatedAt)).not.toBeNaN();
  });

  it("returns the id as a number, not the string node-postgres defaults to", async () => {
    const got = await store.create({ name: "alpha", description: "", active: false });

    expect(typeof got.id).toBe("number");
  });

  it("lets the database CHECK constraint reject a blank name", async () => {
    await expect(store.create({ name: "   ", description: "", active: false })).rejects.toThrow();
  });

  it("returns an empty array from an empty table, never null", async () => {
    const got = await store.list();

    expect(Array.isArray(got)).toBe(true);
    expect(got).toHaveLength(0);
  });

  it("orders the collection by ascending id, as the SQL promises", async () => {
    for (const name of ["first", "second", "third"]) {
      await store.create({ name, description: "", active: false });
    }

    const got = await store.list();

    expect(got.map((r) => r.name)).toEqual(["first", "second", "third"]);
  });

  it("surfaces a missing row as NotFoundError rather than undefined", async () => {
    await expect(store.get(999)).rejects.toBeInstanceOf(NotFoundError);
  });

  // The PUT semantic, in SQL.
  it("replace overwrites every column, resetting the ones the caller omitted", async () => {
    const seeded = await seed();

    const got = await store.replace(seeded.id, { name: "replaced", description: "", active: false });

    expect(got.name).toBe("replaced");
    expect(got.description).toBe("");
    expect(got.active).toBe(false);

    // And it is actually on disk, not just in the RETURNING row.
    const reread = await store.get(seeded.id);
    expect(reread.description).toBe("");
    expect(reread.active).toBe(false);
  });

  // The PATCH semantic, in SQL.
  it("update's COALESCE leaves columns the caller omitted alone", async () => {
    const seeded = await seed();

    const got = await store.update(seeded.id, { name: "patched" });

    expect(got.name).toBe("patched");
    expect(got.description).toBe("seeded description");
    expect(got.active).toBe(true);
  });

  it("writes an explicit false, because only an absent key means leave alone", async () => {
    const seeded = await seed(); // active = true

    const got = await store.update(seeded.id, { active: false });

    expect(got.active).toBe(false);
    expect(got.name).toBe("seeded");
  });

  it("advances updated_at on write but never created_at", async () => {
    const seeded = await seed();
    await new Promise((r) => setTimeout(r, 5));

    const got = await store.update(seeded.id, { name: "touched" });

    expect(Date.parse(got.updatedAt)).toBeGreaterThan(Date.parse(seeded.updatedAt));
    expect(got.createdAt).toBe(seeded.createdAt);
  });

  it("maps a missing row to NotFoundError from both replace and update", async () => {
    await expect(
      store.replace(999, { name: "x", description: "", active: false }),
    ).rejects.toBeInstanceOf(NotFoundError);

    await expect(store.update(999, { name: "x" })).rejects.toBeInstanceOf(NotFoundError);
  });

  it("removes the row from Postgres", async () => {
    const seeded = await seed();

    await store.remove(seeded.id);

    await expect(store.get(seeded.id)).rejects.toBeInstanceOf(NotFoundError);
  });

  it("reports NotFoundError when deleting a row that is already gone", async () => {
    await expect(store.remove(999)).rejects.toBeInstanceOf(NotFoundError);
  });

  // An unhandled 'error' event on a pg Pool terminates the Node process, and
  // idle clients raise one whenever Postgres restarts or an administrator
  // terminates a backend. This crashed the API for real before db.ts grew its
  // handler, so it is pinned here.
  it("survives its idle connections being terminated and reconnects", async () => {
    await store.create({ name: "before", description: "", active: false });

    // Boot every other connection to this database, exactly as
    // `initdb.sh --reset` does.
    await pool.query(
      `SELECT pg_terminate_backend(pid) FROM pg_stat_activity
       WHERE datname = current_database() AND pid <> pg_backend_pid()`,
    );

    // A moment for the error event to fire on the idle clients.
    await new Promise((r) => setTimeout(r, 100));

    const rows = await store.list();
    expect(rows.map((r) => r.name)).toContain("before");
  });
});
