import { useState, type CSSProperties } from "react";

import {
  api,
  RESOURCE_PATH,
  type ApiResult,
  type PatchBody,
  type ReplaceBody,
} from "../api/client";
import { c, size, space } from "../styles/tokens";
import { BoolField, IdField, TextField } from "./Field";
import { OperationPanel } from "./OperationPanel";
import { Button } from "./ui";

type Verb = "PATCH" | "PUT";

interface Props {
  onMutated: () => void;
}

export function UpdatePanel({ onMutated }: Props) {
  const [verb, setVerb] = useState<Verb>("PATCH");

  const [id, setId] = useState("1");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [active, setActive] = useState(false);

  // Which fields PATCH actually puts in the body. Unchecking one omits the key
  // entirely, which is the whole distinction being demonstrated.
  const [includeName, setIncludeName] = useState(true);
  const [includeDescription, setIncludeDescription] = useState(false);
  const [includeActive, setIncludeActive] = useState(false);

  const [sending, setSending] = useState(false);
  const [result, setResult] = useState<ApiResult | null>(null);

  const patchBody: PatchBody = {
    ...(includeName ? { name } : {}),
    ...(includeDescription ? { description } : {}),
    ...(includeActive ? { active } : {}),
  };

  const putBody: ReplaceBody = { name, description, active };

  const body = verb === "PATCH" ? patchBody : putBody;

  async function send() {
    const trimmed = id.trim();

    // Clear the previous response first, so a stale result is never shown next
    // to an in-flight request.
    setResult(null);
    setSending(true);
    setResult(
      verb === "PATCH"
        ? await api.patch(trimmed, patchBody)
        : await api.replace(trimmed, putBody),
    );
    setSending(false);
    onMutated();
  }

  const note =
    verb === "PATCH"
      ? "Partial update. Only the keys present in the body are touched; everything else keeps its current value. Send {} and it is rejected — there is nothing to do."
      : "Full replacement. Every field is overwritten with what you send, so an omitted field is reset to its zero value. Idempotent: sending the same body twice leaves the same row.";

  const hint: CSSProperties = { margin: 0, fontSize: size.small, color: c.muted };

  return (
    <>
      <div
        role="group"
        aria-label="Update method"
        style={{ display: "flex", gap: space.sm, marginBottom: space.lg }}
      >
        {(["PATCH", "PUT"] as const).map((value) => (
          <Button
            key={value}
            type="button"
            buttonSize="sm"
            variant={verb === value ? "primary" : "ghost"}
            aria-pressed={verb === value}
            style={
              verb === value
                ? { borderColor: c.ink, fontFamily: "var(--font-mono)" }
                : { fontFamily: "var(--font-mono)" }
            }
            onClick={() => {
              setVerb(value);
              setResult(null);
            }}
          >
            {value}
          </Button>
        ))}
      </div>

      <OperationPanel
        method={verb}
        path={`${RESOURCE_PATH}/${id.trim() || "{id}"}`}
        body={body}
        note={note}
        sending={sending}
        disabled={!id.trim() || (verb === "PATCH" && Object.keys(patchBody).length === 0)}
        result={result}
        onSend={() => void send()}
      >
        <IdField value={id} onChange={setId} />

        {verb === "PATCH" ? (
          <>
            <p style={hint}>
              Tick a field to include it in the body. Unticked fields are left out of the
              JSON entirely — compare the payload below as you toggle them.
            </p>
            <TextField
              label="name"
              value={name}
              onChange={setName}
              included={includeName}
              onIncludedChange={setIncludeName}
            />
            <TextField
              label="description"
              value={description}
              onChange={setDescription}
              included={includeDescription}
              onIncludedChange={setIncludeDescription}
            />
            <BoolField
              label="active"
              value={active}
              onChange={setActive}
              included={includeActive}
              onIncludedChange={setIncludeActive}
            />
          </>
        ) : (
          <>
            <p style={hint}>
              All three fields are always sent. Leave one blank and it is written as blank,
              not left alone.
            </p>
            <TextField label="name" value={name} onChange={setName} placeholder="required" />
            <TextField
              label="description"
              value={description}
              onChange={setDescription}
              placeholder="blank means empty string"
            />
            <BoolField label="active" value={active} onChange={setActive} />
          </>
        )}
      </OperationPanel>
    </>
  );
}
