import { describe, expect, it } from "vitest";
import { formatEnvFile, parseEnvFile, toEnvValues } from "../src/lib/env";

describe("env file helpers", () => {
  it("parses comments, blank lines, and quoted values", () => {
    const parsed = parseEnvFile(`
# local router settings
QMS_LISTEN_ADDR=127.0.0.1:8321
QMS_UPSTREAM_API_KEY="secret value"
EMPTY=
`);

    expect(parsed).toEqual({
      QMS_LISTEN_ADDR: "127.0.0.1:8321",
      QMS_UPSTREAM_API_KEY: "secret value",
      EMPTY: "",
    });
  });

  it("preserves router key while applying Raycast upstream preferences", () => {
    const values = toEnvValues(
      {
        QMS_ROUTER_API_KEY: "existing-router-key",
        QMS_UPSTREAM_API_KEY: "old-upstream-key",
      },
      {
        listenAddr: "127.0.0.1:9000",
        upstreamBaseUrl: "https://llm.example.test/v1/",
        upstreamApiKey: "new-upstream-key",
        switches: "/mini=gpt-5.4-mini:low:fast",
      },
      () => "generated-router-key",
    );

    expect(values.QMS_ROUTER_API_KEY).toBe("existing-router-key");
    expect(values.QMS_LISTEN_ADDR).toBe("127.0.0.1:9000");
    expect(values.QMS_UPSTREAM_BASE_URL).toBe("https://llm.example.test/v1");
    expect(values.QMS_UPSTREAM_API_KEY).toBe("new-upstream-key");
    expect(values.QMS_SWITCHES).toBe("/mini=gpt-5.4-mini:low:fast");
    expect(values.QMS_VIRTUAL_MODEL).toBe("codex-quick-model-switch");
  });

  it("requires a non-empty upstream key", () => {
    expect(() =>
      toEnvValues(
        {},
        {
          listenAddr: "127.0.0.1:8321",
          upstreamBaseUrl: "http://localhost:8317/v1",
          upstreamApiKey: "",
          switches: "/msm=gpt-5.5:medium:fast",
        },
        () => "generated-router-key",
      ),
    ).toThrow("Upstream API Key is required");
  });

  it("generates router key while writing required upstream key", () => {
    const values = toEnvValues(
      {},
      {
        listenAddr: "127.0.0.1:8321",
        upstreamBaseUrl: "http://localhost:8317/v1",
        upstreamApiKey: "upstream-key",
        switches: "/msm=gpt-5.5:medium:fast",
      },
      () => "generated-router-key",
    );

    expect(values.QMS_ROUTER_API_KEY).toBe("generated-router-key");
    expect(values.QMS_UPSTREAM_API_KEY).toBe("upstream-key");
  });

  it("formats env values in a stable order", () => {
    const text = formatEnvFile({
      QMS_SWITCHES: "/msm=gpt-5.5:medium:fast",
      QMS_ROUTER_API_KEY: "router-key",
      QMS_UPSTREAM_API_KEY: "upstream-key",
      QMS_LISTEN_ADDR: "127.0.0.1:8321",
      QMS_UPSTREAM_BASE_URL: "http://localhost:8317/v1",
      QMS_VIRTUAL_MODEL: "codex-quick-model-switch",
    });

    expect(text).toBe(
      [
        "QMS_LISTEN_ADDR=127.0.0.1:8321",
        "QMS_ROUTER_API_KEY=router-key",
        "QMS_UPSTREAM_BASE_URL=http://localhost:8317/v1",
        "QMS_UPSTREAM_API_KEY=upstream-key",
        "QMS_VIRTUAL_MODEL=codex-quick-model-switch",
        "QMS_SWITCHES=/msm=gpt-5.5:medium:fast",
        "",
      ].join("\n"),
    );
  });
});
