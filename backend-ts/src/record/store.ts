import type { Pool } from "../db.js";
import {
  NotFoundError,
  type CreateParams,
  type Record,
  type ReplaceParams,
  type Store,
  type UpdateParams,
} from "./record.js";

/**
 * Postgres implementation of Store. Same SQL, statement for statement, as
 * backend/internal/record/store.go — including the COALESCE that makes PATCH
 * partial and its deliberate absence from the PUT statement.
 */

const COLUMNS = "id, name, description, active, created_at, updated_at";

interface Row {
  id: number;
  name: string;
  description: string;
  active: boolean;
  created_at: Date;
  updated_at: Date;
}

/** Postgres speaks snake_case; the API speaks camelCase and ISO strings. */
function toRecord(row: Row): Record {
  return {
    id: row.id,
    name: row.name,
    description: row.description,
    active: row.active,
    createdAt: row.created_at.toISOString(),
    updatedAt: row.updated_at.toISOString(),
  };
}

export class PostgresStore implements Store {
  constructor(private readonly pool: Pool) {}

  async list(): Promise<Record[]> {
    const { rows } = await this.pool.query<Row>(
      `SELECT ${COLUMNS} FROM records ORDER BY id ASC`,
    );
    // Always an array — an empty table must serialise as [] rather than null,
    // so the client needs no special case.
    return rows.map(toRecord);
  }

  async get(id: number): Promise<Record> {
    const { rows } = await this.pool.query<Row>(
      `SELECT ${COLUMNS} FROM records WHERE id = $1`,
      [id],
    );
    const row = rows[0];
    if (!row) throw new NotFoundError();
    return toRecord(row);
  }

  async create(params: CreateParams): Promise<Record> {
    const { rows } = await this.pool.query<Row>(
      `INSERT INTO records (name, description, active)
       VALUES ($1, $2, $3)
       RETURNING ${COLUMNS}`,
      [params.name, params.description, params.active],
    );
    return toRecord(rows[0]!);
  }

  /**
   * Overwrites every mutable column — the PUT semantic. No COALESCE here: each
   * parameter is written unconditionally, so anything the client left out of
   * the body lands in the row as its zero value.
   */
  async replace(id: number, params: ReplaceParams): Promise<Record> {
    const { rows } = await this.pool.query<Row>(
      `UPDATE records
       SET name        = $2,
           description = $3,
           active      = $4,
           updated_at  = NOW()
       WHERE id = $1
       RETURNING ${COLUMNS}`,
      [id, params.name, params.description, params.active],
    );
    const row = rows[0];
    if (!row) throw new NotFoundError();
    return toRecord(row);
  }

  /**
   * Applies a partial update in a single statement — the PATCH semantic.
   * COALESCE lets a NULL parameter mean "leave this column alone", which
   * avoids both a read-modify-write race and the usual pile of dynamically
   * built SQL.
   *
   * `?? null` is what turns an absent field into that NULL.
   */
  async update(id: number, params: UpdateParams): Promise<Record> {
    const { rows } = await this.pool.query<Row>(
      `UPDATE records
       SET name        = COALESCE($2, name),
           description = COALESCE($3, description),
           active      = COALESCE($4, active),
           updated_at  = NOW()
       WHERE id = $1
       RETURNING ${COLUMNS}`,
      [id, params.name ?? null, params.description ?? null, params.active ?? null],
    );
    const row = rows[0];
    if (!row) throw new NotFoundError();
    return toRecord(row);
  }

  async remove(id: number): Promise<void> {
    const { rowCount } = await this.pool.query(`DELETE FROM records WHERE id = $1`, [id]);
    if (!rowCount) throw new NotFoundError();
  }
}
