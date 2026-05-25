import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { DEFAULT_ENV_PATH, writeRouterEnv } from "./env";
import { CommandResult, PreferencesLike } from "./types";

const execFileAsync = promisify(execFile);

export type StartOptions = PreferencesLike & {
  binaryPath: string;
  envPath?: string;
};

export type ServiceDeps = {
  writeEnv?: (envPath: string, preferences: PreferencesLike) => Promise<string>;
  runBinary?: (binaryPath: string, args: string[]) => Promise<CommandResult>;
  wait?: (ms: number) => Promise<void>;
};

export async function startServer(options: StartOptions, deps: ServiceDeps = {}): Promise<{ routerKey: string }> {
  const envPath = options.envPath ?? DEFAULT_ENV_PATH;
  const writeEnv = deps.writeEnv ?? writeRouterEnv;
  const runBinary = deps.runBinary ?? runBinaryCommand;

  const routerKey = await writeEnv(envPath, options);
  await runBinary(options.binaryPath, ["service", "install"]);
  if (!(await isServiceRunning(options.binaryPath, runBinary))) {
    await runBinary(options.binaryPath, ["service", "start"]);
  }
  await doctorWithRetry(options.binaryPath, runBinary, deps.wait ?? wait);
  return { routerKey };
}

export async function toggleServer(
  options: StartOptions,
  deps: ServiceDeps = {},
): Promise<{ action: "started" | "stopped" }> {
  const runBinary = deps.runBinary ?? runBinaryCommand;

  if (await isServiceRunning(options.binaryPath, runBinary)) {
    await stopServer(options.binaryPath, runBinary);
    return { action: "stopped" };
  }

  await startServer(options, deps);
  return { action: "started" };
}

export async function stopServer(binaryPath: string, runBinary = runBinaryCommand): Promise<void> {
  await runBinary(binaryPath, ["service", "stop"]);
}

export async function serviceStatus(binaryPath: string, runBinary = runBinaryCommand): Promise<CommandResult> {
  return runBinary(binaryPath, ["service", "status"]);
}

async function doctorWithRetry(
  binaryPath: string,
  runBinary: (binaryPath: string, args: string[]) => Promise<CommandResult>,
  waitFor: (ms: number) => Promise<void>,
): Promise<void> {
  let lastError: unknown;

  for (let attempt = 1; attempt <= 20; attempt += 1) {
    try {
      await runBinary(binaryPath, ["doctor"]);
      return;
    } catch (error) {
      lastError = error;
      if (attempt === 20) {
        throw error;
      }
      await waitFor(250);
    }
  }

  throw lastError;
}

async function isServiceRunning(
  binaryPath: string,
  runBinary: (binaryPath: string, args: string[]) => Promise<CommandResult>,
): Promise<boolean> {
  try {
    const result = await serviceStatus(binaryPath, runBinary);
    return /\bstate\s*=\s*running\b/.test(result.stdout);
  } catch {
    return false;
  }
}

async function wait(ms: number): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, ms));
}

export async function runBinaryCommand(binaryPath: string, args: string[]): Promise<CommandResult> {
  try {
    const result = await execFileAsync(binaryPath, args, { timeout: 15000 });
    return { stdout: result.stdout ?? "", stderr: result.stderr ?? "" };
  } catch (error) {
    if (typeof error === "object" && error !== null && "stdout" in error && "stderr" in error) {
      const err = error as { message?: string; stdout?: string; stderr?: string };
      throw new Error([err.message, err.stdout, err.stderr].filter(Boolean).join("\n"));
    }
    throw error;
  }
}
