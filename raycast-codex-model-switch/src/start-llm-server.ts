import { openExtensionPreferences, showToast, Toast } from "@raycast/api";
import { startServer } from "./lib/service";
import { preferences } from "./preferences";

export default async function Command() {
  const prefs = preferences();
  const toast = await showToast({
    style: Toast.Style.Animated,
    title: "Starting Codex model router",
    message: prefs.upstreamBaseUrl,
  });

  try {
    await startServer(prefs);
    toast.style = Toast.Style.Success;
    toast.title = "Codex model router running";
    toast.message = "LaunchAgent installed and health check passed";
  } catch (error) {
    toast.style = Toast.Style.Failure;
    toast.title = "Could not start router";
    toast.message = error instanceof Error ? error.message : String(error);
    toast.primaryAction = {
      title: "Open Preferences",
      onAction: () => {
        void openExtensionPreferences();
      },
    };
  }
}
