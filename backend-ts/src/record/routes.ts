import type { FastifyInstance, FastifyReply, FastifyRequest } from "fastify";

import { errorBody, validationBody } from "../httpx.js";
import {
  NotFoundError,
  validateCreate,
  validateReplace,
  validateUpdate,
  type CreateParams,
  type Store,
  type UpdateParams,
} from "./record.js";

/**
 * HTTP for one resource. Knows nothing about Postgres — only about the Store
 * interface, which is what lets routes.test.ts run against an in-memory fake.
 *
 * Mirrors backend/internal/record/handler.go.
 */

/**
 * Shape-level validation only. Anything that fails here is a 400 (the client
 * sent something structurally wrong); value-level rules live in record.ts and
 * produce a 422. That split is what keeps the two backends' status codes
 * identical.
 *
 * `additionalProperties: false` is the equivalent of Go's
 * `dec.DisallowUnknownFields()`.
 */
const bodyProperties = {
  name: { type: "string" },
  description: { type: "string" },
  active: { type: "boolean" },
} as const;

const bodySchema = {
  type: "object",
  additionalProperties: false,
  properties: bodyProperties,
} as const;

interface IdParams {
  id: string;
}

/**
 * Pulls {id} off the URL. Replies 400 and returns null if it is not a positive
 * integer — parsed by hand rather than by schema so the message matches the Go
 * implementation exactly.
 */
function parseId(request: FastifyRequest<{ Params: IdParams }>, reply: FastifyReply): number | null {
  const raw = request.params.id;
  const id = Number(raw);

  if (!/^\d+$/.test(raw) || !Number.isSafeInteger(id) || id < 1) {
    void reply.code(400).send(errorBody("id must be a positive integer"));
    return null;
  }

  return id;
}

function notFound(reply: FastifyReply): void {
  void reply.code(404).send(errorBody("record not found"));
}

export function recordRoutes(store: Store) {
  return async function routes(app: FastifyInstance): Promise<void> {
    // READ (collection)
    app.get("/", async (_request, reply) => {
      return reply.code(200).send(await store.list());
    });

    // CREATE
    app.post<{ Body: Partial<CreateParams> }>(
      "/",
      { schema: { body: bodySchema } },
      async (request, reply) => {
        // JSON decoding into a value type gives zero values for absent fields.
        // Go gets that from the language; here it is explicit.
        const { value, problems } = validateCreate({
          name: request.body.name ?? "",
          description: request.body.description ?? "",
          active: request.body.active ?? false,
        });

        if (problems) return reply.code(422).send(validationBody(problems));

        const created = await store.create(value);

        // r.URL.Path may or may not carry a trailing slash inside a mounted
        // subrouter, so normalise before appending the id.
        const base = request.url.split("?")[0]!.replace(/\/$/, "");
        return reply.code(201).header("Location", `${base}/${created.id}`).send(created);
      },
    );

    // READ (single)
    app.get<{ Params: IdParams }>("/:id", async (request, reply) => {
      const id = parseId(request, reply);
      if (id === null) return reply;

      try {
        return reply.code(200).send(await store.get(id));
      } catch (err) {
        if (err instanceof NotFoundError) return notFound(reply);
        throw err;
      }
    });

    // UPDATE (full replace). Every mutable field is overwritten with what the
    // body contained, so omitting a field resets it.
    app.put<{ Params: IdParams; Body: Partial<CreateParams> }>(
      "/:id",
      { schema: { body: bodySchema } },
      async (request, reply) => {
        const id = parseId(request, reply);
        if (id === null) return reply;

        const { value, problems } = validateReplace({
          name: request.body.name ?? "",
          description: request.body.description ?? "",
          active: request.body.active ?? false,
        });

        if (problems) return reply.code(422).send(validationBody(problems));

        try {
          return reply.code(200).send(await store.replace(id, value));
        } catch (err) {
          if (err instanceof NotFoundError) return notFound(reply);
          throw err;
        }
      },
    );

    // UPDATE (partial). Only the keys present in the body are touched.
    app.patch<{ Params: IdParams; Body: UpdateParams }>(
      "/:id",
      { schema: { body: bodySchema } },
      async (request, reply) => {
        const id = parseId(request, reply);
        if (id === null) return reply;

        // Passed through untouched: an absent key must stay absent, which is
        // exactly the distinction PATCH depends on.
        const { value, problems } = validateUpdate(request.body);

        if (problems) return reply.code(422).send(validationBody(problems));

        try {
          return reply.code(200).send(await store.update(id, value));
        } catch (err) {
          if (err instanceof NotFoundError) return notFound(reply);
          throw err;
        }
      },
    );

    // DELETE
    app.delete<{ Params: IdParams }>("/:id", async (request, reply) => {
      const id = parseId(request, reply);
      if (id === null) return reply;

      try {
        await store.remove(id);
        return reply.code(204).send();
      } catch (err) {
        if (err instanceof NotFoundError) return notFound(reply);
        throw err;
      }
    });
  };
}
