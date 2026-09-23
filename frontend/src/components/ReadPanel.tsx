import { useState, type CSSProperties } from "react";

import { api, RESOURCE_PATH, type ApiResult } from "../api/client";
import { IdField } from "./Field";
import { OperationPanel } from "./OperationPanel";
import { c, size } from "../styles/tokens";

const hintStyle: CSSProperties = { margin: 0, fontSize: size.small, color: c.muted };

export function ReadPanel() {
  const [listSending, setListSending] = useState(false);
  const [listResult, setListResult] = useState<ApiResult | null>(null);

  const [id, setId] = useState("1");
  const [getSending, setGetSending] = useState(false);
  const [getResult, setGetResult] = useState<ApiResult | null>(null);

  // Each send clears its previous response first, so a stale result is never
  // shown next to an in-flight request.
  async function sendList() {
    setListResult(null);
    setListSending(true);
    setListResult(await api.list());
    setListSending(false);
  }

  async function sendGet() {
    setGetResult(null);
    setGetSending(true);
    setGetResult(await api.get(id.trim()));
    setGetSending(false);
  }

  return (
    <>
      <OperationPanel
        method="GET"
        path={RESOURCE_PATH}
        note="Returns the whole collection. Always an array — an empty table gives [], never null, so the client needs no special case."
        sending={listSending}
        result={listResult}
        onSend={() => void sendList()}
      >
        <p style={hintStyle}>No request body, no parameters.</p>
      </OperationPanel>

      <OperationPanel
        method="GET"
        path={`${RESOURCE_PATH}/${id.trim() || "{id}"}`}
        note="Returns a single row. 404 when it does not exist, 400 when the id is not a positive integer."
        sending={getSending}
        disabled={!id.trim()}
        result={getResult}
        onSend={() => void sendGet()}
      >
        <IdField value={id} onChange={setId} />
      </OperationPanel>
    </>
  );
}
