import { getPreferenceValues } from "@raycast/api";
import { PreferencesLike } from "./lib/types";

export type ExtensionPreferences = PreferencesLike & {
  binaryPath: string;
};

export function preferences(): ExtensionPreferences {
  return getPreferenceValues<ExtensionPreferences>();
}
