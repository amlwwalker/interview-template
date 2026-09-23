/**
 * Every knob the service reads at startup, from the environment, so the build
 * artefact is identical in dev, CI and prod.
 *
 * Mirrors backend/internal/config/config.go.
 */

export interface Config {
  port: number;
  databaseUrl: string;
  allowedOrigins: string[];
  shutdownTimeoutMs: number;
}

function env(key: string, fallback: string): string {
  const v = process.env[key];
  return v === undefined || v === "" ? fallback : v;
}

/**
 * Throws rather than exiting, so the caller decides how to fail and tests can
 * assert on the message.
 */
export function loadConfig(): Config {
  const databaseUrl = env("DATABASE_URL", "");
  if (!databaseUrl) {
    throw new Error("DATABASE_URL is required (copy .env.example to .env, or run `make db`)");
  }

  const port = Number(env("PORT", "8080"));
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`PORT must be a valid port number, got "${env("PORT", "8080")}"`);
  }

  // The Vite dev server runs on 5173. Both spellings are listed because a
  // browser treats localhost and 127.0.0.1 as different origins.
  const allowedOrigins = env(
    "CORS_ALLOWED_ORIGINS",
    "http://localhost:5173,http://127.0.0.1:5173",
  )
    .split(",")
    .map((o) => o.trim())
    .filter((o) => o.length > 0);

  return { port, databaseUrl, allowedOrigins, shutdownTimeoutMs: 10_000 };
}
