/**
 * The small, boring helpers that keep route handlers readable: one error
 * shape, used everywhere.
 *
 * Mirrors backend/internal/httpx/respond.go, byte for byte on the wire — the
 * two backends are interchangeable behind the same frontend, which is only
 * true if their error bodies agree.
 */

export interface ErrorBody {
  error: string;
  details?: { [field: string]: string };
}

export function errorBody(message: string): ErrorBody {
  return { error: message };
}

export function validationBody(details: { [field: string]: string }): ErrorBody {
  return { error: "validation failed", details };
}
