import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { ApiResult } from "../api/client";
import { ResponseView } from "./ResponseView";

function result(partial: Partial<ApiResult>): ApiResult {
  return { status: 200, ok: true, durationMs: 3, body: null, ...partial };
}

describe("ResponseView", () => {
  it("shows a placeholder before anything has been sent", () => {
    render(<ResponseView result={null} />);

    expect(screen.getByText(/no response yet/i)).toBeInTheDocument();
  });

  it("renders the status code with its reason phrase", () => {
    render(<ResponseView result={result({ status: 422, ok: false })} />);

    expect(screen.getByText("422 Unprocessable Entity")).toBeInTheDocument();
  });

  it("falls back to the bare code for a status it has no phrase for", () => {
    render(<ResponseView result={result({ status: 418, ok: false })} />);

    expect(screen.getByText("418")).toBeInTheDocument();
  });

  // Status 0 is the client's sentinel for "the request never completed", which
  // is a different failure from any HTTP status and must not read as one.
  it("shows the network error message instead of a status line", () => {
    const networkError = "Could not reach http://localhost:8080.";
    render(<ResponseView result={result({ status: 0, ok: false, networkError })} />);

    expect(screen.getByText(/network error/i)).toBeInTheDocument();
    expect(screen.getByText(networkError)).toBeInTheDocument();
    expect(screen.queryByText("0")).toBeNull();
  });

  it("pretty-prints a JSON body", () => {
    render(<ResponseView result={result({ body: { id: 1, name: "alpha" } })} />);

    const pre = screen.getByText(/"name": "alpha"/);
    expect(pre.textContent).toContain('"id": 1');
  });

  // A 204 carries no body; rendering "null" would look like a bug in the API.
  it("says the body is empty rather than printing null", () => {
    render(<ResponseView result={result({ status: 204, body: null })} />);

    expect(screen.getByText("(empty body)")).toBeInTheDocument();
  });

  it("reports how long the request took", () => {
    render(<ResponseView result={result({ durationMs: 42 })} />);

    expect(screen.getByText("42 ms")).toBeInTheDocument();
  });
});
