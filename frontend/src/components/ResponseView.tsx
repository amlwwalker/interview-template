import type { ApiResult } from "../api/client";
import { border, c, size, space } from "../styles/tokens";
import { Badge, CodeBlock, SectionLabel, type BadgeVariant } from "./ui";

interface Props {
  result: ApiResult | null;
}

/**
 * HTTP status class is one of the few places where meaning genuinely depends
 * on colour, so it is allowed the status hues. Everything else here stays
 * monochrome.
 */
function statusVariant(status: number): BadgeVariant {
  if (status === 0) return "error";
  if (status < 300) return "success";
  if (status < 400) return "default";
  if (status < 500) return "warning";
  return "error";
}

const STATUS_TEXT: { [code: number]: string } = {
  200: "OK",
  201: "Created",
  204: "No Content",
  400: "Bad Request",
  404: "Not Found",
  405: "Method Not Allowed",
  422: "Unprocessable Entity",
  500: "Internal Server Error",
  503: "Service Unavailable",
};

export function ResponseView({ result }: Props) {
  if (!result) {
    return (
      <div data-testid="response" style={{ borderTop: border.hair, paddingTop: space.lg }}>
        <p style={{ margin: 0, fontSize: size.small, color: c.faint }}>No response yet</p>
      </div>
    );
  }

  const label =
    result.status === 0
      ? "Network error"
      : `${result.status} ${STATUS_TEXT[result.status] ?? ""}`.trim();

  return (
    <section
      aria-label="Response"
      data-testid="response"
      style={{ borderTop: border.hair, paddingTop: space.lg }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: space.md,
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: space.md }}>
          <SectionLabel>Response</SectionLabel>
          <Badge variant={statusVariant(result.status)} data-testid="status">
            {label}
          </Badge>
        </div>
        <span
          style={{ fontFamily: "var(--font-mono)", fontSize: size.micro, color: c.faint }}
        >
          {result.durationMs} ms
        </span>
      </div>

      {result.networkError ? (
        <p style={{ margin: 0, fontSize: size.small, color: c.danger }}>{result.networkError}</p>
      ) : result.body === null ? (
        <p style={{ margin: 0, fontSize: size.small, color: c.muted }}>(empty body)</p>
      ) : (
        <CodeBlock>{JSON.stringify(result.body, null, 2)}</CodeBlock>
      )}
    </section>
  );
}
