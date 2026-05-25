import { Action, ActionPanel, Detail, Icon, openExtensionPreferences, showToast, Toast } from "@raycast/api";
import { useEffect, useMemo, useState } from "react";
import { DEFAULT_ENV_PATH, readEnvValues } from "./lib/env";
import { getSwitches, healthz, routerBaseUrl } from "./lib/router";
import { serviceStatus, toggleServer } from "./lib/service";
import { preferences } from "./preferences";

type StatusState =
  | { status: "loading" }
  | {
      status: "ready";
      routerHealthy: boolean;
      active: string;
      serviceOutput: string;
      routerKeyPresent: boolean;
    }
  | { status: "error"; message: string };

export default function Command() {
  const prefs = preferences();
  const [state, setState] = useState<StatusState>({ status: "loading" });
  const baseUrl = useMemo(() => routerBaseUrl(prefs.listenAddr), [prefs.listenAddr]);

  async function load() {
    try {
      setState({ status: "loading" });
      const env = await readEnvValues(DEFAULT_ENV_PATH);
      const routerKey = env.QMS_ROUTER_API_KEY ?? "";
      const [routerHealthy, switches, service] = await Promise.all([
        healthz(baseUrl),
        getSwitches(baseUrl, routerKey).catch(() => undefined),
        serviceStatus(prefs.binaryPath).catch((error) => ({
          stdout: "",
          stderr: error instanceof Error ? error.message : String(error),
        })),
      ]);
      const active = switches?.active?.model
        ? `${switches.active.model} (${switches.active.effort}, ${switches.active.service_tier})`
        : "No active switch";
      setState({
        status: "ready",
        routerHealthy,
        active,
        serviceOutput: [service.stdout, service.stderr].filter(Boolean).join("\n").trim() || "No service output",
        routerKeyPresent: routerKey.length > 0,
      });
    } catch (error) {
      setState({ status: "error", message: error instanceof Error ? error.message : String(error) });
    }
  }

  useEffect(() => {
    void load();
  }, [baseUrl]);

  if (state.status === "loading") {
    return <Detail isLoading markdown="# Checking Codex model switch status" />;
  }

  if (state.status === "error") {
    return <Detail markdown={`# Status unavailable\n\n${state.message}`} actions={<StatusActions onRefresh={load} />} />;
  }

  const markdown = [
    "# Codex Model Switch Status",
    "",
    `- Router: ${state.routerHealthy ? "healthy" : "unreachable"}`,
    `- Active: ${state.active}`,
    `- Router key: ${state.routerKeyPresent ? "present" : "missing"}`,
    `- Router URL: ${baseUrl}`,
    `- Env file: ${DEFAULT_ENV_PATH}`,
    `- Binary: ${prefs.binaryPath}`,
    "",
    "## LaunchAgent",
    "",
    "```text",
    state.serviceOutput,
    "```",
  ].join("\n");

  return <Detail markdown={markdown} actions={<StatusActions onRefresh={load} />} />;
}

function StatusActions(props: { onRefresh: () => Promise<void> }) {
  const prefs = preferences();

  return (
    <ActionPanel>
      <Action
        title="Toggle LLM Server"
        icon={Icon.Power}
        onAction={async () => {
          const toast = await showToast({ style: Toast.Style.Animated, title: "Toggling Codex model router" });
          try {
            const result = await toggleServer(prefs);
            toast.style = result.action === "started" ? Toast.Style.Success : Toast.Style.Failure;
            toast.title = result.action === "started" ? "Router running" : "Router stopped";
            await props.onRefresh();
          } catch (error) {
            toast.style = Toast.Style.Failure;
            toast.title = "Toggle failed";
            toast.message = error instanceof Error ? error.message : String(error);
          }
        }}
      />
      <Action title="Refresh" icon={Icon.ArrowClockwise} onAction={props.onRefresh} />
      <Action title="Open Extension Preferences" icon={Icon.Gear} onAction={openExtensionPreferences} />
    </ActionPanel>
  );
}
