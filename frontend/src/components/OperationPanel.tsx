import type { ReactNode } from "react";

import type { ApiResult, HttpMethod } from "../api/client";
import { c, size, space } from "../styles/tokens";
import { ResponseView } from "./ResponseView";
import { Badge, Button, Card, CodeBlock, SectionLabel, type BadgeVariant } from "./ui";

interface Props {
  method: HttpMethod;
  path: string;
  /** Request body to preview, or undefined for verbs that send none. */
  body?: unknown;
  /** Explains what this verb does to the resource. */
  note: string;
  sending: boolean;
  disabled?: boolean;
  result: ApiResult | null;
  onSend: () => void;
  children: ReactNode;
}

/** Only the destructive verb earns colour; the rest stay monochrome. */
function methodVariant(method: HttpMethod): BadgeVariant {
  return method === "DELETE" ? "error" : "active";
}

/**
 * Shared shell for every operation: the form, the exact request that will be
 * sent, and the raw response. Showing the request verbatim is the point — it
 * makes the PUT/PATCH difference visible rather than something you have to
 * take on faith.
 */
export function OperationPanel({
  method,
  path,
  body,
  note,
  sending,
  disabled,
  result,
  onSend,
  children,
}: Props) {
  return (
    <Card style={{ marginBottom: space.lg }} data-testid="panel">
      <div style={{ display: "flex", alignItems: "center", gap: space.md }}>
        <Badge variant={methodVariant(method)}>{method}</Badge>
        <code
          style={{
            fontFamily: "var(--font-mono)",
            fontSize: size.small,
            color: c.ink,
            wordBreak: "break-all",
          }}
        >
          {path}
        </code>
      </div>

      <p style={{ margin: `${space.md}px 0 0`, fontSize: size.small, color: c.body }}>{note}</p>

      <div
        style={{
          display: "flex",
          flexDirection: "column",
          gap: space.md,
          margin: `${space.xl}px 0`,
        }}
      >
        {children}
      </div>

      {body !== undefined && (
        // A labelled section exposes this as a landmark, so tests (and screen
        // readers) can address it by name rather than by class.
        <section aria-label="Request body" data-testid="request-body" style={{ marginBottom: space.lg }}>
          <SectionLabel style={{ marginBottom: space.sm }}>Request body</SectionLabel>
          <CodeBlock>{JSON.stringify(body, null, 2)}</CodeBlock>
        </section>
      )}

      <div style={{ paddingBottom: space.xl }}>
        <Button
          type="button"
          variant={method === "DELETE" ? "danger" : "primary"}
          onClick={onSend}
          disabled={sending || disabled}
          data-testid="send"
        >
          {sending ? "Sending…" : `Send ${method}`}
        </Button>
      </div>

      <ResponseView result={result} />
    </Card>
  );
}
