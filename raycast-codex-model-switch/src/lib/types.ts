export type PreferencesLike = {
  listenAddr: string;
  upstreamBaseUrl: string;
  upstreamApiKey?: string;
  switches: string;
  binaryPath?: string;
};

export type SwitchConfig = {
  shortcut: string;
  model: string;
  effort: string;
  service_tier: string;
};

export type SwitchesResponse = {
  switches: SwitchConfig[];
  active?: SwitchConfig;
};

export type CommandResult = {
  stdout: string;
  stderr: string;
};
