import cors from "@fastify/cors";
import Fastify, { type FastifyError, type FastifyInstance } from "fastify";

import type { Config } from "./config.js";
import type { Pool } from "./db.js";
import { errorBody } from "./httpx.js";
import { recordRoutes } from "./record/routes.js";
import type { Store } from "./record/record.js";

/**
 * Builds the server without starting it, so tests can drive it through
 * `app.inject()` — the equivalent of Go's httptest.NewServer, but without
 * binding a socket.
 *
 * Mirrors newRouter() in backend/cmd/api/main.go.
 */

export interface AppDeps {
  config: Config;
  store: Store;
  /** Only used by /readyz. Absent in tests, which have no database. */
  pool?: Pool;
  logger?: boolean;
}

export async function buildApp({ config, store, pool, logger = false }: AppDeps): Promise<FastifyInstance> {
  const app = Fastify({
    logger,
    ajv: {
      customOptions: {
        // Fastify strips unknown properties by default; schemas here set
        // additionalProperties: false and this makes that an error instead of
        // a silent deletion — the equivalent of DisallowUnknownFields.
        removeAdditional: false,
        // And ajv coerces by default, which would quietly turn {"name": 42}
        // into {"name": "42"} and accept it. Go's decoder rejects that, so
        // coercion is off or the two backends disagree on the wire.
        coerceTypes: false,
        allErrors: true,
      },
    },
  });

  // CORS is registered before the routes it protects. @fastify/cors answers the
  // browser's preflight OPTIONS itself, which is the piece people usually miss
  // when hand-rolling this.
  await app.register(cors, {
    origin: config.allowedOrigins,
    // Every verb the API serves must be listed, or the browser rejects the
    // preflight and the request never reaches a handler. Forgetting PUT or
    // PATCH is the classic version of this bug: GET and POST keep working, so
    // the router looks fine.
    methods: ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
    allowedHeaders: ["Accept", "Authorization", "Content-Type", "X-Request-Id"],
    // Cross-origin JS can only read headers named here. Without Location the
    // browser receives the 201 header but cannot see it.
    exposedHeaders: ["X-Request-Id", "Location"],
    // Leave false unless you move to cookie auth; `true` is incompatible with
    // an origin of "*".
    credentials: false,
    maxAge: 300,
  });

  // Liveness: is the process up? Deliberately does not touch the database, so
  // a database blip cannot cause a restart loop.
  app.get("/healthz", async (_request, reply) => reply.code(200).send({ status: "ok" }));

  // Readiness: can we actually serve traffic? This one does check the database.
  app.get("/readyz", async (_request, reply) => {
    if (!pool) return reply.code(200).send({ status: "ready" });

    try {
      await pool.query("SELECT 1");
      return reply.code(200).send({ status: "ready" });
    } catch (err) {
      app.log.warn({ err }, "readiness check failed");
      return reply.code(503).send(errorBody("database unavailable"));
    }
  });

  // Registered BEFORE the routes: a child context inherits whichever error
  // handler exists at the moment it is registered, so setting one afterwards
  // leaves the routes on Fastify's default (which echoes the raw message).
  app.setNotFoundHandler(async (_request, reply) =>
    reply.code(404).send(errorBody("endpoint not found")),
  );

  // The handler's `error` is typed `unknown` under strict settings, so narrow
  // it once here rather than casting at each branch.
  app.setErrorHandler(async (raw, request, reply) => {
    const error = raw as FastifyError;

    // Schema failures and malformed JSON are the client's fault: 400, and say
    // what was wrong. This is the equivalent of Go's decodeJSON errors.
    if (error.validation || error.statusCode === 400) {
      return reply.code(400).send(errorBody(`request body must be valid JSON: ${error.message}`));
    }

    if (error.statusCode === 405) {
      return reply.code(405).send(errorBody("method not allowed"));
    }

    if (error.statusCode && error.statusCode < 500) {
      return reply.code(error.statusCode).send(errorBody(error.message));
    }

    // Internal failures are logged in full and returned vague — the details
    // are for the operator, not the caller.
    app.log.error(
      { err: error, method: request.method, path: request.url },
      "request failed",
    );
    return reply.code(500).send(errorBody("internal server error"));
  });

  await app.register(recordRoutes(store), { prefix: "/api/v1/records" });

  return app;
}
