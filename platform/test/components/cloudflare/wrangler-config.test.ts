import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import type { Unstable_RawConfig as RawConfig } from "wrangler";
import { VisibleError } from "../../../src/components/error";
import {
  createWranglerConfig,
  type WranglerLink,
} from "../../../src/components/cloudflare/helpers/wrangler";
import {
  composeWranglerConfig,
  LINK_NAME_PLACEHOLDER,
  replaceLinkName,
  type DeepInput,
  validateLinkScopedConfig,
  type WranglerConfigSource,
} from "../../../src/components/cloudflare/helpers/wrangler-config";

const PLACEHOLDER = LINK_NAME_PLACEHOLDER;

function config(value: unknown) {
  return value as Partial<RawConfig>;
}

function source(owner: string, value: unknown): WranglerConfigSource {
  return { owner, config: config(value) };
}

function deepFreeze<T>(value: T): T {
  if (value === null || typeof value !== "object") return value;
  if (Object.isFrozen(value)) return value;

  Object.freeze(value);
  for (const child of Object.values(value as Record<string, unknown>)) {
    deepFreeze(child);
  }
  return value;
}

function expectConflict(
  sources: readonly WranglerConfigSource[],
  path: string,
  firstOwner: string,
  secondOwner: string,
) {
  const compose = () => composeWranglerConfig(sources);
  expect(compose).toThrow(VisibleError);
  expect(compose).toThrow(
    new RegExp(
      `${escapeRegExp(path)}.*${escapeRegExp(firstOwner)}.*${escapeRegExp(
        secondOwner,
      )}`,
    ),
  );
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

describe("Wrangler config baseline", () => {
  function wranglerLink(
    name: string,
    config: Partial<RawConfig>,
    properties: Record<string, unknown> = {},
  ): WranglerLink {
    return {
      name,
      properties,
      include: [{ type: "cloudflare.dev", config }],
    };
  }

  it("composes a new dev section without a kind-specific switch", () => {
    const result = createWranglerConfig({
      appName: "app",
      appStage: "dev",
      name: "Site",
      compatibility: { date: "2025-01-01", flags: [] },
      links: [
        wranglerLink("Custom", {
          custom_section: {
            [PLACEHOLDER]: { enabled: true },
          },
        } as Partial<RawConfig>),
      ],
    });

    expect((result as Record<string, unknown>).custom_section).toEqual({
      Custom: { enabled: true },
    });
  });

  it("does not scope-reject or replace placeholders in non-link sources", () => {
    const result = composeWranglerConfig([
      {
        owner: "framework",
        linkScoped: false,
        config: config({
          vars: {
            [PLACEHOLDER]: "framework-value",
            DEBUG: PLACEHOLDER,
          },
        }),
      },
    ]);

    expect(result).toEqual({
      vars: {
        [PLACEHOLDER]: "framework-value",
        DEBUG: PLACEHOLDER,
      },
    });
  });

  it("uses Wrangler ratelimits and retains Durable Object dev fragments", () => {
    const result = createWranglerConfig({
      appName: "app",
      appStage: "dev",
      name: "Site",
      compatibility: { date: "2025-01-01", flags: [] },
      links: [
        wranglerLink("Limiter", {
          ratelimits: [
            {
              name: PLACEHOLDER,
              namespace_id: "1001",
              simple: { limit: 10, period: 60 },
            },
          ],
        } as Partial<RawConfig>),
        wranglerLink("Counter", {
          durable_objects: {
            bindings: [{ name: PLACEHOLDER, class_name: "Counter" }],
          },
        } as Partial<RawConfig>),
      ],
    });

    expect(result.ratelimits).toEqual([
      {
        name: "Limiter",
        namespace_id: "1001",
        simple: { limit: 10, period: 60 },
      },
    ]);
    expect(result.durable_objects).toEqual({
      bindings: [{ name: "Counter", class_name: "Counter" }],
    });
    expect(result).not.toHaveProperty("rate_limits");
  });

  it("reports owner conflicts across framework, base, and link sources", () => {
    expect(() =>
      createWranglerConfig({
        appName: "app",
        appStage: "dev",
        name: "Site",
        frameworkConfig: { vars: { SST_RESOURCE_App: "framework" } },
        compatibility: { date: "2025-01-01", flags: [] },
      }),
    ).toThrow(/vars\.SST_RESOURCE_App.*framework.*sst/);
  });

  it("keeps ordinary links as resource vars and ignores production bindings", () => {
    const result = createWranglerConfig({
      appName: "app",
      appStage: "dev",
      name: "Site",
      compatibility: { date: "2025-01-01", flags: [] },
      links: [
        {
          name: "External",
          properties: { url: "https://example.com" },
          include: [],
        },
        {
          name: "ProductionOnly",
          properties: { value: "ignored" },
          include: [
            {
              type: "cloudflare.binding",
              binding: { type: "custom" } as any,
            },
          ],
        },
      ],
    });

    expect(result.vars).toMatchObject({
      SST_RESOURCE_External: JSON.stringify({ url: "https://example.com" }),
    });
    expect(result.vars).not.toHaveProperty("SST_RESOURCE_ProductionOnly");
  });

  it("uses VisibleError when a link dev projection is missing", () => {
    expect(() =>
      createWranglerConfig({
        appName: "app",
        appStage: "dev",
        name: "Site",
        compatibility: { date: "2025-01-01", flags: [] },
        links: [
          {
            name: "Broken",
            properties: {},
            include: [{ type: "cloudflare.dev" }],
          },
        ],
      }),
    ).toThrow(VisibleError);
  });

  it("uses the exact Wrangler type dependency without importing a runtime value", () => {
    const packageJson = JSON.parse(
      readFileSync(new URL("../../../package.json", import.meta.url), "utf8"),
    ) as {
      dependencies?: Record<string, string>;
      devDependencies?: Record<string, string>;
    };
    const sourceText = readFileSync(
      new URL(
        "../../../src/components/cloudflare/helpers/wrangler-config.ts",
        import.meta.url,
      ),
      "utf8",
    );

    expect(packageJson.devDependencies?.wrangler).toBe("4.83.0");
    expect(packageJson.dependencies?.wrangler).toBeUndefined();
    expect(sourceText).toContain(
      'import type { Unstable_RawConfig as RawConfig } from "wrangler";',
    );
    expect(sourceText).not.toMatch(/import\s+\{[^}]*Unstable_RawConfig/);
  });

  it("keeps DeepInput recursive while preserving functions", () => {
    type InputShape = {
      values: readonly { value: string }[];
      transform: (value: string) => string;
    };
    const value: DeepInput<InputShape> = {
      values: [{ value: "ok" }],
      transform: (input) => input,
    };

    expect(value.values?.[0]?.value).toBe("ok");
    expect(value.transform?.("ok")).toBe("ok");
  });

  it.each([
    "name",
    "main",
    "account_id",
    "compatibility_date",
    "compatibility_flags",
    "workers_dev",
    "route",
    "routes",
    "triggers",
    "migrations",
    "exports",
    "assets",
    "observability",
    "placement",
    "limits",
    "tail_consumers",
    "streaming_tail_consumers",
  ])("rejects link-scoped %s", (path) => {
    expect(() =>
      validateLinkScopedConfig("owner", config({ [path]: {} })),
    ).toThrow(new RegExp(`owner.*${escapeRegExp(path)}`));
  });

  it("rejects queue consumers but allows queue producers", () => {
    expect(() =>
      validateLinkScopedConfig("owner", config({ queues: { consumers: [] } })),
    ).toThrow(/owner.*queues\.consumers/);

    expect(() =>
      validateLinkScopedConfig(
        "owner",
        config({
          queues: {
            producers: [{ binding: PLACEHOLDER, queue: "events" }],
          },
        }),
      ),
    ).not.toThrow();
  });

  it.each([
    "name",
    "main",
    "account_id",
    "compatibility_date",
    "compatibility_flags",
    "workers_dev",
    "route",
    "routes",
    "triggers",
    "migrations",
    "exports",
    "assets",
    "observability",
    "placement",
    "limits",
    "tail_consumers",
    "streaming_tail_consumers",
    "queues.consumers",
  ])("rejects forbidden lifecycle config nested under env for %s", (path) => {
    const [section, child] = path.split(".");
    const nested = child ? { [section!]: { [child]: [] } } : { [section!]: {} };

    expect(() =>
      validateLinkScopedConfig("owner", config({ env: { prod: nested } })),
    ).toThrow(/owner.*env/);
  });

  it("rejects env entirely for link-scoped config", () => {
    expect(() =>
      validateLinkScopedConfig("owner", config({ env: {} })),
    ).toThrow(/owner.*env/);
  });

  it("uses VisibleError for link-scoped validation failures", () => {
    expect(() =>
      validateLinkScopedConfig("owner", config({ env: {} })),
    ).toThrow(VisibleError);
  });

  it("replaces binding and name identity placeholders", () => {
    const result = replaceLinkName(
      "owner",
      config({
        queues: {
          producers: [{ binding: PLACEHOLDER, queue: "events" }],
        },
        ratelimits: [
          {
            name: PLACEHOLDER,
            namespace_id: "namespace-id",
            simple: { limit: 100, period: 60 },
          },
        ],
      }),
      "Events",
    );

    expect(result).toEqual({
      queues: {
        producers: [{ binding: "Events", queue: "events" }],
      },
      ratelimits: [
        {
          name: "Events",
          namespace_id: "namespace-id",
          simple: { limit: 100, period: 60 },
        },
      ],
    });
  });

  it.each([
    ["binding", "queues", "producers"],
    ["name", "ratelimits", undefined],
  ])("rejects partial %s identity placeholders", (identity, section, child) => {
    const value = child
      ? { [section]: { [child]: [{ [identity]: "prefix-__sst_link_name__" }] } }
      : { [section]: [{ [identity]: "prefix-__sst_link_name__" }] };
    expect(() => replaceLinkName("owner", config(value), "Events")).toThrow(
      VisibleError,
    );
  });

  it("rejects a placeholder identity colliding with an explicit final identity", () => {
    expect(() =>
      replaceLinkName(
        "owner",
        config({
          queues: {
            producers: [
              { binding: PLACEHOLDER, queue: "placeholder" },
              { binding: "Events", queue: "explicit" },
            ],
          },
        }),
        "Events",
      ),
    ).toThrow(/queues\.producers.*owner.*owner/);
  });

  it("rejects a name identity collision after placeholder replacement", () => {
    expect(() =>
      replaceLinkName(
        "owner",
        config({
          ratelimits: [
            { name: PLACEHOLDER, namespace_id: "placeholder" },
            { name: "Events", namespace_id: "explicit" },
          ],
        }),
        "Events",
      ),
    ).toThrow(/ratelimits.*owner.*owner/);
  });

  it("rejects identity duplicates after placeholder replacement in a map", () => {
    expect(() =>
      replaceLinkName(
        "owner",
        config({
          custom_section: {
            [PLACEHOLDER]: { binding: PLACEHOLDER },
            Events: { binding: "Events" },
          },
        }),
        "Events",
      ),
    ).toThrow(/custom_section\.Events.*owner.*owner/);
  });

  it("does not mutate nested maps while composing", () => {
    const input = deepFreeze({
      vars: {
        LINKED: {
          nested: {
            value: "original",
          },
        },
      },
    });
    const snapshot = structuredClone(input);

    const result = composeWranglerConfig([source("owner", input)]);

    expect(input).toEqual(snapshot);
    expect(result).toEqual(input);
    expect(result.vars).not.toBe(input.vars);
    expect(result.vars?.LINKED).not.toBe(input.vars.LINKED);
  });

  it("rejects duplicate vars keys when values are arrays", () => {
    expectConflict(
      [
        source("first-owner", { vars: { SHARED: ["first"] } }),
        source("second-owner", { vars: { SHARED: ["second"] } }),
      ],
      "vars.SHARED",
      "first-owner",
      "second-owner",
    );
  });

  it("merges distinct vars keys when values are arrays", () => {
    expect(
      composeWranglerConfig([
        source("first-owner", { vars: { FIRST: ["one"] } }),
        source("second-owner", { vars: { SECOND: ["two"] } }),
      ]),
    ).toEqual({ vars: { FIRST: ["one"], SECOND: ["two"] } });
  });

  it("does not mutate nested maps during placeholder replacement", () => {
    const input = deepFreeze({
      vars: {
        [PLACEHOLDER]: {
          nested: {
            value: "original",
          },
        },
      },
    });
    const snapshot = structuredClone(input);

    const result = replaceLinkName("owner", config(input), "LinkedResource");

    expect(input).toEqual(snapshot);
    expect(result).toEqual({
      vars: {
        LinkedResource: {
          nested: {
            value: "original",
          },
        },
      },
    });
    expect(result.vars).not.toBe(input.vars);
    expect(result.vars?.LinkedResource).not.toBe(input.vars[PLACEHOLDER]);
  });

  it("does not mutate identity arrays during placeholder replacement", () => {
    const input = deepFreeze({
      queues: {
        producers: [
          {
            binding: PLACEHOLDER,
            queue: "events",
          },
        ],
      },
    });
    const snapshot = structuredClone(input);

    const result = replaceLinkName("owner", config(input), "Events");

    expect(input).toEqual(snapshot);
    expect(result.queues?.producers).toEqual([
      {
        binding: "Events",
        queue: "events",
      },
    ]);
    expect(result.queues?.producers).not.toBe(input.queues.producers);
    expect(result.queues?.producers?.[0]).not.toBe(input.queues.producers[0]);
  });

  it("replaces a complete object map key", () => {
    const result = replaceLinkName(
      "owner",
      config({ vars: { [PLACEHOLDER]: "linked-value" } }),
      "LinkedResource",
    );

    expect(result).toEqual({ vars: { LinkedResource: "linked-value" } });
  });

  it("replaces a complete vars map key when its value is an array", () => {
    const result = replaceLinkName(
      "owner",
      config({ vars: { [PLACEHOLDER]: ["linked-value"] } }),
      "LinkedResource",
    );

    expect(result).toEqual({ vars: { LinkedResource: ["linked-value"] } });
  });

  it("rejects a vars map key collision after replacement when values are arrays", () => {
    expect(() =>
      replaceLinkName(
        "owner",
        config({
          vars: {
            [PLACEHOLDER]: ["placeholder-value"],
            LinkedResource: ["existing-value"],
          },
        }),
        "LinkedResource",
      ),
    ).toThrow(VisibleError);
  });

  it("does not mutate vars arrays while replacing a map key", () => {
    const input = deepFreeze({
      vars: {
        [PLACEHOLDER]: ["linked-value"],
      },
    });
    const snapshot = structuredClone(input);

    const result = replaceLinkName("owner", config(input), "LinkedResource");

    expect(input).toEqual(snapshot);
    expect(result).toEqual({ vars: { LinkedResource: ["linked-value"] } });
    expect(result.vars).not.toBe(input.vars);
    expect(result.vars?.LinkedResource).not.toBe(input.vars[PLACEHOLDER]);
  });

  it("rejects a placeholder embedded in an object map key", () => {
    expect(() =>
      replaceLinkName(
        "owner",
        config({ vars: { [`prefix-${PLACEHOLDER}`]: "value" } }),
        "LinkedResource",
      ),
    ).toThrow(/owner.*vars\.prefix/);
  });

  it("rejects a map key collision after replacement", () => {
    expect(() =>
      replaceLinkName(
        "owner",
        config({
          vars: {
            [PLACEHOLDER]: "placeholder-value",
            LinkedResource: "existing-value",
          },
        }),
        "LinkedResource",
      ),
    ).toThrow(/vars\.LinkedResource.*owner.*owner/);
  });

  it("rejects placeholders in ordinary values", () => {
    expect(() =>
      replaceLinkName(
        "owner",
        config({ vars: { DEBUG: PLACEHOLDER } }),
        "LinkedResource",
      ),
    ).toThrow(VisibleError);

    expect(() =>
      replaceLinkName(
        "owner",
        config({
          queues: {
            producers: [{ binding: PLACEHOLDER, queue: PLACEHOLDER }],
          },
        }),
        "LinkedResource",
      ),
    ).toThrow(/owner.*queues\.producers\[0\]\.queue/);
  });

  it("appends recognized arrays in source order", () => {
    const result = composeWranglerConfig([
      source("first-owner", {
        queues: {
          producers: [{ binding: "first", queue: "first-queue" }],
        },
        ratelimits: [
          {
            name: "first-limit",
            namespace_id: "first-namespace",
            simple: { limit: 10, period: 10 },
          },
        ],
      }),
      source("second-owner", {
        queues: {
          producers: [{ binding: "second", queue: "second-queue" }],
        },
        ratelimits: [
          {
            name: "second-limit",
            namespace_id: "second-namespace",
            simple: { limit: 20, period: 60 },
          },
        ],
      }),
    ]);

    expect(result).toEqual({
      queues: {
        producers: [
          { binding: "first", queue: "first-queue" },
          { binding: "second", queue: "second-queue" },
        ],
      },
      ratelimits: [
        {
          name: "first-limit",
          namespace_id: "first-namespace",
          simple: { limit: 10, period: 10 },
        },
        {
          name: "second-limit",
          namespace_id: "second-namespace",
          simple: { limit: 20, period: 60 },
        },
      ],
    });
  });

  it("keeps queues.producers as identity arrays when composing sources", () => {
    expect(
      composeWranglerConfig([
        source("first-owner", {
          queues: { producers: [{ binding: "first", queue: "first-queue" }] },
        }),
        source("second-owner", {
          queues: {
            producers: [{ binding: "second", queue: "second-queue" }],
          },
        }),
      ]),
    ).toEqual({
      queues: {
        producers: [
          { binding: "first", queue: "first-queue" },
          { binding: "second", queue: "second-queue" },
        ],
      },
    });
  });

  it("rejects duplicate binding identities", () => {
    expectConflict(
      [
        source("first-owner", {
          queues: { producers: [{ binding: "same", queue: "first" }] },
        }),
        source("second-owner", {
          queues: { producers: [{ binding: "same", queue: "second" }] },
        }),
      ],
      "queues.producers",
      "first-owner",
      "second-owner",
    );
  });

  it("rejects duplicate name identities", () => {
    expectConflict(
      [
        source("first-owner", {
          ratelimits: [
            {
              name: "same",
              namespace_id: "first",
              simple: { limit: 10, period: 10 },
            },
          ],
        }),
        source("second-owner", {
          ratelimits: [
            {
              name: "same",
              namespace_id: "second",
              simple: { limit: 20, period: 60 },
            },
          ],
        }),
      ],
      "ratelimits",
      "first-owner",
      "second-owner",
    );
  });

  it("recursively merges distinct object keys", () => {
    expect(
      composeWranglerConfig([
        source("first-owner", { vars: { FIRST: "one" } }),
        source("second-owner", { vars: { SECOND: "two" } }),
      ]),
    ).toEqual({ vars: { FIRST: "one", SECOND: "two" } });
  });

  it("treats an empty object as neutral during recursive merging", () => {
    expect(
      composeWranglerConfig([
        source("first-owner", { queues: {} }),
        source("second-owner", {
          queues: {
            producers: [{ binding: "second", queue: "second-queue" }],
          },
        }),
      ]),
    ).toEqual({
      queues: {
        producers: [{ binding: "second", queue: "second-queue" }],
      },
    });
  });

  it("rejects scalar conflicts with both owners", () => {
    expectConflict(
      [
        source("first-owner", { send_metrics: true }),
        source("second-owner", { send_metrics: false }),
      ],
      "send_metrics",
      "first-owner",
      "second-owner",
    );
  });

  it("rejects singleton conflicts with both owners", () => {
    expectConflict(
      [
        source("first-owner", { ai: { binding: "first", remote: true } }),
        source("second-owner", { ai: { binding: "second" } }),
      ],
      "ai",
      "first-owner",
      "second-owner",
    );
  });

  it("rejects duplicate object map keys without merging their values", () => {
    expectConflict(
      [
        source("first-owner", { vars: { SHARED: { first: true } } }),
        source("second-owner", { vars: { SHARED: { second: true } } }),
      ],
      "vars.SHARED",
      "first-owner",
      "second-owner",
    );
  });

  it("allows one owner for an array without a recognized identity", () => {
    expect(
      composeWranglerConfig([
        source("owner", { custom: [{ value: "one" }, { value: "two" }] }),
      ]),
    ).toEqual({ custom: [{ value: "one" }, { value: "two" }] });
  });

  it("rejects an unknown identity array from multiple owners", () => {
    expectConflict(
      [
        source("first-owner", { custom: [{ value: "one" }] }),
        source("second-owner", { custom: [{ value: "two" }] }),
      ],
      "custom",
      "first-owner",
      "second-owner",
    );
  });

  it.each([
    [
      "empty first",
      source("first-owner", { custom: [] }),
      source("second-owner", { custom: [{ value: "second" }] }),
    ],
    [
      "empty second",
      source("first-owner", { custom: [{ value: "first" }] }),
      source("second-owner", { custom: [] }),
    ],
    [
      "both empty",
      source("first-owner", { custom: [] }),
      source("second-owner", { custom: [] }),
    ],
  ])(
    "rejects an unidentifiable array when %s is supplied by another owner",
    (_case, first, second) => {
      expectConflict([first, second], "custom", "first-owner", "second-owner");
    },
  );
});
