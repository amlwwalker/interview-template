import { useCallback, useEffect, useState } from "react";

import { api, type Record } from "./api/client";
import { CreatePanel } from "./components/CreatePanel";
import { DeletePanel } from "./components/DeletePanel";
import { ReadPanel } from "./components/ReadPanel";
import { UpdatePanel } from "./components/UpdatePanel";
import { Button, Card, CardHeader, td, th, TableRow } from "./components/ui";
import { border, c, motion, size, space, text, weight } from "./styles/tokens";

const TABS = ["create", "read", "update", "delete"] as const;
type Tab = (typeof TABS)[number];

export default function App() {
  const [tab, setTab] = useState<Tab>("create");

  // A live view of the table, refreshed after every mutation. Not part of any
  // operation — just so the ids you need are on screen.
  const [rows, setRows] = useState<Record[] | null>(null);

  const refresh = useCallback(async () => {
    const result = await api.list();
    setRows(result.ok && Array.isArray(result.body) ? (result.body as Record[]) : null);
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return (
    <main
      style={{
        maxWidth: 880,
        margin: "0 auto",
        padding: `${space["4xl"]}px ${space["3xl"]}px`,
      }}
    >
      <header style={{ marginBottom: space["3xl"] }}>
        <h1 style={{ ...text.pageTitle, margin: 0 }}>CRUD</h1>
        <p style={{ ...text.pageSubtitle, margin: `${space.sm}px 0 0` }}>
          React and Vite against a Go API, cross-origin
        </p>
      </header>

      {/* Underline tabs: the active one is a 1px ink rule, never a pill. */}
      <div
        role="tablist"
        aria-label="Operation"
        style={{
          display: "flex",
          gap: space["2xl"],
          borderBottom: border.thin,
          marginBottom: space["2xl"],
        }}
      >
        {TABS.map((value) => {
          const isActive = tab === value;
          return (
            <button
              key={value}
              type="button"
              role="tab"
              aria-selected={isActive}
              onClick={() => setTab(value)}
              style={{
                background: "none",
                border: "none",
                borderBottom: `1px solid ${isActive ? c.ink : "transparent"}`,
                marginBottom: -1,
                padding: `${space.md}px 0`,
                fontFamily: "inherit",
                fontSize: size.label,
                fontWeight: isActive ? weight.medium : weight.regular,
                letterSpacing: "0.15em",
                textTransform: "uppercase",
                color: isActive ? c.ink : c.muted,
                cursor: "pointer",
                transition: motion.fast,
              }}
            >
              {value}
            </button>
          );
        })}
      </div>

      <div role="tabpanel">
        {tab === "create" && <CreatePanel onMutated={() => void refresh()} />}
        {tab === "read" && <ReadPanel />}
        {tab === "update" && <UpdatePanel onMutated={() => void refresh()} />}
        {tab === "delete" && <DeletePanel onMutated={() => void refresh()} />}
      </div>

      <Card padded={false} style={{ marginTop: space["3xl"] }} data-testid="table">
        <CardHeader
          title="Table contents"
          actions={
            <Button type="button" buttonSize="sm" variant="secondary" onClick={() => void refresh()}>
              Refresh
            </Button>
          }
        />

        {rows === null ? (
          <p style={{ margin: 0, padding: space.lg, fontSize: size.body, color: c.muted }}>
            Could not read the table.
          </p>
        ) : rows.length === 0 ? (
          <p style={{ margin: 0, padding: space.lg, fontSize: size.body, color: c.muted }}>
            No rows
          </p>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr>
                <th style={{ ...th, width: 64 }}>id</th>
                <th style={th}>name</th>
                <th style={th}>description</th>
                <th style={{ ...th, width: 90 }}>active</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <TableRow key={row.id}>
                  <td style={{ ...td, fontFamily: "var(--font-mono)", color: c.muted }}>{row.id}</td>
                  <td style={td}>{row.name}</td>
                  <td style={{ ...td, color: row.description ? c.body : c.faint }}>
                    {row.description || '""'}
                  </td>
                  <td
                    style={{
                      ...td,
                      fontFamily: "var(--font-mono)",
                      fontSize: size.small,
                      color: row.active ? c.ink : c.faint,
                    }}
                  >
                    {String(row.active)}
                  </td>
                </TableRow>
              ))}
            </tbody>
          </table>
        )}
      </Card>
    </main>
  );
}
