import { describe, expect, it } from "vitest";
import { DEFAULT_SWITCHES, normalizeSwitches } from "../src/lib/switchDefaults";

const legacyDefaultSwitches = [
  "/m" + "sl=gpt-5.3-codex:medium:none",
  "/m" + "sm=gpt-5.5:medium:fast",
  "/m" + "sh=gpt-5.5:high:standard",
  "/m" + "sxh=gpt-5.5:xhigh:standard",
].join(",");

describe("switch default migration", () => {
  it("uses the easy-to-read default switch names", () => {
    expect(DEFAULT_SWITCHES).toBe(
      "/light=gpt-5.3-codex:medium:none,/medium=gpt-5.5:medium:fast,/high=gpt-5.5:high:standard,/extra-high=gpt-5.5:xhigh:standard",
    );
  });

  it("migrates only the exact legacy default switch list", () => {
    expect(normalizeSwitches(legacyDefaultSwitches)).toBe(DEFAULT_SWITCHES);
    expect(normalizeSwitches("  " + legacyDefaultSwitches + "  ")).toBe(DEFAULT_SWITCHES);
    expect(normalizeSwitches("/custom=gpt-5.5:high:standard")).toBe("/custom=gpt-5.5:high:standard");
  });
});
