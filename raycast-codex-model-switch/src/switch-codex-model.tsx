import { Action, ActionPanel, Detail, Icon, List, openExtensionPreferences, showToast, Toast } from "@raycast/api";
import { useEffect, useMemo, useState } from "react";
import { readEnvValues, DEFAULT_ENV_PATH } from "./lib/env";
import { getSwitches, routerBaseUrl, switchModel } from "./lib/router";
import { SwitchConfig, SwitchesResponse } from "./lib/types";
import { preferences } from "./preferences";

type LoadState =
  | { status: "loading" }
  | { status: "ready"; data: SwitchesResponse; routerKey: string; baseUrl: string }
  | { status: "error"; message: string };

export default function Command() {
  const prefs = preferences();
  const [state, setState] = useState<LoadState>({ status: "loading" });
  const baseUrl = useMemo(() => routerBaseUrl(prefs.listenAddr), [prefs.listenAddr]);

  useEffect(() => {
    async function load() {
      try {
        const env = await readEnvValues(DEFAULT_ENV_PATH);
        const routerKey = env.QMS_ROUTER_API_KEY ?? "";
        const data = await getSwitches(baseUrl, routerKey);
        setState({ status: "ready", data, routerKey, baseUrl });
      } catch (error) {
        setState({ status: "error", message: error instanceof Error ? error.message : String(error) });
      }
    }
    void load();
  }, [baseUrl]);

  if (state.status === "loading") {
    return <List isLoading searchBarPlaceholder="Loading configured model switches..." />;
  }

  if (state.status === "error") {
    return (
      <Detail
        markdown={`# Router unavailable\n\n${state.message}\n\nRun **Start LLM Server** first, then try again.`}
        actions={
          <ActionPanel>
            <Action title="Open Extension Preferences" icon={Icon.Gear} onAction={openExtensionPreferences} />
          </ActionPanel>
        }
      />
    );
  }

  return (
    <List searchBarPlaceholder="Choose a Codex model switch...">
      {state.data.switches.map((item) => (
        <SwitchItem
          key={item.shortcut}
          item={item}
          active={state.data.active?.shortcut === item.shortcut}
          baseUrl={state.baseUrl}
          routerKey={state.routerKey}
          onSwitched={(active) =>
            setState((current) =>
              current.status === "ready" ? { ...current, data: { ...current.data, active } } : current,
            )
          }
        />
      ))}
    </List>
  );
}

function SwitchItem(props: {
  item: SwitchConfig;
  active: boolean;
  baseUrl: string;
  routerKey: string;
  onSwitched: (active: SwitchConfig) => void;
}) {
  const { item, active, baseUrl, routerKey, onSwitched } = props;
  const accessories = [
    { text: item.effort },
    ...(item.service_tier === "none" ? [] : [{ tag: item.service_tier }]),
    ...(active ? [{ icon: Icon.CheckCircle, tooltip: "Active" }] : []),
  ];

  return (
    <List.Item
      title={`${item.model} (${item.effort})`}
      subtitle={item.shortcut}
      icon={active ? Icon.CheckCircle : Icon.Circle}
      accessories={accessories}
      actions={
        <ActionPanel>
          <Action
            title="Switch Model"
            icon={Icon.ArrowRight}
            onAction={async () => {
              const toast = await showToast({ style: Toast.Style.Animated, title: "Switching Codex model" });
              try {
                await switchModel(baseUrl, routerKey, item.shortcut);
                onSwitched(item);
                toast.style = Toast.Style.Success;
                toast.title = "Codex model switched";
                toast.message = `${item.model} (${item.effort})`;
              } catch (error) {
                toast.style = Toast.Style.Failure;
                toast.title = "Switch failed";
                toast.message = error instanceof Error ? error.message : String(error);
              }
            }}
          />
        </ActionPanel>
      }
    />
  );
}
