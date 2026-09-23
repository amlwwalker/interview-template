import "dotenv/config";

import { buildApp } from "./app.js";
import { loadConfig } from "./config.js";
import { connect } from "./db.js";
import { PostgresStore } from "./record/store.js";

/**
 * Entrypoint: config, database, server, graceful shutdown.
 *
 * Mirrors main() and run() in backend/cmd/api/main.go.
 */
async function main(): Promise<void> {
  const config = loadConfig();
  // The pool's error handler is wired before anything else uses it; see db.ts
  // for why an unhandled one is fatal.
  const pool = await connect(config.databaseUrl, (err) => {
    console.error("idle postgres client error:", err.message);
  });

  const app = await buildApp({
    config,
    store: new PostgresStore(pool),
    pool,
    logger: true,
  });

  app.log.info("connected to postgres");

  // Drain in-flight requests before closing the pool. Without this you drop
  // live requests on every deploy.
  let shuttingDown = false;
  const shutdown = async (signal: string): Promise<void> => {
    if (shuttingDown) return;
    shuttingDown = true;

    app.log.info({ signal }, "shutdown signal received, draining connections");

    const timer = setTimeout(() => {
      app.log.error("shutdown timed out, exiting anyway");
      process.exit(1);
    }, config.shutdownTimeoutMs);
    // Do not let the timer itself hold the process open.
    timer.unref();

    try {
      await app.close();
      await pool.end();
      clearTimeout(timer);
      app.log.info("shutdown complete");
      process.exit(0);
    } catch (err) {
      app.log.error({ err }, "error during shutdown");
      process.exit(1);
    }
  };

  process.on("SIGINT", () => void shutdown("SIGINT"));
  process.on("SIGTERM", () => void shutdown("SIGTERM"));

  await app.listen({ port: config.port, host: "0.0.0.0" });
  app.log.info({ allowedOrigins: config.allowedOrigins }, "listening");
}

main().catch((err: unknown) => {
  console.error("fatal:", err instanceof Error ? err.message : err);
  process.exit(1);
});
