import { randomBytes } from "node:crypto";
import { chmod, mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname } from "node:path";
import { PreferencesLike } from "./types";

const ENV_ORDER = [
  "QMS_LISTEN_ADDR",
  "QMS_ROUTER_API_KEY",
  "QMS_UPSTREAM_BASE_URL",
  "QMS_UPSTREAM_API_KEY",
  "QMS_VIRTUAL_MODEL",
  "QMS_SWITCHES",
];

export const DEFAULT_ENV_PATH = `${process.env.HOME ?? ""}/.codex-quick-model-switch.env`;
export const VIRTUAL_MODEL = "codex-quick-model-switch";

export function parseEnvFile(text: string): Record<string, string> {
  const values: Record<string, string> = {};
  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) {
      continue;
    }
    const index = line.indexOf("=");
    if (index < 0) {
      continue;
    }
    const key = line.slice(0, index).trim();
    let value = line.slice(index + 1).trim();
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1);
    }
    values[key] = value;
  }
  return values;
}

export function formatEnvFile(values: Record<string, string>): string {
  const lines: string[] = [];
  const emitted = new Set<string>();
  for (const key of ENV_ORDER) {
    if (Object.prototype.hasOwnProperty.call(values, key)) {
      lines.push(`${key}=${values[key]}`);
      emitted.add(key);
    }
  }
  for (const key of Object.keys(values).sort()) {
    if (!emitted.has(key)) {
      lines.push(`${key}=${values[key]}`);
    }
  }
  return `${lines.join("\n")}\n`;
}

export function toEnvValues(
  existing: Record<string, string>,
  preferences: PreferencesLike,
  generateRouterKey = generateRouterKeyValue,
): Record<string, string> {
  const upstreamKey = preferences.upstreamApiKey?.trim();
  if (!upstreamKey) {
    throw new Error("Upstream API Key is required");
  }

  const values: Record<string, string> = {
    ...existing,
    QMS_LISTEN_ADDR: preferences.listenAddr.trim(),
    QMS_ROUTER_API_KEY: existing.QMS_ROUTER_API_KEY || generateRouterKey(),
    QMS_UPSTREAM_BASE_URL: preferences.upstreamBaseUrl.trim().replace(/\/+$/, ""),
    QMS_UPSTREAM_API_KEY: upstreamKey,
    QMS_VIRTUAL_MODEL: VIRTUAL_MODEL,
    QMS_SWITCHES: preferences.switches.trim(),
  };
  return values;
}

export async function readEnvValues(envPath: string): Promise<Record<string, string>> {
  try {
    return parseEnvFile(await readFile(envPath, "utf8"));
  } catch (error) {
    if (isNotFound(error)) {
      return {};
    }
    throw error;
  }
}

export async function writeRouterEnv(envPath: string, preferences: PreferencesLike): Promise<string> {
  const existing = await readEnvValues(envPath);
  const values = toEnvValues(existing, preferences);
  await mkdir(dirname(envPath), { recursive: true });
  await writeFile(envPath, formatEnvFile(values), { mode: 0o600 });
  await chmod(envPath, 0o600);
  return values.QMS_ROUTER_API_KEY;
}

export function generateRouterKeyValue(): string {
  return randomBytes(32).toString("base64url");
}

function isNotFound(error: unknown): boolean {
  return typeof error === "object" && error !== null && "code" in error && error.code === "ENOENT";
}
