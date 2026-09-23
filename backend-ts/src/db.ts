import pg from "pg";

/**
 * node-postgres returns int8 (bigint) as a *string* by default, because a
 * 64-bit integer does not always fit in a JS number. Our ids are BIGINT, so
 * without this an id would serialise as "1" rather than 1 and the API would
 * silently disagree with the Go implementation.
 *
 * Safe up to 2^53. A table expecting more rows than that should use a string
 * id in the API type rather than lying about the parse.
 */
pg.types.setTypeParser(pg.types.builtins.INT8, (value: string) => {
  const n = Number(value);
  if (!Number.isSafeInteger(n)) {
    throw new Error(`bigint ${value} exceeds Number.MAX_SAFE_INTEGER`);
  }
  return n;
});

export type Pool = pg.Pool;

/**
 * Opens a pool and verifies it with a query, so the process fails fast on a
 * bad DATABASE_URL instead of on the first request.
 *
 * Mirrors backend/internal/database/database.go.
 */
export async function connect(
  databaseUrl: string,
  onError?: (err: Error) => void,
): Promise<Pool> {
  const pool = new pg.Pool({
    connectionString: databaseUrl,
    max: 10,
    idleTimeoutMillis: 30_000,
    connectionTimeoutMillis: 5_000,
  });

  // A Pool is an EventEmitter, and an unhandled 'error' event terminates the
  // Node process. Postgres restarting, a failover, or an administrator running
  // pg_terminate_backend all raise it on an *idle* client — no request is in
  // flight, nothing is broken, and yet the API would die.
  //
  // The pool discards the dead client and opens a new one on the next query,
  // so the correct response is to record it and carry on. Go's pgxpool does
  // this internally, which is why the Go backend needs no equivalent.
  pool.on("error", (err) => {
    if (onError) onError(err);
    else console.error("idle postgres client error:", err.message);
  });

  try {
    const client = await pool.connect();
    await client.query("SELECT 1");
    client.release();
  } catch (err) {
    await pool.end();
    throw new Error(`cannot reach the database: ${(err as Error).message}`);
  }

  return pool;
}
