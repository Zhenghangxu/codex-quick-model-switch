import { randomBytes } from "node:crypto";
import { chmod, mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname } from "node:path";
import { PreferencesLike } from "./types";
import { normalizeSwitches } from "./switchDefaults";

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

export type EnvReadResult = {
  exists: boolean;
  values: Record<string, string>;
};

export type GeneratedEnvStatus = {
  exists: boolean;
  routerKeyPresent: boolean;
  matchesPreferences: boolean;
  differences: string[];
  values: Record<string, string>;
};

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
    QMS_SWITCHES: normalizeSwitches(preferences.switches),
  };
  return values;
}

export async function readEnvValues(envPath: string): Promise<Record<string, string>> {
  return (await readEnvValuesWithPresence(envPath)).values;
}

export async function readEnvValuesWithPresence(envPath: string): Promise<EnvReadResult> {
  try {
    return { exists: true, values: parseEnvFile(await readFile(envPath, "utf8")) };
  } catch (error) {
    if (isNotFound(error)) {
      return { exists: false, values: {} };
    }
    throw error;
  }
}

export async function generatedEnvStatus(
  envPath: string,
  preferences: PreferencesLike,
  readEnv: (envPath: string) => Promise<EnvReadResult> = readEnvValuesWithPresence,
): Promise<GeneratedEnvStatus> {
  const result = await readEnv(envPath);
  if (!result.exists) {
    return {
      exists: false,
      routerKeyPresent: false,
      matchesPreferences: false,
      differences: ["Env file is missing"],
      values: result.values,
    };
  }

  const differences = compareGeneratedEnv(result.values, preferences);
  return {
    exists: result.exists,
    routerKeyPresent: Boolean(result.values.QMS_ROUTER_API_KEY),
    matchesPreferences: result.exists && differences.length === 0,
    differences,
    values: result.values,
  };
}

function compareGeneratedEnv(values: Record<string, string>, preferences: PreferencesLike): string[] {
  const expected = {
    QMS_LISTEN_ADDR: preferences.listenAddr.trim(),
    QMS_UPSTREAM_BASE_URL: preferences.upstreamBaseUrl.trim().replace(/\/+$/, ""),
    QMS_UPSTREAM_API_KEY: preferences.upstreamApiKey?.trim() ?? "",
    QMS_VIRTUAL_MODEL: VIRTUAL_MODEL,
    QMS_SWITCHES: normalizeSwitches(preferences.switches),
  };
  const labels: Record<keyof typeof expected, string> = {
    QMS_LISTEN_ADDR: "Raycast Router Listen Address",
    QMS_UPSTREAM_BASE_URL: "Raycast Upstream Base URL",
    QMS_UPSTREAM_API_KEY: "Raycast Upstream API Key",
    QMS_VIRTUAL_MODEL: "generated virtual model",
    QMS_SWITCHES: "Raycast Model Switches",
  };

  return (Object.keys(expected) as Array<keyof typeof expected>).flatMap((key) => {
    if ((values[key] ?? "") === expected[key]) {
      return [];
    }
    return [`${key} differs from ${labels[key]}`];
  });
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
