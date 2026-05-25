import { getPreferenceValues } from "@raycast/api";
import { PreferencesLike } from "./lib/types";
import { normalizeSwitches } from "./lib/switchDefaults";

export type ExtensionPreferences = PreferencesLike & {
  binaryPath: string;
};

export function preferences(): ExtensionPreferences {
  const prefs = getPreferenceValues<ExtensionPreferences>();
  return { ...prefs, switches: normalizeSwitches(prefs.switches) };
}
