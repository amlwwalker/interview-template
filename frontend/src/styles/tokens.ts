/**
 * Wireframe UI — design tokens.
 *
 * Copied from the global `wireframe-ui` skill (references/tokens.ts) as that
 * skill instructs. Inline styles read these values; nothing here is a CSS
 * class, so no build-time class scanner can strip it.
 *
 * Local change: the source file refers to `React.CSSProperties` without
 * importing React, which does not compile in a `.ts` module. Imported as a
 * type below; everything else is verbatim.
 */
import type { CSSProperties } from "react";

/* ── Colour ─────────────────────────────────────────────────────────────
 * Monochrome by design. Hierarchy comes from grey value alone.
 * Only `danger` and the `status` block introduce hue, and they are used
 * sparingly — a coloured element should feel like an exception.
 */
export const c = {
  /** Primary text, active nav, emphasis borders. */
  ink: "#111",
  /** Secondary text, descriptions, blockquotes. */
  body: "#555",
  /** Labels, placeholders, inactive nav, metadata. The most-used grey. */
  muted: "#999",
  /** Disabled text, chevrons, lowest-emphasis marks. */
  faint: "#ccc",

  /** Standard border. The workhorse — card edges, inputs, dividers. */
  line: "#e8e8e8",
  /** Lighter divider, for rows inside an already-bordered container. */
  hair: "#f0f0f0",

  /** Page and card background. */
  bg: "#fff",
  /** Row hover, table head fill, active nav fill. */
  subtle: "#fafafa",
  /** Code blocks, inline code, pressed states. */
  sunken: "#f5f5f5",
  /** Inverted surfaces (rare — footers, dark callouts). */
  dark: "#111",
  /** Text on a dark surface. */
  onDark: "#fafafa",

  /** Errors and destructive actions. */
  danger: "#cc0000",
} as const;

/** Status hues. Use only where meaning genuinely depends on colour. */
export const status = {
  error: "#dc2626",
  errorSoft: "#ef5350",
  errorBg: "#fef2f2",
  success: "#4caf50",
  warning: "#f59e0b",
  warningBg: "#fff3e0",
  warningLine: "#ffe0b2",
  info: "#2563eb",
} as const;

/* ── Type ───────────────────────────────────────────────────────────────
 * Headings use a distinct display face; body text a neutral sans.
 * Declared as CSS variables so the font files are imported once in
 * globals.css and swapped per project without touching components.
 */
export const font = {
  heading: "var(--font-heading)",
  body: "var(--font-body)",
  mono: "var(--font-mono)",
} as const;

/**
 * Size ramp. Unitless numbers — React appends `px`.
 * 10–14 carry almost the entire interface; the large sizes are for page
 * titles and marketing only.
 */
export const size = {
  micro: 10,  // badges, table headers
  label: 11,  // uppercase section labels
  small: 12,  // meta, helper text, dense tables
  base: 13,   // buttons, controls, most UI text
  body: 14,   // paragraphs, table cells, nav
  lead: 16,   // emphasised body
  h3: 18,
  h2: 20,
  h1: 28,     // page title
  display: 48,
} as const;

/**
 * Weights. Nothing is bold. A page title is *lighter* than its body text
 * is heavy — hierarchy comes from size and space, not thickness.
 */
export const weight = {
  regular: 400,
  medium: 500,
  semibold: 600,
} as const;

/**
 * Tracking. Negative tightens large display type; positive opens the
 * uppercase micro-labels that define the system's look.
 */
export const tracking = {
  tight: "-0.02em",   // page titles
  normal: "0",
  label: "0.1em",     // uppercase labels, badges
  wide: "0.15em",     // table headers
  wider: "0.25em",    // eyebrow text
} as const;

/* ── Space ──────────────────────────────────────────────────────────────
 * 4px base. The layout is intentionally airy.
 */
export const space = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 20,
  "2xl": 24,
  "3xl": 32,
  "4xl": 48,
} as const;

/* ── Border ─────────────────────────────────────────────────────────────
 * Square corners are non-negotiable. `radius.full` exists for avatars and
 * `radius.chip` for tiny inline tags — nothing else should be rounded.
 */
export const border = {
  /** Standard hairline. */
  thin: `1px solid ${c.line}`,
  /** Lighter inner divider. */
  hair: `1px solid ${c.hair}`,
  /** Emphasis border (selected, active). */
  strong: `1px solid ${c.ink}`,
  /** Destructive. */
  danger: `1px solid ${c.danger}`,
} as const;

export const radius = {
  none: 0,
  chip: 2,
  full: "50%",
} as const;

/** Transitions are quick and unfussy. */
export const motion = {
  fast: "all 0.15s",
  base: "all 0.2s",
} as const;

/* ── Composed text styles ───────────────────────────────────────────────
 * Ready-made objects for the shapes that repeat on every screen.
 * Spread them: <h1 style={{ ...text.pageTitle }}>
 */
export const text: Record<string, CSSProperties> = {
  /** Page H1. Light, large, slightly tightened. */
  pageTitle: {
    fontFamily: font.heading,
    fontSize: size.h1,
    fontWeight: weight.regular,
    color: c.ink,
    letterSpacing: tracking.tight,
  },
  /** The muted sentence under a page title. */
  pageSubtitle: {
    marginTop: space.sm,
    fontSize: size.body,
    color: c.muted,
  },
  /** Section heading inside a page. */
  sectionTitle: {
    fontFamily: font.heading,
    fontSize: size.h2,
    fontWeight: weight.regular,
    color: c.ink,
  },
  /** The uppercase micro-label — the system's signature. */
  label: {
    display: "block",
    fontSize: size.label,
    fontWeight: weight.medium,
    color: c.muted,
    textTransform: "uppercase",
    letterSpacing: tracking.label,
  },
  /** Table header cell text. */
  tableHeader: {
    fontSize: size.micro,
    fontWeight: weight.medium,
    color: c.muted,
    textTransform: "uppercase",
    letterSpacing: tracking.wide,
  },
  /** Default body copy. */
  body: {
    fontSize: size.body,
    color: c.ink,
  },
  /** De-emphasised meta text. */
  meta: {
    fontSize: size.small,
    color: c.muted,
  },
};
