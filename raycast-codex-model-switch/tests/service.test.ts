import { describe, expect, it, vi } from "vitest";
import { startServer } from "../src/lib/service";

describe("service orchestration", () => {
  it("writes env then installs, starts when stopped, and doctors the LaunchAgent", async () => {
    const writeEnv = vi.fn(async () => "router-key");
    const runBinary = vi.fn(async (_binaryPath: string, args: string[]) => ({
      stdout: args[1] === "status" ? "state = not running" : "ok",
      stderr: "",
    }));

    const result = await startServer(
      {
        binaryPath: "/tmp/qms",
        envPath: "/tmp/qms.env",
        listenAddr: "127.0.0.1:8321",
        upstreamBaseUrl: "http://localhost:8317/v1",
        upstreamApiKey: "upstream-key",
        switches: "/msm=gpt-5.5:medium:fast",
      },
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
      {
        binaryPath: "/tmp/qms",
        envPath: "/tmp/qms.env",
        listenAddr: "127.0.0.1:8321",
        upstreamBaseUrl: "http://localhost:8317/v1",
        upstreamApiKey: "upstream-key",
        switches: "/msm=gpt-5.5:medium:fast",
      },
      { writeEnv, runBinary },
    );

    expect(runBinary.mock.calls).toEqual([
      ["/tmp/qms", ["service", "install"]],
      ["/tmp/qms", ["service", "status"]],
      ["/tmp/qms", ["doctor"]],
    ]);
  });
});
