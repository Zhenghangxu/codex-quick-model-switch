import { openExtensionPreferences, showToast, Toast } from "@raycast/api";
import { toggleServer } from "./lib/service";
import { preferences } from "./preferences";

export default async function Command() {
  const prefs = preferences();
  const toast = await showToast({
    style: Toast.Style.Animated,
    title: "Toggling Codex model router",
    message: prefs.upstreamBaseUrl,
  });

  try {
    const result = await toggleServer(prefs);
    toast.style = result.action === "started" ? Toast.Style.Success : Toast.Style.Failure;
    toast.title = result.action === "started" ? "Codex model router running" : "Codex model router stopped";
    toast.message =
      result.action === "started" ? "LaunchAgent installed and health check passed" : "LaunchAgent stopped";
  } catch (error) {
    toast.style = Toast.Style.Failure;
    toast.title = "Could not toggle router";
    toast.message = error instanceof Error ? error.message : String(error);
    toast.primaryAction = {
      title: "Open Preferences",
      onAction: () => {
        void openExtensionPreferences();
      },
    };
  }
}
