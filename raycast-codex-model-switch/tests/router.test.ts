import { describe, expect, it, vi } from "vitest";
import { getSwitches, routerBaseUrl, switchModel } from "../src/lib/router";

describe("router client", () => {
  it("derives a local base URL from a listen address", () => {
    expect(routerBaseUrl("127.0.0.1:8321")).toBe("http://127.0.0.1:8321");
    expect(routerBaseUrl("0.0.0.0:8321")).toBe("http://127.0.0.1:8321");
  });

  it("fetches ordered switches with bearer authorization", async () => {
    const fetchImpl = vi.fn(async () => ({
      ok: true,
      status: 200,
      statusText: "OK",
      json: async () => ({
        switches: [
          { shortcut: "/mini", model: "gpt-5.4-mini", effort: "low", service_tier: "fast" },
          { shortcut: "/deep", model: "gpt-5.5", effort: "xhigh", service_tier: "standard" },
        ],
        active: { shortcut: "/deep", model: "gpt-5.5", effort: "xhigh", service_tier: "standard" },
      }),
    }));

    const result = await getSwitches("http://127.0.0.1:8321", "router-key", fetchImpl);

    expect(fetchImpl).toHaveBeenCalledWith("http://127.0.0.1:8321/switches", {
      headers: { Authorization: "Bearer router-key" },
    });
    expect(result.switches.map((item) => item.shortcut)).toEqual(["/mini", "/deep"]);
    expect(result.active?.shortcut).toBe("/deep");
  });

  it("posts selected shortcut to the switch endpoint", async () => {
    const fetchImpl = vi.fn(async () => ({
      ok: true,
      status: 204,
      statusText: "No Content",
      text: async () => "",
    }));

    await switchModel("http://127.0.0.1:8321", "router-key", "/msm", fetchImpl);

    expect(fetchImpl).toHaveBeenCalledWith("http://127.0.0.1:8321/switch", {
      method: "POST",
      headers: {
        Authorization: "Bearer router-key",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ shortcut: "/msm" }),
    });
  });
});
