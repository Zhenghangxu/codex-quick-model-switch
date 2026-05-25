import { describe, expect, it, vi } from "vitest";
import { startServer, toggleServer } from "../src/lib/service";

const testOptions = {
  binaryPath: "/tmp/qms",
  envPath: "/tmp/qms.env",
  listenAddr: "127.0.0.1:8321",
  upstreamBaseUrl: "http://localhost:8317/v1",
  upstreamApiKey: "upstream-key",
  switches: "/msm=gpt-5.5:medium:fast",
};

describe("service orchestration", () => {
  it("writes env then installs, starts when stopped, and doctors the LaunchAgent", async () => {
    const writeEnv = vi.fn(async () => "router-key");
    const runBinary = vi.fn(async (_binaryPath: string, args: string[]) => ({
      stdout: args[1] === "status" ? "state = not running" : "ok",
      stderr: "",
    }));

    const result = await startServer(
      testOptions,
      { writeEnv, runBinary },
    );

    expect(result.routerKey).toBe("router-key");
    expect(writeEnv).toHaveBeenCalledTimes(1);
    expect(runBinary.mock.calls).toEqual([
      ["/tmp/qms", ["service", "install"]],
      ["/tmp/qms", ["service", "status"]],
      ["/tmp/qms", ["service", "start"]],
      ["/tmp/qms", ["doctor"]],
    ]);
  });

  it("skips service start when the LaunchAgent is already running", async () => {
    const writeEnv = vi.fn(async () => "router-key");
    const runBinary = vi.fn(async (_binaryPath: string, args: string[]) => ({
      stdout: args[1] === "status" ? "state = running" : "ok",
      stderr: "",
    }));

    await startServer(
      testOptions,
      { writeEnv, runBinary },
    );

    expect(runBinary.mock.calls).toEqual([
      ["/tmp/qms", ["service", "install"]],
      ["/tmp/qms", ["service", "status"]],
      ["/tmp/qms", ["doctor"]],
    ]);
  });

  it("retries doctor while the LaunchAgent is still becoming healthy", async () => {
    const writeEnv = vi.fn(async () => "router-key");
    const wait = vi.fn(async () => undefined);
    let doctorAttempts = 0;
    const runBinary = vi.fn(async (_binaryPath: string, args: string[]) => {
      if (args[1] === "status") {
        return { stdout: "state = not running", stderr: "" };
      }
      if (args[0] === "doctor") {
        doctorAttempts += 1;
        if (doctorAttempts === 1) {
          throw new Error("connection refused");
        }
      }
      return { stdout: "ok", stderr: "" };
    });

    const result = await startServer(testOptions, { writeEnv, runBinary, wait });

    expect(result.routerKey).toBe("router-key");
    expect(wait).toHaveBeenCalledWith(250);
    expect(runBinary.mock.calls).toEqual([
      ["/tmp/qms", ["service", "install"]],
      ["/tmp/qms", ["service", "status"]],
      ["/tmp/qms", ["service", "start"]],
      ["/tmp/qms", ["doctor"]],
      ["/tmp/qms", ["doctor"]],
    ]);
  });

  it("toggle starts the LaunchAgent when status reports stopped", async () => {
    const writeEnv = vi.fn(async () => "router-key");
    const runBinary = vi.fn(async (_binaryPath: string, args: string[]) => ({
      stdout: args[1] === "status" ? "state = not running" : "ok",
      stderr: "",
    }));

    const result = await toggleServer(testOptions, { writeEnv, runBinary });

    expect(result.action).toBe("started");
    expect(writeEnv).toHaveBeenCalledTimes(1);
    expect(runBinary.mock.calls).toEqual([
      ["/tmp/qms", ["service", "status"]],
      ["/tmp/qms", ["service", "install"]],
      ["/tmp/qms", ["service", "status"]],
      ["/tmp/qms", ["service", "start"]],
      ["/tmp/qms", ["doctor"]],
    ]);
  });

  it("toggle stops the LaunchAgent when status reports running", async () => {
    const writeEnv = vi.fn(async () => "router-key");
    const runBinary = vi.fn(async (_binaryPath: string, args: string[]) => ({
      stdout: args[1] === "status" ? "state = running" : "ok",
      stderr: "",
    }));

    const result = await toggleServer(testOptions, { writeEnv, runBinary });

    expect(result.action).toBe("stopped");
    expect(writeEnv).not.toHaveBeenCalled();
    expect(runBinary.mock.calls).toEqual([
      ["/tmp/qms", ["service", "status"]],
      ["/tmp/qms", ["service", "stop"]],
    ]);
  });

  it("toggle starts the LaunchAgent when status lookup fails", async () => {
    const writeEnv = vi.fn(async () => "router-key");
    const runBinary = vi.fn(async (_binaryPath: string, args: string[]) => {
      if (args[1] === "status") {
        throw new Error("service status failed");
      }
      return { stdout: "ok", stderr: "" };
    });

    const result = await toggleServer(testOptions, { writeEnv, runBinary });

    expect(result.action).toBe("started");
    expect(writeEnv).toHaveBeenCalledTimes(1);
    expect(runBinary.mock.calls).toEqual([
      ["/tmp/qms", ["service", "status"]],
      ["/tmp/qms", ["service", "install"]],
      ["/tmp/qms", ["service", "status"]],
      ["/tmp/qms", ["service", "start"]],
      ["/tmp/qms", ["doctor"]],
    ]);
  });
});
