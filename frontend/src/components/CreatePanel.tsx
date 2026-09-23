import { useState } from "react";

import { api, RESOURCE_PATH, type ApiResult, type CreateBody } from "../api/client";
import { BoolField, TextField } from "./Field";
import { OperationPanel } from "./OperationPanel";

interface Props {
  onMutated: () => void;
}

export function CreatePanel({ onMutated }: Props) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [active, setActive] = useState(false);

  const [sending, setSending] = useState(false);
  const [result, setResult] = useState<ApiResult | null>(null);

  const body: CreateBody = { name, description, active };

  async function send() {
    // Clear the previous response first, so a stale result is never shown
    // next to an in-flight request.
    setResult(null);
    setSending(true);
    setResult(await api.create(body));
    setSending(false);
    onMutated();
  }

  return (
    <OperationPanel
      method="POST"
      path={RESOURCE_PATH}
      body={body}
      note="Inserts a new row. Responds 201 with the created resource and a Location header. Not idempotent — sending twice creates two rows."
      sending={sending}
      result={result}
      onSend={() => void send()}
    >
      <TextField label="name" value={name} onChange={setName} placeholder="required" />
      <TextField
        label="description"
        value={description}
        onChange={setDescription}
        placeholder="optional"
      />
      <BoolField label="active" value={active} onChange={setActive} />
    </OperationPanel>
  );
}
