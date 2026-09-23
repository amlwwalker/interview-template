import { useState } from "react";

import { api, RESOURCE_PATH, type ApiResult } from "../api/client";
import { IdField } from "./Field";
import { OperationPanel } from "./OperationPanel";

interface Props {
  onMutated: () => void;
}

export function DeletePanel({ onMutated }: Props) {
  const [id, setId] = useState("1");
  const [sending, setSending] = useState(false);
  const [result, setResult] = useState<ApiResult | null>(null);

  async function send() {
    // Clear the previous response first, so a stale result is never shown
    // next to an in-flight request.
    setResult(null);
    setSending(true);
    setResult(await api.remove(id.trim()));
    setSending(false);
    onMutated();
  }

  return (
    <OperationPanel
      method="DELETE"
      path={`${RESOURCE_PATH}/${id.trim() || "{id}"}`}
      note="Removes the row. Responds 204 with no body on success, 404 if it was already gone. Deleting twice is safe but the second call reports 404 — the effect is idempotent, the status code is not."
      sending={sending}
      disabled={!id.trim()}
      result={result}
      onSend={() => void send()}
    >
      <IdField value={id} onChange={setId} />
    </OperationPanel>
  );
}
