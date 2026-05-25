import { SwitchesResponse } from "./types";

export type FetchLike = (input: string, init?: RequestInit) => Promise<Pick<Response, "ok" | "status" | "statusText" | "json" | "text">>;

export function routerBaseUrl(listenAddr: string): string {
  const trimmed = listenAddr.trim();
  if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) {
    return trimmed.replace(/\/+$/, "");
  }
  const normalized = trimmed.replace(/^0\.0\.0\.0:/, "127.0.0.1:");
  return `http://${normalized}`;
}

export async function getSwitches(baseUrl: string, routerKey: string, fetchImpl: FetchLike = fetch): Promise<SwitchesResponse> {
  const response = await fetchImpl(`${baseUrl.replace(/\/+$/, "")}/switches`, {
    headers: authHeaders(routerKey),
  });
  if (!response.ok) {
    throw new Error(`GET /switches failed: ${response.status} ${response.statusText}`);
  }
  return (await response.json()) as SwitchesResponse;
}

export async function switchModel(
  baseUrl: string,
  routerKey: string,
  shortcut: string,
  fetchImpl: FetchLike = fetch,
): Promise<void> {
  const response = await fetchImpl(`${baseUrl.replace(/\/+$/, "")}/switch`, {
    method: "POST",
    headers: {
      ...authHeaders(routerKey),
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ shortcut }),
  });
  if (!response.ok) {
    let detail = "";
    try {
      detail = await response.text();
    } catch {
      detail = "";
    }
    throw new Error(`POST /switch failed: ${response.status} ${response.statusText}${detail ? `: ${detail}` : ""}`);
  }
}

export async function healthz(baseUrl: string, fetchImpl: FetchLike = fetch): Promise<boolean> {
  try {
    const response = await fetchImpl(`${baseUrl.replace(/\/+$/, "")}/healthz`);
    return response.ok || response.status === 204;
  } catch {
    return false;
  }
}

function authHeaders(routerKey: string): Record<string, string> {
  return routerKey ? { Authorization: `Bearer ${routerKey}` } : {};
}
