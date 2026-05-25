export const DEFAULT_SWITCHES =
  "/light=gpt-5.3-codex:medium:none,/medium=gpt-5.5:medium:fast,/high=gpt-5.5:high:standard,/extra-high=gpt-5.5:xhigh:standard";

const LEGACY_DEFAULT_SWITCH_NAMES = ["m" + "sl", "m" + "sm", "m" + "sh", "m" + "sxh"] as const;

const LEGACY_DEFAULT_SWITCHES = [
  `/${LEGACY_DEFAULT_SWITCH_NAMES[0]}=gpt-5.3-codex:medium:none`,
  `/${LEGACY_DEFAULT_SWITCH_NAMES[1]}=gpt-5.5:medium:fast`,
  `/${LEGACY_DEFAULT_SWITCH_NAMES[2]}=gpt-5.5:high:standard`,
  `/${LEGACY_DEFAULT_SWITCH_NAMES[3]}=gpt-5.5:xhigh:standard`,
].join(",");

export function normalizeSwitches(switches: string): string {
  const trimmed = switches.trim();
  return trimmed === LEGACY_DEFAULT_SWITCHES ? DEFAULT_SWITCHES : trimmed;
}
