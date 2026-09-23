import { useId, type CSSProperties } from "react";

import { border, c, motion, radius, size, space, text } from "../styles/tokens";

/**
 * A wrapping <label> associates with its FIRST labelable descendant. When a row
 * holds both an include-toggle and a value input, that would silently bind the
 * label to the checkbox and leave the value input with no accessible name.
 * So every control here gets an explicit id/htmlFor pair, and the toggle
 * carries its own aria-label.
 */

const row: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "180px 1fr",
  alignItems: "center",
  gap: space.lg,
};

const labelCell: CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: space.sm,
};

const input: CSSProperties = {
  width: "100%",
  height: 40,
  padding: `0 ${space.md}px`,
  fontFamily: "var(--font-mono)",
  fontSize: size.small,
  color: c.ink,
  backgroundColor: c.bg,
  border: border.thin,
  borderRadius: radius.none,
  outline: "none",
  transition: motion.base,
};

function focusHandlers(disabled?: boolean) {
  return {
    onFocus: (e: React.FocusEvent<HTMLInputElement>) => {
      if (!disabled) e.currentTarget.style.borderColor = c.ink;
    },
    onBlur: (e: React.FocusEvent<HTMLInputElement>) => {
      e.currentTarget.style.borderColor = c.line;
    },
  };
}

interface TextFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  /** Optional include-toggle, used by PATCH to omit fields entirely. */
  included?: boolean;
  onIncludedChange?: (included: boolean) => void;
}

export function TextField({
  label,
  value,
  onChange,
  placeholder,
  included,
  onIncludedChange,
}: TextFieldProps) {
  const id = useId();
  const toggleable = included !== undefined && onIncludedChange !== undefined;
  const active = !toggleable || included;

  return (
    <div style={{ ...row, opacity: active ? 1 : 0.4 }} data-testid={`field-${label}`}>
      <span style={labelCell}>
        {toggleable && (
          <input
            type="checkbox"
            checked={included}
            onChange={(e) => onIncludedChange?.(e.target.checked)}
            aria-label={`Include ${label} in the request body`}
            style={{ accentColor: c.ink, width: 13, height: 13, cursor: "pointer" }}
          />
        )}
        <label htmlFor={id} style={{ ...text.label }}>
          {label}
        </label>
      </span>

      <input
        id={id}
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={!active}
        style={input}
        {...focusHandlers(!active)}
      />
    </div>
  );
}

interface BoolFieldProps {
  label: string;
  value: boolean;
  onChange: (value: boolean) => void;
  included?: boolean;
  onIncludedChange?: (included: boolean) => void;
}

export function BoolField({
  label,
  value,
  onChange,
  included,
  onIncludedChange,
}: BoolFieldProps) {
  const id = useId();
  const toggleable = included !== undefined && onIncludedChange !== undefined;
  const active = !toggleable || included;

  return (
    <div style={{ ...row, opacity: active ? 1 : 0.4 }} data-testid={`field-${label}`}>
      <span style={labelCell}>
        {toggleable && (
          <input
            type="checkbox"
            checked={included}
            onChange={(e) => onIncludedChange?.(e.target.checked)}
            aria-label={`Include ${label} in the request body`}
            style={{ accentColor: c.ink, width: 13, height: 13, cursor: "pointer" }}
          />
        )}
        <label htmlFor={id} style={{ ...text.label }}>
          {label}
        </label>
      </span>

      <span style={{ display: "flex", alignItems: "center", gap: space.sm }}>
        <input
          id={id}
          type="checkbox"
          checked={value}
          disabled={!active}
          onChange={(e) => onChange(e.target.checked)}
          style={{ accentColor: c.ink, width: 15, height: 15, cursor: active ? "pointer" : "default" }}
        />
        <code style={{ fontFamily: "var(--font-mono)", fontSize: size.small, color: c.muted }}>
          {String(value)}
        </code>
      </span>
    </div>
  );
}

export function IdField({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const id = useId();

  return (
    <div style={row} data-testid="field-id">
      <span style={labelCell}>
        <label htmlFor={id} style={{ ...text.label }}>
          id
        </label>
      </span>
      <input
        id={id}
        type="text"
        inputMode="numeric"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="1"
        style={{ ...input, maxWidth: 120 }}
        {...focusHandlers()}
      />
    </div>
  );
}
