/**
 * The resource: model, validation and the Store contract.
 *
 * Mirrors backend/internal/record/record.go. The resource is deliberately
 * anonymous — a row with a name, a description and a flag. It exists to
 * exercise the six HTTP verbs, not to model anything.
 */

/** Thrown by the Store when a row does not exist. Routes map it to a 404. */
export class NotFoundError extends Error {
  constructor() {
    super("record not found");
    this.name = "NotFoundError";
  }
}

export const MAX_NAME = 200;
export const MAX_DESCRIPTION = 2000;

export interface Record {
  id: number;
  name: string;
  description: string;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

/** Per-field problems, or null when the input is fine. */
export type Problems = { [field: string]: string } | null;

/** Body for POST. */
export interface CreateParams {
  name: string;
  description: string;
  active: boolean;
}

/**
 * Body for PUT — a *full replacement*.
 *
 * Fields are required, non-optional values, and that is the entire point:
 * whatever you send is what the row becomes. Omit `description` and it is
 * replaced with "". Omit `active` and it is replaced with false. PUT is
 * idempotent because the outcome depends only on the body, never on the row's
 * current state.
 *
 * This is the TypeScript equivalent of Go's plain-value struct fields.
 */
export type ReplaceParams = CreateParams;

/**
 * Body for PATCH — a *partial* update.
 *
 * Every field is optional, so an absent key is distinguishable from one set to
 * a falsy value. That distinction is the whole reason PATCH exists alongside
 * PUT: without it `{"active": false}` would be indistinguishable from `{}`.
 *
 * Go uses `*bool` for this; TypeScript uses `?:`, and `'active' in body` is
 * the equivalent of a non-nil pointer check.
 */
export interface UpdateParams {
  name?: string;
  description?: string;
  active?: boolean;
}

function checkName(name: string, problems: Record_Problems): void {
  if (name === "") {
    problems["name"] = "name is required";
  } else if (name.length > MAX_NAME) {
    problems["name"] = "name must be 200 characters or fewer";
  }
}

function checkDescription(description: string, problems: Record_Problems): void {
  if (description.length > MAX_DESCRIPTION) {
    problems["description"] = "description must be 2000 characters or fewer";
  }
}

type Record_Problems = { [field: string]: string };

/**
 * Normalises input in place and reports problems. Returning the cleaned value
 * alongside keeps the trim from being forgotten at a call site.
 */
export function validateCreate(raw: CreateParams): { value: CreateParams; problems: Problems } {
  const value: CreateParams = {
    name: raw.name.trim(),
    description: raw.description.trim(),
    active: raw.active,
  };

  const problems: Record_Problems = {};
  checkName(value.name, problems);
  checkDescription(value.description, problems);

  return { value, problems: Object.keys(problems).length ? problems : null };
}

/** PUT replaces the whole resource, so every required field must be present. */
export function validateReplace(raw: ReplaceParams): { value: ReplaceParams; problems: Problems } {
  return validateCreate(raw);
}

/** PATCH checks only the fields that were actually supplied. */
export function validateUpdate(raw: UpdateParams): { value: UpdateParams; problems: Problems } {
  const value: UpdateParams = {};
  const problems: Record_Problems = {};

  if (raw.name !== undefined) {
    const trimmed = raw.name.trim();
    value.name = trimmed;
    if (trimmed === "") {
      problems["name"] = "name cannot be empty";
    } else if (trimmed.length > MAX_NAME) {
      problems["name"] = "name must be 200 characters or fewer";
    }
  }

  if (raw.description !== undefined) {
    const trimmed = raw.description.trim();
    value.description = trimmed;
    checkDescription(trimmed, problems);
  }

  if (raw.active !== undefined) {
    value.active = raw.active;
  }

  if (Object.keys(value).length === 0) {
    problems["_"] = "provide at least one of name, description or active";
  }

  return { value, problems: Object.keys(problems).length ? problems : null };
}

/**
 * The persistence contract. Routes depend on this rather than on Postgres,
 * which is what lets the route tests run without a database.
 */
export interface Store {
  list(): Promise<Record[]>;
  get(id: number): Promise<Record>;
  create(params: CreateParams): Promise<Record>;
  replace(id: number, params: ReplaceParams): Promise<Record>;
  update(id: number, params: UpdateParams): Promise<Record>;
  remove(id: number): Promise<void>;
}
