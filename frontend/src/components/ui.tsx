/**
 * The pieces of the wireframe-ui pattern library this project actually uses.
 *
 * Copied from the global `wireframe-ui` skill (references/patterns.tsx) and
 * trimmed to what the console needs. Import order matters less than the rule
 * every one of these obeys: square corners, hairline borders, no shadow, and
 * colour only where meaning depends on it.
 */
import {
  forwardRef,
  type ButtonHTMLAttributes,
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
} from "react";

import { border, c, motion, radius, size, space, status, text, weight } from "../styles/tokens";

/* ── Button ──────────────────────────────────────────────────────────────
 * Outlined, never filled. Hover darkens the border rather than the fill.
 */

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md";

const buttonSizes: Record<ButtonSize, CSSProperties> = {
  sm: { height: 32, padding: `0 ${space.lg}px`, fontSize: size.small },
  md: { height: 40, padding: `0 ${space["2xl"]}px`, fontSize: size.base },
};

const buttonVariants: Record<ButtonVariant, CSSProperties> = {
  primary: { color: c.ink, backgroundColor: c.bg, border: border.thin },
  secondary: { color: c.body, backgroundColor: c.bg, border: border.thin },
  ghost: { color: c.muted, backgroundColor: "transparent", border: "1px solid transparent" },
  danger: { color: c.danger, backgroundColor: c.bg, border: border.thin },
};

const restingBorder: Record<ButtonVariant, string> = {
  primary: c.line,
  secondary: c.line,
  ghost: "transparent",
  danger: c.line,
};

export const Button = forwardRef<
  HTMLButtonElement,
  ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: ButtonVariant;
    buttonSize?: ButtonSize;
  }
>(function Button({ variant = "primary", buttonSize = "md", style, disabled, ...props }, ref) {
  return (
    <button
      ref={ref}
      disabled={disabled}
      onMouseEnter={(e) => {
        if (disabled) return;
        e.currentTarget.style.borderColor = variant === "danger" ? c.danger : c.muted;
        if (variant === "ghost") e.currentTarget.style.color = c.ink;
      }}
      onMouseLeave={(e) => {
        if (disabled) return;
        e.currentTarget.style.borderColor = restingBorder[variant];
        if (variant === "ghost") e.currentTarget.style.color = c.muted;
      }}
      style={{
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        fontFamily: "inherit",
        fontWeight: weight.medium,
        borderRadius: radius.none,
        // `pointer-events: none` would stop the disabled button from being
        // found by role in tests, so only the cursor and opacity change.
        cursor: disabled ? "default" : "pointer",
        transition: motion.base,
        opacity: disabled ? 0.4 : 1,
        ...buttonVariants[variant],
        ...buttonSizes[buttonSize],
        ...style,
      }}
      {...props}
    />
  );
});

/* ── Card ────────────────────────────────────────────────────────────────
 * A white box with a hairline. That is the whole card.
 */

export function Card({
  children,
  padded = true,
  style,
  ...rest
}: HTMLAttributes<HTMLDivElement> & {
  children: ReactNode;
  padded?: boolean;
}) {
  return (
    <div
      style={{
        backgroundColor: c.bg,
        border: border.thin,
        borderRadius: radius.none,
        padding: padded ? space["2xl"] : 0,
        ...style,
      }}
      {...rest}
    >
      {children}
    </div>
  );
}

/** Hairline-separated header row inside a Card. */
export function CardHeader({ title, actions }: { title: string; actions?: ReactNode }) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        padding: `${space.md}px ${space.lg}px`,
        borderBottom: border.thin,
      }}
    >
      <span style={{ ...text.label, marginBottom: 0 }}>{title}</span>
      {actions}
    </div>
  );
}

/* ── Section label ───────────────────────────────────────────────────────
 * The uppercase micro-label — the signature of the system.
 */

export function SectionLabel({ children, style }: { children: ReactNode; style?: CSSProperties }) {
  return <span style={{ ...text.label, ...style }}>{children}</span>;
}

/* ── Badge ───────────────────────────────────────────────────────────────
 * Outlined uppercase chip. Square, tracked, tiny.
 */

export type BadgeVariant = "default" | "active" | "success" | "warning" | "error";

const badgeVariants: Record<BadgeVariant, CSSProperties> = {
  default: { color: c.muted, borderColor: c.line },
  active: { color: c.ink, borderColor: c.ink },
  success: { color: status.success, borderColor: status.success },
  warning: { color: status.warning, borderColor: status.warning },
  error: { color: c.danger, borderColor: c.danger },
};

export function Badge({
  children,
  variant = "default",
  style,
  ...rest
}: HTMLAttributes<HTMLSpanElement> & {
  children: ReactNode;
  variant?: BadgeVariant;
}) {
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        height: 22,
        padding: `0 ${space.sm}px`,
        fontSize: size.micro,
        fontWeight: weight.medium,
        textTransform: "uppercase",
        letterSpacing: "0.1em",
        borderStyle: "solid",
        borderWidth: 1,
        borderRadius: radius.none,
        whiteSpace: "nowrap",
        ...badgeVariants[variant],
        ...style,
      }}
      {...rest}
    >
      {children}
    </span>
  );
}

/* ── Table ───────────────────────────────────────────────────────────────── */

export const th: CSSProperties = {
  textAlign: "left",
  padding: `${space.md}px ${space.lg}px`,
  ...text.tableHeader,
  borderBottom: border.thin,
};

export const td: CSSProperties = {
  padding: `${space.md}px ${space.lg}px`,
  fontSize: size.body,
  color: c.ink,
  borderBottom: border.hair,
  verticalAlign: "middle",
};

/** Row with the standard hover. Never striped. */
export function TableRow({ children }: { children: ReactNode }) {
  return (
    <tr
      onMouseEnter={(e) => {
        e.currentTarget.style.backgroundColor = c.subtle;
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor = "transparent";
      }}
      style={{ transition: motion.fast }}
    >
      {children}
    </tr>
  );
}

/* ── Monospace block ─────────────────────────────────────────────────────
 * Request and response payloads. Sunken fill, no radius.
 */

export function CodeBlock({ children }: { children: ReactNode }) {
  return (
    <pre
      style={{
        margin: 0,
        padding: space.lg,
        backgroundColor: c.sunken,
        fontFamily: "var(--font-mono)",
        fontSize: size.small,
        lineHeight: 1.6,
        color: c.ink,
        whiteSpace: "pre-wrap",
        wordBreak: "break-word",
      }}
    >
      {children}
    </pre>
  );
}
