import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { UpdatePanel } from "./UpdatePanel";

/**
 * These mirror the Go tests in internal/record/handler_test.go. The contract
 * they pin is the same one: PATCH sends only what you ticked, PUT always sends
 * everything. If the two sides ever disagree, one of these suites goes red.
 *
 * Queries go through roles and labels rather than CSS classes so that
 * restyling the console cannot break them.
 */

let fetchMock: ReturnType<typeof vi.fn>;

function callAt(i: number): [string, RequestInit] {
  const call = fetchMock.mock.calls[i];
  if (!call) throw new Error(`no fetch call at index ${i}`);
  return call as [string, RequestInit];
}

beforeEach(() => {
  fetchMock = vi.fn().mockResolvedValue({
    status: 200,
    ok: true,
    text: async () => "{}",
  } as Response);
  vi.stubGlobal("fetch", fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

/** The panel renders the exact body it will send; that preview is the assertion target. */
function requestBody(): unknown {
  const preview = screen.getByRole("region", { name: /request body/i });
  const json = within(preview).getByText(/^\{/, { selector: "pre" });
  return JSON.parse(json.textContent ?? "{}");
}

function includeToggle(field: string) {
  return screen.getByLabelText(new RegExp(`include ${field} in the request body`, "i"));
}

describe("PATCH", () => {
  it("builds a body containing only the ticked fields", async () => {
    const user = userEvent.setup();
    render(<UpdatePanel onMutated={vi.fn()} />);

    // name is ticked by default; description and active are not.
    await user.type(screen.getByRole("textbox", { name: /^name$/i }), "renamed");

    expect(requestBody()).toEqual({ name: "renamed" });
  });

  it("adds a key when its field is ticked", async () => {
    const user = userEvent.setup();
    render(<UpdatePanel onMutated={vi.fn()} />);

    await user.click(includeToggle("description"));
    await user.type(screen.getByRole("textbox", { name: /^description$/i }), "note");

    expect(requestBody()).toEqual({ name: "", description: "note" });
  });

  it("removes the key entirely when a field is unticked", async () => {
    const user = userEvent.setup();
    render(<UpdatePanel onMutated={vi.fn()} />);

    await user.click(includeToggle("active"));
    expect(requestBody()).toHaveProperty("active");

    await user.click(includeToggle("active"));
    expect(requestBody()).not.toHaveProperty("active");
  });

  // Distinguishing "absent" from "false" is the entire reason PATCH exists
  // alongside PUT, and the reason UpdateParams uses pointers in Go.
  it("can send active:false as a deliberate value", async () => {
    const user = userEvent.setup();
    render(<UpdatePanel onMutated={vi.fn()} />);

    await user.click(includeToggle("active"));

    const body = requestBody() as { active?: boolean };
    expect(body.active).toBe(false);
    expect("active" in body).toBe(true);
  });

  it("disables send when nothing is ticked, because {} is rejected by the API", async () => {
    const user = userEvent.setup();
    render(<UpdatePanel onMutated={vi.fn()} />);

    await user.click(includeToggle("name"));

    expect(requestBody()).toEqual({});
    expect(screen.getByRole("button", { name: /send patch/i })).toBeDisabled();
  });

  it("sends the previewed body verbatim over the wire", async () => {
    const user = userEvent.setup();
    render(<UpdatePanel onMutated={vi.fn()} />);

    await user.type(screen.getByRole("textbox", { name: /^name$/i }), "wire");
    await user.click(screen.getByRole("button", { name: /send patch/i }));

    const [url, init] = callAt(0);
    expect(url).toMatch(/\/api\/v1\/records\/1$/);
    expect(init.method).toBe("PATCH");
    expect(JSON.parse(init.body as string)).toEqual({ name: "wire" });
  });
});

describe("PUT", () => {
  async function switchToPut(user: ReturnType<typeof userEvent.setup>) {
    await user.click(screen.getByRole("button", { name: "PUT" }));
  }

  it("always sends all three fields, however little you filled in", async () => {
    const user = userEvent.setup();
    render(<UpdatePanel onMutated={vi.fn()} />);
    await switchToPut(user);

    await user.type(screen.getByRole("textbox", { name: /^name$/i }), "replaced");

    expect(requestBody()).toEqual({ name: "replaced", description: "", active: false });
  });

  it("offers no include toggles, because a partial PUT is a contradiction", async () => {
    const user = userEvent.setup();
    render(<UpdatePanel onMutated={vi.fn()} />);
    await switchToPut(user);

    expect(screen.queryByLabelText(/include .* in the request body/i)).toBeNull();
  });

  it("sends a blank description as an empty string rather than omitting it", async () => {
    const user = userEvent.setup();
    render(<UpdatePanel onMutated={vi.fn()} />);
    await switchToPut(user);

    await user.type(screen.getByRole("textbox", { name: /^name$/i }), "x");
    await user.click(screen.getByRole("button", { name: /send put/i }));

    const body = JSON.parse(callAt(0)[1].body as string);
    expect(body.description).toBe("");
    expect("description" in body).toBe(true);
  });
});

describe("switching verb", () => {
  it("clears the previous response so a PUT result is never shown under PATCH", async () => {
    const user = userEvent.setup();
    render(<UpdatePanel onMutated={vi.fn()} />);

    await user.type(screen.getByRole("textbox", { name: /^name$/i }), "x");
    await user.click(screen.getByRole("button", { name: /send patch/i }));
    expect(await screen.findByText(/200/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "PUT" }));

    expect(screen.getByText(/no response yet/i)).toBeInTheDocument();
  });
});
